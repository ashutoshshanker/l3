// peer.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"sync"
)

type CONFIG int

const (
	START CONFIG = iota
	STOP
)

type BgpPkt struct {
	connDir config.ConnDir
	pkt     *packet.BGPMessage
}

type FSMManager struct {
	Peer          *Peer
	logger        *syslog.Writer
	gConf         *config.GlobalConfig
	pConf         *config.NeighborConfig
	fsms          map[uint8]*FSM
	acceptCh      chan net.Conn
	tcpConnFailCh chan uint8
	closeCh       chan bool
	stopFSMCh     chan string
	acceptConn    bool
	commandCh     chan int
	activeFSM     uint8
	newConnCh     chan PeerFSMConnState
	fsmMutex      sync.RWMutex
}

func NewFSMManager(peer *Peer, globalConf *config.GlobalConfig, peerConf *config.NeighborConfig) *FSMManager {
	mgr := FSMManager{
		Peer:   peer,
		logger: peer.logger,
		gConf:  globalConf,
		pConf:  peerConf,
	}
	mgr.fsms = make(map[uint8]*FSM)
	mgr.acceptCh = make(chan net.Conn)
	mgr.tcpConnFailCh = make(chan uint8)
	mgr.acceptConn = false
	mgr.closeCh = make(chan bool)
	mgr.stopFSMCh = make(chan string)
	mgr.commandCh = make(chan int)
	mgr.activeFSM = uint8(config.ConnDirInvalid)
	mgr.newConnCh = make(chan PeerFSMConnState)
	mgr.fsmMutex = sync.RWMutex{}
	return &mgr
}

func (mgr *FSMManager) Init() {
	fsmId := uint8(config.ConnDirOut)
	fsm := NewFSM(mgr, fsmId, mgr.Peer)
	go fsm.StartFSM(NewIdleState(fsm))
	mgr.fsms[fsmId] = fsm
	fsm.passiveTcpEstCh <- true

	for {
		select {
		case inConn := <-mgr.acceptCh:
			mgr.logger.Info(fmt.Sprintf("Neighbor %s: Received a connection OPEN from far end",
				mgr.pConf.NeighborAddress))
			if !mgr.acceptConn {
				mgr.logger.Info(fmt.Sprintln("Can't accept connection from ", mgr.pConf.NeighborAddress,
					"yet."))
				inConn.Close()
			} else {
				foundInConn := false
				for _, fsm = range mgr.fsms {
					if fsm != nil && fsm.peerConn != nil && fsm.peerConn.dir == config.ConnDirIn {
						mgr.logger.Info(fmt.Sprintln("A FSM is already created for a incoming connection"))
						foundInConn = true
						inConn.Close()
						break
					}
				}
				if !foundInConn {
					for _, fsm = range mgr.fsms {
						if fsm != nil {
							fsm.inConnCh <- inConn
						}
					}
				}
			}

		case fsmId := <-mgr.tcpConnFailCh:
			mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s: Received a TCP conn failed from FSM %d",
				mgr.pConf.NeighborAddress, fsmId))
			mgr.fsmTcpConnFailed(fsmId)

		case newConn := <-mgr.newConnCh:
			mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s FSM %d Handle another connection",
				mgr.pConf.NeighborAddress, newConn.id))
			newId := mgr.getNewId(newConn.id)
			mgr.handleAnotherConnection(newId, newConn.connDir, newConn.conn)

		case stopMsg := <-mgr.stopFSMCh:
			mgr.StopFSM(stopMsg)

		case <-mgr.closeCh:
			mgr.Cleanup()
			return

		case command := <-mgr.commandCh:
			event := BGPFSMEvent(command)
			if (event == BGPEventManualStart) || (event == BGPEventManualStop) ||
				(event == BGPEventManualStartPassTcpEst) {
				for _, fsm := range mgr.fsms {
					if fsm != nil {
						fsm.eventRxCh <- event
					}
				}
			}
		}
	}
}

func (mgr *FSMManager) AcceptPeerConn() {
	mgr.acceptConn = true
}

func (mgr *FSMManager) RejectPeerConn() {
	mgr.acceptConn = false
}

func (mgr *FSMManager) fsmTcpConnFailed(id uint8) {
	defer mgr.fsmMutex.Unlock()
	mgr.fsmMutex.Lock()

	mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s FSM %d TCP conn failed", mgr.pConf.NeighborAddress.String(), id))
	if len(mgr.fsms) != 1 && mgr.activeFSM != id {
		mgr.fsmClose(id)
	}
}

func (mgr *FSMManager) fsmClose(id uint8) {
	if closeFSM, ok := mgr.fsms[id]; ok {
		mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s, close FSM %d", mgr.pConf.NeighborAddress.String(), id))
		closeFSM.closeCh <- true
		mgr.fsmBroken(id, false)
		mgr.fsms[id] = nil
		delete(mgr.fsms, id)
		mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s, closed FSM %d", mgr.pConf.NeighborAddress.String(), id))
	} else {
		mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s, FSM %d to close is not found in map %v",
			mgr.pConf.NeighborAddress.String(), id, mgr.fsms))
	}
}

func (mgr *FSMManager) fsmEstablished(id uint8, conn *net.Conn) {
	mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s FSM %d connection established", mgr.pConf.NeighborAddress.String(), id))
	mgr.activeFSM = id
	mgr.Peer.PeerConnEstablished(conn)
}

func (mgr *FSMManager) fsmBroken(id uint8, fsmDelete bool) {
	mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s FSM %d connection broken", mgr.pConf.NeighborAddress.String(), id))
	if mgr.activeFSM == id {
		mgr.activeFSM = uint8(config.ConnDirInvalid)
		mgr.Peer.PeerConnBroken(fsmDelete)
	}
}

func (mgr *FSMManager) fsmStateChange(id uint8, state BGPFSMState) {
	if mgr.activeFSM == id || mgr.activeFSM == uint8(config.ConnDirInvalid) {
		mgr.Peer.FSMStateChange(state)
	}
}

func (mgr *FSMManager) SendUpdateMsg(bgpMsg *packet.BGPMessage) {
	defer mgr.fsmMutex.RUnlock()
	mgr.fsmMutex.RLock()

	if mgr.activeFSM == uint8(config.ConnDirInvalid) {
		mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s FSM is not in ESTABLISHED state", mgr.pConf.NeighborAddress))
		return
	}
	mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s FSM %d - send update", mgr.pConf.NeighborAddress, mgr.activeFSM))
	mgr.fsms[mgr.activeFSM].pktTxCh <- bgpMsg
}

func (mgr *FSMManager) Cleanup() {
	defer mgr.fsmMutex.Unlock()
	mgr.fsmMutex.Lock()

	for id, fsm := range mgr.fsms {
		if fsm != nil {
			mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s FSM %d - cleanup FSM", mgr.pConf.NeighborAddress, id))
			fsm.closeCh <- true
			fsm = nil
			mgr.fsmBroken(id, true)
			mgr.fsms[id] = nil
			delete(mgr.fsms, id)
		}
	}
}

func (mgr *FSMManager) StopFSM(stopMsg string) {
	defer mgr.fsmMutex.Unlock()
	mgr.fsmMutex.Lock()

	for id, fsm := range mgr.fsms {
		if fsm != nil {
			mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s FSM %d - Stop FSM", mgr.pConf.NeighborAddress, id))
			fsm.eventRxCh <- BGPEventTcpConnFails
			mgr.fsmBroken(id, false)
		}
	}
}

func (mgr *FSMManager) getNewId(id uint8) uint8 {
	return uint8((id + 1) % 2)
}

func (mgr *FSMManager) createFSMForNewConnection(id uint8, connDir config.ConnDir) (*FSM, BaseStateIface,
	chan net.Conn) {
	defer mgr.fsmMutex.Unlock()
	mgr.fsmMutex.Lock()

	var state BaseStateIface

	if mgr.fsms[id] != nil {
		mgr.logger.Err(fmt.Sprintf("FSMManager: Neighbor %s - FSM with id %d already exists", mgr.pConf.NeighborAddress, id))
		return nil, state, nil
	}

	mgr.logger.Info(fmt.Sprintf("FSMManager: Neighbor %s Creating new FSM with id %d", mgr.pConf.NeighborAddress, id))
	fsm := NewFSM(mgr, id, mgr.Peer)

	state = NewActiveState(fsm)
	connCh := fsm.inConnCh
	if connDir == config.ConnDirOut {
		state = NewConnectState(fsm)
		connCh = fsm.outConnCh
	}
	mgr.fsms[id] = fsm
	return fsm, state, connCh
}

func (mgr *FSMManager) handleAnotherConnection(id uint8, connDir config.ConnDir, conn *net.Conn) {
	fsm, state, connCh := mgr.createFSMForNewConnection(id, connDir)
	if fsm != nil {
		go fsm.StartFSM(state)
		fsm.passiveTcpEstCh <- true
		connCh <- *conn
	}
}

func (mgr *FSMManager) getFSMIdByDir(connDir config.ConnDir) uint8 {
	for id, fsm := range mgr.fsms {
		if fsm != nil && fsm.peerConn != nil && fsm.peerConn.dir == connDir {
			return id
		}
	}

	return uint8(config.ConnDirInvalid)
}

func (mgr *FSMManager) receivedBGPOpenMessage(id uint8, connDir config.ConnDir, openMsg *packet.BGPOpen) bool {
	var closeConnDir config.ConnDir = config.ConnDirInvalid

	defer mgr.fsmMutex.Unlock()
	mgr.fsmMutex.Lock()

	localBGPId := packet.ConvertIPBytesToUint(mgr.gConf.RouterId.To4())
	bgpIdInt := packet.ConvertIPBytesToUint(openMsg.BGPId.To4())
	for fsmId, fsm := range mgr.fsms {
		if fsmId != id && fsm != nil && fsm.State.state() >= BGPFSMOpensent {
			if fsm.State.state() == BGPFSMEstablished {
				closeConnDir = connDir
			} else if localBGPId > bgpIdInt {
				closeConnDir = config.ConnDirIn
			} else {
				closeConnDir = config.ConnDirOut
			}
			closeFSMId := mgr.getFSMIdByDir(closeConnDir)
			mgr.fsmClose(closeFSMId)
		}
	}
	if closeConnDir == config.ConnDirInvalid || closeConnDir != connDir {
		asSize := packet.GetASSize(openMsg)
		mgr.Peer.SetPeerAttrs(openMsg.BGPId, asSize, mgr.fsms[id].holdTime, mgr.fsms[id].keepAliveTime)
	}

	if closeConnDir == connDir {
		mgr.logger.Info(fmt.Sprintf("FSMManager: Peer %s, FSM %d Closing FSM... return false",
			mgr.pConf.NeighborAddress.String(), id))
		return false
	} else {
		return true
	}
}
