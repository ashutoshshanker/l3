// fsm.go
package server

import (
    "fmt"
    "net"
)

type BGP_FSM_STATE int

const (
    BGP_FSM_NONE BGP_FSM_STATE = iota
    BGP_FSM_IDLE
    BGP_FSM_CONNECT
    BGP_FSM_ACTIVE
    BGP_FSM_OPENSENT
    BGP_FSM_OPENCONFIRM
    BGP_FSM_ESTABLISHED
)

type BGP_FSM_EVENT int

const (
    _ BGP_FSM_EVENT = iota
    BGP_EVENT_MANUAL_START
    BGP_EVENT_MANUAL_STOP
    BGP_EVENT_AUTO_START
    BGP_EVENT_MANUAL_START_PASS_TCP_EST
    BGP_EVENT_AUTO_START_PASS_TCP_EST
    BGP_EVENT_AUTO_DAMP_PEER_OSCL
    BGP_EVENT_AUTO_START_DAMP_PEER_OSCL_PASS_TCP_EST
    BGP_EVENT_AUTO_STOP
    BGP_EVENT_CONN_RETRY_TIMER_EXP
    BGP_EVENT_HOLD_TIMER_EXP
    BGP_EVENT_KEEP_ALIVE_TIMER_EXP
    BGP_EVENT_DELAY_OPEN_TIMER_EXP
    BGP_EVENT_IDLE_HOLD_TIMER_EXP
    BGP_EVENT_TCP_CONN_VALID
    BGP_EVENT_TCP_CR_INVALID
    BGP_EVENT_TCP_CR_ACKED
    BGP_EVENT_TCP_CONN_CONFIRMED
    BGP_EVENT_TCP_CONN_FAILS
    BGP_EVENT_BGP_OPEN
    BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER
    BGP_EVENT_HEADER_ERR
    BGP_EVENT_OPEN_MSG_ERR
    BGP_EVENT_OPEN_COLLISION_DUMP
    BGP_EVENT_NOTIF_MSG_VER_ERR
    BGP_EVENT_NOTIF_MSG
    BGP_EVENT_KEEP_ALIVE_MSG
    BGP_EVENT_UPDATE_MSG
    BGP_EVENT_UPDATE_MSG_ERR
)

type BASE_STATE_IFACE interface {
    processEvent(BGP_FSM_EVENT)
    enter()
    leave()
    state() BGP_FSM_STATE
}

type BASE_STATE struct {
    fsm *FSM
    connectRetryCounter int
    connectRetryTimer int
}

func (self *BASE_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("BASE_STATE: processEvent", event)
}

func (self *BASE_STATE) enter() {
    fmt.Println("BASE_STATE: enter")
}

func (self *BASE_STATE) leave() {
    fmt.Println("BASE_STATE: leave")
}

func (self *BASE_STATE) state() BGP_FSM_STATE {
    return BGP_FSM_NONE
}

type IDLE_STATE struct {
    BASE_STATE
}

func NewIdleState(fsm *FSM) *IDLE_STATE {
    state := IDLE_STATE{
        BASE_STATE{
            fsm: fsm,
        },
    }
    return &state
}

func (self *IDLE_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("IDLE_STATE: processEvent", event)
    switch(event) {
        case BGP_EVENT_MANUAL_START, BGP_EVENT_AUTO_START:
            self.BASE_STATE.fsm.SetConnectRetryCounter(0)
            self.BASE_STATE.fsm.Manager.ConnectToPeer(3)
            self.BASE_STATE.fsm.ChangeState(NewConnectState(self.fsm))

        case BGP_EVENT_MANUAL_START_PASS_TCP_EST, BGP_EVENT_AUTO_START_PASS_TCP_EST:
            self.BASE_STATE.fsm.ChangeState(NewActiveState(self.fsm))
    }
}

func (self *IDLE_STATE) enter() {
    fmt.Println("IDLE_STATE: enter")
}

func (self *IDLE_STATE) leave() {
    fmt.Println("IDLE_STATE: leave")
}

func (self *IDLE_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_IDLE
}

type CONNECT_STATE struct {
    BASE_STATE
}

func NewConnectState(fsm *FSM) *CONNECT_STATE {
    state := CONNECT_STATE{
        BASE_STATE {
            fsm: fsm,
        },
    }
    return &state
}

func (self *CONNECT_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("CONNECT_STATE: processEvent", event)
    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP:

        case BGP_EVENT_DELAY_OPEN_TIMER_EXP:
            self.fsm.ChangeState(NewOpenSentState(self.fsm))

        case BGP_EVENT_TCP_CONN_VALID: // Supported later

        case BGP_EVENT_TCP_CR_INVALID: // Supported later

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:
            self.BASE_STATE.fsm.sendOpenMessage()
            self.BASE_STATE.fsm.ChangeState(NewOpenSentState(self.BASE_STATE.fsm))

        case BGP_EVENT_TCP_CONN_FAILS:
            self.fsm.ChangeState(NewActiveState(self.fsm))

        case BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER:

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:

        case BGP_EVENT_AUTO_STOP, BGP_EVENT_HOLD_TIMER_EXP, BGP_EVENT_KEEP_ALIVE_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_OPEN_COLLISION_DUMP, BGP_EVENT_NOTIF_MSG, BGP_EVENT_KEEP_ALIVE_MSG,
             BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 8, 10, 11, 13, 19, 23, 25-28
            self.fsm.ChangeState(NewIdleState(self.fsm))
    }
}

func (self *CONNECT_STATE) enter() {
    fmt.Println("CONNECT_STATE: enter")
}

func (self *CONNECT_STATE) leave() {
    fmt.Println("CONNECT_STATE: leave")
}

func (self *CONNECT_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_CONNECT
}

type ACTIVE_STATE struct {
    BASE_STATE
}

func NewActiveState(fsm *FSM) *ACTIVE_STATE {
    state := ACTIVE_STATE{
        BASE_STATE{
            fsm: fsm,
        },
    }
    return &state
}

func (self *ACTIVE_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("ACTIVE_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP:
            self.fsm.ChangeState(NewConnectState(self.fsm))

        case BGP_EVENT_DELAY_OPEN_TIMER_EXP:
            self.fsm.ChangeState(NewOpenSentState(self.fsm))

        case BGP_EVENT_TCP_CONN_VALID: // Supported later

        case BGP_EVENT_TCP_CR_INVALID: // Supported later

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:

        case BGP_EVENT_TCP_CONN_FAILS:
            self.fsm.ChangeState(NewActiveState(self.fsm))

        case BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER:
            self.fsm.ChangeState(NewOpenConfirmState(self.fsm))

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_AUTO_STOP, BGP_EVENT_HOLD_TIMER_EXP, BGP_EVENT_KEEP_ALIVE_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_OPEN_COLLISION_DUMP, BGP_EVENT_NOTIF_MSG, BGP_EVENT_KEEP_ALIVE_MSG,
             BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 8, 10, 11, 13, 19, 23, 25-28
            self.fsm.ChangeState(NewIdleState(self.fsm))
    }
}

func (self *ACTIVE_STATE) enter() {
    fmt.Println("ACTIVE_STATE: enter")
}

func (self *ACTIVE_STATE) leave() {
    fmt.Println("ACTIVE_STATE: leave")
}

func (self *ACTIVE_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_ACTIVE
}

type OPENSENT_STATE struct {
    BASE_STATE
}

func NewOpenSentState(fsm *FSM) *OPENSENT_STATE {
    state := OPENSENT_STATE{
        BASE_STATE{
            fsm: fsm,
        },
    }
    return &state
}

func (self *OPENSENT_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("OPENSENT_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_AUTO_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_HOLD_TIMER_EXP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_TCP_CONN_VALID: // Supported later

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:

        case BGP_EVENT_TCP_CONN_FAILS:
            self.fsm.ChangeState(NewActiveState(self.fsm))

        case BGP_EVENT_BGP_OPEN:
            self.fsm.ChangeState(NewOpenConfirmState(self.fsm))

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_OPEN_COLLISION_DUMP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP, BGP_EVENT_KEEP_ALIVE_TIMER_EXP, BGP_EVENT_DELAY_OPEN_TIMER_EXP,
             BGP_EVENT_IDLE_HOLD_TIMER_EXP, BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER, BGP_EVENT_NOTIF_MSG,
             BGP_EVENT_KEEP_ALIVE_MSG, BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 9, 11, 12, 13, 20, 25-28
            self.fsm.ChangeState(NewIdleState(self.fsm))
    }
}

func (self *OPENSENT_STATE) enter() {
    fmt.Println("OPENSENT_STATE: enter")
}

func (self *OPENSENT_STATE) leave() {
    fmt.Println("OPENSENT_STATE: leave")
}

func (self *OPENSENT_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_OPENSENT
}

type OPENCONFIRM_STATE struct {
    BASE_STATE
}

func NewOpenConfirmState(fsm *FSM) *OPENCONFIRM_STATE {
    state := OPENCONFIRM_STATE{
        BASE_STATE{
            fsm: fsm,
        },
    }
    return &state
}

func (self *OPENCONFIRM_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("OPENCONFIRM_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_AUTO_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_HOLD_TIMER_EXP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_KEEP_ALIVE_TIMER_EXP:

        case BGP_EVENT_TCP_CONN_VALID: // Supported later

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:

        case BGP_EVENT_TCP_CONN_FAILS, BGP_EVENT_NOTIF_MSG:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_BGP_OPEN:

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_OPEN_COLLISION_DUMP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_KEEP_ALIVE_MSG:
            self.fsm.ChangeState(NewEstablishedState(self.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP, BGP_EVENT_DELAY_OPEN_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER, BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 9, 12, 13, 20, 27, 28
            self.fsm.ChangeState(NewIdleState(self.fsm))
    }
}

func (self *OPENCONFIRM_STATE) enter() {
    fmt.Println("OPENCONFIRM_STATE: enter")
}

func (self *OPENCONFIRM_STATE) leave() {
    fmt.Println("OPENCONFIRM_STATE: leave")
}

func (self *OPENCONFIRM_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_OPENCONFIRM
}

type ESTABLISHED_STATE struct {
    BASE_STATE
}

func NewEstablishedState(fsm *FSM) *ESTABLISHED_STATE {
    state := ESTABLISHED_STATE{
        BASE_STATE{
            fsm: fsm,
        },
    }
    return &state
}

func (self *ESTABLISHED_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("ESTABLISHED_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_AUTO_STOP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_HOLD_TIMER_EXP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_KEEP_ALIVE_TIMER_EXP:

        case BGP_EVENT_TCP_CONN_VALID: // Supported later

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:

        case BGP_EVENT_TCP_CONN_FAILS, BGP_EVENT_NOTIF_MSG_VER_ERR, BGP_EVENT_NOTIF_MSG:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_OPEN_COLLISION_DUMP:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_KEEP_ALIVE_MSG:

        case BGP_EVENT_UPDATE_MSG:

        case BGP_EVENT_UPDATE_MSG_ERR:
            self.fsm.ChangeState(NewIdleState(self.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP, BGP_EVENT_DELAY_OPEN_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER, BGP_EVENT_HEADER_ERR: // 9, 12, 13, 20, 21, 22
            self.fsm.ChangeState(NewIdleState(self.fsm))
    }
}

func (self *ESTABLISHED_STATE) enter() {
    fmt.Println("ESTABLISHED_STATE: enter")
}

func (self *ESTABLISHED_STATE) leave() {
    fmt.Println("ESTABLISHED_STATE: leave")
}

func (self *ESTABLISHED_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_ESTABLISHED
}

type FSM_IFACE interface {
    StartFSM(state BASE_STATE_IFACE)
    ProcessEvent(event BGP_FSM_EVENT)
    ChangeState(state BASE_STATE_IFACE)
}

type FSM struct {
    gConf *GlobalConfig
    pConf *PeerConfig
    Manager *FsmManager
    State BASE_STATE_IFACE

    conn net.Conn
    event BGP_FSM_EVENT
    connectRetryCounter int
    holdTime uint16
}

func NewFSM(fsmManager *FsmManager, gConf *GlobalConfig, pConf *PeerConfig) *FSM {
    fsm := FSM{
        gConf: gConf,
        pConf: pConf,
        Manager: fsmManager,
        holdTime: 240, // seconds
    }
    return &fsm
}

func (fsm *FSM) SetConn(conn net.Conn) {
    fsm.conn = conn
}

func (fsm *FSM) StartFSM(state BASE_STATE_IFACE) {
    fmt.Println("FSM: Starting the stach machine in", state.state(), "state")
    fsm.State = state
    fsm.State.enter()
}

func (fsm *FSM) ProcessEvent(event BGP_FSM_EVENT) {
    fmt.Println("FSM: ProcessEvent", event)
    fsm.event = event
    fsm.State.processEvent(event)
}

func (fsm *FSM) ChangeState(newState BASE_STATE_IFACE) {
    fmt.Println("FSM: ChangeState: Leaving", fsm.State.state(), "Entering", newState.state())
    fsm.State.leave()
    fsm.State = newState
    fsm.State.enter()
}

func (fsm *FSM) SetConnectRetryCounter(value int) {
    fsm.connectRetryCounter = value
}

func (fsm *FSM) sendOpenMessage() {
    bgpOpenMsg := NewBGPOpenMessage(fsm.pConf.AS, fsm.holdTime, IP)
    packet, _ := bgpOpenMsg.Serialize()
    num, err := fsm.conn.Write(packet)
    if err != nil {
        fmt.Println("Conn.Write failed with error:", err)
    }
    fmt.Println("Conn.Write succeeded. sent %d", num, "bytes of OPEN message")
}
