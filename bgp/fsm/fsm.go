// fsm.go
package fsm

import (
	"bytes"
	"fmt"
	"l3/bgp/baseobjects"
	"l3/bgp/config"
	"l3/bgp/packet"
	"net"
	"sync/atomic"
	"time"
	"utils/logging"
)

type BGPFSMState int

const BGPConnectRetryTime uint16 = 120  // seconds
const BGPHoldTimeDefault uint16 = 9     // 240 seconds
const BGPIdleHoldTimeDefault uint16 = 5 // 240 seconds

var IdleHoldTimeInterval = map[uint16]uint16{
	0:   0,
	5:   10,
	10:  30,
	60:  120,
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

var BGPEventTypeToStr = map[BGPFSMEvent]string{
	BGPEventManualStart:                     "ManualStart",
	BGPEventManualStop:                      "ManualStop",
	BGPEventAutoStart:                       "AutoStart",
	BGPEventManualStartPassTcpEst:           "ManualStartPassTcpEst",
	BGPEventAutoStartPassTcpEst:             "AutoStartPassTcpEst",
	BGPEventAutoStartDampPeerOscl:           "AutoStartDampPeerOscl",
	BGPEventAutoStartDampPeerOsclPassTcpEst: "AutoStartDampPeerOsclPassTcpEst",
	BGPEventAutoStop:                        "AutoStop",
	BGPEventConnRetryTimerExp:               "ConnRetryTimerExp",
	BGPEventHoldTimerExp:                    "HoldTimerExp",
	BGPEventKeepAliveTimerExp:               "KeepAliveTimerExp",
	BGPEventDelayOpenTimerExp:               "DelayOpenTimerExp",
	BGPEventIdleHoldTimerExp:                "IdleHoldTimerExp",
	BGPEventTcpConnValid:                    "TcpConnValid",
	BGPEventTcpCrInvalid:                    "TcpCrInvalid",
	BGPEventTcpCrAcked:                      "TcpCrAcked",
	BGPEventTcpConnConfirmed:                "TcpConnConfirmed",
	BGPEventTcpConnFails:                    "TcpConnFails",
	BGPEventBGPOpen:                         "BGPOpen",
	BGPEventBGPOpenDelayOpenTimer:           "BGPOpenDelayOpenTimer",
	BGPEventHeaderErr:                       "HeaderErr",
	BGPEventOpenMsgErr:                      "OpenMsgErr",
	BGPEventOpenCollisionDump:               "OpenCollisionDump",
	BGPEventNotifMsgVerErr:                  "NotifMsgVerErr",
	BGPEventNotifMsg:                        "NotifMsg",
	BGPEventKeepAliveMsg:                    "KeepAliveMsg",
	BGPEventUpdateMsg:                       "UpdateMsg",
	BGPEventUpdateMsgErr:                    "UpdateMsgErr",
}

type BaseStateIface interface {
	processEvent(BGPFSMEvent, interface{})
	enter()
	leave()
	state() BGPFSMState
	String() string
}

type BaseState struct {
	fsm                 *FSM
	logger              *logging.Writer
	connectRetryCounter int
	connectRetryTimer   int
}

func NewBaseState(fsm *FSM) BaseState {
	state := BaseState{
		fsm:    fsm,
		logger: fsm.logger,
	}
	return state
}

func (baseState *BaseState) processEvent(event BGPFSMEvent, data interface{}) {
	baseState.logger.Info(fmt.Sprintln("BaseState: processEvent", event))
}

func (baseState *BaseState) enter() {
	baseState.logger.Info(fmt.Sprintln("BaseState: enter"))
}

func (baseState *BaseState) leave() {
	baseState.logger.Info(fmt.Sprintln("BaseState: leave"))
}

func (baseState *BaseState) state() BGPFSMState {
	return BGPFSMNone
}

type IdleState struct {
	BaseState
	passive bool
}

func NewIdleState(fsm *FSM) *IdleState {
	state := IdleState{
		BaseState: NewBaseState(fsm),
		passive:   false,
	}
	return &state
}

func (st *IdleState) processEvent(event BGPFSMEvent, data interface{}) {
	if st.fsm.neighborConf.Neighbor.State.BfdNeighborState == "down" {
		st.logger.Info(fmt.Sprintln("Bfd is down for neighbor: ", st.fsm.pConf.NeighborAddress,
			" do not process event: ", BGPEventTypeToStr[event]))
		return
	}
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Idle Event:", BGPEventTypeToStr[event]))
	switch event {
	case BGPEventManualStart, BGPEventAutoStart:
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.StartConnectRetryTimer()
		st.fsm.InitiateConnToPeer()
		st.fsm.AcceptPeerConn()
		st.fsm.ChangeState(NewConnectState(st.fsm))

	case BGPEventManualStartPassTcpEst, BGPEventAutoStartPassTcpEst:
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.StartConnectRetryTimer()
		st.fsm.AcceptPeerConn()
		st.fsm.ChangeState(NewActiveState(st.fsm))

	case BGPEventAutoStartDampPeerOscl, BGPEventAutoStartDampPeerOsclPassTcpEst:
		st.fsm.SetIdleHoldTime(IdleHoldTimeInterval[st.fsm.GetIdleHoldTime()])
		st.fsm.StartIdleHoldTimer()
		if event == BGPEventAutoStartDampPeerOsclPassTcpEst {
			st.passive = true
		} else {
			st.passive = false
		}

	case BGPEventIdleHoldTimerExp:
		//st.fsm.SetConnectRetryCounter(0)
		st.fsm.StartConnectRetryTimer()
		if st.passive {
			st.fsm.AcceptPeerConn()
			st.fsm.ChangeState(NewActiveState(st.fsm))
		} else {
			st.fsm.InitiateConnToPeer()
			st.fsm.AcceptPeerConn()
			st.fsm.ChangeState(NewConnectState(st.fsm))
		}
	}
}

func (st *IdleState) enter() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id, "State: Idle - enter"))
	st.logger.Info(fmt.Sprintln("IdleState: enter"))
	st.fsm.StopKeepAliveTimer()
	st.fsm.StopHoldTimer()
	st.fsm.RejectPeerConn()
	st.fsm.ApplyAutomaticStart()
}

func (st *IdleState) leave() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id, "State: Idle - leave"))
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
		BaseState: NewBaseState(fsm),
	}
	return &state
}

func (st *ConnectState) processEvent(event BGPFSMEvent, data interface{}) {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id, "State: Connect Event:",
		BGPEventTypeToStr[event]))
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
		st.fsm.AcceptPeerConn()

	case BGPEventDelayOpenTimerExp: // Supported later

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrInvalid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed:
		st.fsm.StopConnectRetryTimer()
		st.fsm.SetPeerConn(data)
		st.fsm.sendOpenMessage()
		st.fsm.SetHoldTime(st.fsm.neighborConf.RunningConf.HoldTime,
			st.fsm.neighborConf.RunningConf.KeepaliveTime)
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
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Connect - enter"))
}

func (st *ConnectState) leave() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Connect - leave"))
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
		BaseState: NewBaseState(fsm),
	}
	return &state
}

func (st *ActiveState) processEvent(event BGPFSMEvent, data interface{}) {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id, "State: Active Event:",
		BGPEventTypeToStr[event]))

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
		st.fsm.SetHoldTime(st.fsm.neighborConf.RunningConf.HoldTime,
			st.fsm.neighborConf.RunningConf.KeepaliveTime)
		st.fsm.StartHoldTimer()
		st.fsm.ChangeState(NewOpenSentState(st.fsm))

	case BGPEventTcpConnFails:
		st.fsm.StartConnectRetryTimer()
		st.fsm.StopConnToPeer()
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
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Active - enter"))
}

func (st *ActiveState) leave() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Active - leave"))
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
		BaseState: NewBaseState(fsm),
	}
	return &state
}

func (st *OpenSentState) processEvent(event BGPFSMEvent, data interface{}) {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: OpenSent Event:", BGPEventTypeToStr[event]))

	switch event {
	case BGPEventManualStop:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventHoldTimerExp:
		st.fsm.SendNotificationMessage(packet.BGPHoldTimerExpired, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed: // Collistion detection... needs work
		st.fsm.HandleAnotherConnection(data)

	case BGPEventTcpConnFails:
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.StartConnectRetryTimer()
		st.fsm.AcceptPeerConn()
		st.fsm.ChangeState(NewActiveState(st.fsm))

	case BGPEventBGPOpen:
		st.fsm.StopConnectRetryTimer()
		bgpMsg := data.(*packet.BGPMessage)
		if st.fsm.ProcessOpenMessage(bgpMsg) {
			st.fsm.sendKeepAliveMessage()
			st.fsm.StartHoldTimer()
			st.fsm.ChangeState(NewOpenConfirmState(st.fsm))
		}

	case BGPEventHeaderErr, BGPEventOpenMsgErr:
		bgpMsgErr := data.(*packet.BGPMessageError)
		st.fsm.SendNotificationMessage(bgpMsgErr.TypeCode, bgpMsgErr.SubTypeCode, bgpMsgErr.Data)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventOpenCollisionDump:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
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
		st.fsm.SendNotificationMessage(packet.BGPFSMError, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *OpenSentState) enter() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: OpenSent - enter"))
	//st.BaseState.fsm.startRxPkts()
}

func (st *OpenSentState) leave() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: OpenSent - leave"))
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
		BaseState: NewBaseState(fsm),
	}
	return &state
}

func (st *OpenConfirmState) processEvent(event BGPFSMEvent, data interface{}) {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: OpenConfirm Event:", BGPEventTypeToStr[event]))

	switch event {
	case BGPEventManualStop:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.StopConnectRetryTimer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventHoldTimerExp:
		st.fsm.SendNotificationMessage(packet.BGPHoldTimerExpired, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveTimerExp:
		st.fsm.sendKeepAliveMessage()

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed: // Collision Detection... needs work
		st.fsm.HandleAnotherConnection(data)

	case BGPEventTcpConnFails, BGPEventNotifMsg:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventBGPOpen: // Collision Detection... needs work

	case BGPEventHeaderErr, BGPEventOpenMsgErr:
		bgpMsgErr := data.(*packet.BGPMessageError)
		st.fsm.SendNotificationMessage(bgpMsgErr.TypeCode, bgpMsgErr.SubTypeCode, bgpMsgErr.Data)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventOpenCollisionDump:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
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
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *OpenConfirmState) enter() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: OpenConfirm - enter"))
}

func (st *OpenConfirmState) leave() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: OpenConfirm - leave"))
	st.fsm.SetHoldTime(st.fsm.neighborConf.RunningConf.HoldTime,
		st.fsm.neighborConf.RunningConf.KeepaliveTime)
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
		BaseState: NewBaseState(fsm),
	}
	return &state
}

func (st *EstablishedState) processEvent(event BGPFSMEvent, data interface{}) {
	if event != BGPEventKeepAliveMsg && event != BGPEventKeepAliveTimerExp {
		st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
			"State: Established Event:", BGPEventTypeToStr[event]))
	}

	switch event {
	case BGPEventManualStop:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.SetConnectRetryCounter(0)
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventAutoStop:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventHoldTimerExp:
		st.fsm.SendNotificationMessage(packet.BGPHoldTimerExpired, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveTimerExp:
		st.fsm.sendKeepAliveMessage()

	case BGPEventTcpConnValid: // Supported later

	case BGPEventTcpCrAcked, BGPEventTcpConnConfirmed: // Collistion detection... needs work
		st.fsm.HandleAnotherConnection(data)

	case BGPEventTcpConnFails, BGPEventNotifMsgVerErr, BGPEventNotifMsg:
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
			"Established: Stop Connection"))
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventBGPOpen: // Collistion detection... needs work

	case BGPEventOpenCollisionDump:
		st.fsm.SendNotificationMessage(packet.BGPCease, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventKeepAliveMsg:
		st.fsm.StartHoldTimer()

	case BGPEventUpdateMsg:
		st.fsm.StartHoldTimer()
		bgpMsg := data.(*packet.BGPMessage)
		st.fsm.ProcessUpdateMessage(bgpMsg)

	case BGPEventUpdateMsgErr:
		bgpMsgErr := data.(*packet.BGPMessageError)
		st.fsm.SendNotificationMessage(bgpMsgErr.TypeCode, bgpMsgErr.SubTypeCode, bgpMsgErr.Data)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))

	case BGPEventConnRetryTimerExp, BGPEventDelayOpenTimerExp, BGPEventIdleHoldTimerExp,
		BGPEventOpenMsgErr, BGPEventBGPOpenDelayOpenTimer, BGPEventHeaderErr: // 9, 12, 13, 20, 21, 22
		st.fsm.SendNotificationMessage(packet.BGPFSMError, 0, nil)
		st.fsm.StopConnectRetryTimer()
		st.fsm.ClearPeerConn()
		st.fsm.StopConnToPeer()
		st.fsm.IncrConnectRetryCounter()
		st.fsm.ChangeState(NewIdleState(st.fsm))
	}
}

func (st *EstablishedState) enter() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Established - enter"))
	st.fsm.SetIdleHoldTime(BGPIdleHoldTimeDefault)
}

func (st *EstablishedState) leave() {
	st.logger.Info(fmt.Sprintln("Neighbor:", st.fsm.pConf.NeighborAddress, "FSM:", st.fsm.id,
		"State: Established - leave"))
	st.fsm.SetHoldTime(st.fsm.neighborConf.RunningConf.HoldTime,
		st.fsm.neighborConf.RunningConf.KeepaliveTime)
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

type PeerConnDir struct {
	connDir config.ConnDir
	conn    *net.Conn
}

type PeerFSMConnState struct {
	isEstablished bool
	id            uint8
	connDir       config.ConnDir
	conn          *net.Conn
}

type PeerFSMEvent struct {
	event  BGPFSMEvent
	reason int
}

type FSM struct {
	logger       *logging.Writer
	neighborConf *base.NeighborConf
	gConf        *config.GlobalConfig
	pConf        *config.NeighborConfig
	Manager      *FSMManager
	State        BaseStateIface
	id           uint8
	peerType     config.PeerType
	peerConn     *PeerConn

	outConnCh    chan net.Conn
	outConnErrCh chan error
	stopConnCh   chan bool
	inConnCh     chan net.Conn
	closeCh      chan bool
	outTCPConn   *OutTCPConn

	event BGPFSMEvent

	connectRetryCounter int
	connectRetryTime    uint32
	connectRetryTimer   *time.Timer

	holdTime  uint32
	holdTimer *time.Timer

	keepAliveTime  uint32
	keepAliveTimer *time.Timer

	restartTime  uint32
	restartTimer *time.Timer

	autoStart       bool
	autoStop        bool
	passiveTcpEst   bool
	passiveTcpEstCh chan bool
	dampPeerOscl    bool
	idleHoldTime    uint16
	idleHoldTimer   *time.Timer

	delayOpen      bool
	delayOpenTime  uint16
	delayOpenTimer *time.Timer

	afiSafiMap map[uint32]bool
	pktTxCh    chan *packet.BGPMessage
	pktRxCh    chan *packet.BGPPktInfo
	eventRxCh  chan PeerFSMEvent
	rxPktsFlag bool

	cleanup bool
}

func NewFSM(fsmManager *FSMManager, id uint8, neighborConf *base.NeighborConf) *FSM {
	fsm := FSM{
		logger:           fsmManager.logger,
		neighborConf:     neighborConf,
		gConf:            neighborConf.Global,
		pConf:            &neighborConf.RunningConf,
		Manager:          fsmManager,
		id:               id,
		connectRetryTime: neighborConf.RunningConf.ConnectRetryTime, // seconds
		holdTime:         neighborConf.RunningConf.HoldTime,         // seconds
		keepAliveTime:    neighborConf.RunningConf.KeepaliveTime,    // seconds
		restartTime:      0,                                         // seconds
		rxPktsFlag:       false,
		outConnCh:        make(chan net.Conn),
		outConnErrCh:     make(chan error, 2),
		stopConnCh:       make(chan bool),
		inConnCh:         make(chan net.Conn),
		closeCh:          make(chan bool, 2),
		outTCPConn:       nil,
		autoStart:        true,
		autoStop:         true,
		passiveTcpEst:    false,
		passiveTcpEstCh:  make(chan bool, 2),
		dampPeerOscl:     false,
		idleHoldTime:     BGPIdleHoldTimeDefault,
		afiSafiMap:       make(map[uint32]bool),
		cleanup:          false,
	}

	fsm.pktTxCh = make(chan *packet.BGPMessage)
	fsm.pktRxCh = make(chan *packet.BGPPktInfo, 2)
	fsm.eventRxCh = make(chan PeerFSMEvent, 5)
	fsm.connectRetryTimer = time.NewTimer(time.Duration(fsm.connectRetryTime) * time.Second)
	fsm.connectRetryTimer.Stop()

	fsm.holdTimer = time.NewTimer(time.Duration(fsm.holdTime) * time.Second)
	fsm.holdTimer.Stop()

	fsm.keepAliveTimer = time.NewTimer(time.Duration(fsm.keepAliveTime) * time.Second)
	fsm.keepAliveTimer.Stop()

	fsm.idleHoldTimer = time.NewTimer(time.Duration(fsm.idleHoldTime) * time.Second)
	fsm.idleHoldTimer.Stop()

	fsm.restartTimer = time.NewTimer(time.Duration(5) * time.Second)
	fsm.restartTimer.Stop()

	return &fsm
}

func (fsm *FSM) StartFSM(state BaseStateIface) {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM:", fsm.id,
		"Starting the state machine in", state.state(), "state"))
	fsm.State = state
	fsm.State.enter()

	for {
		select {
		case outConnCh := <-fsm.outConnCh:
			fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
				"OUT connection SUCCESS"))
			fsm.outTCPConn = nil
			out := PeerConnDir{config.ConnDirOut, &outConnCh}
			fsm.ProcessEvent(BGPEventTcpCrAcked, out)

		case outConnErrCh := <-fsm.outConnErrCh:
			fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
				"connection FAIL"))
			fsm.outTCPConn = nil
			fsm.ProcessEvent(BGPEventTcpConnFails, outConnErrCh)
			if !fsm.cleanup {
				fsm.Manager.tcpConnFailCh <- fsm.id
			}

		case inConnCh := <-fsm.inConnCh:
			fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
				"IN connection SUCCESS"))
			in := PeerConnDir{config.ConnDirIn, &inConnCh}
			fsm.ProcessEvent(BGPEventTcpConnConfirmed, in)

		case <-fsm.closeCh:
			fsm.logger.Info(fmt.Sprintf("Neighbor: %s FSM %d received close", fsm.pConf.NeighborAddress.String(),
				fsm.id))
			fsm.cleanup = true
			fsm.ProcessEvent(BGPEventManualStop, nil)
			return

		case val := <-fsm.passiveTcpEstCh:
			fsm.SetPassiveTcpEstablishment(val)

		case bgpMsg := <-fsm.pktTxCh:
			if fsm.State.state() != BGPFSMEstablished {
				fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
					"is not in Established state, can't send the UPDATE message"))
				continue
			}
			fsm.sendUpdateMessage(bgpMsg)

		case bgpPktInfo := <-fsm.pktRxCh:
			fsm.ProcessPacket(bgpPktInfo.Msg, bgpPktInfo.MsgError)

		case fsmEvent := <-fsm.eventRxCh:
			fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
				"Received event", fsmEvent.event, "reason", fsmEvent.reason))
			if fsmEvent.reason == BGPCmdReasonMaxPrefixExceeded {
				fsm.restartTime = uint32(fsm.neighborConf.RunningConf.MaxPrefixesRestartTimer)
			}
			fsm.ProcessEvent(fsmEvent.event, nil)
			if fsmEvent.reason != BGPCmdReasonNone {
				fsm.restartTime = 0
			}

		case <-fsm.connectRetryTimer.C:
			fsm.ProcessEvent(BGPEventConnRetryTimerExp, nil)

		case <-fsm.holdTimer.C:
			fsm.ProcessEvent(BGPEventHoldTimerExp, nil)

		case <-fsm.keepAliveTimer.C:
			fsm.ProcessEvent(BGPEventKeepAliveTimerExp, nil)

		case <-fsm.idleHoldTimer.C:
			fsm.ProcessEvent(BGPEventIdleHoldTimerExp, nil)

		case <-fsm.restartTimer.C:
			fsm.ProcessEvent(BGPEventAutoStart, nil)
		}
	}
}

func (fsm *FSM) ProcessEvent(event BGPFSMEvent, data interface{}) {
	fsm.event = event
	fsm.State.processEvent(event, data)
}

func (fsm *FSM) ProcessPacket(msg *packet.BGPMessage, msgErr *packet.BGPMessageError) {
	var event BGPFSMEvent
	var data interface{}

	if msgErr != nil {
		data = msgErr
		switch msgErr.TypeCode {
		case packet.BGPMsgHeaderError:
			event = BGPEventHeaderErr

		case packet.BGPOpenMsgError:
			event = BGPEventOpenMsgErr

		case packet.BGPUpdateMsgError:
			event = BGPEventUpdateMsgErr
		}
	} else {
		data = msg
		switch msg.Header.Type {
		case packet.BGPMsgTypeOpen:
			event = BGPEventBGPOpen

		case packet.BGPMsgTypeUpdate:
			event = BGPEventUpdateMsg

		case packet.BGPMsgTypeNotification:
			fsm.neighborConf.Neighbor.State.Messages.Received.Notification++
			event = BGPEventNotifMsg
			notifyMsg := msg.Body.(*packet.BGPNotification)
			fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
				"Received notification message:", notifyMsg.ErrorCode, notifyMsg.ErrorSubcode, notifyMsg.Data))

		case packet.BGPMsgTypeKeepAlive:
			event = BGPEventKeepAliveMsg
		}
	}
	if event != BGPEventKeepAliveMsg {
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ProcessPacket - event:",
			BGPEventTypeToStr[event]))
	}
	fsm.ProcessEvent(event, data)
}

func (fsm *FSM) ChangeState(newState BaseStateIface) {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ChangeState: Leaving",
		fsm.State, "state Entering", newState, "state"))
	if fsm.cleanup {
		return
	}
	oldState := fsm.State.state()
	fsm.State.leave()
	fsm.State = newState
	fsm.State.enter()
	fsm.Manager.fsmStateChange(fsm.id, fsm.State.state())
	if oldState == BGPFSMEstablished && fsm.State.state() != BGPFSMEstablished {
		fsm.ConnBroken()
	} else if oldState != BGPFSMEstablished && fsm.State.state() == BGPFSMEstablished {
		fsm.ConnEstablished()
	}
}

func (fsm *FSM) sendAutoStartEvent() {
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

func (fsm *FSM) ApplyAutomaticStart() {
	if fsm.restartTime != 0 {
		fsm.StartRestartTimer()
	}
	if fsm.autoStart {
		fsm.sendAutoStartEvent()
	}
}

func (fsm *FSM) StartRestartTimer() {
	fsm.restartTimer.Reset(time.Duration(fsm.restartTime) * time.Second)
}

func (fsm *FSM) StopRestartTimer() {
	fsm.restartTimer.Stop()
}

func (fsm *FSM) SetPassiveTcpEstablishment(flag bool) {
	fsm.passiveTcpEst = flag
}

func (fsm *FSM) StartConnectRetryTimer() {
	fsm.connectRetryTimer.Reset(time.Duration(fsm.connectRetryTime) * time.Second)
}

func (fsm *FSM) StopConnectRetryTimer() {
	fsm.connectRetryTimer.Stop()
}

func (fsm *FSM) SetHoldTime(holdTime uint32, keepaliveTime uint32) {
	if holdTime < 0 || (holdTime > 0 && holdTime < 3) {
		fsm.logger.Err(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"Cannot set hold time. Invalid value", holdTime))
		return
	}

	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "Set hold time to", holdTime,
		"and keepalive time to", keepaliveTime))
	fsm.holdTime = holdTime
	fsm.keepAliveTime = keepaliveTime
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

func (fsm *FSM) HandleAnotherConnection(data interface{}) {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "*** COLLISION DETECTED ***"))
	pConnDir := data.(PeerConnDir)
	fsm.Manager.newConnCh <- PeerFSMConnState{false, fsm.id, pConnDir.connDir, pConnDir.conn}
}

func (fsm *FSM) ProcessOpenMessage(pkt *packet.BGPMessage) bool {
	body := pkt.Body.(*packet.BGPOpen)
	if uint32(body.HoldTime) < fsm.holdTime {
		fsm.SetHoldTime(uint32(body.HoldTime), uint32(body.HoldTime/3))
	}
	if body.MyAS == fsm.Manager.gConf.AS {
		fsm.peerType = config.PeerTypeInternal
	} else {
		fsm.peerType = config.PeerTypeExternal
	}

	afiSafiMap := packet.GetProtocolFromOpenMsg(body)
	for protoFamily, _ := range afiSafiMap {
		if fsm.neighborConf.AfiSafiMap[protoFamily] {
			fsm.afiSafiMap[protoFamily] = true
		}
	}

	return fsm.Manager.receivedBGPOpenMessage(fsm.id, fsm.peerConn.dir, body)
}

func (fsm *FSM) ProcessUpdateMessage(pkt *packet.BGPMessage) {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
		"ProcessUpdateMessage: send message to server"))
	updateMsg := pkt.Body.(*packet.BGPUpdate)
	for _, pa := range updateMsg.PathAttributes {
		if mpReachNLRI, ok := pa.(*packet.BGPPathAttrMPReachNLRI); ok {
			protoFamily := packet.GetProtocolFamily(mpReachNLRI.AFI, mpReachNLRI.SAFI)
			if !fsm.afiSafiMap[protoFamily] {
				fsm.logger.Warning(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
					"Received MP_REACH_NLRI path attr for AFI", mpReachNLRI.AFI, "SAFI", mpReachNLRI.SAFI))
			}
		}
	}
	atomic.AddUint32(&fsm.neighborConf.Neighbor.State.Queues.Input, 1)
	go func() {
		fsm.Manager.bgpPktSrcCh <- packet.NewBGPPktSrc(fsm.Manager.neighborConf.Neighbor.NeighborAddress.String(), pkt)
	}()
}

func (fsm *FSM) sendUpdateMessage(bgpMsg *packet.BGPMessage) {
	updateMsgs := packet.ConstructMaxSizedUpdatePackets(bgpMsg)
	atomic.AddUint32(&fsm.neighborConf.Neighbor.State.Queues.Output, ^uint32(0))

	for idx, _ := range updateMsgs {
		packet, _ := updateMsgs[idx].Encode()
		num, err := (*fsm.peerConn.conn).Write(packet)
		if err != nil {
			fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
				"Conn.Write failed to send Update message with error:", err))
			return
		}
		fsm.neighborConf.Neighbor.State.Messages.Sent.Update++
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"Conn.Write succeeded. sent Update message of", num, "bytes"))
	}
	fsm.StartKeepAliveTimer()
}

func (fsm *FSM) sendOpenMessage() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
		"sendOpenMessage: send address family", fsm.neighborConf.AfiSafiMap))
	optParams := packet.ConstructOptParams(uint32(fsm.pConf.LocalAS), fsm.neighborConf.AfiSafiMap,
		fsm.neighborConf.RunningConf.AddPathsRx, fsm.neighborConf.RunningConf.AddPathsMaxTx)
	bgpOpenMsg := packet.NewBGPOpenMessage(fsm.pConf.LocalAS, uint16(fsm.holdTime), fsm.gConf.RouterId.To4().String(), optParams)
	packet, _ := bgpOpenMsg.Encode()
	num, err := (*fsm.peerConn.conn).Write(packet)
	if err != nil {
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"Conn.Write failed to send Open message with error:", err))
		return
	}
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
		"Conn.Write succeeded. sent Open message of", num, "bytes"))
}

func (fsm *FSM) sendKeepAliveMessage() {
	bgpKeepAliveMsg := packet.NewBGPKeepAliveMessage()
	packet, _ := bgpKeepAliveMsg.Encode()
	_, err := (*fsm.peerConn.conn).Write(packet)
	if err != nil {
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"Conn.Write failed to send KeepAlive message with error:", err))
	}
	fsm.StartKeepAliveTimer()
}

func (fsm *FSM) SendNotificationMessage(code uint8, subCode uint8, data []byte) {
	bgpNotifMsg := packet.NewBGPNotificationMessage(code, subCode, data)
	packet, _ := bgpNotifMsg.Encode()
	num, err := (*fsm.peerConn.conn).Write(packet)
	if err != nil {
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"Conn.Write failed to send Notification message with error:", err))
		return
	}
	fsm.neighborConf.Neighbor.State.Messages.Sent.Notification++
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
		"Conn.Write succeeded. sent Notification message with", num, "bytes"))
}

func (fsm *FSM) SetPeerConn(data interface{}) {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "SetPeerConn called"))
	if fsm.peerConn != nil {
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"SetupPeerConn - Peer conn is already set up"))
		return
	}
	pConnDir := data.(PeerConnDir)
	fsm.peerConn = NewPeerConn(fsm, pConnDir.connDir, pConnDir.conn)
	go fsm.peerConn.StartReading()
}

func (fsm *FSM) ClearPeerConn() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ClearPeerConn called"))
	if fsm.peerConn == nil {
		fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
			"ClearPeerConn - Peer conn is not set up yet"))
		return
	}
	fsm.StopKeepAliveTimer()
	fsm.StopHoldTimer()
	fsm.peerConn.StopReading()
	<-fsm.peerConn.exitCh
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "Conn exited"))
	fsm.peerConn = nil
}

func (fsm *FSM) startRxPkts() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "startRxPkts called"))
	if fsm.peerConn != nil && !fsm.rxPktsFlag {
		fsm.rxPktsFlag = true
		fsm.peerConn.StartReading()
	}
}

func (fsm *FSM) stopRxPkts() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "stopRxPkts called"))
	if fsm.peerConn != nil && fsm.rxPktsFlag {
		fsm.rxPktsFlag = false
		fsm.peerConn.StopReading()
	}
}

func (fsm *FSM) ConnEstablished() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ConnEstablished - start"))
	fsm.Manager.fsmEstablished(fsm.id, fsm.peerConn.conn)
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ConnEstablished - end"))
}

func (fsm *FSM) ConnBroken() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ConnBroken - start"))
	fsm.Manager.fsmBroken(fsm.id, false)
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "ConnBroken - end"))
}

func (fsm *FSM) AcceptPeerConn() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "AcceptPeerConn called"))
	fsm.Manager.AcceptPeerConn()
}

func (fsm *FSM) RejectPeerConn() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "RejectPeerConn called"))
	fsm.Manager.RejectPeerConn()
}

func (fsm *FSM) InitiateConnToPeer() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "InitiateConnToPeer called"))
	if bytes.Equal(fsm.pConf.NeighborAddress, net.IPv4bcast) {
		fsm.logger.Info("Unknown neighbor address")
		return
	}
	addr := net.JoinHostPort(fsm.pConf.NeighborAddress.String(), config.BGPPort)
	if fsm.outTCPConn == nil {
		fsm.outTCPConn = NewOutTCPConn(fsm, fsm.outConnCh, fsm.outConnErrCh)
		go fsm.outTCPConn.ConnectToPeer(fsm.connectRetryTime, addr)
	}
}

func (fsm *FSM) StopConnToPeer() {
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id, "StopConnToPeer called"))
	if fsm.outTCPConn != nil {
		fsm.outTCPConn.StopConnCh <- true
		fsm.outTCPConn = nil
	}
}
