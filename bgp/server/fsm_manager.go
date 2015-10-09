// peer.go
package server

import (
	"fmt"
    "net"
    "time"
)

type CONFIG int

const (
    START CONFIG = iota
    STOP
)
type FsmManager struct {
    gConf *GlobalConfig
    pConf *PeerConfig
    fsms map[CONN_DIR]*FSM
    configCh chan CONFIG
    conns [CONN_DIR_MAX]net.Conn
    connectCh chan net.Conn
    connectErrCh chan error
    acceptCh chan net.Conn
    acceptErrCh chan error
    acceptConn bool
    commandCh chan int
    activeFsm CONN_DIR
}

func NewFsmManager(globalConf *GlobalConfig, peerConf *PeerConfig) *FsmManager {
    fsmManager := FsmManager{
        gConf: globalConf,
        pConf: peerConf,
    }
    fsmManager.conns = [CONN_DIR_MAX] net.Conn{nil, nil}
    fsmManager.connectCh = make(chan net.Conn)
    fsmManager.connectErrCh = make(chan error)
    fsmManager.acceptCh = make(chan net.Conn)
    fsmManager.acceptErrCh = make(chan error)
    fsmManager.acceptConn = false
    fsmManager.commandCh = make(chan int)
    fsmManager.fsms = make(map[CONN_DIR]*FSM)
    fsmManager.activeFsm = CONN_DIR_OUT
    return &fsmManager
}

func (fsmManager *FsmManager) Init() {
    fsmManager.fsms[CONN_DIR_OUT] = NewFSM(fsmManager, fsmManager.gConf, fsmManager.pConf)
    fsmManager.fsms[CONN_DIR_OUT].StartFSM(NewIdleState(fsmManager.fsms[CONN_DIR_OUT]))

    for {
        select {
            case inConn := <- fsmManager.acceptCh:
                if !fsmManager.acceptConn {
                    fmt.Println("Can't accept connection from ", fsmManager.pConf.IP, "yet.")
                    inConn.Close()
                } else if fsmManager.fsms[CONN_DIR_IN] != nil {
                    fmt.Println("A FSM is already created for a incoming connection")
                } else {
                    fsmManager.conns[CONN_DIR_IN] = inConn
                    fsmManager.fsms[CONN_DIR_IN] = NewFSM(fsmManager, fsmManager.gConf, fsmManager.pConf)
                    fsmManager.fsms[CONN_DIR_IN].SetConn(inConn)
                    fsmManager.fsms[CONN_DIR_IN].StartFSM(NewActiveState(fsmManager.fsms[CONN_DIR_IN]))
                    fsmManager.fsms[CONN_DIR_IN].ProcessEvent(BGP_EVENT_TCP_CONN_CONFIRMED)
                }

            case <- fsmManager.acceptErrCh:
                fsmManager.fsms[CONN_DIR_IN].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
                fsmManager.conns[CONN_DIR_IN].Close()
                fsmManager.conns[CONN_DIR_IN] = nil

            case outConn := <- fsmManager.connectCh:
                fsmManager.conns[CONN_DIR_OUT] = outConn
                fsmManager.fsms[CONN_DIR_OUT].SetConn(outConn)
                fsmManager.fsms[CONN_DIR_OUT].ProcessEvent(BGP_EVENT_TCP_CR_ACKED)

            case <- fsmManager.connectErrCh:
                fsmManager.fsms[CONN_DIR_OUT].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
                fsmManager.conns[CONN_DIR_OUT].Close()
                fsmManager.conns[CONN_DIR_OUT] = nil

            case command := <- fsmManager.commandCh:
                event := BGP_FSM_EVENT(command)
                if (event == BGP_EVENT_MANUAL_START) || (event == BGP_EVENT_MANUAL_STOP) ||
                    (event == BGP_EVENT_MANUAL_START_PASS_TCP_EST) {
                    fsmManager.fsms[fsmManager.activeFsm].ProcessEvent(event)
                }
        }
    }
}

func (fsmManager *FsmManager) ConnectToPeer(seconds int) {
    go fsmManager.Connect(seconds)
}

func (fsmManager *FsmManager) AcceptFromPeer() {
    fsmManager.acceptConn = true
}

func (fsmManager *FsmManager) Connect(seconds int) {
    addr := net.JoinHostPort(fsmManager.pConf.IP.String(), BGP_PORT)

    conn, err := net.DialTimeout("tcp", addr, time.Duration(seconds)*time.Second)
    if err != nil {
        fsmManager.connectErrCh <- err
    } else {
        fsmManager.connectCh <- conn
    }
}
