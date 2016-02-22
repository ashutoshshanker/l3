package vrrpServer

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
)

func VrrpReceivePackets() {
	var buf []byte = make([]byte, 1500)
	for {
		bytesRead, ctrlMsg, srcAddr, err := vrrpListener.ReadFrom(buf)
		if err != nil {
			logger.Err(fmt.Sprintln("Reading buffer failed",
				err))
			continue
		}
		logger.Info(fmt.Sprintln("bytesRead:", bytesRead,
			"ctrlMsg:", ctrlMsg,
			"srcAddr:", srcAddr))
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
