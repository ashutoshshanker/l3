// Dhcp Relay Agent Protocol Handling for Packet Send/Receive
package relayServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	_ "github.com/google/gopacket/pcap"
	"golang.org/x/net/ipv4"
	"net"
	"sync"
)

func DhcpRelayAgentDecodeInPkt(data []byte, ethLayer *layers.Ethernet,
	ipLayer *layers.IPv4, udpLayer *layers.UDP,
	payload *gopacket.Payload) {
	//@FIXME: jgheewala getting error on decode
	//Trouble decoding layers:  No decoder for layer type Payload

	logger.Info(fmt.Sprintln("DRA: Decoding PKT"))
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet,
		ethLayer, ipLayer, udpLayer, payload)
	foundLayerTypes := make([]gopacket.LayerType, 0, 10)

	err := parser.DecodeLayers(data, &foundLayerTypes)
	if err != nil {
		logger.Info(fmt.Sprintln("DRA: Trouble decoding layers: ", err))
	}

	for _, layerType := range foundLayerTypes {
		if layerType == layers.LayerTypeEthernet {
			logger.Info(fmt.Sprintln("DRA: Eth: ", ethLayer.SrcMAC,
				"->", ethLayer.DstMAC))

		}
		if layerType == layers.LayerTypeIPv4 {
			logger.Info(fmt.Sprintln("DRA: IPv4: ", ipLayer.SrcIP,
				"->", ipLayer.DstIP))
		}
		if layerType == layers.LayerTypeUDP {
			logger.Info(fmt.Sprintln("DRA: UDP Port: ",
				udpLayer.SrcPort, "->", udpLayer.DstPort))
		}
	}
	logger.Info(fmt.Sprintln("DRA: Decoding of Pkt done"))
}

/*
func DhcpRelayAgentSendPacketToDhcpServer(inputPacket gopacket.Packet,
	handler *pcap.Handle, ethLayer layers.Ethernet, ipLayer layers.IPv4,
	udpLayer layers.UDP, payload gopacket.Payload) {

	logger.Info("DRA: Creating Send Pkt")
	// Send raw bytes over wire
	rawBytes := []byte{10, 20, 30}

	// Ethernet Info
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x12, 0x34},
		DstMAC: ethLayer.DstMAC,
		//DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	logger.Info(fmt.Sprintln("DRA: eth payload", eth))

	// Ip Info
	ip := &layers.IPv4{
		SrcIP:    ipLayer.SrcIP,
		DstIP:    ipLayer.DstIP,
		Version:  4,
		Protocol: layers.IPProtocolUDP,
		TTL:      64,
	}
	logger.Info(fmt.Sprintln("DRA: ip payload", ip))

	// UDP (Port) Info
	udp := &layers.UDP{
		SrcPort: udpLayer.SrcPort,
		DstPort: udpLayer.DstPort,
	}
	udp.SetNetworkLayerForChecksum(ip)
	logger.Info(fmt.Sprintln("DRA: udp payload", udp))

	// Add DRA Option to the packet formed
	// Create the packet with the layers
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		// FixLengths determines whether, during serialization, layers
		// should fix the values for any length field that depends on the
		// payload.
		FixLengths: true,
		// ComputeChecksums determines whether, during serialization, layers
		// should recompute checksums based on their payloads.
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, options, eth, ip, udp,
		gopacket.Payload(rawBytes))
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Serializing gopacket failed", err))
		return
	}
	logger.Info(fmt.Sprintln("DRA: PacketData... ", buffer.Bytes()))
	err = handler.WritePacketData(buffer.Bytes())
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: couldn't write to output data", err))
		return
	}

	logger.Info(fmt.Sprintln("DRA: Create & Send of PKT successfully"))
}
*/

func DhcpRelayAgentReceiveDhcpPktFromClient() {
	var buf []byte = make([]byte, 1500)
	for {
		bytesRead, cm, srcAddr, err := dhcprelayClientConn.ReadFrom(buf)
		if err != nil {
			logger.Err("DRA: reading buffer failed")
		}
		// Will reuse these for each packet
		/*
			var ethLayer layers.Ethernet
			var ipLayer layers.IPv4
			var udpLayer layers.UDP
			var payload gopacket.Payload
		*/
		//Decode the packet...
		//DhcpRelayAgentDecodeInPkt(buf, &ethLayer, &ipLayer, &udpLayer,
		//	&payload)
		logger.Info(fmt.Sprintln("DRA: bytesread is ", bytesRead))
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		logger.Info(fmt.Sprintln("DRA: srcAddr is ", srcAddr))
	}
}

func DhcpRelayAgentCreateClientServerConn() {

	// Client send dhcp packet from port 68 to server port 67
	// So create a filter for udp:67 for messages send out by client to
	// server
	logger.Info("DRA: creating listenPacket for udp port 67")
	saddr := net.UDPAddr{
		Port: 67,
		IP:   net.ParseIP(""),
	}
	/*
		caddr := net.UDPAddr{
			Port: 68,
			IP:   net.ParseIP("0.0.0.0"),
		}*/
	dhcprelayNetHandler, err := net.ListenUDP("udp", &saddr)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Opening udp port for client --> server failed", err))
		return
	}
	dhcprelayClientConn = ipv4.NewPacketConn(dhcprelayNetHandler)
	controlFlag := ipv4.FlagTTL | ipv4.FlagSrc | ipv4.FlagDst | ipv4.FlagInterface
	err = dhcprelayClientConn.SetControlMessage(controlFlag, true)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Setting control flag failed..", err))
		return
	}
	logger.Info("DRA: Connection opened successfully")
	go DhcpRelayAgentReceiveDhcpPktFromClient()

}

func DhcpRelayAgentInitGblHandling(ifName string, ifNum int) {
	logger.Info("DRA: Initializaing Global Info for " + ifName + " " +
		string(ifNum))
	// Created a global Entry for Interface
	gblEntry := dhcprelayGblInfo[ifName]
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IpSubnet = ""
	gblEntry.IntfConfig.Netmask = ""
	gblEntry.IntfConfig.IfIndex = ifName
	gblEntry.IntfConfig.AgentSubType = 0
	gblEntry.IntfConfig.Enable = false
	gblEntry.dhcprelayConfigMutex = sync.RWMutex{}
	// Stats information
	gblEntry.StateDebugInfo.stats = make([]string, 150)
	DhcpRelayAgentUpdateStats(ifName, &gblEntry)
	DhcpRelayAgentUpdateStats("Global Init Done", &gblEntry)

	dhcprelayGblInfo[ifName] = gblEntry

}
