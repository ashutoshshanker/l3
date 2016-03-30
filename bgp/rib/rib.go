// rib.go
package server

import (
	"bgpd"
	"fmt"
	"l3/bgp/baseobjects"
	"l3/bgp/config"
	"l3/bgp/packet"
	"net"
	"ribd"
	"sync"
	"time"
	"utils/logging"
)

const ResetTime int = 120
const AggregatePathId uint32 = 0

type AdjRib struct {
	logger         *logging.Writer
	gConf          *config.GlobalConfig
	ribdClient     *ribd.RIBDServicesClient
	destPathMap    map[string]*Destination
	routeList      []*Route
	routeMutex     sync.RWMutex
	routeListDirty bool
	activeGet      bool
	timer          *time.Timer
}

func NewAdjRib(logger *logging.Writer, ribdClient *ribd.RIBDServicesClient, gConf *config.GlobalConfig) *AdjRib {
	rib := &AdjRib{
		logger:         logger,
		gConf:          gConf,
		ribdClient:     ribdClient,
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

func (adjRib *AdjRib) GetDestFromIPAndLen(ip string, cidrLen uint32) *Destination {
	if dest, ok := adjRib.destPathMap[ip]; ok {
		return dest
	}

	return nil
}

func (adjRib *AdjRib) GetDest(nlri packet.NLRI, createIfNotExist bool) (*Destination, bool) {
	dest, ok := adjRib.destPathMap[nlri.GetPrefix().Prefix.String()]
	if !ok && createIfNotExist {
		dest = NewDestination(adjRib, nlri.GetPrefix(), adjRib.gConf)
		adjRib.destPathMap[nlri.GetPrefix().Prefix.String()] = dest
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

func (adjRib *AdjRib) updateRibOutInfo(action RouteAction, addPathsMod bool, addRoutes, updRoutes, delRoutes []*Route,
	dest *Destination, withdrawn []*Destination, updated map[*Path][]*Destination, updatedAddPaths []*Destination) (
	[]*Destination, map[*Path][]*Destination, []*Destination) {
	if action == RouteActionAdd || action == RouteActionReplace {
		updated[dest.LocRibPath] = append(updated[dest.LocRibPath], dest)
	} else if action == RouteActionDelete {
		withdrawn = append(withdrawn, dest)
	} else if addPathsMod {
		updatedAddPaths = append(updatedAddPaths, dest)
	}

	adjRib.updateRouteList(addRoutes, updRoutes, delRoutes)
	return withdrawn, updated, updatedAddPaths
}

func (adjRib *AdjRib) ProcessRoutes(peerIP string, add []packet.NLRI, addPath *Path, rem []packet.NLRI, remPath *Path,
	addPathCount int) (map[*Path][]*Destination, []*Destination, []*Destination) {
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	// process withdrawn routes
	for _, nlri := range rem {
		if !isIpInList(add, nlri) {
			adjRib.logger.Info(fmt.Sprintln("Processing withdraw destination", nlri.GetPrefix().Prefix.String()))
			dest, ok := adjRib.GetDest(nlri, false)
			if !ok {
				adjRib.logger.Warning(fmt.Sprintln("Can't process withdraw field. Destination does not exist, Dest:",
					nlri.GetPrefix().Prefix.String()))
				continue
			}
			dest.RemovePath(peerIP, nlri.GetPathId(), remPath)
			action, addPathsMod, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib(addPathCount)
			withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
				delRoutes, dest, withdrawn, updated, updatedAddPaths)
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
		dest, _ := adjRib.GetDest(nlri, true)
		dest.AddOrUpdatePath(peerIP, nlri.GetPathId(), addPath)
		action, addPathsMod, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib(addPathCount)
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
			delRoutes, dest, withdrawn, updated, updatedAddPaths)
	}

	return updated, withdrawn, updatedAddPaths
}

func (adjRib *AdjRib) ProcessUpdate(neighborConf *base.NeighborConf, pktInfo *packet.BGPPktSrc, addPathCount int) (
	map[*Path][]*Destination, []*Destination, *Path, []*Destination) {
	body := pktInfo.Msg.Body.(*packet.BGPUpdate)

	remPath := NewPath(adjRib, neighborConf, body.PathAttributes, true, false, RouteTypeEGP)
	addPath := NewPath(adjRib, neighborConf, body.PathAttributes, false, true, RouteTypeEGP)
	//addPath.GetReachabilityInfo()
	if !addPath.IsValid() {
		adjRib.logger.Info(fmt.Sprintf("Received a update with our cluster id %d. Discarding the update.",
			addPath.NeighborConf.RunningConf.RouteReflectorClusterId))
		return nil, nil, nil, nil
	}

	updated, withdrawn, updatedAddPaths := adjRib.ProcessRoutes(pktInfo.Src, body.NLRI, addPath, body.WithdrawnRoutes,
		remPath, addPathCount)
	addPath.updated = false
	return updated, withdrawn, remPath, updatedAddPaths
}

func (adjRib *AdjRib) ProcessConnectedRoutes(src string, path *Path, add []packet.NLRI, remove []packet.NLRI,
	addPathCount int) (map[*Path][]*Destination, []*Destination, *Path, []*Destination) {
	var removePath *Path
	removePath = path.Clone()
	removePath.withdrawn = true
	path.updated = true
	updated, withdrawn, updatedAddPaths := adjRib.ProcessRoutes(src, add, path, remove, removePath, addPathCount)
	path.updated = false
	return updated, withdrawn, removePath, updatedAddPaths
}

func (adjRib *AdjRib) RemoveUpdatesFromNeighbor(peerIP string, neighborConf *base.NeighborConf, addPathCount int) (
	map[*Path][]*Destination, []*Destination, *Path, []*Destination) {
	remPath := NewPath(adjRib, neighborConf, nil, true, false, RouteTypeEGP)
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllPaths(peerIP, remPath)
		action, addPathsMod, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib(addPathCount)
		adjRib.logger.Info(fmt.Sprintln("RemoveUpdatesFromNeighbor - dest", dest.IPPrefix.Prefix.String(),
			"SelectRouteForLocRib returned action", action, "addRoutes", addRoutes, "updRoutes", updRoutes,
			"delRoutes", delRoutes))
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
			delRoutes, dest, withdrawn, updated, updatedAddPaths)
		if action == RouteActionDelete && dest.IsEmpty() {
			adjRib.logger.Info(fmt.Sprintln("All routes removed for dest", dest.IPPrefix.Prefix.String()))
			delete(adjRib.destPathMap, destIP)
		}
	}

	return updated, withdrawn, remPath, updatedAddPaths
}

func (adjRib *AdjRib) RemoveUpdatesFromAllNeighbors(addPathCount int) {
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllNeighborPaths()
		action, addPathsMod, addRoutes, updRoutes, delRoutes := dest.SelectRouteForLocRib(addPathCount)
		adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes, delRoutes, dest, withdrawn, updated,
			updatedAddPaths)
		if action == RouteActionDelete && dest.IsEmpty() {
			delete(adjRib.destPathMap, destIP)
		}
	}
}

func (adjRib *AdjRib) GetLocRib() map[*Path][]*Destination {
	updated := make(map[*Path][]*Destination)
	for _, dest := range adjRib.destPathMap {
		if dest.LocRibPath != nil {
			updated[dest.LocRibPath] = append(updated[dest.LocRibPath], dest)
		}
	}

	return updated
}

func (adjRib *AdjRib) RemoveRouteFromAggregate(ip *packet.IPPrefix, aggIP *packet.IPPrefix, srcIP string,
	bgpAgg *config.BGPAggregate, ipDest *Destination, addPathCount int) (map[*Path][]*Destination, []*Destination,
	*Path, []*Destination) {
	var aggPath, path *Path
	var dest *Destination
	var aggDest *Destination
	var ok bool
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	adjRib.logger.Info(fmt.Sprintf("AdjRib:RemoveRouteFromAggregate - ip %v, aggIP %v", ip, aggIP))
	if dest, ok = adjRib.GetDest(ip, false); !ok {
		if ipDest == nil {
			adjRib.logger.Info(fmt.Sprintln("RemoveRouteFromAggregate: routes ip", ip, "not found"))
			return updated, withdrawn, nil, nil
		}
		dest = ipDest
	}
	adjRib.logger.Info(fmt.Sprintln("RemoveRouteFromAggregate: locRibPath", dest.LocRibPath, "locRibRoutePath", dest.LocRibPathRoute.path))
	path = dest.LocRibPathRoute.path
	remPath := NewPath(adjRib, nil, path.PathAttrs, true, false, path.routeType)

	if aggDest, ok = adjRib.GetDest(aggIP, false); !ok {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:RemoveRouteFromAggregate - dest not found for aggIP %v", aggIP))
		return updated, withdrawn, nil, nil
	}

	if aggPath = aggDest.getPathForIP(srcIP, AggregatePathId); aggPath == nil {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:RemoveRouteFromAggregate - path not found for dest, aggIP %v", aggIP))
		return updated, withdrawn, nil, nil
	}

	aggPath.removePathFromAggregate(ip.Prefix.String(), bgpAgg.GenerateASSet)
	if aggPath.isAggregatePathEmpty() {
		aggDest.RemovePath(srcIP, AggregatePathId, aggPath)
	} else {
		aggDest.setUpdateAggPath(srcIP, AggregatePathId)
	}
	aggDest.removeAggregatedDests(ip.Prefix.String())
	action, addPathsMod, addRoutes, updRoutes, delRoutes := aggDest.SelectRouteForLocRib(addPathCount)
	withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes, delRoutes, aggDest,
		withdrawn, updated, updatedAddPaths)
	if action == RouteActionAdd || action == RouteActionReplace {
		dest.aggPath = aggPath
	}
	if action == RouteActionDelete && aggDest.IsEmpty() {
		delete(adjRib.destPathMap, aggIP.Prefix.String())
	}

	return updated, withdrawn, remPath, updatedAddPaths
}

func (adjRib *AdjRib) AddRouteToAggregate(ip *packet.IPPrefix, aggIP *packet.IPPrefix, srcIP string, ifaceIP net.IP,
	bgpAgg *config.BGPAggregate, addPathCount int) (map[*Path][]*Destination, []*Destination, *Path, []*Destination) {
	var aggPath, path *Path
	var dest *Destination
	var aggDest *Destination
	var ok bool
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - ip %v, aggIP %v", ip, aggIP))
	if dest, ok = adjRib.GetDest(ip, false); !ok {
		adjRib.logger.Info(fmt.Sprintln("AddRouteToAggregate: routes ip", ip, "not found"))
		return updated, withdrawn, nil, nil
	}
	path = dest.LocRibPath
	remPath := NewPath(adjRib, nil, path.PathAttrs, true, false, path.routeType)

	if aggDest, ok = adjRib.GetDest(aggIP, true); ok {
		aggPath = aggDest.getPathForIP(srcIP, AggregatePathId)
		adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - aggIP %v found in dest, agg path %v", aggIP, aggPath))
	}

	if aggPath != nil {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - aggIP %v, agg path found, update path attrs", aggIP))
		aggPath.addPathToAggregate(ip.Prefix.String(), path, bgpAgg.GenerateASSet)
		aggDest.setUpdateAggPath(srcIP, AggregatePathId)
		aggDest.addAggregatedDests(ip.Prefix.String(), dest)
		action, addPathsMod, addRoutes, updRoutes, delRoutes := aggDest.SelectRouteForLocRib(addPathCount)
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
			delRoutes, aggDest, withdrawn, updated, updatedAddPaths)
		if action == RouteActionAdd || action == RouteActionReplace {
			dest.aggPath = aggPath
		}
	} else {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - aggIP %v, agg path NOT found, create new path", aggIP))
		pathAttrs := packet.ConstructPathAttrForAggRoutes(path.PathAttrs, bgpAgg.GenerateASSet)
		packet.SetNextHopPathAttrs(pathAttrs, ifaceIP)
		packet.SetPathAttrAggregator(pathAttrs, adjRib.gConf.AS, adjRib.gConf.RouterId)
		aggPath = NewPath(path.rib, nil, pathAttrs, false, true, RouteTypeAgg)
		aggPath.setAggregatedPath(ip.Prefix.String(), path)
		aggDest, _ := adjRib.GetDest(aggIP, true)
		aggDest.AddOrUpdatePath(srcIP, AggregatePathId, aggPath)
		aggDest.addAggregatedDests(ip.Prefix.String(), dest)
		action, addPathsMod, addRoutes, updRoutes, delRoutes := aggDest.SelectRouteForLocRib(addPathCount)
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
			delRoutes, aggDest, withdrawn, updated, updatedAddPaths)
		if action == RouteActionAdd || action == RouteActionReplace {
			dest.aggPath = aggPath
		}
	}

	if aggPath != nil {
		aggPath.SetUpdate(false)
	}
	return updated, withdrawn, remPath, updatedAddPaths
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
	var modIdx, idx int
	for idx = 0; idx < len(adjRib.routeList); idx++ {
		if adjRib.routeList[idx] == nil {
			for modIdx = lastIdx; modIdx > idx && adjRib.routeList[modIdx] == nil; modIdx-- {
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
	adjRib.routeList = adjRib.routeList[:idx]
	adjRib.routeListDirty = false
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
