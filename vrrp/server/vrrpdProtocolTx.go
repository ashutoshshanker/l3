package vrrpServer

import (
	"encoding/binary"
	_ "errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

/*
Octet Offset--> 0                   1                   2                   3
 |		0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 |		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 V		|                    IPv4 Fields or IPv6 Fields                 |
		...                                                             ...
		|                                                               |
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 0		|Version| Type  | Virtual Rtr ID|   Priority    |Count IPvX Addr|
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 4		|(rsvd) |     Max Adver Int     |          Checksum             |
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 8		|                                                               |
		+                                                               +
12		|                       IPvX Address(es)                        |
		+                                                               +
..		+                                                               +
		+                                                               +
		+                                                               +
		|                                                               |
		+                                                               +
		|                                                               |
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
func (svr *VrrpServer) VrrpEncodeHeader(hdr VrrpPktHeader) ([]byte, uint16) {
	pktLen := VRRP_HEADER_SIZE_EXCLUDING_IPVX + (hdr.CountIPv4Addr * 4)
	if pktLen < VRRP_HEADER_MIN_SIZE {
		pktLen = VRRP_HEADER_MIN_SIZE
	}
	bytes := make([]byte, pktLen)
	bytes[0] = (hdr.Version << 4) | hdr.Type
	bytes[1] = hdr.VirtualRtrId
	bytes[2] = hdr.Priority
	bytes[3] = hdr.CountIPv4Addr
	rsvdAdver := (uint16(hdr.Rsvd) << 13) | hdr.MaxAdverInt
	binary.BigEndian.PutUint16(bytes[4:], rsvdAdver)
	binary.BigEndian.PutUint16(bytes[6:8], hdr.CheckSum)
	baseIpByte := 8
	for i := 0; i < int(hdr.CountIPv4Addr); i++ {
		copy(bytes[baseIpByte:(baseIpByte+4)], hdr.IPv4Addr[i].To4())
		baseIpByte += 4
	}
	// Create Checksum for the header and store it
	binary.BigEndian.PutUint16(bytes[6:8],
		svr.VrrpComputeChecksum(hdr.Version, bytes))
	return bytes, uint16(pktLen)
}

func (svr *VrrpServer) VrrpCreateVrrpHeader(gblInfo VrrpGlobalInfo) ([]byte, uint16) {
	// @TODO: handle v6 packets.....
	vrrpHeader := VrrpPktHeader{
		Version:       VRRP_VERSION2,
		Type:          VRRP_PKT_TYPE,
		VirtualRtrId:  uint8(gblInfo.IntfConfig.VRID),
		Priority:      uint8(gblInfo.IntfConfig.Priority),
		CountIPv4Addr: 1, // FIXME for more than 1 vip
		Rsvd:          VRRP_RSVD,
		MaxAdverInt:   uint16(gblInfo.IntfConfig.AdvertisementInterval),
		CheckSum:      VRRP_HDR_CREATE_CHECKSUM,
	}
	var ip net.IP
	//FIXME with Virtual Ip Addr.... and not IfIndex Ip Addr
	// If no virtual ip then use interface/router ip address as virtual ip
	if gblInfo.IntfConfig.VirtualIPv4Addr == "" {
		ip, _, _ = net.ParseCIDR(gblInfo.IpAddr)
	} else {
		ip = net.ParseIP(gblInfo.IntfConfig.VirtualIPv4Addr)
	}
	vrrpHeader.IPv4Addr = append(vrrpHeader.IPv4Addr, ip)
	vrrpEncHdr, hdrLen := svr.VrrpEncodeHeader(vrrpHeader)
	svr.logger.Info(fmt.Sprintln("vrrp header after enc is",
		svr.VrrpDecodeHeader(vrrpEncHdr)))
	return vrrpEncHdr, hdrLen
}

func (svr *VrrpServer) VrrpCreateSendPkt(gblInfo VrrpGlobalInfo, vrrpEncHdr []byte,
	hdrLen uint16) []byte {
	// Ethernet Layer
	srcMAC, _ := net.ParseMAC(gblInfo.IntfConfig.VirtualRouterMACAddress)
	dstMAC, _ := net.ParseMAC(VRRP_PROTOCOL_MAC)
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// IP Layer
	sip, _, _ := net.ParseCIDR(gblInfo.IpAddr)
	ipv4 := &layers.IPv4{
		Version:  uint8(4),
		IHL:      uint8(VRRP_IPV4_HEADER_MIN_SIZE),
		Protocol: layers.IPProtocol(VRRP_PROTO_ID),
		Length:   uint16(VRRP_IPV4_HEADER_MIN_SIZE + hdrLen),
		TTL:      uint8(VRRP_TTL),
		SrcIP:    sip,
		DstIP:    net.ParseIP(VRRP_GROUP_IP),
	}

	// Construct go Packet Buffer
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	gopacket.SerializeLayers(buffer, options, eth, ipv4,
		gopacket.Payload(vrrpEncHdr))
	return buffer.Bytes()
}

func (svr *VrrpServer) VrrpSendPkt(rcvdCh <-chan string /*VrrpPktChannelInfo*/) {
	svr.logger.Info("started send packet routine")
	for {
		//pktChannel := <-rcvdCh
		key := <-rcvdCh //pktChannel.key
		gblInfo, found := svr.vrrpGblInfo[key]
		if !found {
			svr.logger.Err("No Entry for " + key)
			continue
		}
		if gblInfo.pHandle == nil {
			svr.logger.Info("Invalid Pcap Handle")
			continue
		}
		vrrpEncHdr, hdrLen := svr.VrrpCreateVrrpHeader(gblInfo)
		vrrpTxPkt := svr.VrrpCreateSendPkt(gblInfo, vrrpEncHdr, hdrLen)
		svr.logger.Info(fmt.Sprintln("send pkt", vrrpTxPkt))
		err := gblInfo.pHandle.WritePacketData(vrrpTxPkt)
		if err != nil {
			svr.logger.Info(fmt.Sprintln("Sending Packet failed", err))
		}
	}
}

func (svr *VrrpServer) VrrpSendGratuitousArp(gblInfo *VrrpGlobalInfo) {
	/*
		// Set up all the layers' fields we can.
		eth := layers.Ethernet{
			SrcMAC:       iface.HardwareAddr,
			DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			EthernetType: layers.EthernetTypeARP,
		}
		arp := layers.ARP{
			AddrType:          layers.LinkTypeEthernet,
			Protocol:          layers.EthernetTypeIPv4,
			HwAddressSize:     6,
			ProtAddressSize:   4,
			Operation:         layers.ARPRequest,
			SourceHwAddress:   []byte(iface.HardwareAddr),
			SourceProtAddress: []byte(addr.IP),
			DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		}
		// Set up buffer and options for serialization.
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}
		// Send one packet for every address.
		for _, ip := range ips(addr) {
			arp.DstProtAddress = []byte(ip)
			gopacket.SerializeLayers(buf, opts, &eth, &arp)
			if err := handle.WritePacketData(buf.Bytes()); err != nil {
				return err
			}
		}
		return nil
	*/
}
