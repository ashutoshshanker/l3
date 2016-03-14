// destination.go
package server

import (
	"bgpd"
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"math"
	"net"
	"utils/logging"
)

const BGP_INTERNAL_PREF = 100
const BGP_EXTERNAL_PREF = 50

type Destination struct {
	server            *BGPServer
	logger            *logging.Writer
	nlri              packet.IPPrefix
	peerPathMap       map[string]*Path
	locRibPath        *Path
	locRibPathRoute   *Route
	aggPath           *Path
	aggregatedDestMap map[string]*Destination
	ecmpPaths         map[*Path]*Route
	recalculate       bool
}

func NewDestination(server *BGPServer, nlri packet.IPPrefix) *Destination {
	dest := &Destination{
		server:            server,
		logger:            server.logger,
		nlri:              nlri,
		peerPathMap:       make(map[string]*Path),
		ecmpPaths:         make(map[*Path]*Route),
		aggregatedDestMap: make(map[string]*Destination),
	}

	return dest
}

func (d *Destination) GetLocRibPathRoute() *bgpd.BGPRoute {
	d.logger.Info(fmt.Sprintf("GetLocRibPathRoute for %s", d.nlri.Prefix.String()))
	return d.locRibPathRoute.GetBGPRoute()
}

func (d *Destination) GetBGPRoutes() []*bgpd.BGPRoute {
	routes := make([]*bgpd.BGPRoute, 0)
	for _, route := range d.ecmpPaths {
		routes = append(routes, route.GetBGPRoute())
	}

	return routes
}

func (d *Destination) IsEmpty() bool {
	return len(d.peerPathMap) == 0
}

func (d *Destination) getPathForIP(peerIP string) *Path {
	return d.peerPathMap[peerIP]
}

func (d *Destination) setUpdateAggPath(peerIP string) {
	path, ok := d.peerPathMap[peerIP]
	if ok {
		if d.locRibPath == nil || path == d.locRibPath || getRouteSource(d.locRibPath.routeType) >= getRouteSource(path.routeType) {
			d.recalculate = true
		}
	}

	if d.locRibPath == nil {
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

func (d *Destination) AddOrUpdatePath(peerIp string, path *Path) bool {
	added := false
	oldPath, ok := d.peerPathMap[peerIp]
	if ok {
		d.logger.Info(fmt.Sprintf("Update path for %s from %s", d.nlri.Prefix.String(), peerIp))
		if d.locRibPath == oldPath {
			d.locRibPath = path
			d.recalculate = true
		}
	} else {
		d.logger.Info(fmt.Sprintf("Add new path for %s from %s", d.nlri.Prefix.String(), peerIp))
		added = true
	}

	if d.locRibPath == nil || getRouteSource(d.locRibPath.routeType) >= getRouteSource(path.routeType) {
		d.recalculate = true
	}

	d.peerPathMap[peerIp] = path
	return added
}

func (d *Destination) RemovePath(peerIp string, path *Path) {
	if oldPath, ok := d.peerPathMap[peerIp]; ok {
		for path, _ := range d.ecmpPaths {
			if path == oldPath {
				d.recalculate = true
				d.locRibPath = path
			}
		}

		if d.locRibPath == oldPath {
			d.recalculate = true
			d.locRibPath = path
		}
		delete(d.peerPathMap, peerIp)
	} else {
		d.logger.Err(fmt.Sprintln("Can't remove path", d.nlri.Prefix.String(), "Path not found from peer", peerIp))
	}
}

func (d *Destination) RemoveAllNeighborPaths() {
	for peerIP, path := range d.peerPathMap {
		if path.peer != nil {
			delete(d.peerPathMap, peerIP)
		}
	}

	if d.locRibPath != nil {
		if d.locRibPath.peer != nil {
			d.recalculate = true
			d.locRibPath = nil
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

func (d *Destination) SelectRouteForLocRib() (RouteAction, []*Route, []*Route, []*Route) {
	updatedPaths := make([]*Path, 0)
	addedRoutes := make([]*Route, 0)
	updatedRoutes := make([]*Route, 0)
	deletedRoutes := make([]*Route, 0)
	maxPref := uint32(0)
	routeSrc := RouteSrcUnknown
	locRibAction := RouteActionNone

	d.logger.Info(fmt.Sprintf("Destination:SelectRouteForLocalRib - peer path map = %v", d.peerPathMap))
	if !d.recalculate {
		return locRibAction, addedRoutes, updatedRoutes, deletedRoutes
	}
	d.recalculate = false

	if d.locRibPath != nil && !d.locRibPath.IsWithdrawn() && !d.locRibPath.IsUpdated() {
		peerIP := d.server.BgpConfig.Global.Config.RouterId.String()
		if d.locRibPath.peer != nil {
			peerIP = d.locRibPath.peer.Neighbor.NeighborAddress.String()
		}
		routeSrc = getRouteSource(d.locRibPath.routeType)
		maxPref = d.locRibPath.GetPreference()
		updatedPaths = append(updatedPaths, d.locRibPath)
		d.logger.Info(fmt.Sprintf("Add loc rib path from %s to the list of selected paths, pref=%d", peerIP, maxPref))
	}

	for peerIP, path := range d.peerPathMap {
		if path.IsUpdated() || (d.locRibPath != nil && (d.locRibPath.IsWithdrawn() || d.locRibPath.IsUpdated())) {
			if !path.IsLocal() && !path.IsReachable() {
				d.logger.Info(fmt.Sprintf("peer %s, NEXT_HOP[%s] is not reachable", peerIP, path.GetNextHop()))
				continue
			}

			if path.HasASLoop() {
				d.logger.Info(fmt.Sprintf("This path has AS loop [%d], removing this path from the selection process", path.peer.PeerConf.LocalAS))
				continue
			}

			currPathSource := getRouteSource(path.routeType)
			if currPathSource > routeSrc {
				continue
			} else if currPathSource < routeSrc {
				if len(updatedPaths) > 0 {
					updatedPaths[0] = path
					// For garbage collection
					for i := 1; i < len(updatedPaths); i++ {
						updatedPaths[i] = nil
					}
					updatedPaths = updatedPaths[:1]
				} else {
					updatedPaths = append(updatedPaths, path)
				}
				d.logger.Info(fmt.Sprintf("route from %s is from a better source type, old type=%d, new type=%d, pref=%d",
					peerIP, routeSrc, currPathSource, path.GetPreference()))
				routeSrc = currPathSource
				maxPref = path.GetPreference()
				continue
			}

			currPref := path.GetPreference()
			if currPref > maxPref {
				if len(updatedPaths) > 0 {
					updatedPaths[0] = path
					// For garbage collection
					for i := 1; i < len(updatedPaths); i++ {
						updatedPaths[i] = nil
					}
					updatedPaths = updatedPaths[:1]
				} else {
					updatedPaths = append(updatedPaths, path)
				}
				d.logger.Info(fmt.Sprintf("route from %s has more preference, old pref=%d, new pref=%d",
					peerIP, maxPref, currPref))
				maxPref = currPref
			} else if currPref == maxPref {
				d.logger.Info(fmt.Sprintf("route from %s has same preference, add to the list, pref=%d",
					peerIP, maxPref))
				updatedPaths = append(updatedPaths, path)
			}
		}
	}

	d.logger.Info(fmt.Sprintln("Destination =", d.nlri.Prefix.String(), "ECMP routes =", d.ecmpPaths, "updated paths =", updatedPaths))
	if len(updatedPaths) > 0 {
		var ecmpPaths [][]*Path
		if len(updatedPaths) > 1 {
			d.logger.Info(fmt.Sprintf("Found multiple paths with same pref, run path selection algorithm"))
			if d.server.BgpConfig.Global.Config.UseMultiplePaths {
				updatedPaths, ecmpPaths = d.calculateBestPath(updatedPaths, d.server.BgpConfig.Global.Config.EBGPMaxPaths > 1,
					d.server.BgpConfig.Global.Config.IBGPMaxPaths > 1)
			} else {
				updatedPaths, ecmpPaths = d.calculateBestPath(updatedPaths, false, false)
			}
		}

		if len(updatedPaths) > 1 {
			d.logger.Err(fmt.Sprintf("Have more than one route after the tie breaking rules... using the first one, routes[%s]", updatedPaths))
		}

		d.removeAndPrepend(&ecmpPaths, updatedPaths[0])
		d.logger.Info(fmt.Sprintln("ecmpPaths =", ecmpPaths))

		for idx, paths := range ecmpPaths {
			found := false
			for _, path := range paths {
				if route, ok := d.ecmpPaths[path]; ok {
					// Update path
					found = true
					if (path.IsAggregate() || !path.IsLocal()) && path.IsUpdated() {
						d.logger.Info(fmt.Sprintf("Update route for ip=%s", d.nlri.Prefix.String()))
						d.updateRoute(path)
						route.update()
					}

					if (idx == 0) && (d.locRibPath != path) {
						locRibAction = RouteActionReplace
					}
					updatedRoutes = append(updatedRoutes, route)
					route.setAction(RouteActionReplace)
					break
				}
			}

			if !found {
				// Add route
				if paths[0].IsAggregate() || !paths[0].IsLocal() {
					d.logger.Info(fmt.Sprintf("Add route for ip=%s, mask=%s, next hop=%s", d.nlri.Prefix.String(),
						constructNetmaskFromLen(int(d.nlri.Length), 32).String(), paths[0].NextHop))
					protocol := "IBGP"
					if paths[0].IsExternal() {
						protocol = "EBGP"
					}
					ret, err := d.server.ribdClient.CreateV4Route(d.nlri.Prefix.String(),
						constructNetmaskFromLen(int(d.nlri.Length), 32).String(),
						paths[0].Metric, paths[0].NextHop, paths[0].NextHopIfType,
						paths[0].NextHopIfIdx, protocol)
					if err != nil {
						d.logger.Err(fmt.Sprintf("CreateV4Route failed with error: %s, retVal: %d", err, ret))
					}
				}
				if idx == 0 {
					locRibAction = RouteActionAdd
				}
				newRoute := NewRoute(d, paths[0], RouteActionAdd)
				d.ecmpPaths[paths[0]] = newRoute
				addedRoutes = append(addedRoutes, newRoute)
			}
		}

		d.locRibPath = ecmpPaths[0][0]
		d.locRibPathRoute = d.ecmpPaths[d.locRibPath]
	} else {
		if d.locRibPath != nil {
			// Remove route
			for path, route := range d.ecmpPaths {
				route.setAction(RouteActionDelete)
				if path.IsAggregate() || !path.IsLocal() {
					d.logger.Info(fmt.Sprintf("Remove route for ip=%s nexthop=%s", d.nlri.Prefix.String(),
						path.NextHop))
					protocol := "IBGP"
					if path.IsExternal() {
						protocol = "EBGP"
					}
					ret, err := d.server.ribdClient.DeleteV4Route(d.nlri.Prefix.String(),
						constructNetmaskFromLen(int(d.nlri.Length), 32).String(), protocol, path.NextHop)
					if err != nil {
						d.logger.Err(fmt.Sprintf("DeleteV4Route failed with error: %s, retVal: %d", err, ret))
					}
				}
			}
			locRibAction = RouteActionDelete
			d.locRibPath = nil
		}
	}

	for path, route := range d.ecmpPaths {
		if route.action == RouteActionNone || route.action == RouteActionDelete {
			if path.IsAggregate() || !path.IsLocal() {
				d.logger.Info(fmt.Sprintln("Remove route from ECMP paths, route =", route, "ip =",
					d.nlri.Prefix.String(), "next hop =", path.NextHop))
				protocol := "IBGP"
				if path.IsExternal() {
					protocol = "EBGP"
				}
				ret, err := d.server.ribdClient.DeleteV4Route(d.nlri.Prefix.String(),
					constructNetmaskFromLen(int(d.nlri.Length), 32).String(), protocol, path.NextHop)
				if err != nil {
					d.logger.Err(fmt.Sprintf("DeleteV4Route failed with error: %s, retVal: %d", err, ret))
				}
			}
			deletedRoutes = append(deletedRoutes, route)
			delete(d.ecmpPaths, path)
		} else {
			route.setAction(RouteActionNone)
		}
	}
	return locRibAction, addedRoutes, updatedRoutes, deletedRoutes
}

func (d *Destination) updateRoute(path *Path) {
	d.logger.Info(fmt.Sprintf("Remove route for ip=%s, mask=%s", d.nlri.Prefix.String(),
		constructNetmaskFromLen(int(d.nlri.Length), 32).String()))
	protocol := "IBGP"
	if path.IsExternal() {
		protocol = "EBGP"
	}
	ret, err := d.server.ribdClient.DeleteV4Route(d.nlri.Prefix.String(),
		constructNetmaskFromLen(int(d.nlri.Length), 32).String(), protocol, path.NextHop)
	if err != nil {
		d.logger.Err(fmt.Sprintf("DeleteV4Route failed with error: %s, retVal: %d", err, ret))
	}

	if path.IsAggregate() || !path.IsLocal() {
		var nextHop string
		if path.IsAggregate() {
			nextHop = "255.255.255.255"
		} else {
			nextHop = path.NextHop
		}
		d.logger.Info(fmt.Sprintf("Add route for ip=%s, mask=%s, next hop=%s", d.nlri.Prefix.String(),
			constructNetmaskFromLen(int(d.nlri.Length), 32).String(), nextHop))
		ret, err = d.server.ribdClient.CreateV4Route(d.nlri.Prefix.String(),
			constructNetmaskFromLen(int(d.nlri.Length), 32).String(),
			path.Metric, nextHop, path.NextHopIfType,
			path.NextHopIfIdx, protocol)
		if err != nil {
			d.logger.Err(fmt.Sprintf("CreateV4Route failed with error: %s, retVal: %d", err, ret))
		}
	}
}

func (d *Destination) getRoutesWithSmallestAS(updatedPaths []*Path) []*Path {
	minASNums := uint32(4096)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		d.logger.Info(fmt.Sprintln("Destination:getRoutesWithSmallestAS - get num ASes from path", updatedPaths[i]))
		asNums := updatedPaths[i].GetNumASes()
		from := ""
		if updatedPaths[i].peer != nil {
			from = updatedPaths[i].peer.Neighbor.NeighborAddress.String()
		}
		d.logger.Info(fmt.Sprintln("Destination:getRoutesWithSmallestAS - Dest =", d.nlri.Prefix, "number of ASes =",
			asNums, "from", from))
		if asNums < minASNums {
			minASNums = asNums
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if asNums == minASNums {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		return updatedPaths[:idx]
	}

	return updatedPaths
}

func (d *Destination) getRoutesWithLowestOrigin(updatedPaths []*Path) []*Path {
	minOrigin := uint8(packet.BGPPathAttrOriginMax)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		origin := updatedPaths[i].GetOrigin()
		if origin < minOrigin {
			minOrigin = origin
			updatedPaths[0] = updatedPaths[i]
			idx++
		} else if origin == minOrigin {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		return updatedPaths[:idx]
	}

	return updatedPaths
}

func deleteIBGPRoutes(updatedPaths []*Path) []*Path {
	n := len(updatedPaths) - 1
	i := 0

	for i <= n {
		if updatedPaths[i].peer.IsInternal() {
			updatedPaths[i] = updatedPaths[n]
			updatedPaths[n] = nil
			n--
			continue
		}
		i++
	}

	return updatedPaths[:i]
}

func (d *Destination) removeIBGPRoutesIfEBGPExist(updatedPaths []*Path) []*Path {
	for _, path := range updatedPaths {
		if path.peer != nil && path.peer.IsExternal() {
			return deleteIBGPRoutes(updatedPaths)
		}
	}

	return updatedPaths
}

func (d *Destination) isEBGPRoute(path *Path) bool {
	if path.peer != nil && path.peer.IsExternal() {
		return true
	}

	return false
}

func (d *Destination) isIBGPRoute(path *Path) bool {
	if path.peer != nil && path.peer.IsInternal() {
		return true
	}

	return false
}

func (d *Destination) getRoutesWithLowestBGPId(updatedPaths []*Path) []*Path {
	n := len(updatedPaths)
	lowestBGPId := uint32(math.MaxUint32)
	idx := 0

	for i := 0; i < n; i++ {
		bgpId := updatedPaths[i].GetBGPId()
		if bgpId < lowestBGPId {
			lowestBGPId = bgpId
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if bgpId == lowestBGPId {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		return updatedPaths[:idx]
	}

	return updatedPaths
}

func (d *Destination) getRoutesWithShorterClusterLen(updatedPaths []*Path) []*Path {
	minClusterLen := uint16(math.MaxUint16)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		clusterLen := updatedPaths[i].GetNumClusters()
		if clusterLen < minClusterLen {
			minClusterLen = clusterLen
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if clusterLen == minClusterLen {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		return updatedPaths[:idx]
	}

	return updatedPaths
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

func (d *Destination) getRoutesWithLowestPeerAddress(updatedPaths []*Path) []*Path {
	n := len(updatedPaths)
	idx := 0

	for i, path := range updatedPaths {
		val, err := CompareNeighborAddress(path.peer.Neighbor.NeighborAddress,
			updatedPaths[0].peer.Neighbor.NeighborAddress)
		if err != nil {
			d.logger.Err(fmt.Sprintf("CompareNeighborAddress failed with %s", err))
		}

		if val < 0 {
			updatedPaths[0] = updatedPaths[i]
			idx = 1
		} else if val == 0 {
			updatedPaths[idx] = updatedPaths[i]
			idx++
		}
	}

	if idx > 0 {
		for i := idx; i < n; i++ {
			updatedPaths[i] = nil
		}
		return updatedPaths[:idx]
	}

	return updatedPaths
}

func (d *Destination) getECMPPaths(updatedPaths []*Path) [][]*Path {
	ecmpPathMap := make(map[string][]*Path)

	for _, path := range updatedPaths {
		if _, ok := ecmpPathMap[path.NextHop]; !ok {
			ecmpPathMap[path.NextHop] = make([]*Path, 1)
			ecmpPathMap[path.NextHop][0] = path
		} else {
			ecmpPathMap[path.NextHop] = append(ecmpPathMap[path.NextHop], path)
		}
	}

	d.logger.Info(fmt.Sprintln("getECMPPaths: update paths =", updatedPaths, "ecmpPathsMap =", ecmpPathMap))
	ecmpPaths := make([][]*Path, 0)
	for _, paths := range ecmpPathMap {
		ecmpPaths = append(ecmpPaths, paths)
	}
	return ecmpPaths
}

func (d *Destination) calculateBestPath(updatedPaths []*Path, ebgpMultiPath, ibgpMultiPath bool) ([]*Path, [][]*Path) {
	var ecmpPaths [][]*Path

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling getRoutesWithSmallestAS, update paths =", updatedPaths))
		updatedPaths = d.getRoutesWithSmallestAS(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling getRoutesWithLowestOrigin, update paths =", updatedPaths))
		updatedPaths = d.getRoutesWithLowestOrigin(updatedPaths)
	}

	if (len(updatedPaths) > 1) && ebgpMultiPath && ibgpMultiPath {
		ecmpPaths = d.getECMPPaths(updatedPaths)
		d.logger.Info(fmt.Sprintln("calculateBestPath: IBGP & EBGP multi paths =", ecmpPaths))
	}

	if len(updatedPaths) > 1 {
		d.logger.Info(fmt.Sprintln("calling removeIBGPRoutesIfEBGPExist, update paths =", updatedPaths))
		updatedPaths = d.removeIBGPRoutesIfEBGPExist(updatedPaths)
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
		updatedPaths = d.getRoutesWithLowestBGPId(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		d.logger.Info("calling getRoutesWithShorterClusterLen")
		updatedPaths = d.getRoutesWithShorterClusterLen(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		d.logger.Info("calling getRoutesWithLowestPeerAddress")
		updatedPaths = d.getRoutesWithLowestPeerAddress(updatedPaths)
	}

	return updatedPaths, ecmpPaths
}
