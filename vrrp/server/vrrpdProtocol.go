package vrrpServer

import (
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
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
func VrrpDecodeVrrpHeader(data []byte) {
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
	logger.Info(fmt.Sprintln("vrrp payload:", vrrpPkt))
}

func VrrpDecodeReceivedPkt(packet gopacket.Packet) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		logger.Err("Not an ip packet?")
		return
	}
	ipPayload := ipLayer.LayerPayload()
	if ipPayload == nil {
		logger.Err("No payload for ip packet")
		return
	}
	VrrpDecodeVrrpHeader(ipPayload)
}

func VrrpReceivePackets(pHandle *pcap.Handle, IfIndex int32) {
	packetSource := gopacket.NewPacketSource(pHandle, pHandle.LinkType())
	inCh := packetSource.Packets()
	for {
		packet, ok := <-inCh
		if ok {
			VrrpDecodeReceivedPkt(packet)
		}
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
	go VrrpReceivePackets(handle, IfIndex)
}
