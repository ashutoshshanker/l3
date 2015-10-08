// peer.go
package server

import (
	"fmt"
    "net"
    "time"
)

type FsmManager struct {
    Global *GlobalConfig
    Peer *PeerConfig
    fsms map[CONN_DIR]*FSM
    conns *[CONN_DIR_MAX] net.Conn
    connectCh chan net.Conn
    connectErrCh chan err
    acceptCh chan net.TCPConn
    acceptErrCh chan err
    acceptConn bool
}

func NewFsmManager(globalConf GlobalConfig, peerConf PeerConfig) *FsmManager {
    fsmManager := FsmManager{
        Global: &globalConf,
        Peer: &peerConf,
    }
    fsmManager.connectCh = make(chan net.Conn)
    fsmManager.connectErrCh = make(chan error)
    fsmManager.acceptCh = make(chan net.TCPConn)
    fsmManager.acceptErrCh = make(chan error)
    fsmManager.acceptConn = false
    fsmManager.fsms[CONN_DIR_OUT] = NewFSM(&globalConf, &peerConf)
    return &fsmManager
}

func (fsmManager *FsmManager) Init() {
    fsmManager.fsms[CONN_DIR_OUT].startFSM(NewIdleState(fsmManager.fsms[CONN_DIR_OUT]))

    conn := make(chan net.Conn)
    err := make(chan err)

    for {
        select {
            case inConn := <- fsmManager.acceptCh:
                if !fsmManager.acceptConn {
                    fmt.Println("Peer", fsmManager.Peer.IP, "can't accept connection yet.")
                    inConn.Close()
                }
                else if fsmManager.fsms[CONN_DIR_IN] != nil {
                    fmt.Println("A FSM is already created for a incoming connection")
                }
                else {
                    fsmManager.inConn = inConn
                    fsmManager.fsms[CONN_DIR_IN] = NewFSM(&globalConf, &peerConf)
                    fsmManager.fsms[CONN_DIR_IN].StartFSM(NewActiveState(fsmManager.fsms[CONN_DIR_IN]))
                    fsmManager.fsms[CONN_DIR_IN].ProcessEvent(BGP_EVENT_TCP_CONN_CONFIRMED)
                }

            case inConnErr := <- fsmManager.acceptErrCh:
                fsmManager.fsms[CONN_DIR_IN].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
                fsmManager.conns[CONN_DIR_IN].Close()
                fsmManager.conns[CONN_DIR_IN] = nil

            case outConn := <- fsmManager.connectCh:
                fsmManager.outConn = outConn
                fsmManager.fsms[CONN_DIR_OUT].ProcessEvent(BGP_EVENT_TCP_CR_ACKED)

            case outConnErr := <- fsmManager.connectErrCh:
                fsmManager.fsms[CONN_DIR_OUT].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
                fsmManager.conns[CONN_DIR_OUT].Close()
                fsmManager.conns[CONN_DIR_OUT] = nil

        }
    }
}

func (fsmManager *FsmManager) ConnectToPeer(seconds int) {
    go peer.Connect(seconds)
}

func (fsmManager *FsmManager) AcceptFromPeer() {
    fsmManager.acceptConn = true
}

func (fsmManager *FsmManager) Connect(seconds int) {
    addr := net.JoinHostPort(peer.Peer.IP.String(), BGP_PORT)

    conn, err := net.DialTimeout("tcp", addr, time.Duration(seconds)*time.Second)
    if err != nil {
        fsmManager.connectErrCh <- err
    }
    else {
        fsmManager.connectCh <- conn
    }
}

func (fsmManager *FsmManager) ReceiveBgpPackets() {
    nil
}
