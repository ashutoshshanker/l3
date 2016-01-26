// Dhcp Relay Agent Protocol Handling for Packet Send/Receive
package relayServer

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
	"sync"
)

// Dhcp OpCodes Types
const (
	BootRequest OpCode = 1 // From Client
	BootReply   OpCode = 2 // From Server
)

// DHCP Packet global constants
const DHCP_PACKET_MIN_SIZE = 272
const DHCP_PACKET_HEADER_SIZE = 16
const DHCP_PACKET_OPTIONS_LEN = 240

// DHCP Client/Server Message Type 53
const (
	DhcpDiscover MessageType = 1 // From Client - Can I have an IP?
	DhcpOffer    MessageType = 2 // From Server - Here's an IP
	DhcpRequest  MessageType = 3 // From Client - I'll take that IP (Also start for renewals)
	DhcpDecline  MessageType = 4 // From Client - Sorry I can't use that IP
	DhcpACK      MessageType = 5 // From Server, Yes you can have that IP
	DhcpNAK      MessageType = 6 // From Server, No you cannot have that IP
	DhcpRelease  MessageType = 7 // From Client, I don't need that IP anymore
	DhcpInform   MessageType = 8 // From Client, I have this IP and there's nothing you can do about it
)

// DHCP Available Options enum type.... This will cover most of the options type
const (
	End                          DhcpOptionCode = 255
	Pad                          DhcpOptionCode = 0
	OptionSubnetMask             DhcpOptionCode = 1
	OptionTimeOffset             DhcpOptionCode = 2
	OptionRouter                 DhcpOptionCode = 3
	OptionTimeServer             DhcpOptionCode = 4
	OptionNameServer             DhcpOptionCode = 5
	OptionDomainNameServer       DhcpOptionCode = 6
	OptionLogServer              DhcpOptionCode = 7
	OptionCookieServer           DhcpOptionCode = 8
	OptionLPRServer              DhcpOptionCode = 9
	OptionImpressServer          DhcpOptionCode = 10
	OptionResourceLocationServer DhcpOptionCode = 11
	OptionHostName               DhcpOptionCode = 12
	OptionBootFileSize           DhcpOptionCode = 13
	OptionMeritDumpFile          DhcpOptionCode = 14
	OptionDomainName             DhcpOptionCode = 15
	OptionSwapServer             DhcpOptionCode = 16
	OptionRootPath               DhcpOptionCode = 17
	OptionExtensionsPath         DhcpOptionCode = 18

	// IP Layer Parameters per Host
	OptionIPForwardingEnableDisable          DhcpOptionCode = 19
	OptionNonLocalSourceRoutingEnableDisable DhcpOptionCode = 20
	OptionPolicyFilter                       DhcpOptionCode = 21
	OptionMaximumDatagramReassemblySize      DhcpOptionCode = 22
	OptionDefaultIPTimeToLive                DhcpOptionCode = 23
	OptionPathMTUAgingTimeout                DhcpOptionCode = 24
	OptionPathMTUPlateauTable                DhcpOptionCode = 25

	// IP Layer Parameters per Interface
	OptionInterfaceMTU              DhcpOptionCode = 26
	OptionAllSubnetsAreLocal        DhcpOptionCode = 27
	OptionBroadcastAddress          DhcpOptionCode = 28
	OptionPerformMaskDiscovery      DhcpOptionCode = 29
	OptionMaskSupplier              DhcpOptionCode = 30
	OptionPerformRouterDiscovery    DhcpOptionCode = 31
	OptionRouterSolicitationAddress DhcpOptionCode = 32
	OptionStaticRoute               DhcpOptionCode = 33

	// Link Layer Parameters per Interface
	OptionTrailerEncapsulation  DhcpOptionCode = 34
	OptionARPCacheTimeout       DhcpOptionCode = 35
	OptionEthernetEncapsulation DhcpOptionCode = 36

	// TCP Parameters
	OptionTCPDefaultTTL        DhcpOptionCode = 37
	OptionTCPKeepaliveInterval DhcpOptionCode = 38
	OptionTCPKeepaliveGarbage  DhcpOptionCode = 39

	// Application and Service Parameters
	OptionNetworkInformationServiceDomain            DhcpOptionCode = 40
	OptionNetworkInformationServers                  DhcpOptionCode = 41
	OptionNetworkTimeProtocolServers                 DhcpOptionCode = 42
	OptionVendorSpecificInformation                  DhcpOptionCode = 43
	OptionNetBIOSOverTCPIPNameServer                 DhcpOptionCode = 44
	OptionNetBIOSOverTCPIPDatagramDistributionServer DhcpOptionCode = 45
	OptionNetBIOSOverTCPIPNodeType                   DhcpOptionCode = 46
	OptionNetBIOSOverTCPIPScope                      DhcpOptionCode = 47
	OptionXWindowSystemFontServer                    DhcpOptionCode = 48
	OptionXWindowSystemDisplayManager                DhcpOptionCode = 49
	OptionNetworkInformationServicePlusDomain        DhcpOptionCode = 64
	OptionNetworkInformationServicePlusServers       DhcpOptionCode = 65
	OptionMobileIPHomeAgent                          DhcpOptionCode = 68
	OptionSimpleMailTransportProtocol                DhcpOptionCode = 69
	OptionPostOfficeProtocolServer                   DhcpOptionCode = 70
	OptionNetworkNewsTransportProtocol               DhcpOptionCode = 71
	OptionDefaultWorldWideWebServer                  DhcpOptionCode = 72
	OptionDefaultFingerServer                        DhcpOptionCode = 73
	OptionDefaultInternetRelayChatServer             DhcpOptionCode = 74
	OptionStreetTalkServer                           DhcpOptionCode = 75
	OptionStreetTalkDirectoryAssistance              DhcpOptionCode = 76

	OptionRelayAgentInformation DhcpOptionCode = 82

	// DHCP Extensions
	OptionRequestedIPAddress     DhcpOptionCode = 50
	OptionIPAddressLeaseTime     DhcpOptionCode = 51
	OptionOverload               DhcpOptionCode = 52
	OptionDHCPMessageType        DhcpOptionCode = 53
	OptionServerIdentifier       DhcpOptionCode = 54
	OptionParameterRequestList   DhcpOptionCode = 55
	OptionMessage                DhcpOptionCode = 56
	OptionMaximumDHCPMessageSize DhcpOptionCode = 57
	OptionRenewalTimeValue       DhcpOptionCode = 58
	OptionRebindingTimeValue     DhcpOptionCode = 59
	OptionVendorClassIdentifier  DhcpOptionCode = 60
	OptionClientIdentifier       DhcpOptionCode = 61

	OptionTFTPServerName DhcpOptionCode = 66
	OptionBootFileName   DhcpOptionCode = 67

	OptionUserClass DhcpOptionCode = 77

	OptionClientArchitecture DhcpOptionCode = 93

	OptionTZPOSIXString    DhcpOptionCode = 100
	OptionTZDatabaseString DhcpOptionCode = 101

	OptionClasslessRouteFormat DhcpOptionCode = 121
)

/* ========================HELPER FUNCTIONS FOR DHCP =========================*/
/*
   0               1               2               3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |     op (1)    |   htype (1)   |   hlen (1)    |   hops (1)    |
   +---------------+---------------+---------------+---------------+
   |                            xid (4)                            |
   +-------------------------------+-------------------------------+
   |           secs (2)            |           flags (2)           |
   +-------------------------------+-------------------------------+
   |                          ciaddr  (4)                          |
   +---------------------------------------------------------------+
   |                          yiaddr  (4)                          |
   +---------------------------------------------------------------+
   |                          siaddr  (4)                          |
   +---------------------------------------------------------------+
   |                          giaddr  (4)                          |
   +---------------------------------------------------------------+
   |                                                               |
   |                          chaddr  (16)                         |
   |                                                               |
   |                                                               |
   +---------------------------------------------------------------+
   |                                                               |
   |                          sname   (64)                         |
   +---------------------------------------------------------------+
   |                                                               |
   |                          file    (128)                        |
   +---------------------------------------------------------------+
   |                                                               |
   |                          options (variable)                   |
   +---------------------------------------------------------------+
*/
/*
 * API to return Header Lenght of the incoming packet
 */
func (p DhcpRelayAgentPacket) HeaderLen() byte {
	return p[2]
}

func (p DhcpRelayAgentPacket) OpCode() OpCode {
	return OpCode(p[0])
}
func (p DhcpRelayAgentPacket) HeaderType() byte {
	return p[1]
}
func (p DhcpRelayAgentPacket) Hops() byte {
	return p[3]
}
func (p DhcpRelayAgentPacket) XId() []byte {
	return p[4:8]
}
func (p DhcpRelayAgentPacket) Secs() []byte {
	return p[8:10]
}
func (p DhcpRelayAgentPacket) Flags() []byte {
	return p[10:12]
}
func (p DhcpRelayAgentPacket) CIAddr() net.IP {
	return net.IP(p[12:16])
}
func (p DhcpRelayAgentPacket) YIAddr() net.IP {
	return net.IP(p[16:20])
}
func (p DhcpRelayAgentPacket) SIAddr() net.IP {
	return net.IP(p[20:24])
}
func (p DhcpRelayAgentPacket) GIAddr() net.IP {
	return net.IP(p[24:28])
}
func (p DhcpRelayAgentPacket) CHAddr() net.HardwareAddr {
	hLen := p.HeaderLen()
	if hLen > DHCP_PACKET_HEADER_SIZE { // Prevent chaddr exceeding p boundary
		hLen = DHCP_PACKET_HEADER_SIZE
	}
	return net.HardwareAddr(p[28 : 28+hLen]) // max endPos 44
}

func ParseMessageTypeToString(mtype MessageType) {
	switch mtype {
	case 1:
		logger.Info("DRA: Message Type: DhcpDiscover")
	case 2:
		logger.Info("DRA: Message Type: DhcpOffer")
	case 3:
		logger.Info("DRA: Message Type: DhcpRequest")
	case 4:
		logger.Info("DRA: Message Type: DhcpDecline")
	case 5:
		logger.Info("DRA: Message Type: DhcpACK")
	case 6:
		logger.Info("DRA: Message Type: DhcpNAK")
	case 7:
		logger.Info("DRA: Message Type: DhcpRelease")
	case 8:
		logger.Info("DRA: Message Type: DhcpInform")
	default:
		logger.Info("DRA: Message Type: UnKnown...Discard the Packet")
	}
}

func (p DhcpRelayAgentPacket) AllocateOptions() []byte {
	if len(p) > DHCP_PACKET_OPTIONS_LEN {
		return p[DHCP_PACKET_OPTIONS_LEN:]
	}
	return nil
}

func (p *DhcpRelayAgentPacket) PadToMinSize() {
	sizeofPacket := len(*p)
	if sizeofPacket < DHCP_PACKET_MIN_SIZE {
		// adding whatever is left out to the padder
		*p = append(*p, dhcprelayPadder[:DHCP_PACKET_MIN_SIZE-sizeofPacket]...)
	}
}

// Parses the packet's options into an Options map
func (p DhcpRelayAgentPacket) ParseDhcpOptions() DhcpRelayAgentOptions {
	opts := p.AllocateOptions()
	// create basic dhcp options...
	doptions := make(DhcpRelayAgentOptions, 15)
	for len(opts) >= 2 && DhcpOptionCode(opts[0]) != End {
		if DhcpOptionCode(opts[0]) == Pad {
			opts = opts[1:]
			continue
		}
		size := int(opts[1])
		if len(opts) < 2+size {
			break
		}
		doptions[DhcpOptionCode(opts[0])] = opts[2 : 2+size]
		opts = opts[2+size:]
	}
	return doptions
}

/*========================= END OF HELPER FUNCTION ===========================*/
/*
 * APT to decode incoming Packet by converting the byte into DHCP packet format
 */
func DhcpRelayAgentDecodeInPkt(data []byte, bytesRead int) {
	logger.Info(fmt.Sprintln("DRA: Decoding PKT"))
	inRequest := DhcpRelayAgentPacket(data[:bytesRead])
	if inRequest.HeaderLen() > DHCP_PACKET_HEADER_SIZE {
		logger.Warning("Header Lenght is invalid... don't do anything")
		return
	}
	reqOptions := inRequest.ParseDhcpOptions()
	logger.Info("DRA: CIAddr is " + inRequest.CIAddr().String())
	logger.Info("DRA: CHaddr is " + inRequest.CHAddr().String())
	logger.Info("DRA: YIAddr is " + inRequest.YIAddr().String())
	logger.Info("DRA: GIAddr is " + inRequest.GIAddr().String())
	mType := reqOptions[OptionDHCPMessageType]
	ParseMessageTypeToString(MessageType(mType[0]))

	logger.Info(fmt.Sprintln("DRA: Decoding of Pkt done"))
}

func DhcpRelayAgentSendPacketToDhcpServer(controlMessage *ipv4.ControlMessage,
	data []byte) {
	logger.Info("DRA: Creating Send Pkt")
	/*
	   rawBytes := []byte{10, 20, 30}
	           // Ethernet Info
	           eth := &layers.Ethernet{
	                   SrcMAC:       net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x12, 0x34},
	                   DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	                   EthernetType: layers.EthernetTypeIPv4,
	           }
	           logger.Info(fmt.Sprintln("DRA: eth payload", eth))

	               // Ip Info
	               ip := &layers.IPv4{
	                       SrcIP:    net.IP{0, 0, 0, 0},
	                       DstIP:    net.IP{255, 255, 255, 255},
	                       Version:  4,
	                       Protocol: layers.IPProtocolUDP,
	                       TTL:      64,
	               }
	               logger.Info(fmt.Sprintln("DRA: ip payload", ip))

	               // UDP (Port) Info
	               udp := &layers.UDP{
	                       SrcPort: 67,
	                       DstPort: 68,
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
	*/
	logger.Info(fmt.Sprintln("DRA: Create & Send of PKT successfully"))
}

func DhcpRelayAgentReceiveDhcpPktFromClient() {
	var buf []byte = make([]byte, 1500)
	for {
		bytesRead, cm, srcAddr, err := dhcprelayClientConn.ReadFrom(buf)
		if err != nil {
			logger.Err("DRA: reading buffer failed")
			continue
		} else if bytesRead < 240 {
			// This is not dhcp packet as the minimum size is 240
			continue
		}
		//Decode the packet...
		DhcpRelayAgentDecodeInPkt(buf, bytesRead)
		//logger.Info(fmt.Sprintln("DRA: bytesread is ", bytesRead))
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		logger.Info(fmt.Sprintln("DRA: srcAddr is ", srcAddr))
		//logger.Info(fmt.Sprintln("DRA: buffer is ", buf))
		// Send Packet
		DhcpRelayAgentSendPacketToDhcpServer(cm, buf)
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
