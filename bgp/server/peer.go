// peer.go
package server

import (
	"fmt"
	"l3/bgp/baseobjects"
	"l3/bgp/config"
	"l3/bgp/fsm"
	"l3/bgp/packet"
	bgprib "l3/bgp/rib"
	"net"
	"sync/atomic"
	"utils/logging"
)

type Peer struct {
	Server       *BGPServer
	logger       *logging.Writer
	NeighborConf *base.NeighborConf
	fsmManager   *fsm.FSMManager
	ifIdx        int32
	ribOut       map[string]map[uint32]*bgprib.Path
}

func NewPeer(server *BGPServer, globalConf *config.GlobalConfig, peerGroup *config.PeerGroupConfig, peerConf config.NeighborConfig) *Peer {
	peer := Peer{
		Server: server,
		logger: server.logger,
		ifIdx:  -1,
		ribOut: make(map[string]map[uint32]*bgprib.Path),
	}

	peer.NeighborConf = base.NewNeighborConf(peer.logger, globalConf, peerGroup, peerConf)
	peer.fsmManager = fsm.NewFSMManager(peer.logger, peer.NeighborConf, server.BGPPktSrcCh, server.PeerFSMConnCh,
		server.ReachabilityCh)
	return &peer
}

func (p *Peer) UpdatePeerGroup(peerGroup *config.PeerGroupConfig) {
	p.NeighborConf.UpdatePeerGroup(peerGroup)
}

func (p *Peer) UpdateNeighborConf(nConf config.NeighborConfig, bgp *config.Bgp) {
	p.NeighborConf.UpdateNeighborConf(nConf, bgp)
}

func (p *Peer) Init() {
	if p.NeighborConf.Neighbor.State.BfdNeighborState == "down" {
		p.logger.Info(fmt.Sprintf("Neighbor's bfd state is down for %s\n", p.NeighborConf.Neighbor.NeighborAddress))
		return
	}
	if p.fsmManager == nil {
		p.logger.Info(fmt.Sprintf("Instantiating new FSM Manager for neighbor %s\n",
			p.NeighborConf.Neighbor.NeighborAddress))
		p.fsmManager = fsm.NewFSMManager(p.logger, p.NeighborConf, p.Server.BGPPktSrcCh, p.Server.PeerFSMConnCh,
			p.Server.ReachabilityCh)
	}

	go p.fsmManager.Init()
}

func (p *Peer) Cleanup() {
	p.fsmManager.CloseCh <- true
	p.fsmManager = nil
}

func (p *Peer) StopFSM(msg string) {
	p.fsmManager.StopFSMCh <- msg
}

func (p *Peer) MaxPrefixesExceeded() {
	if p.NeighborConf.RunningConf.MaxPrefixesDisconnect {
		p.Command(int(fsm.BGPEventAutoStop), fsm.BGPCmdReasonMaxPrefixExceeded)
	}
}
func (p *Peer) setIfIdx(ifIdx int32) {
	p.ifIdx = ifIdx
}

func (p *Peer) getIfIdx() int32 {
	return p.ifIdx
}

func (p *Peer) AcceptConn(conn *net.TCPConn) {
	if p.NeighborConf.Neighbor.State.BfdNeighborState == "down" {
		p.logger.Info(fmt.Sprintf("Neighbor's bfd state is down for %s\n", p.NeighborConf.Neighbor.NeighborAddress))
		(*conn).Close()
		return
	}
	if p.fsmManager == nil {
		p.logger.Info(fmt.Sprintf("FSM Manager is not instantiated yet for neighbor %s\n",
			p.NeighborConf.Neighbor.NeighborAddress))
		(*conn).Close()
		return
	}
	p.fsmManager.AcceptCh <- conn
}

func (p *Peer) Command(command int, reason int) {
	if p.fsmManager == nil {
		p.logger.Info(fmt.Sprintf("FSM Manager is not instantiated yet for neighbor %s\n",
			p.NeighborConf.Neighbor.NeighborAddress))
		return
	}
	p.fsmManager.CommandCh <- fsm.PeerFSMCommand{command, reason}
}

func (p *Peer) getAddPathsMaxTx() int {
	return int(p.NeighborConf.Neighbor.State.AddPathsMaxTx)
}

func (p *Peer) updatePathAttrs(bgpMsg *packet.BGPMessage, path *bgprib.Path) bool {
	if p.NeighborConf.Neighbor.Transport.Config.LocalAddress == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Can't send Update message, FSM is not in Established state\n",
			p.NeighborConf.Neighbor.NeighborAddress))
		return false
	}

	if bgpMsg == nil || bgpMsg.Body.(*packet.BGPUpdate).PathAttributes == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Path attrs not found in BGP Update message\n",
			p.NeighborConf.Neighbor.NeighborAddress))
		return false
	}

	if len(bgpMsg.Body.(*packet.BGPUpdate).NLRI) == 0 {
		return true
	}

	if p.NeighborConf.ASSize == 2 {
		packet.Convert4ByteTo2ByteASPath(bgpMsg)
	}

	if p.NeighborConf.IsInternal() {
		if path.NeighborConf != nil && (path.NeighborConf.IsRouteReflectorClient() ||
			p.NeighborConf.IsRouteReflectorClient()) {
			packet.AddOriginatorId(bgpMsg, path.NeighborConf.BGPId)
			packet.AddClusterId(bgpMsg, path.NeighborConf.RunningConf.RouteReflectorClusterId)
		} else {
			packet.SetNextHop(bgpMsg, p.NeighborConf.Neighbor.Transport.Config.LocalAddress)
			packet.SetLocalPref(bgpMsg, path.GetPreference())
		}
	} else {
		// Do change these path attrs for local routes
		if path.NeighborConf != nil {
			packet.RemoveMultiExitDisc(bgpMsg)
		}
		packet.PrependAS(bgpMsg, p.NeighborConf.RunningConf.LocalAS, p.NeighborConf.ASSize)
		packet.SetNextHop(bgpMsg, p.NeighborConf.Neighbor.Transport.Config.LocalAddress)
		packet.RemoveLocalPref(bgpMsg)
	}

	return true
}

func (p *Peer) clearRibOut() {
	for ip, pathIdMap := range p.ribOut {
		for pathId, _ := range pathIdMap {
			delete(p.ribOut[ip], pathId)
		}
		delete(p.ribOut, ip)
	}
}

func (p *Peer) PeerConnEstablished(conn *net.Conn) {
	host, _, err := net.SplitHostPort((*conn).LocalAddr().String())
	if err != nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Can't find local address from the peer connection: %s",
			p.NeighborConf.Neighbor.NeighborAddress, (*conn).LocalAddr()))
		return
	}
	p.NeighborConf.Neighbor.Transport.Config.LocalAddress = net.ParseIP(host)
	p.clearRibOut()
	//p.Server.PeerConnEstCh <- p.Neighbor.NeighborAddress.String()
}

func (p *Peer) PeerConnBroken(fsmCleanup bool) {
	if p.NeighborConf.Neighbor.Transport.Config.LocalAddress != nil {
		p.NeighborConf.Neighbor.Transport.Config.LocalAddress = nil
		//p.Server.PeerConnBrokenCh <- p.Neighbor.NeighborAddress.String()
	}

	p.NeighborConf.Neighbor.State.ConnectRetryTime = p.NeighborConf.RunningConf.ConnectRetryTime
	p.NeighborConf.Neighbor.State.HoldTime = p.NeighborConf.RunningConf.HoldTime
	p.NeighborConf.Neighbor.State.KeepaliveTime = p.NeighborConf.RunningConf.KeepaliveTime
	p.NeighborConf.Neighbor.State.AddPathsRx = false
	p.NeighborConf.Neighbor.State.AddPathsMaxTx = 0
	p.NeighborConf.Neighbor.State.TotalPrefixes = 0
	p.clearRibOut()
}

func (p *Peer) sendUpdateMsg(msg *packet.BGPMessage, path *bgprib.Path) {
	if path != nil && path.NeighborConf != nil {
		if path.NeighborConf.IsInternal() {

			if p.NeighborConf.IsInternal() && !path.NeighborConf.IsRouteReflectorClient() &&
				!p.NeighborConf.IsRouteReflectorClient() {
				return
			}
		}

		// Don't send the update to the peer that sent the update.
		if p.NeighborConf.RunningConf.NeighborAddress.String() == path.NeighborConf.RunningConf.NeighborAddress.String() {
			return
		}
	}

	if p.updatePathAttrs(msg, path) {
		atomic.AddUint32(&p.NeighborConf.Neighbor.State.Queues.Output, 1)
		p.fsmManager.SendUpdateMsg(msg)
	}

}

func (p *Peer) isAdvertisable(path *bgprib.Path) bool {
	if path != nil && path.NeighborConf != nil {
		if path.NeighborConf.IsInternal() {

			if p.NeighborConf.IsInternal() && !path.NeighborConf.IsRouteReflectorClient() &&
				!p.NeighborConf.IsRouteReflectorClient() {
				return false
			}
		}

		// Don't send the update to the peer that sent the update.
		if p.NeighborConf.RunningConf.NeighborAddress.String() == path.NeighborConf.RunningConf.NeighborAddress.String() {
			return false
		}
	}

	return true
}

func (p *Peer) calculateAddPathsAdvertisements(dest *bgprib.Destination, path *bgprib.Path, newUpdated map[*bgprib.Path][]packet.NLRI,
	withdrawList []packet.NLRI, addPathsTx int) (map[*bgprib.Path][]packet.NLRI, []packet.NLRI) {
	pathIdMap := make(map[uint32]*bgprib.Path)
	ip := dest.IPPrefix.Prefix.String()

	if _, ok := p.ribOut[ip]; !ok {
		p.logger.Info(fmt.Sprintf("Neighbor %s: calculateAddPathsAdvertisements - processing updates, dest %s not"+
			"found in rib out", p.NeighborConf.Neighbor.NeighborAddress, ip))
		p.ribOut[ip] = make(map[uint32]*bgprib.Path)
	}

	if p.isAdvertisable(path) {
		route := dest.LocRibPathRoute
		if path != nil { // Loc-RIB path changed
			if _, ok := newUpdated[path]; !ok {
				newUpdated[path] = make([]packet.NLRI, 0)
			}
			nlri := packet.NewExtNLRI(route.OutPathId, *dest.IPPrefix)
			newUpdated[path] = append(newUpdated[path], nlri)
		} else {
			path = dest.LocRibPath
		}
		pathIdMap[route.OutPathId] = path
	}

	for i := 0; i < len(dest.AddPaths) && len(pathIdMap) < (addPathsTx-1); i++ {
		route := dest.GetPathRoute(dest.AddPaths[i])
		if route != nil && p.isAdvertisable(dest.AddPaths[i]) {
			pathIdMap[route.OutPathId] = dest.AddPaths[i]
		}
	}

	ribPathMap, _ := p.ribOut[ip]
	for ribPathId, ribPath := range ribPathMap {
		if path, ok := pathIdMap[ribPathId]; !ok {
			nlri := packet.NewExtNLRI(ribPathId, *dest.IPPrefix)
			withdrawList = append(withdrawList, nlri)
			delete(p.ribOut[ip], ribPathId)
		} else if ribPath == path {
			delete(pathIdMap, ribPathId)
		} else if ribPath != path {
			if _, ok := newUpdated[path]; !ok {
				newUpdated[path] = make([]packet.NLRI, 0)
			}
			nlri := packet.NewExtNLRI(ribPathId, *dest.IPPrefix)
			newUpdated[path] = append(newUpdated[path], nlri)
			p.ribOut[ip][ribPathId] = path
			delete(pathIdMap, ribPathId)
		}
	}

	for pathId, path := range pathIdMap {
		if _, ok := newUpdated[path]; !ok {
			newUpdated[path] = make([]packet.NLRI, 0)
		}
		nlri := packet.NewExtNLRI(pathId, *dest.IPPrefix)
		newUpdated[path] = append(newUpdated[path], nlri)
		p.ribOut[ip][pathId] = path
		delete(pathIdMap, pathId)
	}

	return newUpdated, withdrawList
}

func (p *Peer) SendUpdate(updated map[*bgprib.Path][]*bgprib.Destination, withdrawn []*bgprib.Destination, withdrawPath *bgprib.Path,
	updatedAddPaths []*bgprib.Destination) {
	p.logger.Info(fmt.Sprintf("Neighbor %s: Send update message valid routes:%v, withdraw routes:%v",
		p.NeighborConf.Neighbor.NeighborAddress, updated, withdrawn))
	if p.NeighborConf.Neighbor.Transport.Config.LocalAddress == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Can't send Update message, FSM is not in Established state",
			p.NeighborConf.Neighbor.NeighborAddress))
		return
	}

	addPathsTx := p.getAddPathsMaxTx()
	withdrawList := make([]packet.NLRI, 0)
	newUpdated := make(map[*bgprib.Path][]packet.NLRI)
	if len(withdrawn) > 0 {
		for _, dest := range withdrawn {
			if dest != nil {
				ip := dest.IPPrefix.Prefix.String()
				if addPathsTx > 0 {
					pathIdMap, ok := p.ribOut[ip]
					if !ok {
						p.logger.Err(fmt.Sprintf("Neighbor %s: SendUpdate - processing withdraws, dest %s not found in rib out",
							p.NeighborConf.Neighbor.NeighborAddress, ip))
						continue
					}
					for pathId, _ := range pathIdMap {
						nlri := packet.NewExtNLRI(pathId, *dest.IPPrefix)
						withdrawList = append(withdrawList, nlri)
					}
					delete(p.ribOut, ip)
				} else {
					withdrawList = append(withdrawList, dest.IPPrefix)
					delete(p.ribOut, ip)
				}
			}
		}
	}

	for path, destinations := range updated {
		for _, dest := range destinations {
			if dest != nil {
				ip := dest.IPPrefix.Prefix.String()
				if addPathsTx > 0 {
					newUpdated, withdrawList = p.calculateAddPathsAdvertisements(dest, path, newUpdated, withdrawList,
						addPathsTx)
				} else {
					if !p.isAdvertisable(path) {
						withdrawList = append(withdrawList, dest.IPPrefix)
						delete(p.ribOut, ip)
					} else {
						route := dest.LocRibPathRoute
						pathId := route.OutPathId
						if _, ok := p.ribOut[ip]; !ok {
							p.ribOut[ip] = make(map[uint32]*bgprib.Path)
						}
						for ribPathId, _ := range p.ribOut[ip] {
							if pathId != ribPathId {
								delete(p.ribOut[ip], ribPathId)
							}
						}
						if ribPath, ok := p.ribOut[ip][pathId]; !ok || ribPath != path {
							if _, ok := newUpdated[path]; !ok {
								newUpdated[path] = make([]packet.NLRI, 0)
							}
							newUpdated[path] = append(newUpdated[path], dest.IPPrefix)
						}
						p.ribOut[ip][pathId] = path
					}
				}
			}
		}
	}

	if addPathsTx > 0 {
		for _, dest := range updatedAddPaths {
			newUpdated, withdrawList = p.calculateAddPathsAdvertisements(dest, nil, newUpdated, withdrawList, addPathsTx)
		}
	}

	if len(withdrawList) > 0 {
		p.logger.Info(fmt.Sprintf("Neighbor %s: Send update message withdraw routes:%+v",
			p.NeighborConf.Neighbor.NeighborAddress, withdrawList))
		updateMsg := packet.NewBGPUpdateMessage(withdrawList, nil, nil)
		p.sendUpdateMsg(updateMsg.Clone(), withdrawPath)
		withdrawList = withdrawList[:0]
	}

	for path, nlriList := range newUpdated {
		p.logger.Info(fmt.Sprintf("Neighbor %s: Send update message valid routes:%+v",
			p.NeighborConf.Neighbor.NeighborAddress, nlriList))
		updateMsg := packet.NewBGPUpdateMessage(make([]packet.NLRI, 0), path.PathAttrs, nlriList)
		p.sendUpdateMsg(updateMsg.Clone(), path)
	}
}
