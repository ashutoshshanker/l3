// rib.go
package server

import (
	"bgpd"
	"fmt"
	"l3/bgp/packet"
	"log/syslog"
	"sync"
	"time"
)

const ResetTime int = 120

type AdjRib struct {
	server      *BGPServer
	logger      *syslog.Writer
	destPathMap map[string]*Destination
	routeList   []*Destination
	routeMutex  sync.RWMutex
	activeGet   bool
	timer       *time.Timer
}

func NewAdjRib(server *BGPServer) *AdjRib {
	rib := &AdjRib{
		server:      server,
		logger:      server.logger,
		destPathMap: make(map[string]*Destination),
		routeList:   make([]*Destination, 0),
		activeGet:   false,
		routeMutex:  sync.RWMutex{},
	}

	rib.timer = time.AfterFunc(time.Duration(100)*time.Second, rib.ResetRouteList)
	rib.timer.Stop()
	return rib
}

func isIpInList(ipPrefix []packet.IPPrefix, ip packet.IPPrefix) bool {
	for _, prefix := range ipPrefix {
		if prefix.Prefix.Equal(ip.Prefix) {
			return true
		}
	}
	return false
}

func (adjRib *AdjRib) getDest(nlri packet.IPPrefix, createIfNotExist bool) (*Destination, bool) {
	dest, ok := adjRib.destPathMap[nlri.Prefix.String()]
	if !ok && createIfNotExist {
		dest = NewDestination(adjRib.server, nlri)
		adjRib.destPathMap[nlri.Prefix.String()] = dest
		ok = true
	}

	return dest, ok
}

func updateRibOutInfo(action RouteSelectionAction, dest *Destination, withdrawn []packet.IPPrefix,
	updated map[*Path][]packet.IPPrefix) ([]packet.IPPrefix, map[*Path][]packet.IPPrefix) {
	if action == RouteSelectionAdd || action == RouteSelectionReplace {
		updated[dest.locRibPath] = append(updated[dest.locRibPath], dest.nlri)
	} else if action == RouteSelectionDelete {
		withdrawn = append(withdrawn, dest.nlri)
	}

	return withdrawn, updated
}

func (adjRib *AdjRib) ProcessRoutes(peerIP string, add []packet.IPPrefix, addPath *Path, rem []packet.IPPrefix,
	remPath *Path) (map[*Path][]packet.IPPrefix, []packet.IPPrefix) {
	var action RouteSelectionAction
	withdrawn := make([]packet.IPPrefix, 0)
	updated := make(map[*Path][]packet.IPPrefix)

	// process withdrawn routes
	for _, nlri := range rem {
		if !isIpInList(add, nlri) {
			adjRib.logger.Info(fmt.Sprintln("Processing withdraw destination", nlri.Prefix.String()))
			dest, ok := adjRib.getDest(nlri, false)
			if !ok {
				adjRib.logger.Warning(fmt.Sprintln("Can't process withdraw field. Destination does not exist, Dest:", nlri.Prefix.String()))
				continue
			}
			dest.RemovePath(peerIP, remPath)
			action = dest.SelectRouteForLocRib()
			withdrawn, updated = updateRibOutInfo(action, dest, withdrawn, updated)
			if action == RouteSelectionDelete {

				adjRib.removeDestFromRouteList(dest)
				if dest.IsEmpty() {
					delete(adjRib.destPathMap, nlri.Prefix.String())
				}
			}
		} else {
			adjRib.logger.Info(fmt.Sprintln("Can't withdraw destination", nlri.Prefix.String(),
				"Destination is part of NLRI in the UDPATE"))
		}
	}

	for _, nlri := range add {
		adjRib.logger.Info(fmt.Sprintln("Processing nlri", nlri.Prefix.String()))
		dest, _ := adjRib.getDest(nlri, true)
		dest.AddOrUpdatePath(peerIP, addPath)
		action = dest.SelectRouteForLocRib()
		withdrawn, updated = updateRibOutInfo(action, dest, withdrawn, updated)
	}

	return updated, withdrawn
}

func (adjRib *AdjRib) ProcessUpdate(peer *Peer, pktInfo *packet.BGPPktSrc) (map[*Path][]packet.IPPrefix, []packet.IPPrefix, *Path) {
	body := pktInfo.Msg.Body.(*packet.BGPUpdate)

	remPath := NewPath(adjRib.server, peer, body.PathAttributes, true, false, RouteTypeEGP)
	addPath := NewPath(adjRib.server, peer, body.PathAttributes, false, true, RouteTypeEGP)
	addPath.GetReachabilityInfo()
	if !addPath.IsValid() {
		adjRib.logger.Info(fmt.Sprintf("Received a update with our cluster id %d. Discarding the update.", addPath.peer.PeerConf.RouteReflectorClusterId))
		return nil, nil, nil
	}

	updated, withdrawn := adjRib.ProcessRoutes(pktInfo.Src, body.NLRI, addPath, body.WithdrawnRoutes, remPath)
	addPath.updated = false
	return updated, withdrawn, remPath
}

func (adjRib *AdjRib) ProcessConnectedRoutes(src string, path *Path, add []packet.IPPrefix, remove []packet.IPPrefix) (
	map[*Path][]packet.IPPrefix, []packet.IPPrefix, *Path) {
	var removePath *Path
	removePath = path.Clone()
	removePath.withdrawn = true
	path.updated = true
	updated, withdrawn := adjRib.ProcessRoutes(src, add, path, remove, removePath)
	path.updated = false
	return updated, withdrawn, removePath
}

func (adjRib *AdjRib) RemoveUpdatesFromNeighbor(peerIP string, peer *Peer) (map[*Path][]packet.IPPrefix, []packet.IPPrefix, *Path) {
	remPath := NewPath(adjRib.server, peer, nil, true, false, RouteTypeEGP)
	withdrawn := make([]packet.IPPrefix, 0)
	updated := make(map[*Path][]packet.IPPrefix)
	var action RouteSelectionAction

	for destIP, dest := range adjRib.destPathMap {
		dest.RemovePath(peerIP, remPath)
		action = dest.SelectRouteForLocRib()
		withdrawn, updated = updateRibOutInfo(action, dest, withdrawn, updated)
		if action == RouteSelectionDelete && dest.IsEmpty() {
			delete(adjRib.destPathMap, destIP)
		}
	}

	return updated, withdrawn, remPath
}

func (adjRib *AdjRib) RemoveUpdatesFromAllNeighbors() {
	withdrawn := make([]packet.IPPrefix, 0)
	updated := make(map[*Path][]packet.IPPrefix)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllNeighborPaths()
		action := dest.SelectRouteForLocRib()
		updateRibOutInfo(action, dest, withdrawn, updated)
		if action == RouteSelectionDelete && dest.IsEmpty() {
			delete(adjRib.destPathMap, destIP)
		}
	}
}

func (adjRib *AdjRib) GetLocRib() map[*Path][]packet.IPPrefix {
	updated := make(map[*Path][]packet.IPPrefix)
	for _, dest := range adjRib.destPathMap {
		if dest.locRibPath != nil {
			updated[dest.locRibPath] = append(updated[dest.locRibPath], dest.nlri)
		}
	}

	return updated
}

func (adjRib *AdjRib) removeDestFromRouteList(dest *Destination) {
	idx := dest.routeListIdx
	if idx != -1 {
		defer adjRib.routeMutex.Unlock()
		adjRib.routeMutex.Lock()
		if !adjRib.activeGet {
			adjRib.routeList[idx] = adjRib.routeList[len(adjRib.routeList)-1]
			adjRib.routeList[len(adjRib.routeList)-1] = nil
			adjRib.routeList = append(adjRib.routeList[:idx], adjRib.routeList[idx+1:]...)
			adjRib.routeList = adjRib.routeList[:len(adjRib.routeList)-1]
		} else {
			adjRib.routeList[idx] = nil
		}
	}
}

func (adjRib *AdjRib) addDestToRouteList(dest *Destination) {
	defer adjRib.routeMutex.Unlock()
	adjRib.routeMutex.Lock()
	adjRib.routeList = append(adjRib.routeList, dest)
}

func (adjRib *AdjRib) ResetRouteList() {
	defer adjRib.routeMutex.Unlock()
	adjRib.routeMutex.Lock()
	adjRib.activeGet = false

	lastIdx := len(adjRib.routeList) - 1
	var modIdx int
	for idx := 0; idx < len(adjRib.routeList); idx++ {
		if adjRib.routeList[idx] == nil {
			for modIdx := lastIdx; modIdx > idx && adjRib.routeList[modIdx] == nil; modIdx-- {
			}
			if modIdx <= idx {
				break
			}
			adjRib.routeList[idx] = adjRib.routeList[modIdx]
			adjRib.routeList[modIdx] = nil
			lastIdx = modIdx
		}
	}
	adjRib.routeList = adjRib.routeList[:lastIdx]
}

func (adjRib *AdjRib) GetBGPRoute(prefix string) *bgpd.BGPRoute {
	defer adjRib.routeMutex.RUnlock()
	adjRib.routeMutex.RLock()

	if dest, ok := adjRib.destPathMap[prefix]; ok {
		return dest.GetBGPRoute()
	}

	return nil
}

func (adjRib *AdjRib) BulkGetBGPRoutes(index int, count int) (int, int, []*bgpd.BGPRoute) {
	defer adjRib.routeMutex.RUnlock()

	adjRib.routeMutex.RLock()
	adjRib.timer.Stop()
	if index == 0 && adjRib.activeGet {
		adjRib.ResetRouteList()
	}
	adjRib.activeGet = true

	var i int
	n := 0
	result := make([]*bgpd.BGPRoute, count)
	for i = index; i < len(adjRib.routeList) && n < count; i++ {
		if adjRib.routeList[i] != nil && adjRib.routeList[i].locRibPath != nil {
			result[n] = adjRib.routeList[i].GetBGPRoute()
			n++
		}
	}
	result = result[:n]

	if i >= len(adjRib.routeList) {
		i = 0
	}

	adjRib.timer.Reset(time.Duration(ResetTime) * time.Second)
	return i, n, result
}
