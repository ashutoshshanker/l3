package vrrpServer

import (
	"asicdServices"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	_ "net"
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
