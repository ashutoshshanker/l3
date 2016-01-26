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
 * ========================GET API's FOR ABOVE MESSAGE FORMAT==================
 */
func (p DhcpRelayAgentPacket) GetHeaderLen() byte {
	return p[2]
}

func (p DhcpRelayAgentPacket) GetOpCode() OpCode {
	return OpCode(p[0])
}
func (p DhcpRelayAgentPacket) GetHeaderType() byte {
	return p[1]
}
func (p DhcpRelayAgentPacket) GetHops() byte {
	return p[3]
}
func (p DhcpRelayAgentPacket) GetXId() []byte {
	return p[4:8]
}
func (p DhcpRelayAgentPacket) GetSecs() []byte {
	return p[8:10]
}
func (p DhcpRelayAgentPacket) GetFlags() []byte {
	return p[10:12]
}
func (p DhcpRelayAgentPacket) GetCIAddr() net.IP {
	return net.IP(p[12:16])
}
func (p DhcpRelayAgentPacket) GetYIAddr() net.IP {
	return net.IP(p[16:20])
}
func (p DhcpRelayAgentPacket) GetSIAddr() net.IP {
	return net.IP(p[20:24])
}
func (p DhcpRelayAgentPacket) GetGIAddr() net.IP {
	return net.IP(p[24:28])
}
func (p DhcpRelayAgentPacket) GetCHAddr() net.HardwareAddr {
	hLen := p.GetHeaderLen()
	if hLen > DHCP_PACKET_HEADER_SIZE { // Prevent chaddr exceeding p boundary
		hLen = DHCP_PACKET_HEADER_SIZE
	}
	return net.HardwareAddr(p[28 : 28+hLen]) // max endPos 44
}
func (p DhcpRelayAgentPacket) GetCookie() []byte {
	return p[236:240]
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

/*
 * ========================SET API's FOR ABOVE MESSAGE FORMAT==================
 */
func (p DhcpRelayAgentPacket) SetOpCode(c OpCode) {
	p[0] = byte(c)
}

func (p DhcpRelayAgentPacket) SetCHAddr(a net.HardwareAddr) {
	copy(p[28:44], a)
	p[2] = byte(len(a))
}

func (p DhcpRelayAgentPacket) SetHType(hType byte) {
	p[1] = hType
}

func (p DhcpRelayAgentPacket) SetCookie(cookie []byte) {
	copy(p.GetCookie(), cookie)
}

func (p DhcpRelayAgentPacket) SetHops(hops byte) {
	p[3] = hops
}

func (p DhcpRelayAgentPacket) SetXId(xId []byte) {
	copy(p.GetXId(), xId)
}

func (p DhcpRelayAgentPacket) SetSecs(secs []byte) {
	copy(p.GetSecs(), secs)
}

func (p DhcpRelayAgentPacket) SetFlags(flags []byte) {
	copy(p.GetFlags(), flags)
}

func (p DhcpRelayAgentPacket) SetCIAddr(ip net.IP) {
	copy(p.GetCIAddr(), ip.To4())
}

func (p DhcpRelayAgentPacket) SetYIAddr(ip net.IP) {
	copy(p.GetYIAddr(), ip.To4())
}

func (p DhcpRelayAgentPacket) SetSIAddr(ip net.IP) {
	copy(p.GetSIAddr(), ip.To4())
}

func (p DhcpRelayAgentPacket) SetGIAddr(ip net.IP) {
	copy(p.GetGIAddr(), ip.To4())
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
func DhcpRelayAgentDecodeInPkt(data []byte, bytesRead int) (DhcpRelayAgentPacket,
	DhcpRelayAgentOptions) {
	logger.Info(fmt.Sprintln("DRA: Decoding PKT"))
	inRequest := DhcpRelayAgentPacket(data[:bytesRead])
	if inRequest.GetHeaderLen() > DHCP_PACKET_HEADER_SIZE {
		logger.Warning("Header Lenght is invalid... don't do anything")
		return nil, nil
	}
	reqOptions := inRequest.ParseDhcpOptions()
	logger.Info("DRA: CIAddr is " + inRequest.GetCIAddr().String())
	logger.Info("DRA: CHaddr is " + inRequest.GetCHAddr().String())
	logger.Info("DRA: YIAddr is " + inRequest.GetYIAddr().String())
	logger.Info("DRA: GIAddr is " + inRequest.GetGIAddr().String())
	logger.Info(fmt.Sprintln("DRA: Cookie is ", inRequest.GetCookie()))
	mType := reqOptions[OptionDHCPMessageType]
	ParseMessageTypeToString(MessageType(mType[0]))
	logger.Info(fmt.Sprintln("DRA: Decoding of Pkt done"))

	return inRequest, reqOptions
}

/*
 * API to create a new Dhcp packet with Relay Agent information in it
 */
func DhcpRelayAgentCreateNewPacket(opCode OpCode) DhcpRelayAgentPacket {
	p := make(DhcpRelayAgentPacket, DHCP_PACKET_OPTIONS_LEN+1) //241
	//p.SetOpCode(opCode)
	//p.SetHType(1) // Ethernet
	//p.SetCookie([]byte{99, 130, 83, 99}) @TODO: do we want to set
	//cookies???
	p[DHCP_PACKET_OPTIONS_LEN] = byte(End) // set opcode END at the very last
	return p
}

func DhcpRelayAgentSendPacketToDhcpServer(controlMessage *ipv4.ControlMessage,
	data []byte, inReq DhcpRelayAgentPacket, reqOptions DhcpRelayAgentOptions) {
	logger.Info("DRA: Creating Send Pkt")
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
		inReq, reqOptions := DhcpRelayAgentDecodeInPkt(buf, bytesRead)
		if inReq == nil || reqOptions == nil {
			logger.Warning("Couldn't decode dhcp packet...continue")
			continue
		}
		//logger.Info(fmt.Sprintln("DRA: bytesread is ", bytesRead))
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		logger.Info(fmt.Sprintln("DRA: srcAddr is ", srcAddr))
		//logger.Info(fmt.Sprintln("DRA: buffer is ", buf))
		// Send Packet
		DhcpRelayAgentSendPacketToDhcpServer(cm, buf, inReq, reqOptions)
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
