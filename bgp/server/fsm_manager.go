// peer.go
package server

import (
	"fmt"
	"l3/bgp/packet"
	"net"
)

type CONFIG int

const (
	START CONFIG = iota
	STOP
)

type BgpPkt struct {
	connDir  ConnDir
	pkt *packet.BGPMessage
}

type FSMManager struct {
	Peer         *Peer
	gConf        *GlobalConfig
	pConf        *PeerConfig
	fsms         map[ConnDir]*FSM
	configCh     chan CONFIG
	conns        [ConnDirMax]net.Conn
	connectCh    chan net.Conn
	connectErrCh chan error
	acceptCh     chan net.Conn
	acceptErrCh  chan error
	acceptConn   bool
	commandCh    chan int
	activeFSM    ConnDir
	pktRxCh      chan BgpPkt
}

func NewFSMManager(peer *Peer, globalConf *GlobalConfig, peerConf *PeerConfig) *FSMManager {
	fsmManager := FSMManager{
		Peer: peer,
		gConf: globalConf,
		pConf: peerConf,
	}
	fsmManager.conns = [ConnDirMax]net.Conn{nil, nil}
	fsmManager.connectCh = make(chan net.Conn)
	fsmManager.connectErrCh = make(chan error)
	fsmManager.acceptCh = make(chan net.Conn)
	fsmManager.acceptErrCh = make(chan error)
	fsmManager.acceptConn = false
	fsmManager.commandCh = make(chan int)
	fsmManager.fsms = make(map[ConnDir]*FSM)
	fsmManager.activeFSM = ConnDirOut
	fsmManager.pktRxCh = make(chan BgpPkt)
	return &fsmManager
}

func (fsmManager *FSMManager) Init() {
	fsmManager.fsms[ConnDirOut] = NewFSM(fsmManager, ConnDirOut, fsmManager.gConf, fsmManager.pConf)
	go fsmManager.fsms[ConnDirOut].StartFSM(NewIdleState(fsmManager.fsms[ConnDirOut]))

	for {
		select {
		case inConn := <-fsmManager.acceptCh:
			if !fsmManager.acceptConn {
				fmt.Println("Can't accept connection from ", fsmManager.pConf.IP, "yet.")
				inConn.Close()
			} else if fsmManager.fsms[ConnDirIn] != nil {
				fmt.Println("A FSM is already created for a incoming connection")
			} else {
				fsmManager.conns[ConnDirIn] = inConn
				//fsmManager.fsms[ConnDirOut] = NewFSM(fsmManager, ConnDirIn, fsmManager.gConf, fsmManager.pConf)
				//fsmManager.fsms[ConnDirOut].SetConn(inConn)
				//go fsmManager.fsms[ConnDirIn].StartFSM(NewActiveState(fsmManager.fsms[ConnDirIn]))
				//fsmManager.fsms[ConnDirIn].eventRxCh <- BGPEventTcpConnConfirmed
				//fsmManager.fsms[ConnDirIn].ProcessEvent(BGP_EVENT_TCP_CONN_CONFIRMED)
				fsmManager.fsms[ConnDirOut].inConnCh <- inConn
			}

		case <-fsmManager.acceptErrCh:
			fsmManager.fsms[ConnDirIn].eventRxCh <- BGPEventTcpConnFails
			//fsmManager.fsms[ConnDirIn].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
			fsmManager.conns[ConnDirIn].Close()
			fsmManager.conns[ConnDirIn] = nil

		/*case outConn := <-fsmManager.connectCh:
			fsmManager.conns[ConnDirOut] = outConn
			fsmManager.fsms[ConnDirOut].SetConn(outConn)
			fsmManager.fsms[ConnDirOut].eventRxCh <- BGP_EVENT_TCP_CR_ACKED
			//fsmManager.fsms[ConnDirOut].ProcessEvent(BGP_EVENT_TCP_CR_ACKED)

		case <-fsmManager.connectErrCh:
			fsmManager.fsms[ConnDirOut].eventRxCh <- BGP_EVENT_TCP_CONN_FAILS
			//fsmManager.fsms[ConnDirOut].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
			fsmManager.conns[ConnDirOut].Close()
			fsmManager.conns[ConnDirOut] = nil*/

		case command := <-fsmManager.commandCh:
			event := BGPFSMEvent(command)
			if (event == BGPEventManualStart) || (event == BGPEventManualStop) ||
				(event == BGPEventManualStartPassTcpEst) {
				fsmManager.fsms[fsmManager.activeFSM].eventRxCh <- event
				//fsmManager.fsms[fsmManager.activeFSM].ProcessEvent(event)
			}

		case <-fsmManager.pktRxCh:
			fmt.Println("FSMManager:Init - Rx a BGP packets")
			//fsmManager.fsms[pktRx.id].pktRxCh <- pktRx.pkt
			//fsmManager.fsms[pktRx.id].ProcessPacket(pktRx.pkt, nil)
		}
	}
}

func (fsmManager *FSMManager) AcceptPeerConn() {
	fsmManager.acceptConn = true
}

func (fsmManager *FSMManager) RejectPeerConn() {
	fsmManager.acceptConn = false
}
