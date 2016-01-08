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
	connDir  config.ConnDir
	pkt *packet.BGPMessage
}

type FSMManager struct {
	Peer         *Peer
	logger       *syslog.Writer
	gConf        *config.GlobalConfig
	pConf        *config.NeighborConfig
	fsms         map[uint8]*FSM
	acceptCh     chan net.Conn
	acceptErrCh  chan error
	closeCh      chan *sync.WaitGroup
	acceptConn   bool
	commandCh    chan int
	activeFSM    uint8
	newConnCh    chan PeerFSMConnState
	fsmStateCh   chan PeerFSMConnState
	fsmOpenMsgCh chan PeerFSMOpenMsg
}

func NewFSMManager(peer *Peer, globalConf *config.GlobalConfig, peerConf *config.NeighborConfig) *FSMManager {
	fsmManager := FSMManager{
		Peer: peer,
		logger: peer.logger,
		gConf: globalConf,
		pConf: peerConf,
	}
	fsmManager.fsms = make(map[uint8]*FSM)
	fsmManager.acceptCh = make(chan net.Conn)
	fsmManager.acceptErrCh = make(chan error)
	fsmManager.acceptConn = false
	fsmManager.closeCh = make(chan *sync.WaitGroup)
	fsmManager.commandCh = make(chan int)
	fsmManager.activeFSM = uint8(config.ConnDirInvalid)
	fsmManager.newConnCh = make(chan PeerFSMConnState)
	fsmManager.fsmStateCh = make(chan PeerFSMConnState)
	fsmManager.fsmOpenMsgCh = make(chan PeerFSMOpenMsg)
	return &fsmManager
}

func (fsmManager *FSMManager) Init() {
	fsmId := uint8(config.ConnDirOut)
	fsm := NewFSM(fsmManager, fsmId, fsmManager.Peer)
	go fsm.StartFSM(NewIdleState(fsm))
	fsmManager.fsms[fsmId] = fsm
	fsm.passiveTcpEstCh <- true

	for {
		select {
		case inConn := <-fsmManager.acceptCh:
			fsmManager.logger.Info(fmt.Sprintf("Neighbor %s: Received a connection OPEN from far end",
				fsmManager.pConf.NeighborAddress))
			if !fsmManager.acceptConn {
				fsmManager.logger.Info(fmt.Sprintln("Can't accept connection from ", fsmManager.pConf.NeighborAddress,
					"yet."))
				inConn.Close()
			} else {
				foundInConn := false
				for _, fsm = range fsmManager.fsms {
					if fsm != nil && fsm.peerConn != nil && fsm.peerConn.dir == config.ConnDirIn {
						fsmManager.logger.Info(fmt.Sprintln("A FSM is already created for a incoming connection"))
						foundInConn = true
						break
					}
				}
				if !foundInConn {
					for _, fsm = range fsmManager.fsms {
						if fsm != nil {
							fsm.inConnCh <- inConn
						}
					}
				}
			}

		case <-fsmManager.acceptErrCh:
			fsmManager.logger.Info(fmt.Sprintf("Neighbor %s: Received a connection CLOSE from far end",
				fsmManager.pConf.NeighborAddress))
			for _, fsm := range fsmManager.fsms {
				if fsm != nil && fsm.peerConn != nil && fsm.peerConn.dir == config.ConnDirIn {
					fsm.eventRxCh <- BGPEventTcpConnFails
				}
			}

		case newConn := <-fsmManager.newConnCh:
			fsmManager.logger.Info(fmt.Sprintf("Neighbor %s: Handle another connection",
				fsmManager.pConf.NeighborAddress))
			newId := fsmManager.getNewId(newConn.id)
			fsmManager.handleAnotherConnection(newId, newConn.connDir, newConn.conn)

		case fsmState := <-fsmManager.fsmStateCh:
			fsmManager.logger.Info(fmt.Sprintf("Neighbor %s: FSM %d state changed",
				fsmManager.pConf.NeighborAddress, fsmState.id))
			if fsmState.isEstablished {
				fsmManager.fsmEstablished(fsmState.id, fsmState.conn)
			} else {
				fsmManager.fsmBroken(fsmState.id)
			}

		case fsmOpenMsg := <- fsmManager.fsmOpenMsgCh:
			fsmManager.logger.Info(fmt.Sprintf("Neighbor %s: FSM %d received OPEN message",
				fsmManager.pConf.NeighborAddress, fsmOpenMsg.id))
			fsmManager.receivedBGPOpenMessage(fsmOpenMsg.id, fsmOpenMsg.connDir, fsmOpenMsg.bgpId)

		case wg := <-fsmManager.closeCh:
			fsmManager.Cleanup(wg)
			return

		case command := <-fsmManager.commandCh:
			event := BGPFSMEvent(command)
			if (event == BGPEventManualStart) || (event == BGPEventManualStop) ||
				(event == BGPEventManualStartPassTcpEst) {
				for _, fsm := range fsmManager.fsms {
					if fsm != nil {
						fsm.eventRxCh <- event
					}
				}
			}
		}
	}
}

func (fsmManager *FSMManager) AcceptPeerConn() {
	fsmManager.acceptConn = true
}

func (fsmManager *FSMManager) RejectPeerConn() {
	fsmManager.acceptConn = false
}

func (fsmManager *FSMManager) fsmEstablished(id uint8, conn *net.Conn) {
	fsmManager.activeFSM = id
	fsmManager.Peer.PeerConnEstablished(conn)
}

func (fsmManager *FSMManager) fsmBroken(id uint8) {
	if fsmManager.activeFSM == id {
		fsmManager.activeFSM = uint8(config.ConnDirInvalid)
	}

	fsmManager.Peer.PeerConnBroken()
}

func (fsmManager *FSMManager) setBGPId(bgpId net.IP) {
	fsmManager.Peer.SetBGPId(bgpId)
}

func (mgr *FSMManager) SendUpdateMsg(bgpMsg *packet.BGPMessage) {
	if mgr.activeFSM == uint8(config.ConnDirInvalid) {
		mgr.logger.Info(fmt.Sprintf("FSMManager: FSM for peer %s is not in ESTABLISHED state", mgr.pConf.NeighborAddress))
		return
	}
	mgr.fsms[mgr.activeFSM].pktTxCh <- bgpMsg
}

func (mgr *FSMManager) Cleanup(wg *sync.WaitGroup) {
	var fsmWG sync.WaitGroup
	for id, fsm := range mgr.fsms {
		if fsm != nil {
			mgr.logger.Info(fmt.Sprintf("FSMManager: Cleanup FSM for peer:%s conn:%d", mgr.pConf.NeighborAddress, id))
			fsmWG.Add(1)
			fsm.closeCh <- &fsmWG
			fsm = nil
			mgr.fsmBroken(id)
			mgr.fsms[id] = nil
			delete(mgr.fsms, id)
		}
	}
	mgr.logger.Info(fmt.Sprintf("FSMManager: waiting for FSM to cleanup %s", mgr.pConf.NeighborAddress.String()))
	fsmWG.Wait()
	mgr.logger.Info(fmt.Sprintf("FSMManager: calling Done() for FSMManager %s", mgr.pConf.NeighborAddress.String()))
	(*wg).Done()
}

func (mgr *FSMManager) getNewId(id uint8) uint8 {
	return uint8((id + 1) % 2)
}

func (mgr *FSMManager) handleAnotherConnection(id uint8, connDir config.ConnDir, conn *net.Conn) {
	if mgr.fsms[id] != nil {
		mgr.logger.Err(fmt.Sprintf("Neighbor %s: A FSM with id %d already exists", mgr.pConf.NeighborAddress, id))
		return
	}

	fsm := NewFSM(mgr, id, mgr.Peer)

	var state BaseStateIface
	state = NewActiveState(fsm)
	connCh := fsm.inConnCh
	if connDir == config.ConnDirOut {
		state = NewConnectState(fsm)
		connCh = fsm.outConnCh
	}
	mgr.fsms[id] = fsm
	go fsm.StartFSM(state)
	connCh <- *conn
	fsm.passiveTcpEstCh <- true
}

func (mgr *FSMManager) getFSMIdByDir(connDir config.ConnDir) uint8 {
	for id, fsm := range mgr.fsms {
		if fsm != nil && fsm.peerConn != nil && fsm.peerConn.dir == connDir {
			return id
		}
	}

	return uint8(config.ConnDirInvalid)
}

func (mgr *FSMManager) receivedBGPOpenMessage(id uint8, connDir config.ConnDir, bgpId net.IP) {
	var wg sync.WaitGroup
	var closeConnDir config.ConnDir = config.ConnDirInvalid

	localBGPId := packet.ConvertIPBytesToUint(mgr.gConf.RouterId.To4())
	bgpIdInt := packet.ConvertIPBytesToUint(bgpId.To4())
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
			if closeFSM, ok := mgr.fsms[closeFSMId]; ok {
				wg.Add(1)
				closeFSM.closeCh <- &wg
				mgr.fsmBroken(closeFSMId)
				mgr.fsms[closeFSMId] = nil
				delete(mgr.fsms, closeFSMId)
			}
		}
	}
	wg.Wait()
	if closeConnDir == config.ConnDirInvalid || closeConnDir != connDir {
		mgr.setBGPId(bgpId)
	}
}
