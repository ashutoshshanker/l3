package vrrpServer

import (
	"asicdServices"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"time"
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
func VrrpDecodeHeader(data []byte) *VrrpPktHeader {
	var vrrpPkt VrrpPktHeader
	vrrpPkt.Version = uint8(data[0]) >> 4
	vrrpPkt.Type = uint8(data[0]) & 0x0F
	vrrpPkt.VirtualRtrId = data[1]
	vrrpPkt.Priority = data[2]
	vrrpPkt.CountIPv4Addr = data[3]
	rsvdAdver := binary.BigEndian.Uint16(data[4:6])
	vrrpPkt.Rsvd = uint8(rsvdAdver >> 13)
	vrrpPkt.MaxAdverInt = rsvdAdver & 0x1FFF
	vrrpPkt.CheckSum = binary.BigEndian.Uint16(data[6:8])
	baseIpByte := 8
	for i := 0; i < int(vrrpPkt.CountIPv4Addr); i++ {
		vrrpPkt.IPv4Addr = append(vrrpPkt.IPv4Addr,
			data[baseIpByte:(baseIpByte+4)])
		baseIpByte += 4
	}
	return &vrrpPkt
}

func VrrpEncodeHeader(hdr VrrpPktHeader) ([]byte, uint16) {
	pktLen := VRRP_HEADER_SIZE_EXCLUDING_IPVX + (hdr.CountIPv4Addr * 4)
	pkt := make([]byte, pktLen)
	logger.Info(fmt.Sprintln("no.of bytes for vrrp tx header is", len(pkt)))
	pkt[0] = hdr.Version << 4
	pkt[0] = hdr.Type & 0x0F
	pkt[1] = hdr.VirtualRtrId
	pkt[2] = hdr.Priority
	pkt[3] = hdr.CountIPv4Addr
	rsvdAdver := (uint16(hdr.Rsvd) << 13) | hdr.MaxAdverInt
	binary.BigEndian.PutUint16(pkt[4:], rsvdAdver)
	binary.BigEndian.PutUint16(pkt[6:8], hdr.CheckSum)
	j := 0
	for i := VRRP_HEADER_SIZE_EXCLUDING_IPVX; i < int(hdr.CountIPv4Addr); i = i + 4 {
		//binary.BigEndian.PutUint32(pkt[i:(i+4)], hdr.IPv4Addr[i].String())
		copy(pkt[i:(i+4)], hdr.IPv4Addr[j])
		j++
	}
	return pkt, uint16(pktLen)
}

func VrrpComputeChecksum(version uint8, content []byte) uint16 {
	var csum uint32
	var rv uint16
	if version == VRRP_VERSION2 {
		for i := 0; i < len(content); i += 2 {
			csum += uint32(content[i]) << 8
			csum += uint32(content[i+1])
		}
		rv = ^uint16((csum >> 16) + csum)
	} else if version == VRRP_VERSION3 {
		//@TODO: .....
	}

	return rv
}

func VrrpCheckHeader(hdr *VrrpPktHeader, layerContent []byte, key string) error {
	// @TODO: need to check for version 2 type...RFC requests to drop the packet
	// but cisco uses version 2...
	if hdr.Version != VRRP_VERSION2 && hdr.Version != VRRP_VERSION3 {
		return errors.New(VRRP_INCORRECT_VERSION)
	}
	logger.Info(fmt.Sprintln("vrrp rx hdr is", hdr))
	// Set Checksum to 0 for verifying checksum
	binary.BigEndian.PutUint16(layerContent[6:8], 0)
	// Verify checksum
	chksum := VrrpComputeChecksum(hdr.Version, layerContent)
	if chksum != hdr.CheckSum {
		logger.Err(fmt.Sprintln(chksum, "!=", hdr.CheckSum))
		return errors.New(VRRP_CHECKSUM_ERR)
	}

	// Verify VRRP fields
	if hdr.CountIPv4Addr == 0 ||
		hdr.MaxAdverInt == 0 ||
		hdr.Type == 0 {
		return errors.New(VRRP_INCORRECT_FIELDS)
	}
	gblInfo := vrrpGblInfo[key]
	for i := 0; i < int(hdr.CountIPv4Addr); i++ {
		// @TODO: confirm this check with HARI
		if gblInfo.IntfConfig.VirtualIPv4Addr == hdr.IPv4Addr[i].String() {
			return errors.New(VRRP_SAME_OWNER)
		}
	}
	if gblInfo.IntfConfig.VRID == 0 {
		return errors.New(VRRP_MISSING_VRID_CONFIG)
	}
	return nil
}

func VrrpCheckIpInfo(rcvdCh <-chan VrrpPktChannelInfo) {
	logger.Info("started pre-fsm check")
	for {
		pktChannel := <-rcvdCh
		packet := pktChannel.pkt
		key := pktChannel.key
		IfIndex := pktChannel.IfIndex
		// Get Entire IP layer Info
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			logger.Err("Not an ip packet?")
			continue
		}
		// Get Ip Hdr and start doing basic check according to RFC
		ipHdr := ipLayer.(*layers.IPv4)
		if ipHdr.TTL != VRRP_TTL {
			logger.Err(fmt.Sprintln("ttl should be 255 instead of", ipHdr.TTL,
				"dropping packet from", ipHdr.SrcIP))
			continue
		}
		// Get Payload as checks are succesful
		ipPayload := ipLayer.LayerPayload()
		if ipPayload == nil {
			logger.Err("No payload for ip packet")
			continue
		}
		// Get VRRP header from IP Payload
		vrrpHeader := VrrpDecodeHeader(ipPayload)
		// Do Basic Vrrp Header Check
		if err := VrrpCheckHeader(vrrpHeader, ipPayload, key); err != nil {
			logger.Err(fmt.Sprintln(err.Error(),
				". Dropping received packet from", ipHdr.SrcIP))
			continue
		}

		logger.Info("Vrrp Info Check Pass...Start FSM")
		vrrpTxPktCh <- VrrpPktChannelInfo{
			pkt:     packet,
			key:     key,
			IfIndex: IfIndex,
		}
	}
}

func VrrpReceivePackets(pHandle *pcap.Handle, key string, IfIndex int32) {
	packetSource := gopacket.NewPacketSource(pHandle, pHandle.LinkType())
	for packet := range packetSource.Packets() {
		vrrpRxPktCh <- VrrpPktChannelInfo{
			pkt:     packet,
			key:     key,
			IfIndex: IfIndex,
		}
	}
	logger.Info("Exiting Receive Packets")
}

func VrrpFormVrrpHeader(gblInfo VrrpGlobalInfo) ([]byte, uint16) {
	// @TODO: handle v6 packets.....
	var vip []net.IP
	//@TODO: Update check from string to len of configured virtual ip's
	if gblInfo.IntfConfig.VirtualIPv4Addr != "" {
		vip = append(vip,
			net.ParseIP(gblInfo.IntfConfig.VirtualIPv4Addr))
	} else {
		vip = append(vip, net.ParseIP(gblInfo.IpAddr))
	}
	vrrpHeader := VrrpPktHeader{
		Version:       VRRP_VERSION2,
		Type:          VRRP_PKT_TYPE,
		VirtualRtrId:  uint8(gblInfo.IntfConfig.VRID),
		Priority:      uint8(gblInfo.IntfConfig.Priority),
		CountIPv4Addr: 1, // @TODO: FIXME for more than 1 vip
		Rsvd:          VRRP_RSVD,
		MaxAdverInt:   uint16(gblInfo.IntfConfig.AdvertisementInterval),
		CheckSum:      VRRP_HDR_CREATE_CHECKSUM,
		IPv4Addr:      vip,
	}
	logger.Info(fmt.Sprintln("vrrp send hdr is", vrrpHeader))
	vrrpEncHdr, hdrLen := VrrpEncodeHeader(vrrpHeader)
	logger.Info(fmt.Sprintln("vrrp send enc hdr is", vrrpEncHdr))
	// Create Checksum for the header and store it
	chksum := VrrpComputeChecksum(vrrpHeader.Version, vrrpEncHdr)
	binary.BigEndian.PutUint16(vrrpEncHdr[6:8], chksum)

	return vrrpEncHdr, hdrLen
}

func VrrpFormPkt(gblInfo VrrpGlobalInfo, vrrpEncHdr []byte, hdrLen uint16) []byte {
	srcMAC, _ := net.ParseMAC(gblInfo.IntfConfig.VirtualRouterMACAddress)
	dstMAC, _ := net.ParseMAC(VRRP_PROTOCOL_MAC)
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	logger.Info(fmt.Sprintln("Send Eth layer:", eth))
	ipv4 := &layers.IPv4{
		SrcIP:    net.ParseIP(gblInfo.IpAddr),
		DstIP:    net.ParseIP(VRRP_GROUP_IP),
		Version:  4,
		Protocol: VRRP_PROTO_ID,
		TTL:      VRRP_TTL,
		Length:   uint16(VRRP_IPV4_HEADER_MIN_SIZE + hdrLen),
	}
	logger.Info(fmt.Sprintln("Send IP layer:", ipv4))

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	gopacket.SerializeLayers(buffer, options, eth, ipv4, gopacket.Payload(vrrpEncHdr))
	return buffer.Bytes()
}

func VrrpSendPkt(rcvdCh <-chan VrrpPktChannelInfo) {
	logger.Info("started send packet routine")
	for {
		pktChannel := <-rcvdCh
		//packet := pktChannel.pkt
		key := pktChannel.key
		gblInfo, found := vrrpGblInfo[key]
		if !found {
			logger.Err("No Entry for " + key)
			continue
		}
		logger.Info("Found gblInfo entry for " + key)
		vrrpEncHdr, hdrLen := VrrpFormVrrpHeader(gblInfo)
		vrrpTxPkt := VrrpFormPkt(gblInfo, vrrpEncHdr, hdrLen)
		logger.Info(fmt.Sprintln("send pkt", vrrpTxPkt))
	}
}

func VrrpInitPacketListener(key string, IfIndex int32) {
	linuxInterface, ok := vrrpLinuxIfIndex2AsicdIfIndex[IfIndex]
	if ok == false {
		logger.Err(fmt.Sprintln("no linux interface for ifindex",
			IfIndex))
		return
	}
	handle, err := pcap.OpenLive(linuxInterface.Name, vrrpSnapshotLen,
		vrrpPromiscuous, vrrpTimeout)
	if err != nil {
		logger.Err(fmt.Sprintln("Creating VRRP listerner failed",
			err))
		return
	}
	err = handle.SetBPFFilter(VRRP_BPF_FILTER)
	if err != nil {
		logger.Err(fmt.Sprintln("Setting filter", VRRP_BPF_FILTER,
			"failed with", "err:", err))
	}
	gblInfo := vrrpGblInfo[key]
	gblInfo.pHandle = handle
	vrrpGblInfo[key] = gblInfo
	logger.Info(fmt.Sprintln("VRRP listener running for", IfIndex))
	if vrrpRxChStarted == false {
		go VrrpCheckIpInfo(vrrpRxPktCh)
		vrrpRxChStarted = true
	}
	if vrrpTxChStarted == false {
		go VrrpSendPkt(vrrpTxPktCh)
		vrrpTxChStarted = true
	}
	go VrrpReceivePackets(handle, key, IfIndex)
}

func VrrpAddMacEntry(add bool) {
	for !asicdClient.IsConnected {
		time.Sleep(time.Millisecond * 750)
		logger.Info("Waiting for vrrp to connect to asicd")
	}
	macConfig := asicdServices.RsvdProtocolMacConfig{
		MacAddr:     VRRP_PROTOCOL_MAC,
		MacAddrMask: VRRP_MAC_MASK,
	}
	if add {
		inserted, _ := asicdClient.ClientHdl.EnablePacketReception(&macConfig)
		if !inserted {
			logger.Info("Adding reserved mac failed")
		}
	} else {
		deleted, _ := asicdClient.ClientHdl.DisablePacketReception(&macConfig)
		if !deleted {
			logger.Info("Adding reserved mac failed")
		}
	}
}
