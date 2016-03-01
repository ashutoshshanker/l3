package vrrpServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func VrrpDecodeVrrpInfo(data []byte) {
	var vrrpPkt VrrpPktFormat
	vrrpPkt.Version = uint8(data[0]) >> 4
	vrrpPkt.Type = uint8(data[0]) & 0x0F
	vrrpPkt.VirtualRtrId = data[1]
	vrrpPkt.Priority = data[2]
	vrrpPkt.CountIPv4Addr = data[3]
	logger.Info(fmt.Sprintln("vrrp payload:", vrrpPkt))
}

func VrrpDecodeReceivedPkt(packet gopacket.Packet) {
	//var err error
	/*
			eth := packet.LinkLayer()
			net := packet.NetworkLayer()
			srcIp, dstIp := net.NetworkFlow().Endpoints()
			srcMac, dstMac := eth.LinkFlow().Endpoints()
			logger.Info(fmt.Sprintln("src", srcIp, "dst", dstIp))
			logger.Info(fmt.Sprintln("src", srcMac, "dst", dstMac))
			VrrpDecodeVrrpInfo(net.Layer().LayerPayload())
		ethLayer := packet.Layer(layers.LayerTypeEthernet)
		if ethLayer == nil {
			logger.Err("No ethernet frame")
			return
		}
	*/
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
	VrrpDecodeVrrpInfo(ipPayload)
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
