package vrrpServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"net"
)

func VrrpDecodeReceivedPkt(InData []byte, bytesRead int) {
	var eth layers.Ethernet
	var ip4 layers.IPv4
	var payload gopacket.Payload
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet,
		&eth, &ip4, &payload)
	decodedLayers := make([]gopacket.LayerType, 0, 10)
	err := parser.DecodeLayers(InData, &decodedLayers)
	if err != nil {
		logger.Err(fmt.Sprintln("Decoding of Packet failed",
			err))
		return
	}
}

func VrrpReceivePackets() {
	var buf []byte = make([]byte, 1500)
	for {
		if vrrpListener == nil || vrrpNetPktConn == nil {
			logger.Info("Listerner is not set...")
			return
		}
		bytesRead, ctrlMsg, srcAddr, err := vrrpListener.ReadFrom(buf)
		if err != nil {
			logger.Err(fmt.Sprintln("Reading buffer failed",
				err))
			continue
		}
		logger.Info(fmt.Sprintln("bytesRead:", bytesRead,
			"ctrlMsg:", ctrlMsg,
			"srcAddr:", srcAddr))
		VrrpDecodeReceivedPkt(buf, bytesRead)
	}
}

func VrrpInitPacketListener() {
	var err error
	vrrpNetPktConn, err = net.ListenPacket("ip4:112", "224.0.0.18")
	if err != nil {
		logger.Err(fmt.Sprintln("Creating VRRP listerner failed",
			err))
		return
	}
	vrrpListener = ipv4.NewPacketConn(vrrpNetPktConn)
	err = vrrpListener.SetControlMessage(vrrpCtrlFlag, true)
	if err != nil {
		logger.Err(fmt.Sprintln("Setting control flag failed",
			err))
		return
	}
	logger.Info("VRRP listener UP and running")
	go VrrpReceivePackets()
}
