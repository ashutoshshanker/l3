package vrrpServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func VrrpDecodeReceivedPkt(packet gopacket.Packet) {
	var err error
	var eth layers.Ethernet
	var ip4 layers.IPv4
	var payload gopacket.Payload
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet,
		&eth, &ip4, &payload)
	decodedLayers := []gopacket.LayerType{}
	err = parser.DecodeLayers(packet.Data(), &decodedLayers)
	if err != nil {
		logger.Err(fmt.Sprintln("Decoding of Packet failed",
			err))
		return
	}
	logger.Info(fmt.Sprintln("DecodeLayers: ", decodedLayers))
	logger.Info(fmt.Sprintln("Payload is", payload))
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
	//filter := "ip host " + VRRP_GROUP_IP
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
