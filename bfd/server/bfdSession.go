package server

import (
	"fmt"
	"math/rand"
	"time"
)

func (server *BFDServer) GetNewSessionId() int32 {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	sessionId := r1.Int31n(MAX_NUM_SESSIONS)
	return sessionId
}

func (server *BFDServer) GetIfIndexFromDestIp(DestIp string) int32 {
	reachabilityInfo, err := server.ribdClient.ClientHdl.GetRouteReachabilityInfo(DestIp)
	if err != nil {
		server.logger.Info(fmt.Sprintf("%s is not reachable", DestIp))
		return int32(0)
	}
	return int32(reachabilityInfo.NextHopIfIndex)
}

func (server *BFDServer) NewBfdSession(DestIp, protocol string) *BfdSession {
	bfdSession := &BfdSession{}
	bfdSession.state.SessionId = server.GetNewSessionId()
	bfdSession.state.RemoteIpAddr = DestIp
	bfdSession.state.InterfaceId = server.GetIfIndexFromDestIp(DestIp)
	bfdSession.state.RegisteredProtocols += protocol + ","
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
	}
	server.logger.Info(fmt.Sprintln("new sesstion : ", bfdSession))
	return bfdSession
}

// CreateBfdSession initializes a session and starts cpntrol packets exchange.
// This function is called when a protocol registers with BFD to monitor a destination IP.
func (server *BFDServer) CreateBfdSession(DestIp, protocol string) error {
	bfdSession := server.NewBfdSession(DestIp, protocol)
	sessionTimeoutMS := time.Duration((bfdSession.state.RequiredMinRxInterval * bfdSession.state.DetectionMultiplier) / 1000)
	bfdSession.timer = time.NewTimer(time.Millisecond * sessionTimeoutMS)
	server.logger.Info(fmt.Sprintln("Bfd session created ", bfdSession))
	return nil
}

// DeleteBfdSession ceases the session.
// A session down control packet is sent to BFD neighbor before deleting the session.
// This function is called when a protocol decides to stop monitoring the destination IP.
func (server *BFDServer) DeleteBfdSession(DestIp string) error {
	return nil
}

// This function handles NextHop change from RIB.
// Subsequent control packets will be sent using the BFD attributes configuration on the new IfIndex.
// A Poll control packet will be sent to BFD neighbor and expact a Final control packet.
func (server *BFDServer) HandleNextHopChange(DestIp string) error {
	return nil
}

/* State Machine
                             +--+
                             |  | UP, ADMIN DOWN, TIMER
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
   DOWN|    | INIT |--------------------->|  UP  |    |INIT, UP
       +--->|      | INIT, UP             |      |<---+
            +------+                      +------+
*/
// EventHandler is called after receiving a BFD packet from remote.
func (session *BfdSession) EventHandler(event BfdSessionEvent) error {
	switch session.state.SessionState {
	case STATE_ADMIN_DOWN, STATE_DOWN:
		switch event {
		case REMOTE_DOWN:
			session.MoveToInitState()
		case REMOTE_INIT:
			session.MoveToUpState()
		case ADMIN_DOWN, TIMEOUT, REMOTE_UP:
			fmt.Printf("Received %d event in DOWN state. No change in state", event)
		}
	case STATE_INIT:
		switch event {
		case REMOTE_INIT, REMOTE_UP:
			session.MoveToUpState()
		case ADMIN_DOWN, TIMEOUT:
			session.MoveToDownState()
		case REMOTE_DOWN:
			fmt.Printf("Received %d event in INIT state. No change in state", event)
		}
	case STATE_UP:
		switch event {
		case REMOTE_DOWN, ADMIN_DOWN, TIMEOUT:
			session.MoveToDownState()
		case REMOTE_INIT, REMOTE_UP:
			fmt.Printf("Received %d event in UP state. No change in state", event)
		}
	}
	return nil
}

func (session *BfdSession) MoveToDownState() error {
	return nil
}

func (session *BfdSession) MoveToInitState() error {
	return nil
}

func (session *BfdSession) MoveToUpState() error {
	return nil
}
