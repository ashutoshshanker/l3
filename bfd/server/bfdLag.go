package server

import (
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"time"
)

func (session *BfdSession) StartPerLinkSessionServer(bfdServer *BFDServer) error {
	var ifName string
	var err error
	var myMacAddr net.HardwareAddr
	bfdServer.logger.Info(fmt.Sprintln("Starting perlink session ", session.state.SessionId, " on ", ifName))
	sessionId := session.state.SessionId
	ifName, err = bfdServer.getLinuxIntfName(session.state.InterfaceId)
	if err != nil {
		bfdServer.logger.Info(fmt.Sprintln("Failed to get ifname for ", session.state.InterfaceId))
		return err
	}
	myMacAddr, err = bfdServer.getMacAddrFromIntfName(ifName)
	if err != nil {
		bfdServer.logger.Info(fmt.Sprintln("Unable to get the MAC addr of ", ifName, err))
		return err
	}
	bfdServer.logger.Info(fmt.Sprintln("MAC is  ", myMacAddr, " on ", ifName))
	bfdPcapTimeout := time.Duration(session.state.RequiredMinRxInterval / 1000000)
	session.recvPcapHandle, err = pcap.OpenLive(ifName, bfdSnapshotLen, bfdPromiscuous, bfdPcapTimeout)
	if session.recvPcapHandle == nil {
		bfdServer.logger.Info(fmt.Sprintln("Failed to open recvPcapHandle for ", ifName, err))
		return err
	} else {
		err = session.recvPcapHandle.SetBPFFilter(bfdPcapFilter)
		if err != nil {
			bfdServer.logger.Info(fmt.Sprintln("Unable to set filter on", ifName, err))
			return err
		}
	}
	bfdPacketSrc := gopacket.NewPacketSource(session.recvPcapHandle, layers.LayerTypeEthernet)
	defer session.recvPcapHandle.Close()
	for receivedPacket := range bfdPacketSrc.Packets() {
		if bfdServer.bfdGlobal.Sessions[sessionId] == nil {
			return nil
		}
		bfdServer.logger.Info(fmt.Sprintln("Receive packet ", receivedPacket))

		ethLayer := receivedPacket.Layer(layers.LayerTypeEthernet)
		ethPacket, _ := ethLayer.(*layers.Ethernet)
		bfdServer.logger.Info(fmt.Sprintln("Ethernet ", ethPacket.SrcMAC, ethPacket.DstMAC))
		nwLayer := receivedPacket.Layer(layers.LayerTypeIPv4)
		ipPacket, _ := nwLayer.(*layers.IPv4)
		bfdServer.logger.Info(fmt.Sprintln("Network ", ipPacket.SrcIP, ipPacket.DstIP))
		transLayer := receivedPacket.Layer(layers.LayerTypeUDP)
		udpPacket, _ := transLayer.(*layers.UDP)
		bfdServer.logger.Info(fmt.Sprintln("Transport ", udpPacket.SrcPort, udpPacket.DstPort))
		appLayer := receivedPacket.ApplicationLayer()
		bfdServer.logger.Info(fmt.Sprintln("Application ", appLayer))

		if bytes.Equal(ethPacket.SrcMAC, myMacAddr) {
			bfdServer.logger.Info(fmt.Sprintln("My packet looped back"))
			continue
		}

		buf := transLayer.LayerPayload()
		if len(buf) >= DEFAULT_CONTROL_PACKET_LEN {
			bfdPacket, err := DecodeBfdControlPacket(buf)
			if err == nil {
				sessionId := int32(bfdPacket.YourDiscriminator)
				if sessionId == 0 {
					bfdServer.logger.Info(fmt.Sprintln("Ignore bfd packet for session ", sessionId))
				} else {
					bfdSession := bfdServer.bfdGlobal.Sessions[sessionId]
					match := bytes.Equal(bfdSession.state.RemoteMacAddr, ethPacket.SrcMAC)
					if !match {
						bfdSession.state.RemoteMacAddr = ethPacket.DstMAC
					}
					bfdSession.state.NumRxPackets++
					bfdSession.ProcessBfdPacket(bfdPacket)
				}
			} else {
				bfdServer.logger.Info(fmt.Sprintln("Failed to decode packet - ", err))
			}
		}
	}
	return nil
}

func (session *BfdSession) StartPerLinkSessionClient(bfdServer *BFDServer) error {
	var ifName string
	var err error
	var myMacAddr net.HardwareAddr
	bfdServer.logger.Info(fmt.Sprintln("Starting perlink session ", session.state.SessionId, " on ", ifName))
	ifName, err = bfdServer.getLinuxIntfName(session.state.InterfaceId)
	if err != nil {
		bfdServer.logger.Info(fmt.Sprintln("Failed to get ifname for ", session.state.InterfaceId))
		bfdServer.FailedSessionClientCh <- session.state.SessionId
		return err
	}
	myMacAddr, err = bfdServer.getMacAddrFromIntfName(ifName)
	if err != nil {
		bfdServer.logger.Info(fmt.Sprintln("Unable to get the MAC addr of ", ifName, err))
		bfdServer.FailedSessionClientCh <- session.state.SessionId
		return err
	}
	bfdServer.logger.Info(fmt.Sprintln("MAC is  ", myMacAddr, " on ", ifName))
	bfdPcapTimeout := time.Duration(session.state.DesiredMinTxInterval / 1000000)
	session.sendPcapHandle, err = pcap.OpenLive(ifName, bfdSnapshotLen, bfdPromiscuous, bfdPcapTimeout)
	if session.sendPcapHandle == nil {
		bfdServer.logger.Info(fmt.Sprintln("Failed to open sendPcapHandle for ", ifName, err))
		bfdServer.FailedSessionClientCh <- session.state.SessionId
		return err
	}
	session.TxTimeoutCh = make(chan int32)
	session.SessionTimeoutCh = make(chan int32)
	sessionTimeoutMS := time.Duration(session.state.RequiredMinRxInterval * session.state.DetectionMultiplier / 1000)
	txTimerMS := time.Duration(session.state.DesiredMinTxInterval / 1000)
	session.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { session.SessionTimeoutCh <- session.state.SessionId })
	session.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { session.TxTimeoutCh <- session.state.SessionId })
	defer session.sendPcapHandle.Close()
	for {
		select {
		case sessionId := <-session.TxTimeoutCh:
			var destMac net.HardwareAddr
			bfdSession := bfdServer.bfdGlobal.Sessions[sessionId]
			if bfdSession.useDedicatedMac {
				destMac, _ = net.ParseMAC(bfdDedicatedMac)
			} else {
				destMac = bfdSession.state.RemoteMacAddr
			}
			ethLayer := &layers.Ethernet{
				SrcMAC:       bfdSession.state.LocalMacAddr,
				DstMAC:       destMac,
				EthernetType: layers.EthernetTypeIPv4,
			}
			ipLayer := &layers.IPv4{
				SrcIP:    net.ParseIP(bfdSession.state.LocalIpAddr),
				DstIP:    net.ParseIP(bfdSession.state.IpAddr),
				Protocol: layers.IPProtocolUDP,
			}
			udpLayer := &layers.UDP{
				SrcPort: layers.UDPPort(SRC_PORT_LAG),
				DstPort: layers.UDPPort(DEST_PORT_LAG),
			}
			options := gopacket.SerializeOptions{
				FixLengths:       true,
				ComputeChecksums: true,
			}
			bfdSession.UpdateBfdSessionControlPacket()
			bfdPacket, err := bfdSession.bfdPacket.CreateBfdControlPacket()
			if err != nil {
				bfdServer.logger.Info(fmt.Sprintln("Failed to create bfd control packet for session ", bfdSession.state.SessionId))
			}
			buffer := gopacket.NewSerializeBuffer()
			gopacket.SerializeLayers(buffer, options, ethLayer, ipLayer, udpLayer, gopacket.Payload(bfdPacket))
			outgoingPacket := buffer.Bytes()
			err = bfdSession.sendPcapHandle.WritePacketData(outgoingPacket)
			if err != nil {
				bfdServer.logger.Info(fmt.Sprintln("Failed to create complete packet for session ", bfdSession.state.SessionId))
			} else {
				if bfdSession.state.SessionState == STATE_UP {
					bfdSession.useDedicatedMac = false
				}
				bfdSession.state.NumTxPackets++
				bfdSession.txTimer.Stop()
				txTimerMS = time.Duration(bfdSession.state.DesiredMinTxInterval / 1000)
				bfdSession.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { bfdSession.TxTimeoutCh <- bfdSession.state.SessionId })
			}
		case sessionId := <-session.SessionTimeoutCh:
			bfdSession := bfdServer.bfdGlobal.Sessions[sessionId]
			bfdSession.state.LocalDiagType = DIAG_TIME_EXPIRED
			bfdSession.EventHandler(TIMEOUT)
			bfdSession.sessionTimer.Stop()
			sessionTimeoutMS = time.Duration(bfdSession.state.RequiredMinRxInterval * bfdSession.state.DetectionMultiplier / 1000)
			bfdSession.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { bfdSession.SessionTimeoutCh <- bfdSession.state.SessionId })
		case <-session.SessionStopClientCh:
			return nil
		}
	}
	return nil
}
