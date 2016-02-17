package server

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"math/rand"
	"net"
	"strconv"
	"time"
)

func (server *BFDServer) processSessionConfig(sessionConfig bfddCommonDefs.BfdSessionConfig) error {
	var sessionMgmt BfdSessionMgmt
	sessionMgmt.DestIp = sessionConfig.DestIp
	sessionMgmt.Protocol = sessionConfig.Protocol
	if sessionConfig.Operation == bfddCommonDefs.CREATE {
		server.CreateSessionCh <- sessionMgmt
	}
	if sessionConfig.Operation == bfddCommonDefs.DELETE {
		server.DeleteSessionCh <- sessionMgmt
	}
	if sessionConfig.Operation == bfddCommonDefs.ADMINDOWN {
		server.AdminDownSessionCh <- sessionMgmt
	}
	return nil
}

func (server *BFDServer) StartSessionHandler() error {
	server.CreateSessionCh = make(chan BfdSessionMgmt)
	server.DeleteSessionCh = make(chan BfdSessionMgmt)
	server.AdminUpSessionCh = make(chan BfdSessionMgmt)
	server.AdminDownSessionCh = make(chan BfdSessionMgmt)
	for {
		select {
		case sessionMgmt := <-server.CreateSessionCh:
			session, _ := server.CreateBfdSession(sessionMgmt)
			if session != nil {
				session.TxTimeoutCh = make(chan int32)
				session.SessionTimeoutCh = make(chan int32)
				session.SessionDeleteCh = make(chan bool)
				go session.StartSessionServer(server)
				go session.StartSessionClient(server)
			} else {
				server.logger.Info(fmt.Sprintf("Bfd session could not be established to ", sessionMgmt))
			}
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

func (server *BFDServer) GetNewSessionId() int32 {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	sessionId := r1.Int31n(MAX_NUM_SESSIONS)
	return sessionId
}

func (server *BFDServer) GetIfIndexAndLocalIpFromDestIp(DestIp string) (int32, string) {
	reachabilityInfo, err := server.ribdClient.ClientHdl.GetRouteReachabilityInfo(DestIp)
	if err != nil {
		server.logger.Info(fmt.Sprintf("%s is not reachable", DestIp))
		return int32(0), ""
	}
	return int32(reachabilityInfo.NextHopIfIndex), reachabilityInfo.NextHopIp
}

func (server *BFDServer) NewBfdSession(DestIp string, protocol int) *BfdSession {
	ifIndex, _ := server.GetIfIndexAndLocalIpFromDestIp(DestIp)
	// Hack to test BFD. RIB is not able to provide ifIndex for a destination IP at this point
	if ifIndex == 0 {
		ifIndex = 46
		//return nil
	}
	if server.bfdGlobal.Interfaces[ifIndex].Enabled {
		bfdSession := &BfdSession{}
		sessionId := server.GetNewSessionId()
		bfdSession.state.SessionId = sessionId
		bfdSession.state.RemoteIpAddr = DestIp
		bfdSession.state.InterfaceId = ifIndex
		bfdSession.state.RegisteredProtocols = make([]bool, bfddCommonDefs.MAX_NUM_PROTOCOLS)
		bfdSession.state.RegisteredProtocols[protocol] = true
		bfdSession.state.SessionState = STATE_DOWN
		bfdSession.state.RemoteSessionState = STATE_DOWN
		bfdSession.state.LocalDiscriminator = uint32(bfdSession.state.SessionId)
		bfdSession.state.LocalDiagType = DIAG_NONE
		intf, exist := server.bfdGlobal.Interfaces[bfdSession.state.InterfaceId]
		if exist {
			bfdSession.state.LocalIpAddr = intf.property.IpAddr.String()
			bfdSession.state.DesiredMinTxInterval = intf.conf.DesiredMinTxInterval
			bfdSession.state.RequiredMinRxInterval = intf.conf.RequiredMinRxInterval
			bfdSession.state.DetectionMultiplier = intf.conf.LocalMultiplier
			bfdSession.state.DemandMode = intf.conf.DemandEnabled
			bfdSession.authEnabled = intf.conf.AuthenticationEnabled
			bfdSession.authType = AuthenticationType(intf.conf.AuthenticationType)
			bfdSession.authSeqNum = 1
			bfdSession.authKeyId = uint32(intf.conf.AuthenticationKeyId)
			bfdSession.authData = intf.conf.AuthenticationData
		}
		server.bfdGlobal.Sessions[sessionId] = bfdSession
		server.bfdGlobal.SessionsIdSlice = append(server.bfdGlobal.SessionsIdSlice, sessionId)
		server.logger.Info(fmt.Sprintln("New session : ", sessionId, " created on : ", ifIndex))
		return bfdSession
	} else {
		server.logger.Info(fmt.Sprintln("Bfd not enabled on interface ", ifIndex))
	}
	return nil
}

func (server *BFDServer) UpdateBfdSessionsOnInterface(ifIndex int32) error {
	intf, exist := server.bfdGlobal.Interfaces[ifIndex]
	if exist {
		for _, session := range server.bfdGlobal.Sessions {
			if session.state.InterfaceId == ifIndex {
				session.state.LocalIpAddr = intf.property.IpAddr.String()
				session.state.DesiredMinTxInterval = intf.conf.DesiredMinTxInterval
				session.state.RequiredMinRxInterval = intf.conf.RequiredMinRxInterval
				session.state.DetectionMultiplier = intf.conf.LocalMultiplier
				session.state.DemandMode = intf.conf.DemandEnabled
				session.authEnabled = intf.conf.AuthenticationEnabled
				session.authType = AuthenticationType(intf.conf.AuthenticationType)
				session.authKeyId = uint32(intf.conf.AuthenticationKeyId)
				session.authData = intf.conf.AuthenticationData
				session.InitiatePollSequence()
			}
		}
	}
	return nil
}

func (server *BFDServer) FindBfdSession(DestIp string) (sessionId int32, found bool) {
	found = false
	for sessionId, session := range server.bfdGlobal.Sessions {
		if session.state.RemoteIpAddr == DestIp {
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
	Protocol := sessionMgmt.Protocol
	sessionId, found := server.FindBfdSession(DestIp)
	if !found {
		server.logger.Info(fmt.Sprintln("CreateSession ", DestIp, Protocol))
		bfdSession = server.NewBfdSession(DestIp, Protocol)
		if bfdSession != nil {
			bfdSession.bfdPacket = NewBfdControlPacketDefault()
			server.bfdGlobal.Sessions[bfdSession.state.SessionId] = bfdSession
			server.logger.Info(fmt.Sprintln("Bfd session created ", bfdSession.state.SessionId, bfdSession.state.RemoteIpAddr))
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

// DeleteBfdSession ceases the session.
// A session down control packet is sent to BFD neighbor before deleting the session.
// This function is called when a protocol decides to stop monitoring the destination IP.
func (server *BFDServer) DeleteBfdSession(sessionMgmt BfdSessionMgmt) error {
	var i int
	DestIp := sessionMgmt.DestIp
	Protocol := sessionMgmt.Protocol
	server.logger.Info(fmt.Sprintln("DeleteSession ", DestIp, Protocol))
	sessionId, found := server.FindBfdSession(DestIp)
	if found {
		server.bfdGlobal.Sessions[sessionId].state.RegisteredProtocols[Protocol] = false
		if server.bfdGlobal.Sessions[sessionId].CheckIfAnyProtocolRegistered() == false {
			server.bfdGlobal.Sessions[sessionId].txTimer.Stop()
			server.bfdGlobal.Sessions[sessionId].sessionTimer.Stop()
			server.bfdGlobal.Sessions[sessionId].SessionDeleteCh <- true
			delete(server.bfdGlobal.Sessions, sessionId)
			for i = 0; i < len(server.bfdGlobal.SessionsIdSlice); i++ {
				if server.bfdGlobal.SessionsIdSlice[i] == sessionId {
					break
				}
			}
			server.bfdGlobal.SessionsIdSlice = append(server.bfdGlobal.SessionsIdSlice[:i], server.bfdGlobal.SessionsIdSlice[i+1:]...)
		}
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
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
		server.bfdGlobal.Sessions[sessionId].StartBfdSession()
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
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
		server.bfdGlobal.Sessions[sessionId].StopBfdSession()
	} else {
		server.logger.Info(fmt.Sprintln("Bfd session not found ", sessionId))
	}
	return nil
}

// This function handles NextHop change from RIB.
// Subsequent control packets will be sent using the BFD attributes configuration on the new IfIndex.
// A Poll control packet will be sent to BFD neighbor and expact a Final control packet.
func (server *BFDServer) HandleNextHopChange(DestIp string) error {
	return nil
}

func (session *BfdSession) StartSessionServer(bfdServer *BFDServer) error {
	destAddr := session.state.LocalIpAddr + ":" + strconv.Itoa(DEST_PORT)
	ServerAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		fmt.Println("Failed ResolveUDPAddr ", destAddr, err)
		return nil
	}
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		fmt.Println("Failed ListenUDP ", err)
		return nil
	}
	defer ServerConn.Close()
	buf := make([]byte, 1024)
	for {
		len, addr, err := ServerConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Failed to read from ", ServerAddr)
		} else {
			if len >= DEFAULT_CONTROL_PACKET_LEN {
				bfdPacket, err := DecodeBfdControlPacket(buf[0:len])
				if err != nil {
					//fmt.Println("Received ", string(buf[0:len]), " from ", addr, " bfdPacket ", bfdPacket)
					sessionId := int32(bfdPacket.YourDiscriminator)
					fmt.Println("Received bfd packet for session ", sessionId, " from ", addr)
					if sessionId == 0 {
						fmt.Println("Ignore bfd packet for session ", sessionId, " from ", addr)
					} else {
						bfdSession := bfdServer.bfdGlobal.Sessions[sessionId]
						bfdSession.ProcessBfdPacket(bfdPacket)
					}
				}
			}
		}
	}
	return nil
}

func (session *BfdSession) CanProcessBfdControlPacket(bfdPacket *BfdControlPacket) bool {
	var canProcess bool
	canProcess = true
	/*
		sessionId := bfdPacket.YourDiscriminator
		session := server.bfdGlobal.Sessions[int32(sessionId)]
		if session != nil {
			if session.state.SessionState == STATE_ADMIN_DOWN {
				canProcess = false
			}
		}
	*/
	if bfdPacket.Version != DEFAULT_BFD_VERSION {
		canProcess = false
		fmt.Sprintln("Can't process version mismatch ", bfdPacket.Version, DEFAULT_BFD_VERSION)
	}
	if bfdPacket.DetectMult == 0 {
		canProcess = false
		fmt.Sprintln("Can't process detect multi ", bfdPacket.DetectMult)
	}
	if bfdPacket.Multipoint {
		canProcess = false
		fmt.Sprintln("Can't process Multipoint ", bfdPacket.Multipoint)
	}
	if bfdPacket.MyDiscriminator == 0 {
		canProcess = false
		fmt.Sprintln("Can't process remote discriminator ", bfdPacket.MyDiscriminator)
	}
	if bfdPacket.YourDiscriminator == 0 {
		canProcess = false
		fmt.Sprintln("Can't process local discriminator ", bfdPacket.YourDiscriminator)
	}
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
		fmt.Sprintln("Can't authenticatereceived bfd packet for session ", session.state.SessionId)
		return nil
	}
	canProcess := session.CanProcessBfdControlPacket(bfdPacket)
	if canProcess == false {
		fmt.Sprintln("Can't process received bfd packet for session ", session.state.SessionId)
		return nil
	}
	session.state.RemoteSessionState = bfdPacket.State
	session.state.RemoteDiscriminator = bfdPacket.MyDiscriminator
	session.state.RemoteMinRxInterval = int32(bfdPacket.RequiredMinRxInterval)
	switch session.state.RemoteSessionState {
	case STATE_DOWN:
		event = REMOTE_DOWN
	case STATE_INIT:
		event = REMOTE_INIT
	case STATE_UP:
		event = REMOTE_UP
	}
	session.EventHandler(event)
	session.RemoteChangedDemandMode(bfdPacket)
	session.ProcessPollSequence(bfdPacket)
	session.sessionTimer.Stop()
	sessionTimeoutMS := time.Duration(session.state.RequiredMinRxInterval * session.state.DetectionMultiplier)
	session.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { session.SessionTimeoutCh <- session.state.SessionId })
	return nil
}

func (session *BfdSession) UpdateBfdSessionControlPacket() error {
	session.bfdPacket.Diagnostic = session.state.LocalDiagType
	session.bfdPacket.State = session.state.SessionState
	session.bfdPacket.DetectMult = uint8(session.state.DetectionMultiplier)
	session.bfdPacket.MyDiscriminator = session.state.LocalDiscriminator
	session.bfdPacket.YourDiscriminator = session.state.RemoteDiscriminator
	session.bfdPacket.DesiredMinTxInterval = time.Duration(session.state.DesiredMinTxInterval)
	session.bfdPacket.RequiredMinRxInterval = time.Duration(session.state.RequiredMinRxInterval)
	if session.state.SessionState == STATE_UP && session.state.RemoteSessionState == STATE_UP {
		session.bfdPacket.Demand = session.state.DemandMode
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
	return nil
}

func (session *BfdSession) CheckIfAnyProtocolRegistered() bool {
	for i := 0; i < bfddCommonDefs.MAX_NUM_PROTOCOLS; i++ {
		if session.state.RegisteredProtocols[i] == true {
			return true
		}
	}
	return false
}

// Stop session as Bfd is disabled globally. Do not delete
func (session *BfdSession) StopBfdSession() error {
	session.EventHandler(ADMIN_DOWN)
	session.txTimer.Stop()
	session.sessionTimer.Stop()
	return nil
}

// Restart session that was stopped earlier due to global Bfd disable.
func (session *BfdSession) StartBfdSession() error {
	sessionTimeoutMS := time.Duration(session.state.RequiredMinRxInterval * session.state.DetectionMultiplier)
	txTimerMS := time.Duration(session.state.DesiredMinTxInterval)
	session.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { session.SessionTimeoutCh <- session.state.SessionId })
	session.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { session.TxTimeoutCh <- session.state.SessionId })
	session.state.SessionState = STATE_DOWN
	session.EventHandler(ADMIN_UP)
	return nil
}

/* State Machine
                                    +--+
                                    |  | UP, ADMIN DOWN, TIMER, ADMIN_UP
                                    |  V
                            DOWN  +------+  INIT
                     +------------|      |------------+
                     |            | DOWN |            |
                     |  +-------->|      |<--------+  |
                     |  |         +------+         |  |
                     |  |                          |  |
                     |  |               ADMIN DOWN,|  |
                     |  |ADMIN DOWN,          DOWN,|  |
                     |  |TIMER                TIMER|  |
                     V  |                          |  V
                   +------+                      +------+
              +----|      |                      |      |----+
ADMIN_UP, DOWN|    | INIT |--------------------->|  UP  |    |INIT, UP, ADMIN_UP
              +--->|      | INIT, UP             |      |<---+
                   +------+                      +------+
*/
// EventHandler is called after receiving a BFD packet from remote.
func (session *BfdSession) EventHandler(event BfdSessionEvent) error {
	switch session.state.SessionState {
	case STATE_ADMIN_DOWN:
		fmt.Printf("Received %d event in ADMINDOWN state. No change in state\n", event)
	case STATE_DOWN:
		switch event {
		case REMOTE_DOWN:
			session.MoveToInitState()
		case REMOTE_INIT:
			session.MoveToUpState()
		case ADMIN_UP:
			session.MoveToDownState()
		case ADMIN_DOWN, TIMEOUT, REMOTE_UP:
			fmt.Printf("Received %d event in DOWN state. No change in state\n", event)
		}
	case STATE_INIT:
		switch event {
		case REMOTE_INIT, REMOTE_UP:
			session.MoveToUpState()
		case ADMIN_DOWN, TIMEOUT:
			session.MoveToDownState()
		case REMOTE_DOWN, ADMIN_UP:
			fmt.Printf("Received %d event in INIT state. No change in state\n", event)
		}
	case STATE_UP:
		switch event {
		case REMOTE_DOWN, ADMIN_DOWN, TIMEOUT:
			session.MoveToDownState()
		case REMOTE_INIT, REMOTE_UP, ADMIN_UP:
			fmt.Printf("Received %d event in UP state. No change in state\n", event)
		}
	}
	return nil
}

func (session *BfdSession) MoveToDownState() error {
	session.state.SessionState = STATE_DOWN
	if session.authType == BFD_AUTH_TYPE_KEYED_MD5 || session.authType == BFD_AUTH_TYPE_KEYED_SHA1 {
		session.authSeqNum++
	}
	//session.txTimer.Reset(0)
	return nil
}

func (session *BfdSession) MoveToInitState() error {
	session.state.SessionState = STATE_INIT
	//session.txTimer.Reset(0)
	return nil
}

func (session *BfdSession) MoveToUpState() error {
	session.state.SessionState = STATE_UP
	//session.txTimer.Reset(0)
	return nil
}

func (session *BfdSession) StartSessionClient(server *BFDServer) error {
	destAddr := session.state.RemoteIpAddr + ":" + strconv.Itoa(DEST_PORT)
	ServerAddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		fmt.Println("Failed ResolveUDPAddr ", destAddr, err)
	}
	localAddr := session.state.LocalIpAddr + ":" + strconv.Itoa(SRC_PORT)
	ClientAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		fmt.Println("Failed ResolveUDPAddr ", localAddr, err)
	}
	Conn, err := net.DialUDP("udp", ClientAddr, ServerAddr)
	if err != nil {
		fmt.Println("Failed DialUDP ", ClientAddr, ServerAddr, err)
	}
	sessionTimeoutMS := time.Duration(session.state.RequiredMinRxInterval * session.state.DetectionMultiplier)
	txTimerMS := time.Duration(session.state.DesiredMinTxInterval)
	session.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { session.SessionTimeoutCh <- session.state.SessionId })
	session.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { session.TxTimeoutCh <- session.state.SessionId })
	//session.txTimer.Reset(0)
	defer Conn.Close()
	for {
		select {
		case sessionId := <-session.TxTimeoutCh:
			bfdSession := server.bfdGlobal.Sessions[sessionId]
			bfdSession.UpdateBfdSessionControlPacket()
			buf, err := bfdSession.bfdPacket.CreateBfdControlPacket()
			if err != nil {
				fmt.Println("Failed to create control packet for session ", bfdSession.state.SessionId)
			} else {
				_, err = Conn.Write(buf)
				if err != nil {
					fmt.Println("failed to send control packet for session ", bfdSession.state.SessionId)
				}
				txTimerMS = time.Duration(bfdSession.state.DesiredMinTxInterval)
				bfdSession.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { bfdSession.TxTimeoutCh <- bfdSession.state.SessionId })
			}
		case sessionId := <-session.SessionTimeoutCh:
			bfdSession := server.bfdGlobal.Sessions[sessionId]
			bfdSession.EventHandler(TIMEOUT)
			sessionTimeoutMS = time.Duration(bfdSession.state.RequiredMinRxInterval * bfdSession.state.DetectionMultiplier)
			bfdSession.sessionTimer = time.AfterFunc(time.Millisecond*sessionTimeoutMS, func() { bfdSession.SessionTimeoutCh <- bfdSession.state.SessionId })
		case <-session.SessionDeleteCh:
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
		txTimerMS := time.Duration(session.state.DesiredMinTxInterval)
		session.txTimer = time.AfterFunc(time.Millisecond*txTimerMS, func() { session.TxTimeoutCh <- session.state.SessionId })
	}
	return nil
}

func (session *BfdSession) InitiatePollSequence() error {
	session.pollSequence = true
	session.txTimer.Reset(0)
	return nil
}

func (session *BfdSession) ProcessPollSequence(bfdPacket *BfdControlPacket) error {
	if bfdPacket.Poll {
		session.pollSequenceFinal = true
	}
	if bfdPacket.Final {
		session.pollSequence = false
	}
	session.txTimer.Reset(0)
	return nil
}
