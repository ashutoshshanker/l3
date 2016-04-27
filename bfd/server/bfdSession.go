package server

import (
	"asicd/asicdCommonDefs"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"math/rand"
	"net"
	"runtime"
	"strconv"
	"time"
	"utils/commonDefs"
)

func (server *BFDServer) StartSessionHandler() error {
	server.CreateSessionCh = make(chan BfdSessionMgmt)
	server.DeleteSessionCh = make(chan BfdSessionMgmt)
	server.AdminUpSessionCh = make(chan BfdSessionMgmt)
	server.AdminDownSessionCh = make(chan BfdSessionMgmt)
	server.CreatedSessionCh = make(chan int32)
	server.FailedSessionClientCh = make(chan int32)
	go server.StartBfdSesionServer()
	go server.StartBfdSesionServerQueuer()
	go server.StartBfdSessionRxTx()
	go server.StartSessionRetryHandler()
	for {
		select {
		case sessionMgmt := <-server.CreateSessionCh:
			server.CreateBfdSession(sessionMgmt)
		case sessionMgmt := <-server.DeleteSessionCh:
			server.DeleteBfdSession(sessionMgmt)
		case sessionMgmt := <-server.AdminUpSessionCh:
			server.AdminUpBfdSession(sessionMgmt)
		case sessionMgmt := <-server.AdminDownSessionCh:
			server.AdminDownBfdSession(sessionMgmt)

		}
	}
	return nil
}

func (server *BFDServer) DispatchReceivedBfdPacket(ipAddr string, bfdPacket *BfdControlPacket) error {
	var session *BfdSession
	sessionId := int32(bfdPacket.YourDiscriminator)
	if sessionId == 0 {
		for _, session = range server.bfdGlobal.Sessions {
			if session.state.IpAddr == ipAddr {
				break
			}
		}
	} else {
		session = server.bfdGlobal.Sessions[sessionId]
	}
	if session != nil {
		session.ReceivedPacketCh <- bfdPacket
	}
	return nil
}

func (server *BFDServer) StartBfdSesionServerQueuer() error {
	server.BfdPacketRecvCh = make(chan RecvedBfdPacket, 10)
	for {
		select {
		case packet := <-server.BfdPacketRecvCh:
			ip := packet.IpAddr
			len := packet.Len
			buf := packet.PacketBuf
			if len >= DEFAULT_CONTROL_PACKET_LEN {
				bfdPacket, err := DecodeBfdControlPacket(buf[0:len])
				if err != nil {
					server.logger.Info(fmt.Sprintln("Failed to decode packet - ", err))
				} else {
					err = server.DispatchReceivedBfdPacket(ip, bfdPacket)
					if err != nil {
						server.logger.Info(fmt.Sprintln("Failed to dispatch received packet"))
					}
				}
			}
		}
	}
}

func (server *BFDServer) StartBfdSesionServer() error {
	destAddr := ":" + strconv.Itoa(DEST_PORT)
	ServerAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed ResolveUDPAddr ", destAddr, err))
		return err
	}
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed ListenUDP ", err))
		return err
	}
	defer ServerConn.Close()
	buf := make([]byte, 1024)
	server.logger.Info(fmt.Sprintln("Started BFD session server on ", destAddr))
	for {
		len, uda, err := ServerConn.ReadFromUDP(buf)
		if err != nil {
			server.logger.Info(fmt.Sprintln("Failed to read from ", ServerAddr))
		} else {
			packet := RecvedBfdPacket{
				IpAddr:    uda.IP.String(),
				Len:       int32(len),
				PacketBuf: buf[0:len],
			}
			server.BfdPacketRecvCh <- packet
		}
	}
	return nil
}

func (server *BFDServer) StartBfdSessionRxTx() error {
	for {
		select {
		case createdSessionId := <-server.CreatedSessionCh:
			session := server.bfdGlobal.Sessions[createdSessionId]
			if session != nil {
				session.SessionStopClientCh = make(chan bool)
				session.ReceivedPacketCh = make(chan *BfdControlPacket, 10)
				if session.state.PerLinkSession {
					//server.logger.Info(fmt.Sprintln("Starting PerLink server for session ", createdSessionId))
					go session.StartPerLinkSessionServer(server)
					server.logger.Info(fmt.Sprintln("Starting PerLink client for session ", createdSessionId))
					go session.StartPerLinkSessionClient(server)
				} else {
					//server.logger.Info(fmt.Sprintln("Starting server for session ", createdSessionId))
					go session.StartSessionServer()
					server.logger.Info(fmt.Sprintln("Starting client for session ", createdSessionId))
					go session.StartSessionClient(server)
				}
				session.isClientActive = true
			} else {
				server.logger.Info(fmt.Sprintf("Bfd session could not be initiated for ", createdSessionId))
			}
		case failedClientSessionId := <-server.FailedSessionClientCh:
			session := server.bfdGlobal.Sessions[failedClientSessionId]
			if session != nil {
				session.isClientActive = false
				server.bfdGlobal.InactiveSessionsIdSlice = append(server.bfdGlobal.InactiveSessionsIdSlice, failedClientSessionId)
			}
		}
	}
	return nil
}

func (server *BFDServer) StartSessionRetryHandler() error {
	server.logger.Info("Starting session retry handler")
	retryTimer := time.NewTicker(time.Second * 5)
	for t := range retryTimer.C {
		_ = t
		for i := 0; i < len(server.bfdGlobal.InactiveSessionsIdSlice); i++ {
			if i%10 == 0 {
				runtime.Gosched()
			}
			sessionId := server.bfdGlobal.InactiveSessionsIdSlice[i]
			session := server.bfdGlobal.Sessions[sessionId]
			if session != nil {
				if session.isClientActive == false {
					if session.state.PerLinkSession {
						server.logger.Info(fmt.Sprintln("Starting PerLink client for inactive session ", sessionId))
						go session.StartPerLinkSessionClient(server)
					} else {
						server.logger.Info(fmt.Sprintln("Starting client for inactive session ", sessionId))
						go session.StartSessionClient(server)
					}
					session.isClientActive = true
					server.bfdGlobal.InactiveSessionsIdSlice = append(server.bfdGlobal.InactiveSessionsIdSlice[:i], server.bfdGlobal.InactiveSessionsIdSlice[i+1:]...)
				}
			}
		}
	}
	server.logger.Info("Session retry handler exiting ...")
	return nil
}

func (server *BFDServer) processSessionConfig(sessionConfig SessionConfig) error {
	sessionMgmt := BfdSessionMgmt{
		DestIp:    sessionConfig.DestIp,
		ParamName: sessionConfig.ParamName,
		Interface: sessionConfig.Interface,
		Protocol:  sessionConfig.Protocol,
		PerLink:   sessionConfig.PerLink,
	}
	switch sessionConfig.Operation {
	case bfddCommonDefs.CREATE:
		server.CreateSessionCh <- sessionMgmt
	case bfddCommonDefs.DELETE:
		server.DeleteSessionCh <- sessionMgmt
	case bfddCommonDefs.ADMINUP:
		server.AdminUpSessionCh <- sessionMgmt
	case bfddCommonDefs.ADMINDOWN:
		server.AdminDownSessionCh <- sessionMgmt
	}
	return nil
}

func (server *BFDServer) SendAdminDownToAllNeighbors() error {
	for _, session := range server.bfdGlobal.Sessions {
		session.StopBfdSession()
	}
	return nil
}

func (server *BFDServer) SendDeleteToAllSessions() error {
	for _, session := range server.bfdGlobal.Sessions {
		session.SessionStopClientCh <- true
	}
	return nil
}

func (server *BFDServer) GetNewSessionId() int32 {
	var sessionIdUsed bool
	var sessionId int32
	sessionId = 0
	if server.bfdGlobal.NumSessions < MAX_NUM_SESSIONS {
		sessionIdUsed = true //By default assume the sessionId is already used.
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		for sessionIdUsed {
			sessionId = r1.Int31n(MAX_NUM_SESSIONS)
			if _, exist := server.bfdGlobal.Sessions[sessionId]; exist {
				server.logger.Info(fmt.Sprintln("GetNewSessionId: sessionId ", sessionId, " is in use, Generating a new one"))
			} else {
				if sessionId != 0 {
					sessionIdUsed = false
				}
			}
		}
	}
	return sessionId
}

func (server *BFDServer) GetIfIndexAndLocalIpFromDestIp(DestIp string) (int32, string) {
	reachabilityInfo, err := server.ribdClient.ClientHdl.GetRouteReachabilityInfo(DestIp)
	server.logger.Info(fmt.Sprintln("Reachability info ", reachabilityInfo))
	if err != nil || !reachabilityInfo.IsReachable {
		server.logger.Info(fmt.Sprintf("%s is not reachable", DestIp))
		return int32(0), ""
	}
	server.ribdClient.ClientHdl.TrackReachabilityStatus(DestIp, "BFD", "add")
	ifIndex := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(reachabilityInfo.NextHopIfIndex), int(reachabilityInfo.NextHopIfType))
	server.logger.Info(fmt.Sprintln("GetIfIndexAndLocalIpFromDestIp: DestIp: ", DestIp, "IfIndex: ", ifIndex))
	return ifIndex, reachabilityInfo.NextHopIp
}

func (server *BFDServer) GetTxJitter() int32 {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	jitter := r1.Int31n(TX_JITTER)
	return jitter
}

func (server *BFDServer) NewNormalBfdSession(IfIndex int32, DestIp string, ParamName string, PerLink bool, Protocol bfddCommonDefs.BfdSessionOwner) *BfdSession {
	bfdSession := &BfdSession{}
	sessionId := server.GetNewSessionId()
	if sessionId == 0 {
		server.logger.Info("Failed to get sessionId")
		return nil
	}
	bfdSession.state.SessionId = sessionId
	bfdSession.state.IpAddr = DestIp
	bfdSession.state.InterfaceId = IfIndex
	bfdSession.state.PerLinkSession = PerLink
	if PerLink {
		IfName, _ := server.getLinuxIntfName(IfIndex)
		bfdSession.state.LocalMacAddr, _ = server.getMacAddrFromIntfName(IfName)
		bfdSession.state.RemoteMacAddr, _ = net.ParseMAC(bfdDedicatedMac)
		bfdSession.useDedicatedMac = true
	}
	bfdSession.state.RegisteredProtocols = make([]bool, bfddCommonDefs.MAX_NUM_PROTOCOLS)
	bfdSession.state.RegisteredProtocols[Protocol] = true
	bfdSession.state.SessionState = STATE_DOWN
	bfdSession.state.RemoteSessionState = STATE_DOWN
	bfdSession.state.LocalDiscriminator = uint32(bfdSession.state.SessionId)
	bfdSession.state.LocalDiagType = DIAG_NONE
	bfdSession.txInterval = STARTUP_TX_INTERVAL / 1000
	bfdSession.txJitter = server.GetTxJitter()
	sessionParam, exist := server.bfdGlobal.SessionParams[ParamName]
	if exist {
		bfdSession.state.ParamName = ParamName
	} else {
		bfdSession.state.ParamName = "default"
		sessionParam, _ = server.bfdGlobal.SessionParams["default"]
	}
	sessionParam.state.NumSessions++
	bfdSession.rxInterval = (STARTUP_RX_INTERVAL * sessionParam.state.LocalMultiplier) / 1000
	bfdSession.state.DesiredMinTxInterval = sessionParam.state.DesiredMinTxInterval
	bfdSession.state.RequiredMinRxInterval = sessionParam.state.RequiredMinRxInterval
	bfdSession.state.DetectionMultiplier = sessionParam.state.LocalMultiplier
	bfdSession.state.DemandMode = sessionParam.state.DemandEnabled
	bfdSession.authEnabled = sessionParam.state.AuthenticationEnabled
	bfdSession.authType = AuthenticationType(sessionParam.state.AuthenticationType)
	bfdSession.authSeqNum = 1
	bfdSession.authKeyId = uint32(sessionParam.state.AuthenticationKeyId)
	bfdSession.authData = sessionParam.state.AuthenticationData
	bfdSession.paramConfigChanged = true
	bfdSession.server = server
	bfdSession.bfdPacket = NewBfdControlPacketDefault()
	server.bfdGlobal.Sessions[sessionId] = bfdSession
	server.bfdGlobal.NumSessions++
	server.bfdGlobal.SessionsIdSlice = append(server.bfdGlobal.SessionsIdSlice, sessionId)
	server.logger.Info(fmt.Sprintln("New session : ", sessionId, " created on : ", IfIndex))
	server.CreatedSessionCh <- sessionId
	return bfdSession
}

func (server *BFDServer) NewPerLinkBfdSessions(IfIndex int32, DestIp string, ParamName string, Protocol bfddCommonDefs.BfdSessionOwner) error {
	lag, exist := server.lagPropertyMap[IfIndex]
	if exist {
		for _, link := range lag.Links {
			bfdSession := server.NewNormalBfdSession(IfIndex, DestIp, ParamName, true, Protocol)
			if bfdSession == nil {
				server.logger.Info(fmt.Sprintln("Failed to create perlink session on ", link))
			}
		}
	} else {
		server.logger.Info(fmt.Sprintln("Unknown lag ", IfIndex, " can not create perlink sessions"))
	}
	return nil
}

func (server *BFDServer) NewBfdSession(DestIp string, ParamName string, Interface string, Protocol bfddCommonDefs.BfdSessionOwner, PerLink bool) *BfdSession {
	var IfType int
	IfIndex, _ := server.GetIfIndexAndLocalIpFromDestIp(DestIp)
	if IfIndex == 0 {
		server.logger.Info(fmt.Sprintln("RemoteIP ", DestIp, " is not reachable"))
		return nil
	} else {
		IfType = asicdCommonDefs.GetIntfTypeFromIfIndex(IfIndex)
		IfName, err := server.getLinuxIntfName(IfIndex)
		if err == nil {
			if Interface != "" && IfName != Interface {
				server.logger.Info(fmt.Sprintln("Bfd session to ", DestIp, " cannot be created on interface ", Interface))
				return nil
			}
		}
	}
	if IfType == commonDefs.IfTypeLag && PerLink {
		server.NewPerLinkBfdSessions(IfIndex, DestIp, ParamName, Protocol)
	} else {
		bfdSession := server.NewNormalBfdSession(IfIndex, DestIp, ParamName, false, Protocol)
		return bfdSession
	}
	return nil
}

func (server *BFDServer) UpdateBfdSessionsOnInterface(ifIndex int32) error {
	intf, exist := server.bfdGlobal.Interfaces[ifIndex]
	if exist {
		intfEnabled := intf.Enabled
		for _, session := range server.bfdGlobal.Sessions {
			if session.state.InterfaceId == ifIndex {
				session.state.DesiredMinTxInterval = intf.conf.DesiredMinTxInterval
				session.state.RequiredMinRxInterval = intf.conf.RequiredMinRxInterval
				session.state.DetectionMultiplier = intf.conf.LocalMultiplier
				session.state.DemandMode = intf.conf.DemandEnabled
				session.authEnabled = intf.conf.AuthenticationEnabled
				session.authType = AuthenticationType(intf.conf.AuthenticationType)
				session.authKeyId = uint32(intf.conf.AuthenticationKeyId)
				session.authData = intf.conf.AuthenticationData
				session.intfConfigChanged = true
				if intfEnabled {
					session.InitiatePollSequence()
				} else {
					session.StopBfdSession()
				}
			}
		}
	}
	return nil
}

func (server *BFDServer) UpdateBfdSessionsUsingParam(paramName string) error {
	sessionParam, paramExist := server.bfdGlobal.SessionParams[paramName]
	for _, session := range server.bfdGlobal.Sessions {
		if session.state.ParamName == paramName {
			if paramExist {
				session.state.DesiredMinTxInterval = sessionParam.state.DesiredMinTxInterval
				session.state.RequiredMinRxInterval = sessionParam.state.RequiredMinRxInterval
				session.state.DetectionMultiplier = sessionParam.state.LocalMultiplier
				session.state.DemandMode = sessionParam.state.DemandEnabled
				session.authEnabled = sessionParam.state.AuthenticationEnabled
				session.authType = AuthenticationType(sessionParam.state.AuthenticationType)
				session.authKeyId = uint32(sessionParam.state.AuthenticationKeyId)
				session.authData = sessionParam.state.AuthenticationData
			} else {
				session.state.DesiredMinTxInterval = DEFAULT_DESIRED_MIN_TX_INTERVAL
				session.state.RequiredMinRxInterval = DEFAULT_REQUIRED_MIN_RX_INTERVAL
				session.state.DetectionMultiplier = DEFAULT_DETECT_MULTI
				session.state.DemandMode = false
				session.authEnabled = false
			}
			session.paramConfigChanged = true
			session.InitiatePollSequence()
		}
	}
	return nil
}

func (server *BFDServer) FindBfdSession(DestIp string) (sessionId int32, found bool) {
	found = false
	for sessionId, session := range server.bfdGlobal.Sessions {
		if session.state.IpAddr == DestIp {
			return sessionId, true
		}
	}
	return sessionId, found
}

func NewBfdControlPacketDefault() *BfdControlPacket {
	bfdControlPacket := &BfdControlPacket{
		Version:    DEFAULT_BFD_VERSION,
		Diagnostic: DIAG_NONE,
		State:      STATE_DOWN,
		Poll:       false,
		Final:      false,
		ControlPlaneIndependent:   false,
		AuthPresent:               false,
		Demand:                    false,
		Multipoint:                false,
		DetectMult:                DEFAULT_DETECT_MULTI,
		MyDiscriminator:           0,
		YourDiscriminator:         0,
		DesiredMinTxInterval:      DEFAULT_DESIRED_MIN_TX_INTERVAL,
		RequiredMinRxInterval:     DEFAULT_REQUIRED_MIN_RX_INTERVAL,
		RequiredMinEchoRxInterval: DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL,
		AuthHeader:                nil,
	}
	return bfdControlPacket
}

// CreateBfdSession initializes a session and starts cpntrol packets exchange.
// This function is called when a protocol registers with BFD to monitor a destination IP.
func (server *BFDServer) CreateBfdSession(sessionMgmt BfdSessionMgmt) (*BfdSession, error) {
	var bfdSession *BfdSession
	DestIp := sessionMgmt.DestIp
	ParamName := sessionMgmt.ParamName
	Interface := sessionMgmt.Interface
	Protocol := sessionMgmt.Protocol
	PerLink := sessionMgmt.PerLink
	sessionId, found := server.FindBfdSession(DestIp)
	if !found {
		server.logger.Info(fmt.Sprintln("CreateSession ", DestIp, ParamName, Interface, Protocol, PerLink))
		bfdSession = server.NewBfdSession(DestIp, ParamName, Interface, Protocol, PerLink)
		if bfdSession != nil {
			server.logger.Info(fmt.Sprintln("Bfd session created ", bfdSession.state.SessionId, bfdSession.state.IpAddr))
		} else {
			server.logger.Info(fmt.Sprintln("CreateSession failed for ", DestIp, Protocol))
		}
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session already exists ", DestIp, Protocol, sessionId))
		bfdSession = server.bfdGlobal.Sessions[sessionId]
		if !bfdSession.state.RegisteredProtocols[Protocol] {
			bfdSession.state.RegisteredProtocols[Protocol] = true
		}
	}
	return bfdSession, nil
}

func (server *BFDServer) SessionDeleteHandler(session *BfdSession, Protocol bfddCommonDefs.BfdSessionOwner, ForceDel bool) error {
	var i int
	sessionId := session.state.SessionId
	session.state.RegisteredProtocols[Protocol] = false
	if ForceDel || session.CheckIfAnyProtocolRegistered() == false {
		session.txTimer.Stop()
		session.sessionTimer.Stop()
		session.SessionStopClientCh <- true
		server.bfdGlobal.SessionParams[session.state.ParamName].state.NumSessions--
		server.bfdGlobal.NumSessions--
		delete(server.bfdGlobal.Sessions, sessionId)
		for i = 0; i < len(server.bfdGlobal.SessionsIdSlice); i++ {
			if server.bfdGlobal.SessionsIdSlice[i] == sessionId {
				break
			}
		}
		server.bfdGlobal.SessionsIdSlice = append(server.bfdGlobal.SessionsIdSlice[:i], server.bfdGlobal.SessionsIdSlice[i+1:]...)
	}
	return nil
}

func (server *BFDServer) DeletePerLinkSessions(DestIp string, Protocol bfddCommonDefs.BfdSessionOwner, ForceDel bool) error {
	for _, session := range server.bfdGlobal.Sessions {
		if session.state.IpAddr == DestIp {
			server.SessionDeleteHandler(session, Protocol, ForceDel)
		}
	}
	return nil
}

// DeleteBfdSession ceases the session.
// A session down control packet is sent to BFD neighbor before deleting the session.
// This function is called when a protocol decides to stop monitoring the destination IP.
func (server *BFDServer) DeleteBfdSession(sessionMgmt BfdSessionMgmt) error {
	DestIp := sessionMgmt.DestIp
	Protocol := sessionMgmt.Protocol
	ForceDel := sessionMgmt.ForceDel
	server.logger.Info(fmt.Sprintln("DeleteSession ", DestIp, Protocol))
	sessionId, found := server.FindBfdSession(DestIp)
	if found {
		session := server.bfdGlobal.Sessions[sessionId]
		if session.state.PerLinkSession {
			server.DeletePerLinkSessions(DestIp, Protocol, ForceDel)
		} else {
			server.SessionDeleteHandler(session, Protocol, ForceDel)
		}
		server.ribdClient.ClientHdl.TrackReachabilityStatus(DestIp, "BFD", "del")
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
	}
	return nil
}

func (server *BFDServer) AdminUpPerLinkBfdSessions(DestIp string) error {
	for _, session := range server.bfdGlobal.Sessions {
		if session.state.IpAddr == DestIp {
			session.StartBfdSession()
		}
	}
	return nil
}

// AdminUpBfdSession ceases the session.
func (server *BFDServer) AdminUpBfdSession(sessionMgmt BfdSessionMgmt) error {
	DestIp := sessionMgmt.DestIp
	Protocol := sessionMgmt.Protocol
	server.logger.Info(fmt.Sprintln("AdminDownSession ", DestIp, Protocol))
	sessionId, found := server.FindBfdSession(DestIp)
	if found {
		session := server.bfdGlobal.Sessions[sessionId]
		if session.state.PerLinkSession {
			server.AdminUpPerLinkBfdSessions(DestIp)
		} else {
			server.bfdGlobal.Sessions[sessionId].StartBfdSession()
		}
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
	}
	return nil
}

func (server *BFDServer) AdminDownPerLinkBfdSessions(DestIp string) error {
	for _, session := range server.bfdGlobal.Sessions {
		if session.state.IpAddr == DestIp {
			session.StopBfdSession()
		}
	}
	return nil
}

// AdminDownBfdSession ceases the session.
func (server *BFDServer) AdminDownBfdSession(sessionMgmt BfdSessionMgmt) error {
	DestIp := sessionMgmt.DestIp
	Protocol := sessionMgmt.Protocol
	server.logger.Info(fmt.Sprintln("AdminDownSession ", DestIp, Protocol))
	sessionId, found := server.FindBfdSession(DestIp)
	if found {
		session := server.bfdGlobal.Sessions[sessionId]
		if session.state.PerLinkSession {
			server.AdminDownPerLinkBfdSessions(DestIp)
		} else {
			server.bfdGlobal.Sessions[sessionId].StopBfdSession()
		}
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
	}
	return nil
}

// This function handles NextHop change from RIB.
// Subsequent control packets will be sent using the BFD attributes configuration on the new IfIndex.
// A Poll control packet will be sent to BFD neighbor and expect a Final control packet.
func (server *BFDServer) HandleNextHopChange(DestIp string, IfIndex int32) error {
	return nil
}

func (session *BfdSession) StartSessionServer() error {
	session.server.logger.Info(fmt.Sprintln("Started session server for ", session.state.SessionId))
	for {
		select {
		case bfdPacket := <-session.ReceivedPacketCh:
			session.state.NumRxPackets++
			session.ProcessBfdPacket(bfdPacket)
		}
	}

	return nil
}

func (session *BfdSession) CanProcessBfdControlPacket(bfdPacket *BfdControlPacket) bool {
	var canProcess bool
	canProcess = true
	if bfdPacket.Version != DEFAULT_BFD_VERSION {
		canProcess = false
		session.server.logger.Info(fmt.Sprintln("Can't process version mismatch ", bfdPacket.Version, DEFAULT_BFD_VERSION))
	}
	if bfdPacket.DetectMult == 0 {
		canProcess = false
		session.server.logger.Info(fmt.Sprintln("Can't process detect multi ", bfdPacket.DetectMult))
	}
	if bfdPacket.Multipoint {
		canProcess = false
		session.server.logger.Info(fmt.Sprintln("Can't process Multipoint ", bfdPacket.Multipoint))
	}
	if bfdPacket.MyDiscriminator == 0 {
		canProcess = false
		session.server.logger.Info(fmt.Sprintln("Can't process remote discriminator ", bfdPacket.MyDiscriminator))
	}
	if bfdPacket.YourDiscriminator == 0 {
		if session.state.SessionState != STATE_DOWN &&
			session.state.SessionState != STATE_ADMIN_DOWN {
			canProcess = false
			session.server.logger.Info(fmt.Sprintln("Can't process my discriminator ", bfdPacket.YourDiscriminator))
		}
	}
	/*
		if bfdPacket.YourDiscriminator == 0 {
			canProcess = false
			fmt.Sprintln("Can't process local discriminator ", bfdPacket.YourDiscriminator)
		} else {
			sessionId := bfdPacket.YourDiscriminator
			session := server.bfdGlobal.Sessions[int32(sessionId)]
			if session != nil {
				if session.state.SessionState == STATE_ADMIN_DOWN {
					canProcess = false
				}
			}
		}
	*/
	return canProcess
}

func (session *BfdSession) AuthenticateReceivedControlPacket(bfdPacket *BfdControlPacket) bool {
	var authenticated bool
	if !bfdPacket.AuthPresent {
		authenticated = true
	} else {
		copiedPacket := &BfdControlPacket{}
		*copiedPacket = *bfdPacket
		authType := bfdPacket.AuthHeader.Type
		keyId := uint32(bfdPacket.AuthHeader.AuthKeyID)
		authData := bfdPacket.AuthHeader.AuthData
		seqNum := bfdPacket.AuthHeader.SequenceNumber
		if authType == session.authType {
			if authType == BFD_AUTH_TYPE_SIMPLE {
				fmt.Sprintln("Authentication type simple: keyId, authData ", keyId, string(authData))
				if keyId == session.authKeyId && string(authData) == session.authData {
					authenticated = true
				}
			} else {
				if seqNum >= session.state.ReceivedAuthSeq && keyId == session.authKeyId {
					var binBuf bytes.Buffer
					copiedPacket.AuthHeader.AuthData = []byte(session.authData)
					binary.Write(&binBuf, binary.BigEndian, copiedPacket)
					switch authType {
					case BFD_AUTH_TYPE_KEYED_MD5, BFD_AUTH_TYPE_METICULOUS_MD5:
						var authDataSum [16]byte
						authDataSum = md5.Sum(binBuf.Bytes())
						if bytes.Equal(authData[:], authDataSum[:]) {
							authenticated = true
						} else {
							fmt.Sprintln("Authentication data did't match for type: ", authType)
						}
					case BFD_AUTH_TYPE_KEYED_SHA1, BFD_AUTH_TYPE_METICULOUS_SHA1:
						var authDataSum [20]byte
						authDataSum = sha1.Sum(binBuf.Bytes())
						if bytes.Equal(authData[:], authDataSum[:]) {
							authenticated = true
						} else {
							fmt.Sprintln("Authentication data did't match for type: ", authType)
						}
					}
				} else {
					fmt.Sprintln("Sequence number and key id check failed: ", seqNum, session.state.ReceivedAuthSeq, keyId, session.authKeyId)
				}
			}
		} else {
			fmt.Sprintln("Authentication type did't match: ", authType, session.authType)
		}
	}
	return authenticated
}

func (session *BfdSession) ProcessBfdPacket(bfdPacket *BfdControlPacket) error {
	var event BfdSessionEvent
	authenticated := session.AuthenticateReceivedControlPacket(bfdPacket)
	if authenticated == false {
		session.server.logger.Info(fmt.Sprintln("Can't authenticatereceived bfd packet for session ", session.state.SessionId))
		return nil
	}
	canProcess := session.CanProcessBfdControlPacket(bfdPacket)
	if canProcess == false {
		session.server.logger.Info(fmt.Sprintln("Can't process received bfd packet for session ", session.state.SessionId))
		return nil
	}
	if session.state.SessionState != STATE_UP || session.state.RemoteSessionState != STATE_UP {
		session.rxInterval = (STARTUP_RX_INTERVAL * int32(bfdPacket.DetectMult)) / 1000
	} else {
		session.rxInterval = (int32(bfdPacket.DesiredMinTxInterval) * int32(bfdPacket.DetectMult)) / 1000
	}
	session.state.RemoteSessionState = bfdPacket.State
	session.state.RemoteDiscriminator = bfdPacket.MyDiscriminator
	session.state.RemoteMinRxInterval = int32(bfdPacket.RequiredMinRxInterval)
	session.RemoteChangedDemandMode(bfdPacket)
	session.ProcessPollSequence(bfdPacket)
	switch session.state.RemoteSessionState {
	case STATE_DOWN:
		event = REMOTE_DOWN
		session.state.LocalDiagType = DIAG_NEIGHBOR_SIGNAL_DOWN
	case STATE_INIT:
		event = REMOTE_INIT
	case STATE_UP:
		event = REMOTE_UP
		if session.state.SessionState == STATE_UP {
			session.txInterval = session.state.DesiredMinTxInterval / 1000
		}
	case STATE_ADMIN_DOWN:
		event = REMOTE_ADMIN_DOWN
	}
	session.EventHandler(event)
	if session.state.SessionState == STATE_ADMIN_DOWN ||
		session.state.RemoteSessionState == STATE_ADMIN_DOWN {
		session.sessionTimer.Stop()
	} else {
		session.sessionTimer.Reset(time.Duration(session.rxInterval) * time.Millisecond)
	}
	return nil
}

func (session *BfdSession) UpdateBfdSessionControlPacket() error {
	session.bfdPacket.Diagnostic = session.state.LocalDiagType
	session.bfdPacket.State = session.state.SessionState
	session.bfdPacket.DetectMult = uint8(session.state.DetectionMultiplier)
	session.bfdPacket.MyDiscriminator = session.state.LocalDiscriminator
	session.bfdPacket.YourDiscriminator = session.state.RemoteDiscriminator
	session.bfdPacket.RequiredMinRxInterval = time.Duration(session.state.RequiredMinRxInterval)
	if session.state.SessionState == STATE_UP && session.state.RemoteSessionState == STATE_UP {
		session.bfdPacket.DesiredMinTxInterval = time.Duration(session.state.DesiredMinTxInterval)
		wasDemand := session.bfdPacket.Demand
		session.bfdPacket.Demand = session.state.DemandMode
		isDemand := session.bfdPacket.Demand
		if !wasDemand && isDemand {
			fmt.Sprintln("Enabled demand for session ", session.state.SessionId)
			session.sessionTimer.Stop()
		}
		if wasDemand && !isDemand {
			fmt.Sprintln("Disabled demand for session ", session.state.SessionId)
			session.sessionTimer.Reset(time.Duration(session.rxInterval) * time.Millisecond)
		}
	} else {
		session.bfdPacket.DesiredMinTxInterval = time.Duration(STARTUP_TX_INTERVAL)
	}
	session.bfdPacket.Poll = session.pollSequence
	session.bfdPacket.Final = session.pollSequenceFinal
	session.pollSequenceFinal = false
	if session.authEnabled {
		session.bfdPacket.AuthPresent = true
		session.bfdPacket.AuthHeader.Type = session.authType
		if session.authType != BFD_AUTH_TYPE_SIMPLE {
			session.bfdPacket.AuthHeader.SequenceNumber = session.authSeqNum
		}
		if session.authType == BFD_AUTH_TYPE_METICULOUS_MD5 || session.authType == BFD_AUTH_TYPE_METICULOUS_SHA1 {
			session.authSeqNum++
		}
		session.bfdPacket.AuthHeader.AuthKeyID = uint8(session.authKeyId)
		session.bfdPacket.AuthHeader.AuthData = []byte(session.authData)
	} else {
		session.bfdPacket.AuthPresent = false
	}
	session.intfConfigChanged = false
	session.paramConfigChanged = false
	session.stateChanged = false
	return nil
}

func (session *BfdSession) CheckIfAnyProtocolRegistered() bool {
	for i := bfddCommonDefs.BfdSessionOwner(1); i < bfddCommonDefs.MAX_NUM_PROTOCOLS; i++ {
		if session.state.RegisteredProtocols[i] == true {
			return true
		}
	}
	return false
}

// Stop session as Bfd is disabled globally. Do not delete
func (session *BfdSession) StopBfdSession() error {
	session.EventHandler(ADMIN_DOWN)
	session.state.LocalDiagType = DIAG_ADMIN_DOWN
	return nil
}

func (session *BfdSession) GetBfdSessionNotification() bool {
	var bfdState bool
	bfdState = false
	if session.state.SessionState == STATE_UP ||
		session.state.SessionState == STATE_ADMIN_DOWN ||
		session.state.RemoteSessionState == STATE_ADMIN_DOWN {
		bfdState = true
	}
	return bfdState
}

func (session *BfdSession) SendBfdNotification() error {
	bfdState := session.GetBfdSessionNotification()
	bfdNotification := bfddCommonDefs.BfddNotifyMsg{
		DestIp: session.state.IpAddr,
		State:  bfdState,
	}
	bfdNotificationBuf, err := json.Marshal(bfdNotification)
	if err != nil {
		session.server.logger.Err(fmt.Sprintln("Failed to marshal BfdSessionNotification message for session ", session.state.SessionId))
	}
	session.server.notificationCh <- bfdNotificationBuf
	return nil
}

// Restart session that was stopped earlier due to global Bfd disable.
func (session *BfdSession) StartBfdSession() error {
	session.sessionTimer.Reset(time.Duration(session.rxInterval) * time.Millisecond)
	txInterval := session.ApplyTxJitter()
	session.txTimer.Reset(time.Duration(txInterval) * time.Millisecond)
	session.state.SessionState = STATE_DOWN
	session.EventHandler(ADMIN_UP)
	return nil
}

func (session *BfdSession) IsSessionActive() bool {
	if session.isClientActive {
		return true
	} else {
		return false
	}
}

/* State Machine
                             +--+
                             |  | UP, TIMER
                             |  V
                     DOWN  +------+  INIT
              +------------|      |------------+
              |            | DOWN |            |
              |  +-------->|      |<--------+  |
              |  |         +------+         |  |
              |  |                          |  |
              |  |                          |  |
              |  |                     DOWN,|  |
              |  |TIMER                TIMER|  |
              V  |                          |  V
            +------+                      +------+
       +----|      |                      |      |----+
   DOWN|    | INIT |--------------------->|  UP  |    |INIT, UP
       +--->|      | INIT, UP             |      |<---+
            +------+                      +------+
*/
// EventHandler is called after receiving a BFD packet from remote.
func (session *BfdSession) EventHandler(event BfdSessionEvent) error {
	var err error
	if session.IsSessionActive() == false {
		session.server.logger.Info(fmt.Sprintln("Cannot process event ", event, " Session ", session.state.SessionId, " not active"))
		err = errors.New("Session is not active. No event can be processed.")
		return err
	}
	switch session.state.SessionState {
	case STATE_ADMIN_DOWN:
		session.server.logger.Info(fmt.Sprintln("Received ", event, " event for an admindown session"))
	case STATE_DOWN:
		switch event {
		case REMOTE_DOWN:
			session.MoveToInitState()
		case REMOTE_INIT:
			session.MoveToUpState()
		case ADMIN_UP:
			session.MoveToDownState()
		case ADMIN_DOWN:
			session.LocalAdminDown()
		case REMOTE_ADMIN_DOWN:
			session.RemoteAdminDown()
		case TIMEOUT, REMOTE_UP:
		}
	case STATE_INIT:
		switch event {
		case REMOTE_INIT, REMOTE_UP:
			session.MoveToUpState()
		case TIMEOUT:
			session.MoveToDownState()
		case ADMIN_DOWN:
			session.LocalAdminDown()
		case REMOTE_ADMIN_DOWN:
			session.RemoteAdminDown()
		case REMOTE_DOWN, ADMIN_UP:
		}
	case STATE_UP:
		switch event {
		case REMOTE_DOWN, TIMEOUT:
			session.MoveToDownState()
		case ADMIN_DOWN:
			session.LocalAdminDown()
		case REMOTE_ADMIN_DOWN:
			session.RemoteAdminDown()
		case REMOTE_INIT, REMOTE_UP, ADMIN_UP:
		}
	}
	return err
}

func (session *BfdSession) LocalAdminDown() error {
	session.state.SessionState = STATE_ADMIN_DOWN
	session.state.RemoteDiscriminator = 0
	session.stateChanged = true
	session.SendBfdNotification()
	session.txInterval = STARTUP_TX_INTERVAL / 1000
	session.rxInterval = (STARTUP_RX_INTERVAL * session.state.DetectionMultiplier) / 1000
	session.sessionTimer.Stop()
	return nil
}

func (session *BfdSession) RemoteAdminDown() error {
	session.state.RemoteSessionState = STATE_ADMIN_DOWN
	session.state.RemoteDiscriminator = 0
	session.state.LocalDiagType = DIAG_NEIGHBOR_SIGNAL_DOWN
	session.SendBfdNotification()
	session.txInterval = STARTUP_TX_INTERVAL / 1000
	session.rxInterval = (STARTUP_RX_INTERVAL * session.state.DetectionMultiplier) / 1000
	session.sessionTimer.Stop()
	return nil
}

func (session *BfdSession) MoveToDownState() error {
	session.state.SessionState = STATE_DOWN
	session.state.RemoteDiscriminator = 0
	session.useDedicatedMac = true
	session.stateChanged = true
	if session.authType == BFD_AUTH_TYPE_KEYED_MD5 || session.authType == BFD_AUTH_TYPE_KEYED_SHA1 {
		session.authSeqNum++
	}
	session.SendBfdNotification()
	session.txInterval = STARTUP_TX_INTERVAL / 1000
	session.rxInterval = (STARTUP_RX_INTERVAL * session.state.DetectionMultiplier) / 1000
	session.txTimer.Reset(time.Duration(session.txInterval) * time.Millisecond)
	session.sessionTimer.Reset(time.Duration(session.rxInterval) * time.Millisecond)
	return nil
}

func (session *BfdSession) MoveToInitState() error {
	session.state.SessionState = STATE_INIT
	session.stateChanged = true
	session.useDedicatedMac = true
	return nil
}

func (session *BfdSession) MoveToUpState() error {
	session.state.SessionState = STATE_UP
	session.stateChanged = true
	session.state.LocalDiagType = DIAG_NONE
	session.SendBfdNotification()
	return nil
}

func (session *BfdSession) ApplyTxJitter() int32 {
	var txInterval int32
	if session.state.DetectionMultiplier == 1 {
		txInterval = int32(float32(session.txInterval) * (1 - float32(session.txJitter)/100))
	} else {
		txInterval = int32(float32(session.txInterval) * (1 + float32(session.txJitter)/100))
	}
	return txInterval
}

func (session *BfdSession) NeedBfdPacketUpdate() bool {
	if session.intfConfigChanged == true ||
		session.paramConfigChanged == true ||
		session.stateChanged == true ||
		session.pollSequence == true ||
		session.pollSequenceFinal == true {
		return true
	}
	return false
}

func (session *BfdSession) SendPeriodicControlPackets() {
	var err error
	var packetUpdated bool
	if session.NeedBfdPacketUpdate() {
		packetUpdated = true
		session.UpdateBfdSessionControlPacket()
		session.bfdPacketBuf, err = session.bfdPacket.CreateBfdControlPacket()
		if err != nil {
			session.server.logger.Info(fmt.Sprintln("Failed to create control packet for session ", session.state.SessionId))
		}
	}
	_, err = session.txConn.Write(session.bfdPacketBuf)
	if err != nil {
		session.server.logger.Info(fmt.Sprintln("failed to send control packet for session ", session.state.SessionId))
	} else {
		session.state.NumTxPackets++
	}
	if packetUpdated {
		// Re-compute the packet to clear any flag set in the previously sent packet
		session.UpdateBfdSessionControlPacket()
		session.bfdPacketBuf, err = session.bfdPacket.CreateBfdControlPacket()
		if err != nil {
			session.server.logger.Info(fmt.Sprintln("Failed to create control packet for session ", session.state.SessionId))
		}
		packetUpdated = false
	}
	if session.state.SessionState != STATE_ADMIN_DOWN &&
		session.state.RemoteSessionState != STATE_ADMIN_DOWN {
		txTimer := session.ApplyTxJitter()
		session.txTimer.Reset(time.Duration(txTimer) * time.Millisecond)
	}
}

func (session *BfdSession) HandleSessionTimeout() {
	if session.state.SessionState != STATE_DOWN ||
		session.state.SessionState != STATE_ADMIN_DOWN {
		session.server.logger.Info(fmt.Sprintln("Timer expired for: ", session.state.IpAddr, " session id ", session.state.SessionId, " prev state ", session.server.ConvertBfdSessionStateValToStr(session.state.SessionState), " at ", time.Now().String()))
	}
	session.state.LocalDiagType = DIAG_TIME_EXPIRED
	session.EventHandler(TIMEOUT)
	session.sessionTimer.Reset(time.Duration(session.rxInterval) * time.Millisecond)
}

func (session *BfdSession) StartSessionClient(server *BFDServer) error {
	var err error
	destAddr := session.state.IpAddr + ":" + strconv.Itoa(DEST_PORT)
	ServerAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed ResolveUDPAddr ", destAddr, err))
		server.FailedSessionClientCh <- session.state.SessionId
		return err
	}
	localAddr := ":" + strconv.Itoa(int(SRC_PORT+session.state.SessionId))
	ClientAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed ResolveUDPAddr ", localAddr, err))
		server.FailedSessionClientCh <- session.state.SessionId
		return err
	}
	Conn, err := net.DialUDP("udp", ClientAddr, ServerAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed DialUDP ", ClientAddr, ServerAddr, err))
		server.FailedSessionClientCh <- session.state.SessionId
		return err
	}
	session.txConn = Conn
	server.logger.Info(fmt.Sprintln("Started session client for ", destAddr, localAddr))
	defer session.txConn.Close()
	session.txTimer = time.AfterFunc(time.Duration(session.txInterval)*time.Millisecond, func() { session.SendPeriodicControlPackets() })
	session.sessionTimer = time.AfterFunc(time.Duration(session.rxInterval)*time.Millisecond, func() { session.HandleSessionTimeout() })
	for {
		select {
		case <-session.SessionStopClientCh:
			return nil
		}
	}
}

func (session *BfdSession) RemoteChangedDemandMode(bfdPacket *BfdControlPacket) error {
	var wasDemandMode, isDemandMode bool
	wasDemandMode = session.state.RemoteDemandMode
	session.state.RemoteDemandMode = bfdPacket.Demand
	if session.state.RemoteDemandMode {
		isDemandMode = true
		session.txTimer.Stop()
	}
	if wasDemandMode && !isDemandMode {
		txInterval := session.ApplyTxJitter()
		session.txTimer.Reset(time.Duration(txInterval) * time.Millisecond)
	}
	return nil
}

func (session *BfdSession) InitiatePollSequence() error {
	session.server.logger.Info(fmt.Sprintln("Starting poll sequence for session ", session.state.SessionId))
	session.pollSequence = true
	return nil
}

func (session *BfdSession) ProcessPollSequence(bfdPacket *BfdControlPacket) error {
	if session.state.SessionState != STATE_ADMIN_DOWN {
		if bfdPacket.Poll {
			session.server.logger.Info(fmt.Sprintln("Received packet with poll bit for session ", session.state.SessionId))
			session.pollSequenceFinal = true
		}
		if bfdPacket.Final {
			session.server.logger.Info(fmt.Sprintln("Received packet with final bit for session ", session.state.SessionId))
			session.pollSequence = false
		}
	}
	return nil
}
