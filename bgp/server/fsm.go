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
    state()
}

type BASE_STATE struct {
    fsm *FSM
    connectRetryCounter int
    connectRetryTimer int
}

func (state *BASE_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("BASE_STATE: processEvent", event)
}

func (state *BASE_STATE) enter() {
    fmt.Println("BASE_STATE: enter")
}

func (state *BASE_STATE) leave() {
    fmt.Println("BASE_STATE: leave")
}

func (state *BASE_STATE) state() BGP_FSM_STATE {
    return iota
}

type IDLE_STATE struct {
    BASE_STATE
}

func NewIdleState(fsm *FSM) *IDLE_STATE {
    state := IDLE_STATE{
        state: fsm,
    }
    return &state
}

func (state *IDLE_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("IDLE_STATE: processEvent", event)
    switch(event) {
        case BGP_EVENT_MANUAL_START, BGP_EVENT_AUTO_START:
            state.fsm.SetState(NewConnectState(state.fsm))

        case BGP_EVENT_MANUAL_START_PASS_TCP_EST, BGP_EVENT_AUTO_START_PASS_TCP_EST:
            state.fsm.SetState(NewActiveState(state.fsm))
    }
}

func (state *IDLE_STATE) enter() {
    fmt.Println("IDLE_STATE: enter")
}

func (state *IDLE_STATE) leave() {
    fmt.Println("IDLE_STATE: leave")
}

func (state *IDLE_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_IDLE
}

type CONNECT_STATE struct {
    BASE_STATE
}

func NewConnectState(fsm *FSM) *CONNECT_STATE {
    state := CONNECT_STATE{
        state: fsm,
    }
    return &state
}

func (state *CONNECT_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("CONNECT_STATE: processEvent", event)
    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP:
            nil

        case BGP_EVENT_DELAY_OPEN_TIMER_EXP:
            state.fsm.SetState(NewOpenSent(state.fsm))

        case BGP_EVENT_TCP_CONN_VALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_INVALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:
            nil

        case BGP_EVENT_TCP_CONN_FAILS:
            state.fsm.SetState(NewActiveState(state.fsm))

        case BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER:
            nil

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            nil

        case BGP_EVENT_AUTO_STOP, BGP_EVENT_HOLD_TIMER_EXP, BGP_EVENT_KEEP_ALIVE_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_OPEN_COLLISION_DAMP, BGP_EVENT_NOTIF_MSG, BGP_EVENT_KEEP_ALIVE_MSG,
             BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 8, 10, 11, 13, 19, 23, 25-28
            state.fsm.SetState(NewIdleState(state.fsm))
    }
}

func (state *CONNECT_STATE) enter() {
    fmt.Println("CONNECT_STATE: enter")
}

func (state *CONNECT_STATE) leave() {
    fmt.Println("CONNECT_STATE: leave")
}

func (state *CONNECT_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_CONNECT
}

type ACTIVE_STATE struct {
    BASE_STATE
}

func NewActiveState(fsm *FSM) *ACTIVE_STATE {
    state := ACTIVE_STATE{
        state: fsm,
    }
    return &state
}

func (state *ACTIVE_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("ACTIVE_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP:
            state.fsm.SetState(NewConnectState(state.fsm))

        case BGP_EVENT_DELAY_OPEN_TIMER_EXP:
            state.fsm.SetState(NewOpenSentState(state.fsm))

        case BGP_EVENT_TCP_CONN_VALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_INVALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:
            nil

        case BGP_EVENT_TCP_CONN_FAILS:
            state.fsm.SetState(NewActiveState(state.fsm))

        case BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER:
            state.fsm.SetState(NewOpenConfirmState(state.fsm))

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_AUTO_STOP, BGP_EVENT_HOLD_TIMER_EXP, BGP_EVENT_KEEP_ALIVE_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_OPEN_COLLISION_DAMP, BGP_EVENT_NOTIF_MSG, BGP_EVENT_KEEP_ALIVE_MSG,
             BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 8, 10, 11, 13, 19, 23, 25-28
            state.fsm.SetState(NewIdleState(state.fsm))
    }
}

func (state *ACTIVE_STATE) enter() {
    fmt.Println("ACTIVE_STATE: enter")
}

func (state *ACTIVE_STATE) leave() {
    fmt.Println("ACTIVE_STATE: leave")
}

func (state *ACTIVE_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_ACTIVE
}

type OPENSENT_STATE struct {
    BASE_STATE
}

func NewOpenSentState(fsm *FSM) *OPENSENT_STATE {
    state := OPENSENT_STATE{
        state: fsm,
    }
    return &state
}

func (state *OPENSENT_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("OPENSENT_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_AUTO_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_HOLD_TIMER_EXP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_TCP_CONN_VALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:
            nil

        case BGP_EVENT_TCP_CONN_FAILS:
            state.fsm.SetState(NewActiveState(state.fsm))

        case BGP_EVENT_BGP_OPEN:
            state.fsm.SetState(NewOpenConfirmState(state.fsm))

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_OPEN_COLLISION_DUMP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP, BGP_EVENT_KEEP_ALIVE_TIMER_EXP, BGP_EVENT_DELAY_OPEN_TIMER_EXP,
             BGP_EVENT_IDLE_HOLD_TIMER_EXP, BGP_EVENT_BGP_OPEN, BGP_EVENT_OPEN_COLLISION_DAMP, BGP_EVENT_NOTIF_MSG,
             BGP_EVENT_KEEP_ALIVE_MSG, BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 9, 11, 12, 13, 20, 25-28
            state.fsm.SetState(NewIdleState(state.fsm))
    }
}

func (state *OPENSENT_STATE) enter() {
    fmt.Println("OPENSENT_STATE: enter")
}

func (state *OPENSENT_STATE) leave() {
    fmt.Println("OPENSENT_STATE: leave")
}

func (state *OPENSENT_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_OPENSENT
}

type OPENCONFIRM_STATE struct {
    BASE_STATE
}

func NewOpenConfirmState(fsm *FSM) *OPENCONFIRM_STATE {
    state := OPENCONFIRM_STATE{
        state: fsm,
    }
    return &state
}

func (state *OPENCONFIRM_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("OPENCONFIRM_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_AUTO_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_HOLD_TIMER_EXP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_KEEP_ALIVE_TIMER_EXP:
            nil

        case BGP_EVENT_TCP_CONN_VALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:
            nil

        case BGP_EVENT_TCP_CONN_FAILS, BGP_EVENT_NOTIF_MSG:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_BGP_OPEN:
            nil

        case BGP_EVENT_HEADER_ERR, BGP_EVENT_OPEN_MSG_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_OPEN_COLLISION_DUMP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_NOTIF_MSG_VER_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_KEEP_ALIVE_MSG:
            state.fsm.SetState(NewEstablishedState(state.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP, BGP_EVENT_DELAY_OPEN_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_UPDATE_MSG, BGP_EVENT_UPDATE_MSG_ERR: // 9, 12, 13, 20, 27, 28
            state.fsm.SetState(NewIdleState(state.fsm))
    }
}

func (state *OPENCONFIRM_STATE) enter() {
    fmt.Println("OPENCONFIRM_STATE: enter")
}

func (state *OPENCONFIRM_STATE) leave() {
    fmt.Println("OPENCONFIRM_STATE: leave")
}

func (state *OPENCONFIRM_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_OPENCONFIRM
}

type ESTABLISHED_STATE struct {
    BASE_STATE
}

func NewEstablishedState(fsm *FSM) *ESTABLISHED_STATE {
    state := ESTABLISHED_STATE{
        state: fsm,
    }
    return &state
}

func (state *ESTABLISHED_STATE) processEvent(event BGP_FSM_EVENT) {
    fmt.Println("ESTABLISHED_STATE: processEvent", event)

    switch(event) {
        case BGP_EVENT_MANUAL_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_AUTO_STOP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_HOLD_TIMER_EXP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_KEEP_ALIVE_TIMER_EXP:
            nil

        case BGP_EVENT_TCP_CONN_VALID: // Supported later
            nil

        case BGP_EVENT_TCP_CR_ACKED, BGP_EVENT_TCP_CONN_CONFIRMED:
            nil

        case BGP_EVENT_TCP_CONN_FAILS, BGP_EVENT_NOTIF_MSG_VER_ERR, BGP_EVENT_NOTIF_MSG:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_OPEN_COLLISION_DUMP:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_KEEP_ALIVE_MSG:
            nil

        case BGP_EVENT_UPDATE_MSG:
            nil

        case BGP_EVENT_UPDATE_MSG_ERR:
            state.fsm.SetState(NewIdleState(state.fsm))

        case BGP_EVENT_CONN_RETRY_TIMER_EXP, BGP_EVENT_DELAY_OPEN_TIMER_EXP, BGP_EVENT_IDLE_HOLD_TIMER_EXP,
             BGP_EVENT_BGP_OPEN, BGP_EVENT_BGP_OPEN_DELAY_OPEN_TIMER, BGP_EVENT_HEADER_ERR: // 9, 12, 13, 20, 21, 22
            state.fsm.SetState(NewIdleState(state.fsm))
    }
}

func (state *ESTABLISHED_STATE) enter() {
    fmt.Println("ESTABLISHED_STATE: enter")
}

func (state *ESTABLISHED_STATE) leave() {
    fmt.Println("ESTABLISHED_STATE: leave")
}

func (state *ESTABLISHED_STATE) state() BGP_FSM_STATE{
    return BGP_FSM_ESTABLISHED
}

type FSM struct {
    Global *GlobalConfig
    Peer *PeerConfig
    State *BASE_STATE
    Event BGP_FSM_EVENT
}

func NewFSM(gConf *GlobalConfig, pConf *PeerConfig) *FSM {
    fsm := FSM{
        Global: gConf,
        Peer: pConf,
    }
    return &fsm
}

func (fsm *FSM) StartFSM(state BGP_FSM_STATE) {
    fsm.State = NewIdleState(fsm)
    fsm.OldState = nil
    fsm.State.enter()
}

func (fsm *FSM) ProcessEvent(event BGP_FSM_EVENT) {
    fmt.Println("FSM: ProcessEvent", event)
    fsm.Event = event
    fsm.State.processEvent(event)
}

func (fsm *FSM) SetState(newState *BASE_STATE) {
    fmt.Println("FSM: SetState: Leaving", fsm.State.state()(), "Entering", newState.state())
    fsm.State.leave()
    fsm.State = newState
    fsm.State.enter()
}
