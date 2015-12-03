// peer.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
)

type CONFIG int

const (
	START CONFIG = iota
	STOP
)

type BgpPkt struct {
	connDir  config.ConnDir
	pkt *packet.BGPMessage
}

type FSMManager struct {
	Peer         *Peer
	logger       *syslog.Writer
	gConf        *config.GlobalConfig
	pConf        *config.NeighborConfig
	fsms         map[config.ConnDir]*FSM
	configCh     chan CONFIG
	conns        [config.ConnDirMax]net.Conn
	connectCh    chan net.Conn
	connectErrCh chan error
	acceptCh     chan net.Conn
	acceptErrCh  chan error
	acceptConn   bool
	commandCh    chan int
	activeFSM    config.ConnDir
	pktRxCh      chan BgpPkt
}

func NewFSMManager(peer *Peer, globalConf *config.GlobalConfig, peerConf *config.NeighborConfig) *FSMManager {
	fsmManager := FSMManager{
		Peer: peer,
		logger: peer.logger,
		gConf: globalConf,
		pConf: peerConf,
	}
	fsmManager.conns = [config.ConnDirMax]net.Conn{nil, nil}
	fsmManager.connectCh = make(chan net.Conn)
	fsmManager.connectErrCh = make(chan error)
	fsmManager.acceptCh = make(chan net.Conn)
	fsmManager.acceptErrCh = make(chan error)
	fsmManager.acceptConn = false
	fsmManager.commandCh = make(chan int)
	fsmManager.fsms = make(map[config.ConnDir]*FSM)
	fsmManager.activeFSM = config.ConnDirOut
	fsmManager.pktRxCh = make(chan BgpPkt)
	return &fsmManager
}

func (fsmManager *FSMManager) Init() {
	fsmManager.fsms[config.ConnDirOut] = NewFSM(fsmManager, config.ConnDirOut, fsmManager.gConf, fsmManager.pConf)
	go fsmManager.fsms[config.ConnDirOut].StartFSM(NewIdleState(fsmManager.fsms[config.ConnDirOut]))

	for {
		select {
		case inConn := <-fsmManager.acceptCh:
			if !fsmManager.acceptConn {
				fsmManager.logger.Info(fmt.Sprintln("Can't accept connection from ", fsmManager.pConf.NeighborAddress,
					"yet."))
				inConn.Close()
			} else if fsmManager.fsms[config.ConnDirIn] != nil {
				fsmManager.logger.Info(fmt.Sprintln("A FSM is already created for a incoming connection"))
			} else {
				fsmManager.conns[config.ConnDirIn] = inConn
				//fsmManager.fsms[ConnDirOut] = NewFSM(fsmManager, ConnDirIn, fsmManager.gConf, fsmManager.pConf)
				//fsmManager.fsms[ConnDirOut].SetConn(inConn)
				//go fsmManager.fsms[ConnDirIn].StartFSM(NewActiveState(fsmManager.fsms[ConnDirIn]))
				//fsmManager.fsms[ConnDirIn].eventRxCh <- BGPEventTcpConnConfirmed
				//fsmManager.fsms[ConnDirIn].ProcessEvent(BGP_EVENT_TCP_CONN_CONFIRMED)
				fsmManager.fsms[config.ConnDirOut].inConnCh <- inConn
			}

		case <-fsmManager.acceptErrCh:
			fsmManager.fsms[config.ConnDirIn].eventRxCh <- BGPEventTcpConnFails
			//fsmManager.fsms[ConnDirIn].ProcessEvent(BGP_EVENT_TCP_CONN_FAILS)
			fsmManager.conns[config.ConnDirIn].Close()
			fsmManager.conns[config.ConnDirIn] = nil

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
			fsmManager.logger.Info(fmt.Sprintln("FSMManager:Init - Rx a BGP packets"))
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

func (fsmManager *FSMManager) PeerConnEstablished(conn *net.Conn) {
	fsmManager.Peer.PeerConnEstablished(conn)
}

func (fsmManager *FSMManager) PeerConnBroken() {
	fsmManager.Peer.PeerConnBroken()
}

func (fsmManager *FSMManager) SetBGPId(bgpId uint32) {
	fsmManager.Peer.SetBGPId(bgpId)
}

func (mgr *FSMManager) SendUpdateMsg(bgpMsg *packet.BGPMessage) {
	mgr.fsms[config.ConnDirOut].pktTxCh <- bgpMsg
}
