// fsm.go
package server

import (
	"fmt"
	"net"
	"time"
)

type BGPFSMState int

const BGPConnectRetryTime uint16 = 120 // seconds
const BGPHoldTimeDefault uint16 = 9    // 240 seconds
const BGPIdleHoldTimeDefault uint16 = 5    // 240 seconds

var IdleHoldTimeInterval = map[uint16]uint16 {
	0: 0,
	5: 10,
	10: 30,
	60: 120,
	120: 180,
	180: 300,
	300: 500,
	500: 0,
}

const (
	BGPFSMNone BGPFSMState = iota
	BGPFSMIdle
	BGPFSMConnect
	BGPFSMActive
	BGPFSMOpensent
	BGPFSMOpenconfirm
	BGPFSMEstablished
)

type BGPFSMEvent int

const (
	_ BGPFSMEvent = iota
	BGPEventManualStart
	BGPEventManualStop
	BGPEventAutoStart
	BGPEventManualStartPassTcpEst
	BGPEventAutoStartPassTcpEst
	BGPEventAutoStartDampPeerOscl
	BGPEventAutoStartDampPeerOsclPassTcpEst
	BGPEventAutoStop
	BGPEventConnRetryTimerExp
	BGPEventHoldTimerExp
	BGPEventKeepAliveTimerExp
	BGPEventDelayOpenTimerExp
	BGPEventIdleHoldTimerExp
	BGPEventTcpConnValid
	BGPEventTcpCrInvalid
	BGPEventTcpCrAcked
	BGPEventTcpConnConfirmed
	BGPEventTcpConnFails
	BGPEventBGPOpen
	BGPEventBGPOpenDelayOpenTimer
	BGPEventHeaderErr
	BGPEventOpenMsgErr
	BGPEventOpenCollisionDump
	BGPEventNotifMsgVerErr
	BGPEventNotifMsg
	BGPEventKeepAliveMsg
	BGPEventUpdateMsg
	BGPEventUpdateMsgErr
)

type BaseStateIface interface {
	processEvent(BGPFSMEvent, interface{})
	enter()
	leave()
	state() BGPFSMState
	String() string
}

type BaseState struct {
	fsm                 *FSM
	connectRetryCounter int
	connectRetryTimer   int
}

func (baseState *BaseState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("BaseState: processEvent", event)
}

func (baseState *BaseState) enter() {
	fmt.Println("BaseState: enter")
}

func (baseState *BaseState) leave() {
	fmt.Println("BaseState: leave")
}

func (baseState *BaseState) state() BGPFSMState {
	return BGPFSMNone
}

type IdleState struct {
	BaseState
}

func NewIdleState(fsm *FSM) *IdleState {
	state := IdleState{
		BaseState{
			fsm: fsm,
		},
	}
	return &state
}

func (st *IdleState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("IdleState: processEvent", event)
	switch event {
	case BGPEventManualStart, BGPEventAutoStart:
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.StartConnectRetryTimer()
		st.fsm.InitiateConnToPeer()
		st.fsm.AcceptPeerConn()
		st.fsm.ChangeState(NewConnectState(st.fsm))

	case BGPEventManualStartPassTcpEst, BGPEventAutoStartPassTcpEst:
		st.fsm.ChangeState(NewActiveState(st.fsm))

	case BGPEventAutoStartDampPeerOscl, BGPEventAutoStartDampPeerOsclPassTcpEst:
		st.fsm.SetIdleHoldTime(IdleHoldTimeInterval[st.fsm.GetIdleHoldTime()])
		st.fsm.StartIdleHoldTimer()

	case BGPEventIdleHoldTimerExp:
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.StartConnectRetryTimer()
		st.fsm.InitiateConnToPeer()
		st.fsm.AcceptPeerConn()
		st.fsm.ChangeState(NewConnectState(st.fsm))
	}
}

func (st *IdleState) enter() {
	fmt.Println("IdleState: enter")
	st.fsm.StopKeepAliveTimer()
	st.fsm.StopHoldTimer()
	st.fsm.RejectPeerConn()
	st.fsm.ApplyAutomaticStart()
}

func (st *IdleState) leave() {
	fmt.Println("IdleState: leave")
	st.fsm.StopIdleHoldTimer()
}

func (st *IdleState) state() BGPFSMState {
	return BGPFSMIdle
}

func (st *IdleState) String() string {
	return fmt.Sprintf("Idle")
}

type ConnectState struct {
	BaseState
}

func NewConnectState(fsm *FSM) *ConnectState {
	state := ConnectState{
		BaseState{
			fsm: fsm,
		},
	}
	return &state
}

func (st *ConnectState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("ConnectState: processEvent", event)
	switch event {
	case BGPEventManualStop:
		st.fsm.StopConnToPeer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventConnRetryTimerExp:
		st.fsm.StopConnToPeer()
		st.fsm.StartConnectRetryTimer()
		st.fsm.InitiateConnToPeer()

	case BGPEventDelayOpenTimerExp: // Supported later

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrInvalid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed:
		st.fsm.StopConnectRetryTimer()
		st.fsm.SetPeerConn(data)
		st.fsm.sendOpenMessage()
		st.fsm.SetHoldTime(BGPHoldTimeDefault)
		st.fsm.StartHoldTimer()
		st.BaseState.fsm.ChangeState(NewOpenSentState(st.BaseState.fsm))

	case BGPEventTcpConnFails:
		st.fsm.StopConnectRetryTimer()
		st.fsm.StopConnToPeer()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventBGPOpenDelayOpenTimer: // Supported later

	case BGPEventHeaderErr, BGPEventOpenMsgErr:
		st.fsm.StopConnectRetryTimer()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventNotifMsgVerErr:
		st.fsm.StopConnectRetryTimer()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop, BGPEventHoldTimerExp, BGPEventKeepAliveTimerExp, BGPEventIdleHoldTimerExp,
		BGPEventBGPOpen, BGPEventOpenCollisionDump, BGPEventNotifMsg, BGPEventKeepAliveMsg,
		BGPEventUpdateMsg, BGPEventUpdateMsgErr: // 8, 10, 11, 13, 19, 23, 25-28
		st.fsm.StopConnectRetryTimer()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *ConnectState) enter() {
	fmt.Println("ConnectState: enter")
}

func (st *ConnectState) leave() {
	fmt.Println("ConnectState: leave")
}

func (st *ConnectState) state() BGPFSMState {
	return BGPFSMConnect
}

func (st *ConnectState) String() string {
	return fmt.Sprintf("Connect")
}

type ActiveState struct {
	BaseState
}

func NewActiveState(fsm *FSM) *ActiveState {
	state := ActiveState{
		BaseState{
			fsm: fsm,
		},
	}
	return &state
}

func (st *ActiveState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("ActiveState: processEvent", event)

	switch event {
	case BGPEventManualStop:
		st.fsm.StopConnToPeer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventConnRetryTimerExp:
		st.fsm.StartConnectRetryTimer()
		st.fsm.InitiateConnToPeer()
		st.fsm.ChangeState(NewConnectState(st.fsm))

	case BGPEventDelayOpenTimerExp: // Supported later

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrInvalid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed:
		st.fsm.StopConnectRetryTimer()
		st.fsm.SetPeerConn(data)
		st.fsm.sendOpenMessage()
		st.fsm.SetHoldTime(BGPHoldTimeDefault)
		st.fsm.StartHoldTimer()
		st.fsm.ChangeState(NewOpenSentState(st.fsm))

	case BGPEventTcpConnFails:
		st.fsm.StartConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventBGPOpenDelayOpenTimer: // Supported later

	case BGPEventHeaderErr, BGPEventOpenMsgErr:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventNotifMsgVerErr:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop, BGPEventHoldTimerExp, BGPEventKeepAliveTimerExp, BGPEventIdleHoldTimerExp,
		BGPEventBGPOpen, BGPEventOpenCollisionDump, BGPEventNotifMsg, BGPEventKeepAliveMsg,
		BGPEventUpdateMsg, BGPEventUpdateMsgErr: // 8, 10, 11, 13, 19, 23, 25-28
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *ActiveState) enter() {
	fmt.Println("ActiveState: enter")
}

func (st *ActiveState) leave() {
	fmt.Println("ActiveState: leave")
}

func (st *ActiveState) state() BGPFSMState {
	return BGPFSMActive
}

func (st *ActiveState) String() string {
	return fmt.Sprintf("Active")
}

type OpenSentState struct {
	BaseState
}

func NewOpenSentState(fsm *FSM) *OpenSentState {
	state := OpenSentState{
		BaseState{
			fsm: fsm,
		},
	}
	return &state
}

func (st *OpenSentState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("OpenSentState: processEvent", event)

	switch event {
	case BGPEventManualStop:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventHoldTimerExp:
		st.fsm.SendNotificationMessage(BGPHoldTimerExpired, 0, nil	)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed: // Collistion detection... needs work

	case BGPEventTcpConnFails:
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.StartConnectRetryTimer()
		st.fsm.ChangeState(NewActiveState(st.fsm))

	case BGPEventBGPOpen:
		st.fsm.StopConnectRetryTimer()
		bgpMsg := data.(*BGPMessage)
		st.fsm.ProcessOpenMessage(bgpMsg)
		st.fsm.sendKeepAliveMessage()
		st.fsm.StartHoldTimer()
		st.fsm.ChangeState(NewOpenConfirmState(st.fsm))

	case BGPEventHeaderErr, BGPEventOpenMsgErr:
		bgpMsgErr := data.(*BGPMessageError)
		st.fsm.SendNotificationMessage(bgpMsgErr.TypeCode, bgpMsgErr.SubTypeCode, bgpMsgErr.Data)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventOpenCollisionDump:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventNotifMsgVerErr:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventConnRetryTimerExp, BGPEventKeepAliveTimerExp, BGPEventDelayOpenTimerExp,
		BGPEventIdleHoldTimerExp, BGPEventBGPOpenDelayOpenTimer, BGPEventNotifMsg,
		BGPEventKeepAliveMsg, BGPEventUpdateMsg, BGPEventUpdateMsgErr: // 9, 11, 12, 13, 20, 25-28
		st.fsm.SendNotificationMessage(BGPFSMError, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *OpenSentState) enter() {
	fmt.Println("OpenSentState: enter")
	//st.BaseState.fsm.startRxPkts()
}

func (st *OpenSentState) leave() {
	fmt.Println("OpenSentState: leave")
}

func (st *OpenSentState) state() BGPFSMState {
	return BGPFSMOpensent
}

func (st *OpenSentState) String() string {
	return fmt.Sprintf("Opensent")
}

type OpenConfirmState struct {
	BaseState
}

func NewOpenConfirmState(fsm *FSM) *OpenConfirmState {
	state := OpenConfirmState{
		BaseState{
			fsm: fsm,
		},
	}
	return &state
}

func (st *OpenConfirmState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("OpenConfirmState: processEvent", event)

	switch event {
	case BGPEventManualStop:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.StopConnectRetryTimer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventHoldTimerExp:
		st.fsm.SendNotificationMessage(BGPHoldTimerExpired, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveTimerExp:
		st.fsm.sendKeepAliveMessage()

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed: // Collision Detection... needs work

	case BGPEventTcpConnFails, BGPEventNotifMsg:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventBGPOpen: // Collision Detection... needs work

	case BGPEventHeaderErr, BGPEventOpenMsgErr:
		bgpMsgErr := data.(BGPMessageError)
		st.fsm.SendNotificationMessage(bgpMsgErr.TypeCode, bgpMsgErr.SubTypeCode, bgpMsgErr.Data)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventOpenCollisionDump:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventNotifMsgVerErr:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveMsg:
		st.fsm.StartHoldTimer()
		st.fsm.ChangeState(NewEstablishedState(st.fsm))

	case BGPEventConnRetryTimerExp, BGPEventDelayOpenTimerExp, BGPEventIdleHoldTimerExp,
		BGPEventBGPOpenDelayOpenTimer, BGPEventUpdateMsg, BGPEventUpdateMsgErr: // 9, 12, 13, 20, 27, 28
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *OpenConfirmState) enter() {
	fmt.Println("OpenConfirmState: enter")
}

func (st *OpenConfirmState) leave() {
	fmt.Println("OpenConfirmState: leave")
}

func (st *OpenConfirmState) state() BGPFSMState {
	return BGPFSMOpenconfirm
}

func (st *OpenConfirmState) String() string {
	return fmt.Sprintf("Openconfirm")
}

type EstablishedState struct {
	BaseState
}

func NewEstablishedState(fsm *FSM) *EstablishedState {
	state := EstablishedState{
		BaseState{
			fsm: fsm,
		},
	}
	return &state
}

func (st *EstablishedState) processEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("EstablishedState: processEvent", event)

	switch event {
	case BGPEventManualStop:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop:
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventHoldTimerExp:
		st.fsm.SendNotificationMessage(BGPHoldTimerExpired, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveTimerExp:
		st.fsm.sendKeepAliveMessage()

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed: // Collistion detection... needs work

	case BGPEventTcpConnFails, BGPEventNotifMsgVerErr, BGPEventNotifMsg:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		fmt.Println("Established: Stop Connection")
		st.fsm.StopConnToPeer()
		fmt.Println("Established: Stopped Connection")
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventBGPOpen: // Collistion detection... needs work

	case BGPEventOpenCollisionDump: // Collistion detection... needs work
		st.fsm.SendNotificationMessage(BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveMsg:
		st.fsm.StartHoldTimer()

	case BGPEventUpdateMsg:
		st.fsm.StartHoldTimer()

	case BGPEventUpdateMsgErr:
		bgpMsgErr := data.(BGPMessageError)
		st.fsm.SendNotificationMessage(bgpMsgErr.TypeCode, bgpMsgErr.SubTypeCode, bgpMsgErr.Data)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventConnRetryTimerExp, BGPEventDelayOpenTimerExp, BGPEventIdleHoldTimerExp,
		BGPEventOpenMsgErr, BGPEventBGPOpenDelayOpenTimer, BGPEventHeaderErr: // 9, 12, 13, 20, 21, 22
		st.fsm.SendNotificationMessage(BGPFSMError, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *EstablishedState) enter() {
	fmt.Println("EstablishedState: enter")
	st.fsm.SetIdleHoldTime(BGPIdleHoldTimeDefault)
}

func (st *EstablishedState) leave() {
	fmt.Println("EstablishedState: leave")
}

func (st *EstablishedState) state() BGPFSMState {
	return BGPFSMEstablished
}

func (st *EstablishedState) String() string {
	return fmt.Sprintf("Established")
}

type FSMIface interface {
	StartFSM(state BaseStateIface)
	ProcessEvent(event BGPFSMEvent)
	ChangeState(state BaseStateIface)
}

type BGPPktInfo struct {
	msg *BGPMessage
	msgError *BGPMessageError
}

type PeerConnDir struct {
	connDir ConnDir
	conn *net.Conn
}

type FSM struct {
	gConf    *GlobalConfig
	pConf    *PeerConfig
	Manager  *FSMManager
	State    BaseStateIface
	connDir  ConnDir
	peerType PeerType
    peerConn *PeerConn

	outConnCh    chan net.Conn
	outConnErrCh chan error
	stopConnCh   chan bool
	inConnCh     chan net.Conn
	connInProgress bool

	conn  net.Conn
	event BGPFSMEvent

	connectRetryCounter int
	connectRetryTime    uint16
	connectRetryTimer   *time.Timer

	holdTime  uint16
	holdTimer *time.Timer

	keepAliveTime  uint16
	keepAliveTimer *time.Timer

	autoStart bool
	autoStop bool
	passiveTcpEst bool
	dampPeerOscl bool
	idleHoldTime uint16
	idleHoldTimer *time.Timer

	delayOpen      bool
	delayOpenTime  uint16
	delayOpenTimer *time.Timer

	pktRxCh    chan *BGPPktInfo
	eventRxCh  chan BGPFSMEvent
	rxPktsFlag bool
}

func NewFSM(fsmManager *FSMManager, connDir ConnDir, gConf *GlobalConfig, pConf *PeerConfig) *FSM {
	fsm := FSM{
		gConf:            gConf,
		pConf:            pConf,
		Manager:          fsmManager,
		connDir:          connDir,
		connectRetryTime: BGPConnectRetryTime,      // seconds
		holdTime:         BGPHoldTimeDefault,       // seconds
		keepAliveTime:    (BGPHoldTimeDefault / 3), // seconds
		rxPktsFlag:       false,
		outConnCh:        make(chan net.Conn),
		outConnErrCh:     make(chan error),
		stopConnCh:       make(chan bool),
		inConnCh:         make(chan net.Conn),
		connInProgress:   false,
		autoStart:        true,
		autoStop:         true,
		passiveTcpEst:    false,
		dampPeerOscl:     true,
		idleHoldTime:     BGPIdleHoldTimeDefault,
	}
	fsm.pktRxCh = make(chan *BGPPktInfo)
	fsm.eventRxCh = make(chan BGPFSMEvent)
	fsm.connectRetryTimer = time.NewTimer(time.Duration(fsm.connectRetryTime) * time.Second)
	fsm.holdTimer = time.NewTimer(time.Duration(fsm.holdTime) * time.Second)
	fsm.keepAliveTimer = time.NewTimer(time.Duration(fsm.keepAliveTime) * time.Second)
	fsm.idleHoldTimer = time.NewTimer(time.Duration(fsm.idleHoldTime) * time.Second)

	fsm.connectRetryTimer.Stop()
	fsm.holdTimer.Stop()
	fsm.keepAliveTimer.Stop()
	fsm.idleHoldTimer.Stop()
	return &fsm
}

func (fsm *FSM) SetConn(conn net.Conn) {
	fsm.conn = conn
}

func (fsm *FSM) StartFSM(state BaseStateIface) {
	fmt.Println("FSM: Starting the stach machine in", state.state(), "state")
	fsm.State = state
	fsm.State.enter()

	for {
		select {
		case outConnCh := <-fsm.outConnCh:
			fsm.connInProgress = false
			out := PeerConnDir{ConnDirOut, &outConnCh}
			fsm.ProcessEvent(BGPEventTcpCrAcked, out)

		case outConnErrCh := <-fsm.outConnErrCh:
			fsm.connInProgress = false
			fsm.ProcessEvent(BGPEventTcpConnFails, outConnErrCh)

		case inConnCh := <-fsm.inConnCh:
			in := PeerConnDir{ConnDirOut, &inConnCh}
			fsm.ProcessEvent(BGPEventTcpConnConfirmed, in)

		case bgpPktInfo := <-fsm.pktRxCh:
			fsm.ProcessPacket(bgpPktInfo.msg, bgpPktInfo.msgError)

		case event := <-fsm.eventRxCh:
			fsm.ProcessEvent(event, nil)

		case <-fsm.connectRetryTimer.C:
			fsm.ProcessEvent(BGPEventConnRetryTimerExp, nil)

		case <-fsm.holdTimer.C:
			fsm.ProcessEvent(BGPEventHoldTimerExp, nil)

		case <-fsm.keepAliveTimer.C:
			fsm.ProcessEvent(BGPEventKeepAliveTimerExp, nil)

		case <-fsm.idleHoldTimer.C:
			fsm.ProcessEvent(BGPEventIdleHoldTimerExp, nil)
		}
	}
}

func (fsm *FSM) ProcessEvent(event BGPFSMEvent, data interface{}) {
	fmt.Println("FSM: ProcessEvent", event)
	fsm.event = event
	fsm.State.processEvent(event, data)
}

func (fsm *FSM) ProcessPacket(msg *BGPMessage, msgErr *BGPMessageError) {
	var event BGPFSMEvent
	var data interface{}

	if msgErr != nil {
		data = msgErr
		switch msgErr.TypeCode {
			case BGPMsgHeaderError:
				event = BGPEventHeaderErr

			case BGPOpenMsgError:
				event = BGPEventOpenMsgErr

			case BGPUpdateMsgError:
				event = BGPEventUpdateMsgErr
		}
	} else {
		data = msg
		switch msg.Header.Type {
		case BGPMsgTypeOpen:
			event = BGPEventBGPOpen

		case BGPMsgTypeUpdate:
			event = BGPEventUpdateMsg

		case BGPMsgTypeNotification:
			event = BGPEventNotifMsg

		case BGPMsgTypeKeepAlive:
			event = BGPEventKeepAliveMsg
		}
	}
	fmt.Println("FSM:ProcessPacket - event =", event)
	fsm.ProcessEvent(event, data)
}

func (fsm *FSM) ChangeState(newState BaseStateIface) {
	fmt.Println("FSM: ChangeState: Leaving", fsm.State, "state Entering", newState, "state")
	fsm.State.leave()
	fsm.State = newState
	fsm.State.enter()
}

func (fsm *FSM) ApplyAutomaticStart() {
	if fsm.autoStart {
		event := BGPEventAutoStart

		if fsm.passiveTcpEst {
			if fsm.dampPeerOscl {
				event = BGPEventAutoStartDampPeerOsclPassTcpEst
			} else {
				event = BGPEventAutoStartPassTcpEst
			}
		} else if fsm.dampPeerOscl {
			event = BGPEventAutoStartDampPeerOscl
		}

		fsm.ProcessEvent(event, nil)
	}
}
func (fsm *FSM) StartConnectRetryTimer() {
	fsm.connectRetryTimer.Reset(time.Duration(fsm.connectRetryTime) * time.Second)
}

func (fsm *FSM) StopConnectRetryTimer() {
	fsm.connectRetryTimer.Stop()
}

func (fsm *FSM) SetHoldTime(holdTime uint16) {
	if holdTime < 0 || (holdTime > 0 && holdTime < 3) {
		fmt.Println("Cannot set hold time. Invalid value", holdTime)
		return
	}

	fsm.holdTime = holdTime
	fsm.keepAliveTime = holdTime / 3
}

func (fsm *FSM) StartHoldTimer() {
	if fsm.holdTime != 0 {
		fsm.holdTimer.Reset(time.Duration(fsm.holdTime) * time.Second)
	}
}

func (fsm *FSM) StopHoldTimer() {
	fsm.holdTimer.Stop()
}

func (fsm *FSM) StartKeepAliveTimer() {
	if fsm.keepAliveTime != 0 {
		fsm.keepAliveTimer.Reset(time.Duration(fsm.keepAliveTime) * time.Second)
	}
}

func (fsm *FSM) StopKeepAliveTimer() {
	fsm.keepAliveTimer.Stop()
}

func (fsm *FSM) SetConnectRetryCounter(value int) {
	fsm.connectRetryCounter = value
}

func (fsm *FSM) IncrConnectRetryCounter() {
	fsm.connectRetryCounter++
}

func (fsm *FSM) GetIdleHoldTime() uint16 {
	return fsm.idleHoldTime
}

func (fsm *FSM) SetIdleHoldTime(seconds uint16) {
	fsm.idleHoldTime = seconds
}

func (fsm *FSM) StartIdleHoldTimer() {
	if fsm.idleHoldTime > 0 && fsm.idleHoldTime <= 300 {
		fsm.idleHoldTimer.Reset(time.Duration(fsm.idleHoldTime) * time.Second)
	}
}

func (fsm *FSM) StopIdleHoldTimer() {
	fsm.idleHoldTimer.Stop()
}

func (fsm *FSM) ProcessOpenMessage(pkt *BGPMessage) {
	body := pkt.Body.(*BGPOpen)
	if body.HoldTime < fsm.holdTime {
		fsm.holdTime = body.HoldTime
		fsm.keepAliveTime = fsm.holdTime / 3
	}
	if body.MyAS == fsm.Manager.gConf.AS {
		fsm.peerType = PeerTypeInternal
	} else {
		fsm.peerType = PeerTypeExternal
	}
}

func (fsm *FSM) sendOpenMessage() {
	bgpOpenMsg := NewBGPOpenMessage(fsm.pConf.AS, fsm.holdTime, IP)
	packet, _ := bgpOpenMsg.Encode()
	num, err := (*fsm.peerConn.conn).Write(packet)
	if err != nil {
		fmt.Println("Conn.Write failed to send Open message with error:", err)
	}
	fmt.Println("Conn.Write succeeded. sent Open message of", num, "bytes")
}

func (fsm *FSM) sendKeepAliveMessage() {
	bgpKeepAliveMsg := NewBGPKeepAliveMessage()
	packet, _ := bgpKeepAliveMsg.Encode()
	num, err := (*fsm.peerConn.conn).Write(packet)
	if err != nil {
		fmt.Println("Conn.Write failed to send KeepAlive message with error:", err)
	}
	fmt.Println("Conn.Write succeeded. sent KeepAlive message of", num, "bytes")
	fsm.StartKeepAliveTimer()
}

func (fsm *FSM) SendNotificationMessage(code uint8, subCode uint8, data []byte) {
	bgpNotifMsg := NewBGPNotificationMessage(code, subCode, data)
	packet, _ := bgpNotifMsg.Encode()
	num, err := (*fsm.peerConn.conn).Write(packet)
	if err != nil {
		fmt.Println("Conn.Write failed to send Notification message with error:", err)
	}
	fmt.Println("Conn.Write succeeded. sent Notification message with", num, "bytes")
}

func (fsm *FSM) SetPeerConn(data interface{}) {
	fmt.Println("SetPeerConn called")
	if fsm.peerConn != nil {
		fmt.Println("FSM:SetupPeerConn - Peer conn is already set up")
		return
	}
	pConnDir := data.(PeerConnDir)
	fsm.peerConn = NewPeerConn(fsm, pConnDir.connDir, pConnDir.conn)
	go fsm.peerConn.StartReading()
}

func (fsm *FSM) ClearPeerConn() {
	fmt.Println("ClearPeerConn called")
	if fsm.peerConn == nil {
		fmt.Println("FSM:ClearPeerConn - Peer conn is not set up yet")
		return
	}
	fsm.StopKeepAliveTimer()
	fsm.StopHoldTimer()
	fsm.peerConn.StopReading()
	fsm.peerConn = nil
}

func (fsm *FSM) startRxPkts() {
	fmt.Println("fsm:startRxPkts called")
	if fsm.peerConn != nil && !fsm.rxPktsFlag {
		fsm.rxPktsFlag = true
		fsm.peerConn.StartReading()
	}
}

func (fsm *FSM) stopRxPkts() {
	fmt.Println("fsm:stopRxPkts called")
	if fsm.peerConn != nil && fsm.rxPktsFlag {
		fsm.rxPktsFlag = false
		fsm.peerConn.StopReading()
	}
}

func (fsm *FSM) AcceptPeerConn() {
	fmt.Println("AcceptPeerConn called")
    fsm.Manager.AcceptPeerConn()
}

func (fsm *FSM) RejectPeerConn() {
	fmt.Println("RejectPeerConn called")
    fsm.Manager.RejectPeerConn()
}

func (fsm *FSM) InitiateConnToPeer() {
	fmt.Println("InitiateConnToPeer called")
	addr := net.JoinHostPort(fsm.pConf.IP.String(), BGPPort)
	if !fsm.connInProgress {
		fsm.connInProgress = true
		go ConnectToPeer(fsm.connectRetryTime, addr, fsm.outConnCh, fsm.outConnErrCh, fsm.stopConnCh)
	}
}

func (fsm *FSM) StopConnToPeer() {
	fmt.Println("StopConnToPeer called")
	if fsm.connInProgress {
		fsm.stopConnCh <- true
	}
}

func Connect(seconds uint16, addr string, connCh chan net.Conn, errCh chan error) {
	fmt.Println("Connect called... calling DialTimeout with", seconds, "second timeout")
	conn, err := net.DialTimeout("tcp", addr, time.Duration(seconds) * time.Second)
	if err != nil {
		errCh <- err
	} else {
		connCh <- conn
	}
}

func ConnectToPeer(seconds uint16, addr string, fsmConnCh chan net.Conn, fsmConnErrCh chan error, fsmStopConnCh chan bool) {
	var stopConn bool = false
	connCh := make(chan net.Conn)
	errCh := make(chan error)

	fmt.Println("ConnectToPeer called")
	connTime := seconds - 3
	if connTime <= 0 {
		connTime = seconds
	}

	go Connect(seconds, addr, connCh, errCh)

	for {
		select {
		case conn := <-connCh:
			fmt.Println("ConnectToPeer: Connected to peer", addr)
			if stopConn {
				conn.Close()
				return
			}

			fsmConnCh <- conn
			return

		case err := <-errCh:
			fmt.Println("ConnectToPeer: Failed to connect to peer", addr)
			if stopConn {
				return
			}

			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				fmt.Println("Connect to peer timed out, retrying...")
				go Connect(3, addr, connCh, errCh)
			} else {
				fmt.Println("Connect to peer failed with error:", err)
				fsmConnErrCh <- err
			}

		case <-fsmStopConnCh:
			fmt.Println("ConnectToPeer: Recieved stop connecting to peer", addr)
			stopConn = true
		}
	}
}
