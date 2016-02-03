package relayServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/ipv4"
	"net"
	"strconv"
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
		dup, ok := doptions[DhcpOptionCode(opts[0])]
		if ok {
			logger.Info(fmt.Sprintln("DRA: jgheewala entry already exists", dup))
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

// SelectOrder returns a slice of options ordered and selected by a byte array
// usually defined by OptionParameterRequestList.  This result is expected to be
// used in ReplyPacket()'s []Option parameter.
func (o DhcpRelayAgentOptions) SelectOrder(order []byte) []Option {
	opts := make([]Option, 0, len(order))
	for _, v := range order {
		if data, ok := o[DhcpOptionCode(v)]; ok {
			opts = append(opts, Option{Code: DhcpOptionCode(v),
				Value: data})
		}
	}
	return opts
}

// SelectOrderOrAll has same functionality as SelectOrder, except if the order
// param is nil, whereby all options are added (in arbitary order).
func (o DhcpRelayAgentOptions) SelectOrderOrAll(order []byte) []Option {
	if order == nil {
		opts := make([]Option, 0, len(o))
		for i, v := range o {
			opts = append(opts, Option{Code: i, Value: v})
		}
		return opts
	}
	return o.SelectOrder(order)
}

func (p *DhcpRelayAgentPacket) CopyDhcpOptions(value []byte) {

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
	logger.Info(fmt.Sprintln("DRA: mtype is", mType, "mtype [0] is ", MessageType(mType[0])))
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

func DhcpRelayAgentAddOptionsToPacket(reqOptions DhcpRelayAgentOptions, mt MessageType,
	outPacket *DhcpRelayAgentPacket) {
	outPacket.AddDhcpOptions(OptionDHCPMessageType, []byte{byte(mt)})
	var dummyDup map[DhcpOptionCode]int
	dummyDup = make(map[DhcpOptionCode]int, len(reqOptions))
	for i := 0; i < len(reqOptions); i++ {
		opt := reqOptions.SelectOrderOrAll(reqOptions[DhcpOptionCode(i)])
		for _, option := range opt {
			_, ok := dummyDup[option.Code]
			if ok {
				logger.Err(fmt.Sprintln("DRA: jgheewala duplicate entry",
					option.Code))
				continue
			}
			outPacket.AddDhcpOptions(option.Code, option.Value)
			dummyDup[option.Code] = 9999
		}
	}
}
func DhcpRelayAgentSendPacketToDhcpServer(ch *net.UDPConn, gblEntry DhcpRelayAgentGlobalInfo,
	inReq DhcpRelayAgentPacket, reqOptions DhcpRelayAgentOptions,
	mt MessageType) {
	logger.Info("DRA: Creating Send Pkt client ----> server")

	serverIpPort := gblEntry.IntfConfig.ServerIp + ":" + strconv.Itoa(DHCP_SERVER_PORT)
	logger.Info("DRA: Sending DHCP PACKET to server: " + serverIpPort)
	serverAddr, err := net.ResolveUDPAddr("udp", serverIpPort)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: couldn't resolved udp addr for and err is", err))
		return
	}
	outPacket := DhcpRelayAgentCreateNewPacket(Request, inReq)
	outPacket.SetGIAddr(net.ParseIP(gblEntry.IntfConfig.IpSubnet))
	DhcpRelayAgentAddOptionsToPacket(reqOptions, mt, &outPacket)
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

func DhcpRelayAgentSendPacketToDhcpClient(gblEntry DhcpRelayAgentGlobalInfo,
	logicalId int, inReq DhcpRelayAgentPacket,
	reqOptions DhcpRelayAgentOptions, mt MessageType) {

	// Get the interface from reverse mapping to send the unicast
	// packet...
	linuxInterface, ok := dhcprelayReverseMap[inReq.GetCHAddr().String()]
	if !ok {
		logger.Err("DRA: rever map didn't cache linux interface for " +
			inReq.GetCHAddr().String())
		return
	}
	logger.Info(fmt.Sprintln("DRA: using cached linuxInterface",
		linuxInterface))
	logger.Info("DRA: Creating Payload server -----> client")
	outPacket := DhcpRelayAgentCreateNewPacket(Reply, inReq)
	DhcpRelayAgentAddOptionsToPacket(reqOptions, mt, &outPacket)
	// subnet ip is the interface ip address copy the message...
	// by creating new packet Decode outpacket...
	logger.Info("DRA: Decoding out pkt for client")
	logger.Info("DRA: CIAddr is " + outPacket.GetCIAddr().String())
	logger.Info("DRA: CHaddr is " + outPacket.GetCHAddr().String())
	logger.Info("DRA: YIAddr is " + outPacket.GetYIAddr().String())
	logger.Info("DRA: GIAddr is " + outPacket.GetGIAddr().String())
	logger.Info(fmt.Sprintln("DRA: Cookie is ", outPacket.GetCookie()))
	outPacket.PadToMinSize()
	logger.Info("DRA: Creating go packet server ------> client")
	eth := &layers.Ethernet{
		SrcMAC:       linuxInterface.HardwareAddr,
		DstMAC:       outPacket.GetCHAddr(),
		EthernetType: layers.EthernetTypeIPv4,
	}
	logger.Info(fmt.Sprintln("DRA: ethernet info:", eth))
	ipv4 := &layers.IPv4{
		SrcIP:    net.ParseIP(gblEntry.IntfConfig.IpSubnet),
		DstIP:    outPacket.GetYIAddr(),
		Version:  4,
		Protocol: layers.IPProtocolUDP,
		TTL:      64,
	}
	logger.Info(fmt.Sprintln("DRA: ipv4 info:", ipv4))
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(DHCP_SERVER_PORT),
		DstPort: layers.UDPPort(DHCP_CLIENT_PORT),
	}
	udp.SetNetworkLayerForChecksum(ipv4)
	logger.Info(fmt.Sprintln("DRA: udp info:", udp))

	goOpts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	buffer := gopacket.NewSerializeBuffer()
	gopacket.SerializeLayers(buffer, goOpts, eth, ipv4, udp, gopacket.Payload(outPacket))
	var pHandle *pcap.Handle
	var err error
	if gblEntry.PcapHandle == nil {
		logger.Info(fmt.Sprintln("DRA: opening pcap handle for", linuxInterface.Name))
		pHandle, err = pcap.OpenLive(linuxInterface.Name, snapshot_len,
			promiscuous, timeout)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: opening pcap for",
				linuxInterface.Name, " failed with Error:", err))
			return
		}
		gblEntry.PcapHandle = pHandle
		dhcprelayGblInfo[logicalId] = gblEntry
	} else {
		pHandle = gblEntry.PcapHandle
	}

	if gblEntry.PcapHandle == nil {
		logger.Info("DRA: jgheewala...pcap handler is nul....")
	}
	err = pHandle.WritePacketData(buffer.Bytes())
	if err != nil {
		logger.Info(fmt.Sprintln("DRA: WritePacketData failed with error:", err))
		return
	}

	logger.Info(fmt.Sprintln("DRA: Create & Send of PKT successfully to client"))
}

func DhcpRelayAgentSendPacket(clientHandler *net.UDPConn, cm *ipv4.ControlMessage,
	inReq DhcpRelayAgentPacket, reqOptions DhcpRelayAgentOptions, mType MessageType) {
	logicalId, ok := dhcprelayLogicalIntfId2LinuxIntId[cm.IfIndex]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: linux id", cm.IfIndex,
			" has no mapping...drop packet"))
		return
	}
	// Use obtained logical id to find the global interface object
	logger.Info(fmt.Sprintln("DRA: linux id ----> logical id is success for", logicalId))
	gblEntry, ok := dhcprelayGblInfo[logicalId]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: is dra enabled on if_index ????",
			logicalId))
		logger.Err("DRA: not sending packet.... :(")
		return
	}
	switch mType {
	case 1, 3, 4, 7, 8:
		// Updating reverse mapping with logical interface id
		linuxInterface, err := net.InterfaceByIndex(cm.IfIndex)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting interface by id failed", err))
			return
		}
		dhcprelayReverseMap[inReq.GetCHAddr().String()] = linuxInterface
		logger.Info(fmt.Sprintln("DRA: cached linux interface is",
			linuxInterface))
		// get logical interface id from linux id...
		// Send Packet
		DhcpRelayAgentSendPacketToDhcpServer(clientHandler, gblEntry,
			inReq, reqOptions, mType)
		break
	case 2, 5, 6:
		DhcpRelayAgentSendPacketToDhcpClient(gblEntry, logicalId, inReq,
			reqOptions, mType)
		break
	default:
		logger.Info("DRA: any new message type")
	}

}

func DhcpRelayAgentReceiveDhcpPkt(clientHandler *net.UDPConn) {
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
		logger.Info(fmt.Sprintln("DRA: Received Packet from ", srcAddr))
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		//Decode the packet...
		inReq, reqOptions, mType := DhcpRelayAgentDecodeInPkt(buf, bytesRead)
		if inReq == nil || reqOptions == nil {
			logger.Warning("DRA: Couldn't decode dhcp packet...continue")
			continue
		}

		// Based on Packet type decide whether to send packet to server
		// or to client
		logger.Info(fmt.Sprintln("DRA: mtype is", mType))
		DhcpRelayAgentSendPacket(clientHandler, cm, inReq, reqOptions,
			mType)
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
	dhcprelayReverseMap = make(map[string]*net.Interface, 30)
	go DhcpRelayAgentReceiveDhcpPkt(dhcprelayClientHandler)
}
