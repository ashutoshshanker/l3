// rib.go
package server

import (
	"bgpd"
	"fmt"
	"l3/bgp/baseobjects"
	"l3/bgp/config"
	"l3/bgp/packet"
	"net"
	"sync"
	"time"
	"utils/logging"
)

var totalRoutes int

const ResetTime int = 120
const AggregatePathId uint32 = 0

type ReachabilityInfo struct {
	NextHop       string
	NextHopIfType int32
	NextHopIfIdx  int32
	Metric        int32
}

func NewReachabilityInfo(nextHop string, nhIfType, nhIfIdx, metric int32) *ReachabilityInfo {
	return &ReachabilityInfo{
		NextHop:       nextHop,
		NextHopIfType: nhIfType,
		NextHopIfIdx:  nhIfIdx,
		Metric:        metric,
	}
}

type AdjRib struct {
	logger           *logging.Writer
	gConf            *config.GlobalConfig
	routeMgr         config.RouteMgrIntf
	destPathMap      map[string]*Destination
	reachabilityMap  map[string]*ReachabilityInfo
	unreachablePaths map[string]map[*Path]map[*Destination][]uint32
	routeList        []*Route
	routeMutex       sync.RWMutex
	routeListDirty   bool
	activeGet        bool
	timer            *time.Timer
}

func NewAdjRib(logger *logging.Writer, rMgr config.RouteMgrIntf,
	gConf *config.GlobalConfig) *AdjRib {
	rib := &AdjRib{
		logger:           logger,
		gConf:            gConf,
		routeMgr:         rMgr,
		destPathMap:      make(map[string]*Destination),
		reachabilityMap:  make(map[string]*ReachabilityInfo),
		unreachablePaths: make(map[string]map[*Path]map[*Destination][]uint32),
		routeList:        make([]*Route, 0),
		routeListDirty:   false,
		activeGet:        false,
		routeMutex:       sync.RWMutex{},
	}

	rib.timer = time.AfterFunc(time.Duration(100)*time.Second, rib.ResetRouteList)
	rib.timer.Stop()
	return rib
}

func isIpInList(prefixes []packet.NLRI, ip packet.NLRI) bool {
	for _, nlri := range prefixes {
		if nlri.GetPathId() == ip.GetPathId() &&
			nlri.GetPrefix().Prefix.Equal(ip.GetPrefix().Prefix) {
			return true
		}
	}
	return false
}

func (adjRib *AdjRib) GetReachabilityInfo(path *Path) *ReachabilityInfo {
	ipStr := path.GetNextHop().String()
	if reachabilityInfo, ok := adjRib.reachabilityMap[ipStr]; ok {
		return reachabilityInfo
	}

	adjRib.logger.Info(fmt.Sprintf("GetReachabilityInfo: Reachability info not cached",
		"for Next hop %s", ipStr))
	ribdReachabilityInfo, err := adjRib.routeMgr.GetNextHopInfo(ipStr)
	if err != nil {
		adjRib.logger.Info(fmt.Sprintf("NEXT_HOP[%s] is not reachable", ipStr))
		return nil
	}
	nextHop := ribdReachabilityInfo.NextHopIp
	if nextHop == "" || nextHop[0] == '0' {
		adjRib.logger.Info(fmt.Sprintf("Next hop for %s is %s. Using %s as the next hop",
			ipStr, nextHop, ipStr))
		nextHop = ipStr
	}

	reachabilityInfo := NewReachabilityInfo(nextHop, ribdReachabilityInfo.NextHopIfType,
		ribdReachabilityInfo.NextHopIfIndex, ribdReachabilityInfo.Metric)
	adjRib.reachabilityMap[ipStr] = reachabilityInfo
	return reachabilityInfo
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

func (adjRib *AdjRib) updateRibOutInfo(action RouteAction, addPathsMod bool, addRoutes,
	updRoutes, delRoutes []*Route, dest *Destination, withdrawn []*Destination,
	updated map[*Path][]*Destination, updatedAddPaths []*Destination) (
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

func (adjRib *AdjRib) ProcessRoutes(peerIP string, add []packet.NLRI, addPath *Path,
	rem []packet.NLRI, remPath *Path, addPathCount int) (map[*Path][]*Destination,
	[]*Destination, []*Destination, bool) {
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)
	addedAllPrefixes := true

	// process withdrawn routes
	for _, nlri := range rem {
		if !isIpInList(add, nlri) {
			adjRib.logger.Info(fmt.Sprintln("Processing withdraw destination",
				nlri.GetPrefix().Prefix.String()))
			dest, ok := adjRib.GetDest(nlri, false)
			if !ok {
				adjRib.logger.Warning(fmt.Sprintln("Can't process withdraw field.",
					"Destination does not exist, Dest:",
					nlri.GetPrefix().Prefix.String()))
				continue
			}
			oldPath := dest.RemovePath(peerIP, nlri.GetPathId(), remPath)
			if oldPath != nil && !oldPath.IsReachable() {
				nextHopStr := oldPath.GetNextHop().String()
				if _, ok := adjRib.unreachablePaths[nextHopStr]; ok {
					if _, ok := adjRib.unreachablePaths[nextHopStr][oldPath]; ok {
						if pathIds, ok :=
							adjRib.unreachablePaths[nextHopStr][oldPath][dest]; ok {
							for idx, pathId := range pathIds {
								if pathId == nlri.GetPathId() {
									adjRib.unreachablePaths[nextHopStr][oldPath][dest][idx] =
										pathIds[len(pathIds)-1]
									adjRib.unreachablePaths[nextHopStr][oldPath][dest] =
										adjRib.unreachablePaths[nextHopStr][oldPath][dest][:len(pathIds)-1]
									break
								}
							}
							if len(adjRib.unreachablePaths[nextHopStr][oldPath][dest]) ==
								0 {
								delete(adjRib.unreachablePaths[nextHopStr][oldPath],
									dest)
							}
						}
						if len(adjRib.unreachablePaths[nextHopStr][oldPath]) == 0 {
							delete(adjRib.unreachablePaths[nextHopStr], oldPath)
						}
					}
					if len(adjRib.unreachablePaths[nextHopStr]) == 0 {
						delete(adjRib.unreachablePaths, nextHopStr)
					}

				}
			}
			action, addPathsMod, addRoutes, updRoutes, delRoutes :=
				dest.SelectRouteForLocRib(addPathCount)
			withdrawn, updated, updatedAddPaths =
				adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
					delRoutes, dest, withdrawn, updated, updatedAddPaths)

			if oldPath != nil && remPath != nil {
				if neighborConf := remPath.GetNeighborConf(); neighborConf != nil {
					adjRib.logger.Info(fmt.Sprintln("Decrement prefix count for",
						"destination %s from Peer %s",
						nlri.GetPrefix().Prefix.String(), peerIP))
					neighborConf.DecrPrefixCount()
				}
			}
			if action == RouteActionDelete {
				if dest.IsEmpty() {
					delete(adjRib.destPathMap, nlri.GetPrefix().Prefix.String())
				}
			}
		} else {
			adjRib.logger.Info(fmt.Sprintln("Can't withdraw destination",
				nlri.GetPrefix().Prefix.String(),
				"Destination is part of NLRI in the UDPATE"))
		}
	}

	nextHopStr := addPath.GetNextHop().String()
	for _, nlri := range add {
		if nlri.GetPrefix().Prefix.String() == "0.0.0.0" {
			adjRib.logger.Info(fmt.Sprintf("Can't process NLRI 0.0.0.0"))
			continue
		}

		adjRib.logger.Info(fmt.Sprintln("Processing nlri", nlri.GetPrefix().Prefix.String()))
		dest, _ := adjRib.GetDest(nlri, true)
		if oldPath := dest.getPathForIP(peerIP, nlri.GetPathId()); oldPath == nil &&
			addPath.NeighborConf != nil {
			if !addPath.NeighborConf.CanAcceptNewPrefix() {
				adjRib.logger.Info(fmt.Sprintf("Max prefixes limit reached for",
					"peer %s, can't process %s",
					peerIP, nlri.GetPrefix().Prefix.String()))
				addedAllPrefixes = false
				continue
			}
			adjRib.logger.Info(fmt.Sprintf("Increment prefix count for destination %s",
				"from Peer %s", nlri.GetPrefix().Prefix.String(), peerIP))
			addPath.NeighborConf.IncrPrefixCount()
		}

		dest.AddOrUpdatePath(peerIP, nlri.GetPathId(), addPath)
		if !addPath.IsReachable() {
			if _, ok := adjRib.unreachablePaths[nextHopStr][addPath][dest]; !ok {
				adjRib.unreachablePaths[nextHopStr][addPath][dest] = make([]uint32, 0)
			}

			adjRib.unreachablePaths[nextHopStr][addPath][dest] =
				append(adjRib.unreachablePaths[nextHopStr][addPath][dest],
					nlri.GetPathId())
			continue
		}

		action, addPathsMod, addRoutes, updRoutes, delRoutes :=
			dest.SelectRouteForLocRib(addPathCount)
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod,
			addRoutes, updRoutes, delRoutes, dest, withdrawn, updated, updatedAddPaths)
	}

	return updated, withdrawn, updatedAddPaths, addedAllPrefixes
}

func (adjRib *AdjRib) ProcessRoutesForReachableRoutes(nextHop string, reachabilityInfo *ReachabilityInfo,
	addPathCount int, updated map[*Path][]*Destination, withdrawn []*Destination,
	updatedAddPaths []*Destination) (map[*Path][]*Destination, []*Destination, []*Destination) {

	if _, ok := adjRib.unreachablePaths[nextHop]; ok {
		for path, destinations := range adjRib.unreachablePaths[nextHop] {
			path.SetReachabilityInfo(reachabilityInfo)
			peerIP := path.GetPeerIP()
			if peerIP == "" {
				adjRib.logger.Err(fmt.Sprintln("ProcessRoutesForReachableRoutes:",
					"nexthop %s peer ip not found for path %+v",
					nextHop, path))
				continue
			}

			for dest, pathIds := range destinations {
				adjRib.logger.Info(fmt.Sprintln("Processing dest",
					dest.IPPrefix.Prefix.String()))
				for _, pathId := range pathIds {
					dest.AddOrUpdatePath(peerIP, pathId, path)
				}
				action, addPathsMod, addRoutes, updRoutes, delRoutes :=
					dest.SelectRouteForLocRib(addPathCount)
				withdrawn, updated, updatedAddPaths =
					adjRib.updateRibOutInfo(action, addPathsMod, addRoutes,
						updRoutes, delRoutes, dest, withdrawn,
						updated, updatedAddPaths)
			}
		}
	}

	return updated, withdrawn, updatedAddPaths
}

func (adjRib *AdjRib) ProcessUpdate(neighborConf *base.NeighborConf, pktInfo *packet.BGPPktSrc,
	addPathCount int) (map[*Path][]*Destination, []*Destination, *Path, []*Destination, bool) {
	body := pktInfo.Msg.Body.(*packet.BGPUpdate)

	remPath := NewPath(adjRib, neighborConf, body.PathAttributes, true, false, RouteTypeEGP)
	addPath := NewPath(adjRib, neighborConf, body.PathAttributes, false, true, RouteTypeEGP)

	reachabilityInfo := adjRib.GetReachabilityInfo(addPath)
	addPath.SetReachabilityInfo(reachabilityInfo)

	//addPath.GetReachabilityInfo()
	if !addPath.IsValid() {
		adjRib.logger.Info(fmt.Sprintf("Received a update with our cluster id %d.",
			"Discarding the update.",
			addPath.NeighborConf.RunningConf.RouteReflectorClusterId))
		return nil, nil, nil, nil, true
	}

	nextHopStr := addPath.GetNextHop().String()
	if reachabilityInfo == nil {
		adjRib.logger.Info(fmt.Sprintf("ProcessUpdate - next hop %s is not reachable",
			nextHopStr))

		if _, ok := adjRib.unreachablePaths[nextHopStr]; !ok {
			adjRib.unreachablePaths[nextHopStr] = make(map[*Path]map[*Destination][]uint32)
		}

		if _, ok := adjRib.unreachablePaths[nextHopStr][addPath]; !ok {
			adjRib.unreachablePaths[nextHopStr][addPath] = make(map[*Destination][]uint32)
		}
	}

	updated, withdrawn, updatedAddPaths, addedAllPrefixes :=
		adjRib.ProcessRoutes(pktInfo.Src, body.NLRI, addPath,
			body.WithdrawnRoutes, remPath, addPathCount)
	addPath.updated = false

	if reachabilityInfo != nil {
		adjRib.logger.Info(fmt.Sprintf("ProcessUpdate - next hop %s is reachable,",
			"so process previously unreachable routes", nextHopStr))
		updated, withdrawn, updatedAddPaths =
			adjRib.ProcessRoutesForReachableRoutes(nextHopStr, reachabilityInfo,
				addPathCount, updated, withdrawn, updatedAddPaths)
	}
	return updated, withdrawn, remPath, updatedAddPaths, addedAllPrefixes
}

func (adjRib *AdjRib) ProcessConnectedRoutes(src string, path *Path, add []packet.NLRI,
	remove []packet.NLRI, addPathCount int) (map[*Path][]*Destination,
	[]*Destination, *Path, []*Destination) {
	var removePath *Path
	removePath = path.Clone()
	removePath.withdrawn = true
	path.updated = true
	updated, withdrawn, updatedAddPaths, addedAllPrefixes :=
		adjRib.ProcessRoutes(src, add, path, remove, removePath, addPathCount)
	path.updated = false
	if !addedAllPrefixes {
		adjRib.logger.Err(fmt.Sprintf("Failed to add connected routes... max",
			"prefixes exceeded for connected routes!"))
	}
	return updated, withdrawn, removePath, updatedAddPaths
}

func (adjRib *AdjRib) RemoveUpdatesFromNeighbor(peerIP string, neighborConf *base.NeighborConf,
	addPathCount int) (
	map[*Path][]*Destination, []*Destination, *Path, []*Destination) {
	remPath := NewPath(adjRib, neighborConf, nil, true, false, RouteTypeEGP)
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllPaths(peerIP, remPath)
		action, addPathsMod, addRoutes, updRoutes, delRoutes :=
			dest.SelectRouteForLocRib(addPathCount)
		adjRib.logger.Info(fmt.Sprintln("RemoveUpdatesFromNeighbor - dest",
			dest.IPPrefix.Prefix.String(), "SelectRouteForLocRib returned action",
			action, "addRoutes", addRoutes, "updRoutes", updRoutes,
			"delRoutes", delRoutes))
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action,
			addPathsMod, addRoutes, updRoutes,
			delRoutes, dest, withdrawn, updated, updatedAddPaths)
		if action == RouteActionDelete && dest.IsEmpty() {
			adjRib.logger.Info(fmt.Sprintln("All routes removed for dest",
				dest.IPPrefix.Prefix.String()))
			delete(adjRib.destPathMap, destIP)
		}
	}

	if neighborConf != nil {
		neighborConf.SetPrefixCount(0)
	}
	return updated, withdrawn, remPath, updatedAddPaths
}

func (adjRib *AdjRib) RemoveUpdatesFromAllNeighbors(addPathCount int) {
	withdrawn := make([]*Destination, 0)
	updated := make(map[*Path][]*Destination)
	updatedAddPaths := make([]*Destination, 0)

	for destIP, dest := range adjRib.destPathMap {
		dest.RemoveAllNeighborPaths()
		action, addPathsMod, addRoutes, updRoutes, delRoutes :=
			dest.SelectRouteForLocRib(addPathCount)
		adjRib.updateRibOutInfo(action, addPathsMod, addRoutes, updRoutes,
			delRoutes, dest, withdrawn, updated,
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

func (adjRib *AdjRib) RemoveRouteFromAggregate(ip *packet.IPPrefix, aggIP *packet.IPPrefix,
	srcIP string, bgpAgg *config.BGPAggregate, ipDest *Destination,
	addPathCount int) (map[*Path][]*Destination, []*Destination,
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
			adjRib.logger.Info(fmt.Sprintln("RemoveRouteFromAggregate: routes ip",
				ip, "not found"))
			return updated, withdrawn, nil, nil
		}
		dest = ipDest
	}
	adjRib.logger.Info(fmt.Sprintln("RemoveRouteFromAggregate: locRibPath", dest.LocRibPath,
		"locRibRoutePath", dest.LocRibPathRoute.path))
	path = dest.LocRibPathRoute.path
	remPath := NewPath(adjRib, nil, path.PathAttrs, true, false, path.routeType)

	if aggDest, ok = adjRib.GetDest(aggIP, false); !ok {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:RemoveRouteFromAggregate - dest not",
			"found for aggIP %v", aggIP))
		return updated, withdrawn, nil, nil
	}

	if aggPath = aggDest.getPathForIP(srcIP, AggregatePathId); aggPath == nil {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:RemoveRouteFromAggregate - path not",
			"found for dest, aggIP %v", aggIP))
		return updated, withdrawn, nil, nil
	}

	aggPath.removePathFromAggregate(ip.Prefix.String(), bgpAgg.GenerateASSet)
	if aggPath.isAggregatePathEmpty() {
		aggDest.RemovePath(srcIP, AggregatePathId, aggPath)
	} else {
		aggDest.setUpdateAggPath(srcIP, AggregatePathId)
	}
	aggDest.removeAggregatedDests(ip.Prefix.String())
	action, addPathsMod, addRoutes, updRoutes, delRoutes :=
		aggDest.SelectRouteForLocRib(addPathCount)
	withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod,
		addRoutes, updRoutes, delRoutes, aggDest, withdrawn, updated, updatedAddPaths)
	if action == RouteActionAdd || action == RouteActionReplace {
		dest.aggPath = aggPath
	}
	if action == RouteActionDelete && aggDest.IsEmpty() {
		delete(adjRib.destPathMap, aggIP.Prefix.String())
	}

	return updated, withdrawn, remPath, updatedAddPaths
}

func (adjRib *AdjRib) AddRouteToAggregate(ip *packet.IPPrefix, aggIP *packet.IPPrefix,
	srcIP string, ifaceIP net.IP,
	bgpAgg *config.BGPAggregate, addPathCount int) (map[*Path][]*Destination,
	[]*Destination, *Path, []*Destination) {
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
		adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - aggIP %v found in",
			"dest, agg path %v", aggIP, aggPath))
	}

	if aggPath != nil {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - aggIP %v,",
			"agg path found, update path attrs", aggIP))
		aggPath.addPathToAggregate(ip.Prefix.String(), path, bgpAgg.GenerateASSet)
		aggDest.setUpdateAggPath(srcIP, AggregatePathId)
		aggDest.addAggregatedDests(ip.Prefix.String(), dest)
		action, addPathsMod, addRoutes, updRoutes, delRoutes :=
			aggDest.SelectRouteForLocRib(addPathCount)
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod,
			addRoutes, updRoutes, delRoutes, aggDest, withdrawn, updated, updatedAddPaths)
		if action == RouteActionAdd || action == RouteActionReplace {
			dest.aggPath = aggPath
		}
	} else {
		adjRib.logger.Info(fmt.Sprintf("AdjRib:AddRouteToAggregate - aggIP %v,",
			"agg path NOT found, create new path", aggIP))
		pathAttrs := packet.ConstructPathAttrForAggRoutes(path.PathAttrs, bgpAgg.GenerateASSet)
		packet.SetNextHopPathAttrs(pathAttrs, ifaceIP)
		packet.SetPathAttrAggregator(pathAttrs, adjRib.gConf.AS, adjRib.gConf.RouterId)
		aggPath = NewPath(path.rib, nil, pathAttrs, false, true, RouteTypeAgg)
		aggPath.setAggregatedPath(ip.Prefix.String(), path)
		aggDest, _ := adjRib.GetDest(aggIP, true)
		aggDest.AddOrUpdatePath(srcIP, AggregatePathId, aggPath)
		aggDest.addAggregatedDests(ip.Prefix.String(), dest)
		action, addPathsMod, addRoutes, updRoutes, delRoutes :=
			aggDest.SelectRouteForLocRib(addPathCount)
		withdrawn, updated, updatedAddPaths = adjRib.updateRibOutInfo(action, addPathsMod,
			addRoutes, updRoutes, delRoutes, aggDest, withdrawn, updated, updatedAddPaths)
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
			adjRib.logger.Info(fmt.Sprintln(
				"removeRoutesFromRouteList: remove route at idx", idx))
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
		adjRib.logger.Info(fmt.Sprintln("addRoutesToRouteList: added route at idx",
			len(adjRib.routeList)-1))
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
			for modIdx = lastIdx; modIdx > idx &&
				adjRib.routeList[modIdx] == nil; modIdx-- {
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

func (adjRib *AdjRib) GetBGPRoute(prefix string) *bgpd.BGPRouteState {
	defer adjRib.routeMutex.RUnlock()
	adjRib.routeMutex.RLock()

	if dest, ok := adjRib.destPathMap[prefix]; ok {
		return dest.GetBGPRoute()
	}

	return nil
}

func (adjRib *AdjRib) BulkGetBGPRoutes(index int, count int) (int, int, []*bgpd.BGPRouteState) {
	adjRib.timer.Stop()
	if index == 0 && adjRib.activeGet {
		adjRib.ResetRouteList()
	}
	adjRib.activeGet = true

	defer adjRib.routeMutex.RUnlock()
	adjRib.routeMutex.RLock()

	var i int
	n := 0
	result := make([]*bgpd.BGPRouteState, count)
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
