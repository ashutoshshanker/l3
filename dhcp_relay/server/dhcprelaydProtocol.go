// Dhcp Relay Agent Protocol Handling for Packet Send/Receive
package relayServer

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
	"strconv"
	_ "sync"
)

// Dhcp OpCodes Types
const (
	Request OpCode = 1 // From Client
	Reply   OpCode = 2 // From Server
)

// DHCP Packet global constants
const DHCP_PACKET_MIN_SIZE = 300 // 272?????? @TODO: fixme....jgheewala
const DHCP_PACKET_HEADER_SIZE = 16
const DHCP_PACKET_MIN_BYTES = 240
const DHCP_SERVER_PORT = 67
const DHCP_CLIENT_PORT = 68

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

func UtiltrimNull(d []byte) []byte {
	for i, v := range d {
		if v == 0 {
			return d[:i]
		}
	}
	return d
}
func (p DhcpRelayAgentPacket) GetCookie() []byte {
	return p[236:240]
}

// BOOTP legacy
func (p DhcpRelayAgentPacket) GetSName() []byte {
	return UtiltrimNull(p[44:108])
}

// BOOTP legacy
func (p DhcpRelayAgentPacket) GetFile() []byte {
	return UtiltrimNull(p[108:236])
}

func ParseMessageTypeToString(mtype MessageType) string {
	switch mtype {
	case 1:
		logger.Info("DRA: Message Type: DhcpDiscover")
		return "DHCPDISCOVER"
	case 2:
		logger.Info("DRA: Message Type: DhcpOffer")
		return "DHCPOFFER"
	case 3:
		logger.Info("DRA: Message Type: DhcpRequest")
		return "DHCPREQUEST"
	case 4:
		logger.Info("DRA: Message Type: DhcpDecline")
		return "DHCPDECLINE"
	case 5:
		logger.Info("DRA: Message Type: DhcpACK")
		return "DHCPACK"
	case 6:
		logger.Info("DRA: Message Type: DhcpNAK")
		return "DHCPNAK"
	case 7:
		logger.Info("DRA: Message Type: DhcpRelease")
		return "DHCPRELEASE"
	case 8:
		logger.Info("DRA: Message Type: DhcpInform")
		return "DHCPINFORM"
	default:
		logger.Info("DRA: Message Type: UnKnown...Discard the Packet")
		return "UNKNOWN REQUEST TYPE"
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

func (p DhcpRelayAgentPacket) SetHeaderType(hType byte) {
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

// BOOTP legacy
func (p DhcpRelayAgentPacket) SetSName(sName []byte) {
	copy(p[44:108], sName)
	if len(sName) < 64 {
		p[44+len(sName)] = 0
	}
}

// BOOTP legacy
func (p DhcpRelayAgentPacket) SetFile(file []byte) {
	copy(p[108:236], file)
	if len(file) < 128 {
		p[108+len(file)] = 0
	}
}

func (p DhcpRelayAgentPacket) AllocateOptions() []byte {
	if len(p) > DHCP_PACKET_MIN_BYTES {
		return p[DHCP_PACKET_MIN_BYTES:]
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

// Appends a DHCP option to the end of a packet
func (p *DhcpRelayAgentPacket) AddDhcpOptions(op DhcpOptionCode, value []byte) {
	// Strip off End, Add OptionCode and Length
	*p = append((*p)[:len(*p)-1], []byte{byte(op), byte(len(value))}...)
	*p = append(*p, value...)  // Add Option Value
	*p = append(*p, byte(End)) // Add on new End
}

/*========================= END OF HELPER FUNCTION ===========================*/
/*
 * APT to decode incoming Packet by converting the byte into DHCP packet format
 */
func DhcpRelayAgentDecodeInPkt(data []byte, bytesRead int) (DhcpRelayAgentPacket,
	DhcpRelayAgentOptions, MessageType) {
	logger.Info(fmt.Sprintln("DRA: Decoding PKT"))
	inRequest := DhcpRelayAgentPacket(data[:bytesRead])
	if inRequest.GetHeaderLen() > DHCP_PACKET_HEADER_SIZE {
		logger.Warning("Header Lenght is invalid... don't do anything")
		return nil, nil, 0
	}
	reqOptions := inRequest.ParseDhcpOptions()
	logger.Info("DRA: CIAddr is " + inRequest.GetCIAddr().String())
	logger.Info("DRA: CHaddr is " + inRequest.GetCHAddr().String())
	logger.Info("DRA: YIAddr is " + inRequest.GetYIAddr().String())
	logger.Info("DRA: GIAddr is " + inRequest.GetGIAddr().String())
	logger.Info(fmt.Sprintln("DRA: Cookie is ", inRequest.GetCookie()))
	mType := reqOptions[OptionDHCPMessageType]
	//	mString := ParseMessageTypeToString(MessageType(mType[0]))
	logger.Info(fmt.Sprintln("DRA: Decoding of Pkt done"))

	return inRequest, reqOptions, MessageType(mType[0])
}

/*
 * API to create a new Dhcp packet with Relay Agent information in it
 */
func DhcpRelayAgentCreateNewPacket(opCode OpCode, inReq DhcpRelayAgentPacket) DhcpRelayAgentPacket {
	p := make(DhcpRelayAgentPacket, DHCP_PACKET_MIN_BYTES+1) //241
	p.SetHeaderType(inReq.GetHeaderType())                   // Ethernet
	p.SetCookie(inReq.GetCookie())                           // copy cookie from original pkt
	p.SetOpCode(opCode)                                      // opcode can be request or reply
	p.SetXId(inReq.GetXId())                                 // copy from org pkt
	p.SetFlags(inReq.GetFlags())                             // copy from org pkt
	p.SetYIAddr(inReq.GetYIAddr())                           // copy from org pkt
	p.SetCHAddr(inReq.GetCHAddr())                           // copy from org pkt
	p.SetSecs(inReq.GetSecs())                               // copy from org pkt
	p.SetSName(inReq.GetSName())                             // copy from org pkt
	p.SetFile(inReq.GetFile())                               // copy from org pkt
	p[DHCP_PACKET_MIN_BYTES] = byte(End)                     // set opcode END at the very last
	return p
}

func DhcpRelayAgentSendPacketToDhcpServer(ch *net.UDPConn, controlMessage *ipv4.ControlMessage,
	data []byte, inReq DhcpRelayAgentPacket, reqOptions DhcpRelayAgentOptions,
	mt MessageType) {
	logger.Info("DRA: Creating Send Pkt")

	// get logical interface id from linux id...
	logicalId, ok := dhcprelayLogicalIntfId2LinuxIntId[controlMessage.IfIndex]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: linux id", controlMessage.IfIndex,
			" has no mapping...did we miss any notification?"))
		return
	}
	// Use obtained logical id to find the global interface object
	logger.Info(fmt.Sprintln("DRA: linux id ----> logical id is success for", logicalId))
	//gblEntry, ok := dhcprelayGblInfo[controlMessage.IfIndex]
	gblEntry, ok := dhcprelayGblInfo[logicalId]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: is dra enabled on if_index ????",
			logicalId))
		logger.Err("DRA: not sending packet.... :(")
		return
	}
	serverIpPort := gblEntry.IntfConfig.ServerIp + ":" + strconv.Itoa(DHCP_SERVER_PORT)
	logger.Info("DRA: Sending DHCP PACKET to server: " + serverIpPort)
	serverAddr, err := net.ResolveUDPAddr("udp", serverIpPort)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: couldn't resolved udp addr for and err is", err))
		return
	}
	outPacket := DhcpRelayAgentCreateNewPacket(Request, inReq)
	outPacket.SetGIAddr(net.ParseIP(gblEntry.IntfConfig.IpSubnet))
	outPacket.AddDhcpOptions(OptionDHCPMessageType, []byte{byte(mt)})

	// Decode outpacket...
	logger.Info("DRA: Decoding out pkt for server")
	logger.Info("DRA: CIAddr is " + outPacket.GetCIAddr().String())
	logger.Info("DRA: CHaddr is " + outPacket.GetCHAddr().String())
	logger.Info("DRA: YIAddr is " + outPacket.GetYIAddr().String())
	logger.Info("DRA: GIAddr is " + outPacket.GetGIAddr().String())
	logger.Info(fmt.Sprintln("DRA: Cookie is ", outPacket.GetCookie()))
	outPacket.PadToMinSize()
	_, err = ch.WriteToUDP(outPacket, serverAddr)
	if err != nil {
		logger.Info(fmt.Sprintln("DRA: WriteToUDP failed with error:", err))
		return
	}
	logger.Info(fmt.Sprintln("DRA: Create & Send of PKT successfully to server"))
}

func DhcpRelayAgentReceiveDhcpPktFromClient(clientHandler *net.UDPConn) {
	var buf []byte = make([]byte, 1500)
	for {
		bytesRead, cm, srcAddr, err := dhcprelayClientConn.ReadFrom(buf)
		if err != nil {
			logger.Err("DRA: reading buffer failed")
			continue
		} else if bytesRead < DHCP_PACKET_MIN_BYTES {
			// This is not dhcp packet as the minimum size is 240
			continue
		}
		logger.Info("DRA: Received PACKET FROM CLIENT")
		//Decode the packet...
		inReq, reqOptions, mType := DhcpRelayAgentDecodeInPkt(buf, bytesRead)
		if inReq == nil || reqOptions == nil {
			logger.Warning("DRA: Couldn't decode dhcp packet...continue")
			continue
		}
		// Updating reverse mapping
		dhcprelayReverseMap[inReq.GetCHAddr().String()] = cm.IfIndex
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		logger.Info(fmt.Sprintln("DRA: srcAddr is ", srcAddr))

		// Send Packet
		DhcpRelayAgentSendPacketToDhcpServer(clientHandler, cm, buf, inReq, reqOptions, mType)
	}
}

func DhcpRelayAgentSendPacketToDhcpClient(ch *net.UDPConn, controlMessage *ipv4.ControlMessage,
	inReq DhcpRelayAgentPacket, reqOptions DhcpRelayAgentOptions,
	gblEntry DhcpRelayAgentGlobalInfo, mt MessageType) {

	clientIpPort := gblEntry.IntfConfig.IpSubnet + ":" + strconv.Itoa(DHCP_CLIENT_PORT)
	logger.Info("DRA: Sending DHCP PACKET to client: " + clientIpPort)
	clientAddr, err := net.ResolveUDPAddr("udp", clientIpPort)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: couldn't resolved udp addr for and err is", err))
		return
	}
	outPacket := DhcpRelayAgentCreateNewPacket(Reply, inReq)
	// subnet ip is the interface ip address
	// copy the message... by creating new packet
	// Decode outpacket...
	logger.Info("DRA: Decoding out pkt for client")
	logger.Info("DRA: CIAddr is " + outPacket.GetCIAddr().String())
	logger.Info("DRA: CHaddr is " + outPacket.GetCHAddr().String())
	logger.Info("DRA: YIAddr is " + outPacket.GetYIAddr().String())
	logger.Info("DRA: GIAddr is " + outPacket.GetGIAddr().String())
	logger.Info(fmt.Sprintln("DRA: Cookie is ", outPacket.GetCookie()))
	outPacket.PadToMinSize()
	_, err = ch.WriteToUDP(outPacket, clientAddr)
	if err != nil {
		logger.Info(fmt.Sprintln("DRA: WriteToUDP failed with error:", err))
		return
	}
	logger.Info(fmt.Sprintln("DRA: Create & Send of PKT successfully to client"))
}

func DhcpRelayAgentReceiveDhcpPktFromServer(serverHandler *net.UDPConn) {
	var buf []byte = make([]byte, 1500)
	for {
		bytesRead, cm, srcAddr, err := dhcprelayServerConn.ReadFrom(buf)
		if err != nil {
			logger.Err("DRA: reading buffer failed")
			continue
		}
		logger.Info("DRA: Received PACKET FROM SERVER")
		inReq, reqOptions, mType := DhcpRelayAgentDecodeInPkt(buf, bytesRead)
		if inReq == nil || reqOptions == nil {
			logger.Warning("DRA: Couldn't decode dhcp packet....continue")
			continue
		}
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		logger.Info(fmt.Sprintln("DRA: srcAddr is ", srcAddr))
		logger.Info(fmt.Sprintln("DRA: MessageType is ", mType))
		// Get the interface from reverse mapping to send the unicast
		// packet...
		outIfId := dhcprelayReverseMap[inReq.GetCHAddr().String()]
		logger.Info(fmt.Sprintln("DRA: Send unicast packet to Interface Id:", outIfId))
		gblEntry, ok := dhcprelayGblInfo[outIfId]
		if !ok {
			// dropping the packet??
			logger.Err(fmt.Sprintln("DRA: dra is not enable on", outIfId, "??"))
			continue
		}
		DhcpRelayAgentSendPacketToDhcpClient(serverHandler, cm, inReq, reqOptions, gblEntry, mType)
	}
}

func DhcpRelayAgentCreateClientServerConn() {

	// Client send dhcp packet from port 68 to server port 67
	// So create a filter for udp:67 for messages send out by client to
	// server
	logger.Info("DRA: creating listenPacket for udp port 67")
	saddr := net.UDPAddr{
		Port: DHCP_SERVER_PORT,
		IP:   net.ParseIP(""),
	}
	dhcprelayClientHandler, err := net.ListenUDP("udp", &saddr)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Opening udp port for client --> server failed", err))
		return
	}
	dhcprelayClientConn = ipv4.NewPacketConn(dhcprelayClientHandler)
	controlFlag := ipv4.FlagTTL | ipv4.FlagSrc | ipv4.FlagDst | ipv4.FlagInterface
	err = dhcprelayClientConn.SetControlMessage(controlFlag, true)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Setting control flag for client failed..", err))
		return
	}
	logger.Info("DRA: Client Connection opened successfully")
	dhcprelayReverseMap = make(map[string]int, 30)
	go DhcpRelayAgentReceiveDhcpPktFromClient(dhcprelayClientHandler)

	// Server sends dhcp packet from port 67 to client port 68
	// so create a filter for udp:68 for message coming from server
	logger.Info("DRA: creating listenPacket for udp port 68")
	caddr := net.UDPAddr{
		Port: DHCP_CLIENT_PORT,
		IP:   net.ParseIP(""),
	}
	dhcprelayServerHandler, err := net.ListenUDP("udp", &caddr)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Opening udp port for server --> client failed", err))
		// do we need to close the client server communication??? ask
		// Hari/Adam
		return
	}
	dhcprelayServerConn = ipv4.NewPacketConn(dhcprelayServerHandler)
	err = dhcprelayServerConn.SetControlMessage(controlFlag, true)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA:Setting control flag for server failed..", err))
		return
	}
	logger.Info("DRA: Server Connection opened successfully")
	go DhcpRelayAgentReceiveDhcpPktFromServer(dhcprelayServerHandler)
}
