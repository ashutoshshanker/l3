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

func (adjRib *AdjRib) updateRouteList(addedRoutes, updatedRoutes, deletedRoutes []*Route) {
	if len(addedRoutes) > 0 {
		adjRib.addRoutesToRouteList(addedRoutes)
	}

	if len(deletedRoutes) > 0 {
		adjRib.removeRoutesFromRouteList(deletedRoutes)
	}
}
func (adjRib *AdjRib) updateRibOutInfo(action RouteAction, addRoutes, updRoutes, delRoutes []*Route,
	dest *Destination, withdrawn []packet.IPPrefix, updated map[*Path][]packet.IPPrefix) ([]packet.IPPrefix,
	map[*Path][]packet.IPPrefix) {
	if action == RouteActionAdd || action == RouteActionReplace {
		updated[dest.locRibPath] = append(updated[dest.locRibPath], dest.nlri)
	} else if action == RouteActionDelete {
		withdrawn = append(withdrawn, dest.nlri)
	}

	adjRib.updateRouteList(addRoutes, updRoutes, delRoutes)
	return withdrawn, updated
}

func (adjRib *AdjRib) ProcessRoutes(peerIP string, add []packet.IPPrefix, addPath *Path, rem []packet.IPPrefix,
	remPath *Path) (map[*Path][]packet.IPPrefix, []packet.IPPrefix) {
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
			action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
			withdrawn, updated = adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
			if action == RouteActionDelete {
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
		action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
		withdrawn, updated = adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
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

	for destIP, dest := range adjRib.destPathMap {
		dest.RemovePath(peerIP, remPath)
		action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
		withdrawn, updated = adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
		if action == RouteActionDelete && dest.IsEmpty() {
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
		action, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib()
		adjRib.updateRibOutInfo(action, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated)
		if action == RouteActionDelete && dest.IsEmpty() {
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
