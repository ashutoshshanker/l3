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
	server         *BGPServer
	logger         *syslog.Writer
	destPathMap    map[string]*Destination
	routeList      []*Route
	routeMutex     sync.RWMutex
	routeListDirty bool
	activeGet      bool
	timer          *time.Timer
}

func NewAdjRib(server *BGPServer) *AdjRib {
	rib := &AdjRib{
		server:         server,
		logger:         server.logger,
		destPathMap:    make(map[string]*Destination),
		routeList:      make([]*Route, 0),
		routeListDirty: false,
		activeGet:      false,
		routeMutex:     sync.RWMutex{},
	}

	rib.timer = time.AfterFunc(time.Duration(100)*time.Second, rib.ResetRouteList)
	rib.timer.Stop()
	return rib
}

func isIpInList(prefixes []packet.NLRI, ip packet.NLRI) bool {
	for _, nlri := range prefixes {
		if nlri.GetPathId() == ip.GetPathId() && nlri.GetPrefix().Prefix.Equal(ip.GetPrefix().Prefix) {
			return true
		}
	}
	return false
}

func (adjRib *AdjRib) getDest(nlri packet.NLRI, createIfNotExist bool) (*Destination, bool) {
	dest, ok := adjRib.destPathMap[nlri.GetPrefix().Prefix.String()]
	if !ok && createIfNotExist {
		dest = NewDestination(adjRib.server, nlri.GetPrefix())
		adjRib.destPathMap[nlri.GetPrefix().Prefix.String()] = dest
		ok = true
	}

	return dest, ok
}

func (adjRib *AdjRib) updateRouteList(addedRoutes, updatedRoutes, deletedRoutes []*Route) {
	if len(addedRoutes) > 0 {
		adjRib.addRoutesToRouteList(addedRoutes)
	}

	if len(deletedRoutes) > 0 {
		adjRib.removeRoutesFromRouteList(deletedRoutes)
	}
}

func (adjRib *AdjRib) updateRibOutInfo(action RouteAction, addRoutes, updRoutes, delRoutes []*Route,
	dest *Destination, withdrawn []packet.NLRI, updated map[*Path][]packet.NLRI) ([]packet.NLRI,
	map[*Path][]packet.NLRI) {
	if action == RouteActionAdd || action == RouteActionReplace {
		updated[dest.locRibPath] = append(updated[dest.locRibPath], dest.ipPrefix)
	} else if action == RouteActionDelete {
		withdrawn = append(withdrawn, dest.ipPrefix)
	}

	adjRib.updateRouteList(addRoutes, updRoutes, delRoutes)
	return withdrawn, updated
}

func (adjRib *AdjRib) ProcessRoutes(peerIP string, add []packet.NLRI, addPath *Path, rem []packet.NLRI,
	remPath *Path) (map[*Path][]packet.NLRI, []packet.NLRI) {
	withdrawn := make([]packet.NLRI, 0)
	updated := make(map[*Path][]packet.NLRI)

	// process withdrawn routes
	for _, nlri := range rem {
		if !isIpInList(add, nlri) {
			adjRib.logger.Info(fmt.Sprintln("Processing withdraw destination", nlri.GetPrefix().Prefix.String()))
			dest, ok := adjRib.getDest(nlri, false)
			if !ok {
				adjRib.logger.Warning(fmt.Sprintln("Can't process withdraw field. Destination does not exist, Dest:",
					nlri.GetPrefix().Prefix.String()))
				continue
			}
			dest.RemovePath(peerIP, nlri.GetPathId(), remPath)
			action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
			withdrawn, updated = adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
			if action == RouteActionDelete {
				if dest.IsEmpty() {
					delete(adjRib.destPathMap, nlri.GetPrefix().Prefix.String())
				}
			}
		} else {
			adjRib.logger.Info(fmt.Sprintln("Can't withdraw destination", nlri.GetPrefix().Prefix.String(),
				"Destination is part of NLRI in the UDPATE"))
		}
	}

	for _, nlri := range add {
		adjRib.logger.Info(fmt.Sprintln("Processing nlri", nlri.GetPrefix().Prefix.String()))
		dest, _ := adjRib.getDest(nlri, true)
		dest.AddOrUpdatePath(peerIP, nlri.GetPathId(), addPath)
		action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
		withdrawn, updated = adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
	}

	return updated, withdrawn
}

func (adjRib *AdjRib) ProcessUpdate(peer *Peer, pktInfo *packet.BGPPktSrc) (map[*Path][]packet.NLRI, []packet.NLRI,
	*Path) {
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

func (adjRib *AdjRib) ProcessConnectedRoutes(src string, path *Path, add []packet.NLRI, remove []packet.NLRI) (
	map[*Path][]packet.NLRI, []packet.NLRI, *Path) {
	var removePath *Path
	removePath = path.Clone()
	removePath.withdrawn = true
	path.updated = true
	updated, withdrawn := adjRib.ProcessRoutes(src, add, path, remove, removePath)
	path.updated = false
	return updated, withdrawn, removePath
}

func (adjRib *AdjRib) RemoveUpdatesFromNeighbor(peerIP string, peer *Peer) (map[*Path][]packet.NLRI, []packet.NLRI,
	*Path) {
	remPath := NewPath(adjRib.server, peer, nil, true, false, RouteTypeEGP)
	withdrawn := make([]packet.NLRI, 0)
	updated := make(map[*Path][]packet.NLRI)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllPaths(peerIP, remPath)
		action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
		withdrawn, updated = adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
		if action == RouteActionDelete && dest.IsEmpty() {
			delete(adjRib.destPathMap, destIP)
		}
	}

	return updated, withdrawn, remPath
}

func (adjRib *AdjRib) RemoveUpdatesFromAllNeighbors() {
	withdrawn := make([]packet.NLRI, 0)
	updated := make(map[*Path][]packet.NLRI)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllNeighborPaths()
		action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
		adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
		if action == RouteActionDelete && dest.IsEmpty() {
			delete(adjRib.destPathMap, destIP)
		}
	}
}

func (adjRib *AdjRib) GetLocRib() map[*Path][]packet.NLRI {
	updated := make(map[*Path][]packet.NLRI)
	for _, dest := range adjRib.destPathMap {
		if dest.locRibPath != nil {
			updated[dest.locRibPath] = append(updated[dest.locRibPath], dest.ipPrefix)
		}
	}

	return updated
}

func (adjRib *AdjRib) removeRoutesFromRouteList(routes []*Route) {
	defer adjRib.routeMutex.Unlock()
	adjRib.routeMutex.Lock()
	adjRib.logger.Info(fmt.Sprintln("removeRoutesFromRouteList: routes =", routes))
	for _, route := range routes {
		idx := route.routeListIdx
		if idx != -1 {
			adjRib.logger.Info(fmt.Sprintln("removeRoutesFromRouteList: remove route at idx", idx, "routeList =", adjRib.routeList))
			if !adjRib.activeGet {
				adjRib.routeList[idx] = adjRib.routeList[len(adjRib.routeList)-1]
				adjRib.routeList[idx].setIdx(idx)
				adjRib.routeList[len(adjRib.routeList)-1] = nil
				adjRib.routeList = adjRib.routeList[:len(adjRib.routeList)-1]
			} else {
				adjRib.routeList[idx] = nil
				adjRib.routeListDirty = true
			}
		}
	}
}

func (adjRib *AdjRib) addRoutesToRouteList(routes []*Route) {
	defer adjRib.routeMutex.Unlock()
	adjRib.routeMutex.Lock()
	adjRib.logger.Info(fmt.Sprintln("addRoutesToRouteList: routes =", routes))
	for _, route := range routes {
		adjRib.routeList = append(adjRib.routeList, route)
		adjRib.logger.Info(fmt.Sprintln("addRoutesToRouteList: added route at idx", len(adjRib.routeList)-1, "routeList =", adjRib.routeList))
		route.routeListIdx = len(adjRib.routeList) - 1
	}
}

func (adjRib *AdjRib) ResetRouteList() {
	defer adjRib.routeMutex.Unlock()
	adjRib.routeMutex.Lock()
	adjRib.activeGet = false

	if !adjRib.routeListDirty {
		return
	}

	lastIdx := len(adjRib.routeList) - 1
	var modIdx int
	for idx := 0; idx < len(adjRib.routeList); idx++ {
		if adjRib.routeList[idx] == nil {
			for modIdx := lastIdx; modIdx > idx && adjRib.routeList[modIdx] == nil; modIdx-- {
			}
			if modIdx <= idx {
				lastIdx = idx
				break
			}
			adjRib.routeList[idx] = adjRib.routeList[modIdx]
			adjRib.routeList[idx].setIdx(idx)
			adjRib.routeList[modIdx] = nil
			lastIdx = modIdx
		}
	}
	adjRib.routeList = adjRib.routeList[:lastIdx]
	adjRib.routeListDirty = false
}

func (adjRib *AdjRib) GetBGPRoutes(prefix string) []*bgpd.BGPRoute {
	defer adjRib.routeMutex.RUnlock()
	adjRib.routeMutex.RLock()

	if dest, ok := adjRib.destPathMap[prefix]; ok {
		return dest.GetBGPRoutes()
	}

	return nil
}

func (adjRib *AdjRib) BulkGetBGPRoutes(index int, count int) (int, int, []*bgpd.BGPRoute) {
	adjRib.timer.Stop()
	if index == 0 && adjRib.activeGet {
		adjRib.ResetRouteList()
	}
	adjRib.activeGet = true

	defer adjRib.routeMutex.RUnlock()
	adjRib.routeMutex.RLock()

	var i int
	n := 0
	result := make([]*bgpd.BGPRoute, count)
	for i = index; i < len(adjRib.routeList) && n < count; i++ {
		if adjRib.routeList[i] != nil && adjRib.routeList[i].path != nil {
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
