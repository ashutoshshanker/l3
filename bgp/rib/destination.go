// destination.go
package server

import (
	"bgpd"
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"l3/rib/ribdCommonDefs"
	"math"
	"net"
	"ribd"
	"ribdInt"
	"sort"
	"strconv"
	"utils/logging"
)

const BGP_INTERNAL_PREF = 100
const BGP_EXTERNAL_PREF = 50

type PathAndRoute struct {
	Path
}

type Destination struct {
	rib               *AdjRib
	logger            *logging.Writer
	gConf             *config.GlobalConfig
	IPPrefix          *packet.IPPrefix
	peerPathMap       map[string]map[uint32]*Path
	LocRibPath        *Path
	LocRibPathRoute   *Route
	aggPath           *Path
	aggregatedDestMap map[string]*Destination
	ecmpPaths         map[*Path]*Route
	pathRouteMap      map[*Path]*Route
	AddPaths          []*Path
	maxPathId         uint32
	pathIds           []uint32
	recalculate       bool
}

func NewDestination(rib *AdjRib, ipPrefix *packet.IPPrefix, gConf *config.GlobalConfig) *Destination {
	dest := &Destination{
		rib:               rib,
		logger:            rib.logger,
		gConf:             gConf,
		IPPrefix:          ipPrefix,
		peerPathMap:       make(map[string]map[uint32]*Path),
		ecmpPaths:         make(map[*Path]*Route),
		aggregatedDestMap: make(map[string]*Destination),
		pathRouteMap:      make(map[*Path]*Route),
		AddPaths:          make([]*Path, 0),
		maxPathId:         1,
		pathIds:           make([]uint32, 0),
	}

	return dest
}

func (d *Destination) GetLocRibPathRoute() *Route {
	d.logger.Info(fmt.Sprintf("GetLocRibPathRoute for %s\n", d.IPPrefix.Prefix.String()))
	return d.LocRibPathRoute
}

func (d *Destination) GetBGPRoute() (route *bgpd.BGPRouteState) {
	if d.LocRibPathRoute != nil {
		route = d.LocRibPathRoute.GetBGPRoute()
	}

	return route
}

func (d *Destination) GetPathRoute(path *Path) *Route {
	if route, ok := d.pathRouteMap[path]; ok {
		return route
	}

	return nil
}

func (d *Destination) IsEmpty() bool {
	return len(d.peerPathMap) == 0
}

func (d *Destination) getNextPathId() uint32 {
	var pathId uint32
	if len(d.pathIds) > 0 {
		pathId = d.pathIds[len(d.pathIds)-1]
		d.pathIds = d.pathIds[:len(d.pathIds)-1]
		return pathId
	}

	pathId = d.maxPathId
	d.maxPathId++
	return pathId
}

func (d *Destination) releasePathId(pathId uint32) {
	if pathId+1 == d.maxPathId {
		d.maxPathId--
		return
	}

	d.pathIds = append(d.pathIds, pathId)
}

func (d *Destination) updateAddPaths(addPaths []*Path) (modified bool) {
	if len(d.AddPaths) != len(addPaths) {
		modified = true
	} else {
		for i := 0; i < len(d.AddPaths); i++ {
			if d.AddPaths[i] != addPaths[i] {
				modified = true
			}
		}
	}
	for i := 0; i < len(d.AddPaths); i++ {
		d.AddPaths[i] = nil
	}
	d.AddPaths = addPaths
	return modified
}

func (d *Destination) getPathForIP(peerIP string, pathId uint32) (path *Path) {
	if pathMap, ok := d.peerPathMap[peerIP]; ok {
		path = pathMap[pathId]
	}
	return path
}

func (d *Destination) getPathIdForPath(path *Path) (uint32, bool) {
	for _, pathMap := range d.peerPathMap {
		for pathId, peerPath := range pathMap {
			if path == peerPath {
				return pathId, true
			}
		}
	}

	d.logger.Err(fmt.Sprintf("Destination:getPathIdForPath - path id not found for path %v\n", path))
	return 0, false
}

func (d *Destination) setUpdateAggPath(peerIP string, pathId uint32) {
	pathMap, ok := d.peerPathMap[peerIP]
	if !ok {
		d.logger.Err(fmt.Sprintf("Destination:setUpdateAggPath - peer ip %s not found in peer path map\n", peerIP))
	} else {
		path, ok := pathMap[pathId]
		if !ok {
			d.logger.Err(fmt.Sprintf("Destination:setUpdateAggPath - pathId %d not found in peer %s path map\n",
				pathId, peerIP))
		} else if d.LocRibPath == nil || path == d.LocRibPath ||
			getRouteSource(d.LocRibPath.routeType) >= getRouteSource(path.routeType) {
			d.recalculate = true
		}
	}

	if d.LocRibPath == nil {
		d.recalculate = true
	}
}

func (d *Destination) setAggPath(path *Path) {
	d.aggPath = path
}

func (d *Destination) addAggregatedDests(peerIP string, dest *Destination) {
	d.aggregatedDestMap[peerIP] = dest
}

func (d *Destination) removeAggregatedDests(peerIP string) {
	delete(d.aggregatedDestMap, peerIP)
}

func (d *Destination) AddOrUpdatePath(peerIp string, pathId uint32, path *Path) bool {
	var pathMap map[uint32]*Path
	added := false
	ok := false

	if pathMap, ok = d.peerPathMap[peerIp]; !ok {
		d.peerPathMap[peerIp] = make(map[uint32]*Path)
	}

	if oldPath, ok := pathMap[pathId]; ok {
		d.logger.Info(fmt.Sprintf("Update path for %s from %s, path id %d\n", d.IPPrefix.Prefix.String(), peerIp, pathId))
		if d.LocRibPath == oldPath {
			d.LocRibPath = path
			d.recalculate = true
		}
	} else {
		d.logger.Info(fmt.Sprintf("Add new path for %s from %s, path id %d\n", d.IPPrefix.Prefix.String(), peerIp, pathId))
		added = true
	}

	if d.LocRibPath == nil || getRouteSource(d.LocRibPath.routeType) >= getRouteSource(path.routeType) {
		d.recalculate = true
	}

	//if added {
	outPathId := d.getNextPathId()
	route := NewRoute(d, path, RouteActionNone, pathId, outPathId)
	d.pathRouteMap[path] = route
	//}
	d.peerPathMap[peerIp][pathId] = path
	return added
}

func (d *Destination) RemovePath(peerIP string, pathId uint32, path *Path) *Path {
	var pathMap map[uint32]*Path
	var oldPath *Path
	ok := false
	if pathMap, ok = d.peerPathMap[peerIP]; !ok {
		d.logger.Err(fmt.Sprintln("Can't remove path", d.IPPrefix.Prefix.String(), "Path not found from peer", peerIP))
		return oldPath
	}

	if oldPath, ok = pathMap[pathId]; ok {
		for ecmpPath, _ := range d.ecmpPaths {
			if ecmpPath == oldPath {
				d.recalculate = true
				d.LocRibPath = path
			}
		}

		if d.LocRibPath == oldPath {
			d.recalculate = true
			d.LocRibPath = path
		}

		route := d.pathRouteMap[oldPath]
		d.releasePathId(route.OutPathId)
		delete(d.pathRouteMap, oldPath)
		delete(d.peerPathMap[peerIP], pathId)
		if len(d.peerPathMap[peerIP]) == 0 {
			delete(d.peerPathMap, peerIP)
		}
	} else {
		d.logger.Err(fmt.Sprintln("Can't remove path", d.IPPrefix.Prefix.String(), "Path with path id", pathId,
			"not found from peer", peerIP))
	}
	return oldPath
}

func (d *Destination) RemoveAllPaths(peerIP string, path *Path) {
	var pathMap map[uint32]*Path
	ok := false
	if pathMap, ok = d.peerPathMap[peerIP]; !ok {
		d.logger.Err(fmt.Sprintln("Can't remove paths for", d.IPPrefix.Prefix.String(), "peer not found in map", peerIP))
		return
	}

	d.logger.Info(fmt.Sprintln("Remove all paths for", d.IPPrefix.Prefix.String(), "from peer", peerIP))
	for pathId, _ := range pathMap {
		d.logger.Err(fmt.Sprintln("Remove path id", pathId, "from peer", peerIP))
		d.RemovePath(peerIP, pathId, path)
	}
}

func (d *Destination) RemoveAllNeighborPaths() {
	for peerIP, pathMap := range d.peerPathMap {
		for pathId, path := range pathMap {
			if path.NeighborConf != nil {
				delete(d.peerPathMap[peerIP], pathId)
				if len(d.peerPathMap[peerIP]) == 0 {
					delete(d.peerPathMap, peerIP)
				}
			}
		}
	}

	if d.LocRibPath != nil {
		if d.LocRibPath.NeighborConf != nil {
			d.recalculate = true
			d.LocRibPath = nil
		}
	}
}

func constructNetmaskFromLen(ones, bits int) net.IP {
	ip := make(net.IP, bits/8)
	bytes := ones / 8
	i := 0
	for ; i < bytes; i++ {
		ip[i] = 255
	}
	rem := ones % 8
	if rem != 0 {
		ip[i] = (255 << uint(8-rem))
	}
	return ip
}

func (d *Destination) removeAndPrepend(pathsList *[][]*Path, item *Path) {
	idx := 0
	found := false
	var paths []*Path

	for idx, paths = range *pathsList {
		var path *Path
		pathIdx := 0
		for pathIdx, path = range paths {
			if path == item {
				found = true
				break
			}
		}
		if found {
			copy(paths[1:pathIdx+1], paths[:pathIdx])
			paths[0] = path
			break
		}
	}

	if !found {
		paths = make([]*Path, 1)
		paths[0] = item
		*pathsList = append(*pathsList, paths)
	}
	copy((*pathsList)[1:idx+1], (*pathsList)[:idx])
	(*pathsList)[0] = paths
}

func (d *Destination) SelectRouteForLocRib(addPathCount int) (RouteAction, bool, []*Route, []*Route, []*Route) {
	updatedPaths := make([]*Path, 0)
	removedPaths := make([]*Path, 0)
	addedRoutes := make([]*Route, 0)
	updatedRoutes := make([]*Route, 0)
	deletedRoutes := make([]*Route, 0)
	maxPref := uint32(0)
	routeSrc := RouteSrcUnknown
	locRibAction := RouteActionNone
	addPathsUpdated := false

	d.logger.Info(fmt.Sprintf("Destination:SelectRouteForLocalRib - network %v, peer path map = %v\n", d.IPPrefix.Prefix.String(), d.peerPathMap))
	if !d.recalculate {
		return locRibAction, addPathsUpdated, addedRoutes, updatedRoutes, deletedRoutes
	}
	d.recalculate = false

	if d.LocRibPath != nil && !d.LocRibPath.IsWithdrawn() && !d.LocRibPath.IsUpdated() {
		peerIP := d.gConf.RouterId.String()
		if d.LocRibPath.NeighborConf != nil {
			peerIP = d.LocRibPath.NeighborConf.Neighbor.NeighborAddress.String()
		}
		routeSrc = getRouteSource(d.LocRibPath.routeType)
		maxPref = d.LocRibPath.GetPreference()
		updatedPaths = append(updatedPaths, d.LocRibPath)
		d.logger.Info(fmt.Sprintf("Add loc rib path from %s to the list of selected paths, pref=%d\n", peerIP, maxPref))
	}

	for peerIP, pathMap := range d.peerPathMap {
		for _, path := range pathMap {
			if path.IsUpdated() || (d.LocRibPath != nil && (d.LocRibPath.IsWithdrawn() || d.LocRibPath.IsUpdated())) {
				if !path.IsLocal() && !path.IsReachable() {
					d.logger.Info(fmt.Sprintf("peer %s, NEXT_HOP[%s] is not reachable\n", peerIP, path.GetNextHop()))
					continue
				}

				if path.HasASLoop() {
					d.logger.Info(fmt.Sprintf("This path has AS loop [%d], removing this path from the selection process\n",
						path.NeighborConf.RunningConf.LocalAS))
					continue
				}

				currPathSource := getRouteSource(path.routeType)
				if currPathSource > routeSrc {
					removedPaths = append(removedPaths, path)
					continue
				} else if currPathSource < routeSrc {
					if len(updatedPaths) > 0 {
						removedPaths = append(removedPaths, updatedPaths...)
						updatedPaths[0] = path
						// For garbage collection
						for i := 0; i < len(updatedPaths); i++ {
							updatedPaths[i] = nil
						}
						updatedPaths = updatedPaths[:1]
					} else {
						updatedPaths = append(updatedPaths, path)
					}
					d.logger.Info(fmt.Sprintf("route from %s is from a better source type, old type=%d, new type=%d, pref=%d\n",
						peerIP, routeSrc, currPathSource, path.GetPreference()))
					routeSrc = currPathSource
					maxPref = path.GetPreference()
					continue
				}

				currPref := path.GetPreference()
				if currPref < maxPref {
					removedPaths = append(removedPaths, path)
				} else if currPref > maxPref {
					if len(updatedPaths) > 0 {
						removedPaths = append(removedPaths, updatedPaths...)
						updatedPaths[0] = path
						// For garbage collection
						for i := 1; i < len(updatedPaths); i++ {
							updatedPaths[i] = nil
						}
						updatedPaths = updatedPaths[:1]
					} else {
						updatedPaths = append(updatedPaths, path)
					}
					d.logger.Info(fmt.Sprintf("route from %s has more preference, old pref=%d, new pref=%d\n",
						peerIP, maxPref, currPref))
					maxPref = currPref
				} else if currPref == maxPref {
					d.logger.Info(fmt.Sprintf("route from %s has same preference, add to the list, pref=%d\n",
						peerIP, maxPref))
					updatedPaths = append(updatedPaths, path)
				}
			}
		}
	}

	d.logger.Info(fmt.Sprintln("Destination =", d.IPPrefix.Prefix.String(), "ECMP routes =", d.ecmpPaths, "updated paths =", updatedPaths))
	if len(updatedPaths) > 0 {
		var ecmpPaths [][]*Path
		var addPaths []*Path
		if len(updatedPaths) > 1 || (addPathCount > 0) {
			d.logger.Info(fmt.Sprintf("Found multiple paths with same pref, run path selection algorithm\n"))
			if d.gConf.UseMultiplePaths {
				updatedPaths, ecmpPaths, addPaths = d.calculateBestPath(updatedPaths, removedPaths,
					d.gConf.EBGPMaxPaths > 1,
					d.gConf.IBGPMaxPaths > 1, addPathCount)
			} else {
				updatedPaths, ecmpPaths, addPaths = d.calculateBestPath(updatedPaths, removedPaths, false, false,
					addPathCount)
			}
		}

		if len(updatedPaths) > 1 {
			d.logger.Err(fmt.Sprintf("Have more than one route after the tie breaking rules... using the first one, routes[%s]\n", updatedPaths))
		}

		addPathsUpdated = d.updateAddPaths(addPaths)
		d.removeAndPrepend(&ecmpPaths, updatedPaths[0])
		d.logger.Info(fmt.Sprintln("ecmpPaths =", ecmpPaths))

		for idx, paths := range ecmpPaths {
			found := false
			for _, path := range paths {
				if route, ok := d.ecmpPaths[path]; ok {
					// Update path
					found = true
					if (path.IsAggregate() || !path.IsLocal()) && path.IsUpdated() {
						d.logger.Info(fmt.Sprintf("Update route for ip=%s\n", d.IPPrefix.Prefix.String()))
						d.updateRoute(path)
						route.update()
					}

					if (idx == 0) && (path.IsAggregate() || (d.LocRibPath != path)) {
						locRibAction = RouteActionReplace
					}
					updatedRoutes = append(updatedRoutes, route)
					route.setAction(RouteActionReplace)
					break
				}
			}

			if !found {
				// Add route
				newRoute := d.pathRouteMap[paths[0]]
				//newRoute := NewRoute(d, paths[0], RouteActionAdd)
				if newRoute == nil {
					continue
				}
				newRoute.setAction(RouteActionAdd)

				if paths[0].IsAggregate() || !paths[0].IsLocal() {
					d.logger.Info(fmt.Sprintf("Add route for ip=%s, mask=%s, next hop=%s\n", d.IPPrefix.Prefix.String(),
						constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(), paths[0].reachabilityInfo.NextHop))
					protocol := "IBGP"
					if paths[0].IsExternal() {
						protocol = "EBGP"
					}
					nextHopIfTypeStr, _ := ribdCommonDefs.GetNextHopIfTypeStr(ribdInt.Int(paths[0].reachabilityInfo.NextHopIfType))
					cfg := ribd.IPv4Route{
						DestinationNw:     d.IPPrefix.Prefix.String(),
						Protocol:          protocol,
						OutgoingInterface: strconv.Itoa(int(paths[0].reachabilityInfo.NextHopIfIdx)),
						OutgoingIntfType:  nextHopIfTypeStr,
						Cost:              int32(paths[0].reachabilityInfo.Metric),
						NetworkMask:       constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(),
						NextHopIp:         paths[0].reachabilityInfo.NextHop}
					d.rib.ribdClient.OnewayCreateIPv4Route(&cfg)
					//d.rib.routeBulkCreateChannel <- cfg
				}
				if idx == 0 {
					locRibAction = RouteActionAdd
				}
				d.ecmpPaths[paths[0]] = newRoute
				addedRoutes = append(addedRoutes, newRoute)
			}
		}

		d.LocRibPath = ecmpPaths[0][0]
		d.LocRibPathRoute = d.ecmpPaths[d.LocRibPath]
	} else {
		if d.LocRibPath != nil {
			// Remove route
			for path, route := range d.ecmpPaths {
				route.setAction(RouteActionDelete)
				if path.IsAggregate() || !path.IsLocal() {
					d.logger.Info(fmt.Sprintf("Remove route for ip=%s nexthop=%s\n", d.IPPrefix.Prefix.String(),
						path.reachabilityInfo.NextHop))
					protocol := "IBGP"
					if path.IsExternal() {
						protocol = "EBGP"
					}
					cfg := ribd.IPv4Route{
						DestinationNw:     d.IPPrefix.Prefix.String(),
						Protocol:          protocol,
						OutgoingInterface: strconv.Itoa(int(path.reachabilityInfo.NextHopIfIdx)),
						OutgoingIntfType:  "",
						Cost:              int32(path.reachabilityInfo.Metric),
						NetworkMask:       constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(),
						NextHopIp:         path.reachabilityInfo.NextHop}

					d.rib.ribdClient.OnewayDeleteIPv4Route(&cfg)
					d.logger.Info(fmt.Sprintf("DeleteV4Route for ip=%s nexthop=%s DONE\n", d.IPPrefix.Prefix.String(),
						path.reachabilityInfo.NextHop))
				}
			}
			locRibAction = RouteActionDelete
			d.LocRibPath = nil
		}
	}

	for path, route := range d.ecmpPaths {
		if route.action == RouteActionNone || route.action == RouteActionDelete {
			if path.IsAggregate() || !path.IsLocal() {
				d.logger.Info(fmt.Sprintln("Remove route from ECMP paths, route =", route, "ip =",
					d.IPPrefix.Prefix.String(), "next hop =", path.reachabilityInfo.NextHop))
				protocol := "IBGP"
				if path.IsExternal() {
					protocol = "EBGP"
				}
				cfg := ribd.IPv4Route{
					DestinationNw:     d.IPPrefix.Prefix.String(),
					Protocol:          protocol,
					OutgoingInterface: strconv.Itoa(int(path.reachabilityInfo.NextHopIfIdx)),
					OutgoingIntfType:  "",
					Cost:              int32(path.reachabilityInfo.Metric),
					NetworkMask:       constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(),
					NextHopIp:         path.reachabilityInfo.NextHop}

				d.rib.ribdClient.OnewayDeleteIPv4Route(&cfg)
				d.logger.Info(fmt.Sprintln("DeleteV4Route from ECMP paths, route =", route, "ip =",
					d.IPPrefix.Prefix.String(), "next hop =", path.reachabilityInfo.NextHop, "DONE"))
			}
			deletedRoutes = append(deletedRoutes, route)
			delete(d.ecmpPaths, path)
		} else {
			route.setAction(RouteActionNone)
		}
	}
	return locRibAction, addPathsUpdated, addedRoutes, updatedRoutes, deletedRoutes
}

func (d *Destination) updateRoute(path *Path) {
	d.logger.Info(fmt.Sprintf("Remove route for ip=%s, mask=%s\n", d.IPPrefix.Prefix.String(),
		constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String()))
	protocol := "IBGP"
	if path.IsExternal() {
		protocol = "EBGP"
	}
	cfg := ribd.IPv4Route{
		DestinationNw:     d.IPPrefix.Prefix.String(),
		Protocol:          protocol,
		OutgoingInterface: strconv.Itoa(int(path.reachabilityInfo.NextHopIfIdx)),
		OutgoingIntfType:  "",
		Cost:              int32(path.reachabilityInfo.Metric),
		NetworkMask:       constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(),
		NextHopIp:         path.reachabilityInfo.NextHop}

	d.rib.ribdClient.OnewayDeleteIPv4Route(&cfg)

	if path.IsAggregate() || !path.IsLocal() {
		var nextHop string
		if path.IsAggregate() {
			nextHop = "255.255.255.255"
		} else {
			nextHop = path.reachabilityInfo.NextHop
		}

		d.logger.Info(fmt.Sprintf("Add route for ip=%s, mask=%s, next hop=%s\n", d.IPPrefix.Prefix.String(),
			constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(), nextHop))
		nextHopIfTypeStr, _ := ribdCommonDefs.GetNextHopIfTypeStr(ribdInt.Int(path.reachabilityInfo.NextHopIfType))
		cfg := ribd.IPv4Route{
			DestinationNw:     d.IPPrefix.Prefix.String(),
			Protocol:          protocol,
			OutgoingInterface: strconv.Itoa(int(path.reachabilityInfo.NextHopIfIdx)),
			OutgoingIntfType:  nextHopIfTypeStr,
			Cost:              int32(path.reachabilityInfo.Metric),
			NetworkMask:       constructNetmaskFromLen(int(d.IPPrefix.Length), 32).String(),
			NextHopIp:         nextHop}

		d.rib.ribdClient.OnewayCreateIPv4Route(&cfg)
		//d.rib.routeBulkCreateChannel <- cfg
	}
}

func (d *Destination) getRoutesWithSmallestAS(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path,
	[]PathSortIface) {
	minASNums := uint32(4096)
	removedPaths := make([]*Path, 0)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		d.logger.Info(fmt.Sprintln("Destination:getRoutesWithSmallestAS - get num ASes from path", updatedPaths[i]))
		asNums := updatedPaths[i].GetNumASes()
		from := ""
		if updatedPaths[i].NeighborConf != nil {
			from = updatedPaths[i].NeighborConf.Neighbor.NeighborAddress.String()
		}
		d.logger.Info(fmt.Sprintln("Destination:getRoutesWithSmallestAS - Dest =", d.IPPrefix.Prefix, "number of ASes =",
			asNums, "from", from))
		if asNums > minASNums {
			removedPaths = append(removedPaths, updatedPaths[i])
		} else if asNums < minASNums {
			removedPaths = append(removedPaths, updatedPaths[:idx]...)
			minASNums = asNums
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if asNums == minASNums {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if len(removedPaths) > 0 {
		pathSortIface := PathSortIface{
			paths: removedPaths,
			iface: BySmallestAS{removedPaths},
		}
		prunedPaths = append(prunedPaths, pathSortIface)
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		updatedPaths = updatedPaths[:idx]
	}

	return updatedPaths, prunedPaths
}

func (d *Destination) getRoutesWithLowestOrigin(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path,
	[]PathSortIface) {
	minOrigin := uint8(packet.BGPPathAttrOriginMax)
	removedPaths := make([]*Path, 0)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		origin := updatedPaths[i].GetOrigin()
		if origin > minOrigin {
			removedPaths = append(removedPaths, updatedPaths[i])
		} else if origin < minOrigin {
			removedPaths = append(removedPaths, updatedPaths[:idx]...)
			minOrigin = origin
			updatedPaths[0] = updatedPaths[i]
			idx++
		} else if origin == minOrigin {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if len(removedPaths) > 0 {
		pathSortIface := PathSortIface{
			paths: removedPaths,
			iface: ByLowestOrigin{removedPaths},
		}
		prunedPaths = append(prunedPaths, pathSortIface)
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		updatedPaths = updatedPaths[:idx]
	}

	return updatedPaths, prunedPaths
}

func deleteIBGPRoutes(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path, []PathSortIface) {
	removedPaths := make([]*Path, 0)
	n := len(updatedPaths) - 1
	i := 0

	for i <= n {
		if updatedPaths[i].NeighborConf.IsInternal() {
			removedPaths = append(removedPaths, updatedPaths[i])
			updatedPaths[i] = updatedPaths[n]
			updatedPaths[n] = nil
			n--
			continue
		}
		i++
	}

	if len(removedPaths) > 0 {
		pathSortIface := PathSortIface{
			paths: removedPaths,
			iface: ByIBGPOrEBGPRoutes{removedPaths},
		}
		prunedPaths = append(prunedPaths, pathSortIface)
	}

	return updatedPaths[:i], prunedPaths
}

func (d *Destination) removeIBGPRoutesIfEBGPExist(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path,
	[]PathSortIface) {
	for _, path := range updatedPaths {
		if path.NeighborConf != nil && path.NeighborConf.IsExternal() {
			return deleteIBGPRoutes(updatedPaths, prunedPaths)
		}
	}

	return updatedPaths, prunedPaths
}

func (d *Destination) isEBGPRoute(path *Path) bool {
	if path.NeighborConf != nil && path.NeighborConf.IsExternal() {
		return true
	}

	return false
}

func (d *Destination) isIBGPRoute(path *Path) bool {
	if path.NeighborConf != nil && path.NeighborConf.IsInternal() {
		return true
	}

	return false
}

func (d *Destination) getRoutesWithLowestBGPId(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path,
	[]PathSortIface) {
	removedPaths := make([]*Path, 0)
	n := len(updatedPaths)
	lowestBGPId := uint32(math.MaxUint32)
	idx := 0

	for i := 0; i < n; i++ {
		bgpId := updatedPaths[i].GetBGPId()
		if bgpId > lowestBGPId {
			removedPaths = append(removedPaths, updatedPaths[i])
		} else if bgpId < lowestBGPId {
			removedPaths = append(removedPaths, updatedPaths[:idx]...)
			lowestBGPId = bgpId
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if bgpId == lowestBGPId {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if len(removedPaths) > 0 {
		pathSortIface := PathSortIface{
			paths: removedPaths,
			iface: ByLowestBGPId{removedPaths},
		}
		prunedPaths = append(prunedPaths, pathSortIface)
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		updatedPaths = updatedPaths[:idx]
	}

	return updatedPaths, prunedPaths
}

func (d *Destination) getRoutesWithShorterClusterLen(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path,
	[]PathSortIface) {
	removedPaths := make([]*Path, 0)
	minClusterLen := uint16(math.MaxUint16)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		clusterLen := updatedPaths[i].GetNumClusters()
		if clusterLen > minClusterLen {
			removedPaths = append(removedPaths, updatedPaths[i])
		} else if clusterLen < minClusterLen {
			removedPaths = append(removedPaths, updatedPaths[:idx]...)
			minClusterLen = clusterLen
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if clusterLen == minClusterLen {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if len(removedPaths) > 0 {
		pathSortIface := PathSortIface{
			paths: removedPaths,
			iface: ByShorterClusterLen{removedPaths},
		}
		prunedPaths = append(prunedPaths, pathSortIface)
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		updatedPaths = updatedPaths[:idx]
	}

	return updatedPaths, prunedPaths
}

func CompareNeighborAddress(a net.IP, b net.IP) (int, error) {
	if len(a) != len(b) {
		return 0, config.AddressError{fmt.Sprintf("Address lenghts not equal, Neighbor Address: %s, compare address: %s",
			a.String(), b.String())}
	}

	for i, val := range a {
		if val < b[i] {
			return -1, nil
		} else if val > b[i] {
			return 1, nil
		}
	}

	return 0, nil
}

func (d *Destination) getRoutesWithLowestPeerAddress(updatedPaths []*Path, prunedPaths []PathSortIface) ([]*Path,
	[]PathSortIface) {
	removedPaths := make([]*Path, 0)
	n := len(updatedPaths)
	idx := 0

	for i, path := range updatedPaths {
		val, err := CompareNeighborAddress(path.NeighborConf.Neighbor.NeighborAddress,
			updatedPaths[0].NeighborConf.Neighbor.NeighborAddress)
		if err != nil {
			d.logger.Err(fmt.Sprintf("CompareNeighborAddress failed with %s", err))
		}

		if val > 0 {
			removedPaths = append(removedPaths, updatedPaths[i])
		} else if val < 0 {
			removedPaths = append(removedPaths, updatedPaths[:idx]...)
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if val == 0 {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if len(removedPaths) > 0 {
		pathSortIface := PathSortIface{
			paths: removedPaths,
			iface: ByLowestPeerAddress{removedPaths},
		}
		prunedPaths = append(prunedPaths, pathSortIface)
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		updatedPaths = updatedPaths[:idx]
	}

	return updatedPaths, prunedPaths
}

func (d *Destination) getECMPPaths(updatedPaths []*Path) [][]*Path {
	ecmpPathMap := make(map[string][]*Path)

	for _, path := range updatedPaths {
		if _, ok := ecmpPathMap[path.reachabilityInfo.NextHop]; !ok {
			ecmpPathMap[path.reachabilityInfo.NextHop] = make([]*Path, 1)
			ecmpPathMap[path.reachabilityInfo.NextHop][0] = path
		} else {
			ecmpPathMap[path.reachabilityInfo.NextHop] = append(ecmpPathMap[path.reachabilityInfo.NextHop], path)
		}
	}

	d.logger.Info(fmt.Sprintln("getECMPPaths: update paths =", updatedPaths, "ecmpPathsMap =", ecmpPathMap))
	ecmpPaths := make([][]*Path, 0)
	for _, paths := range ecmpPathMap {
		ecmpPaths = append(ecmpPaths, paths)
	}
	return ecmpPaths
}

func (d *Destination) addAddPaths(addPaths, currPaths []*Path, pathMap map[string]*Path) ([]*Path, map[string]*Path) {
	currPathMap := make(map[string]*Path)
	for _, path := range currPaths {
		if _, ok := pathMap[path.reachabilityInfo.NextHop]; !ok {
			currPathMap[path.reachabilityInfo.NextHop] = path
			pathMap[path.reachabilityInfo.NextHop] = path
		}
	}

	d.logger.Info(fmt.Sprintln("getAddPaths: add paths =", addPaths, "pathMap =", pathMap))
	for _, path := range currPathMap {
		addPaths = append(addPaths, path)
	}
	return addPaths, pathMap
}

func (d *Destination) calculateBestPath(updatedPaths, removedPaths []*Path, ebgpMultiPath, ibgpMultiPath bool,
	addPathCount int) ([]*Path, [][]*Path, []*Path) {
	var ecmpPaths [][]*Path
	prunedPaths := make([]PathSortIface, 0)
	pathSortIface := PathSortIface{
		paths: removedPaths,
		iface: ByPref{removedPaths},
	}
	prunedPaths = append(prunedPaths, pathSortIface)

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling getRoutesWithSmallestAS, update paths =", updatedPaths))
		updatedPaths, prunedPaths = d.getRoutesWithSmallestAS(updatedPaths, prunedPaths)
	}

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling getRoutesWithLowestOrigin, update paths =", updatedPaths))
		updatedPaths, prunedPaths = d.getRoutesWithLowestOrigin(updatedPaths, prunedPaths)
	}

	if (len(updatedPaths) > 1) && ebgpMultiPath && ibgpMultiPath {
		ecmpPaths = d.getECMPPaths(updatedPaths)
		d.logger.Info(fmt.Sprintln("calculateBestPath: IBGP & EBGP multi paths =", ecmpPaths))
	}

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling removeIBGPRoutesIfEBGPExist, update paths =", updatedPaths))
		updatedPaths, prunedPaths = d.removeIBGPRoutesIfEBGPExist(updatedPaths, prunedPaths)
	}

	if len(updatedPaths) > 1 && ibgpMultiPath != ebgpMultiPath {
		if ebgpMultiPath && d.isEBGPRoute(updatedPaths[0]) {
			ecmpPaths = d.getECMPPaths(updatedPaths)
			d.logger.Info(fmt.Sprintf("calculateBestPath: EBGP multi paths =", ecmpPaths))
		} else if ibgpMultiPath && d.isIBGPRoute(updatedPaths[0]) {
			ecmpPaths = d.getECMPPaths(updatedPaths)
			d.logger.Info(fmt.Sprintf("calculateBestPath: IBGP multi paths =", ecmpPaths))
		}
	}

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling getRoutesWithLowestBGPId, update paths =", updatedPaths))
		updatedPaths, prunedPaths = d.getRoutesWithLowestBGPId(updatedPaths, prunedPaths)
	}

	if len(updatedPaths) > 1 {
		d.logger.Info("calling getRoutesWithShorterClusterLen")
		updatedPaths, prunedPaths = d.getRoutesWithShorterClusterLen(updatedPaths, prunedPaths)
	}

	if len(updatedPaths) > 1 {
		d.logger.Info("calling getRoutesWithLowestPeerAddress")
		updatedPaths, prunedPaths = d.getRoutesWithLowestPeerAddress(updatedPaths, prunedPaths)
	}

	pathMap := make(map[string]*Path)
	addPaths := make([]*Path, 0)
	if len(addPaths) < addPathCount && len(updatedPaths) > 1 {
		addPaths, pathMap = d.addAddPaths(addPaths, updatedPaths[1:], pathMap)
	}

	if len(addPaths) < addPathCount {
		for i := len(prunedPaths) - 1; i >= 0; i-- {
			sort.Sort(prunedPaths[i].iface)
			currPaths := prunedPaths[i].paths
			addPaths, pathMap = d.addAddPaths(addPaths, currPaths, pathMap)
			if len(addPaths) >= addPathCount {
				for idx := addPathCount; idx < len(addPaths); idx++ {
					addPaths[idx] = nil
				}
				addPaths = addPaths[:addPathCount]
				break
			}
		}
	}
	return updatedPaths, ecmpPaths, addPaths
}
