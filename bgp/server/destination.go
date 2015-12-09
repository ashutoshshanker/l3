// rib.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"math"
	"net"
	_ "ribd"
)

const BGP_INTERNAL_PREF = 100
const BGP_EXTERNAL_PREF = 50

type RouteSelectionAction uint8

const (
	RouteSelectionNone RouteSelectionAction = iota
	RouteSelectionAdd
	RouteSelectionReplace
	RouteSelectionDelete
)

type Destination struct {
	server      *BGPServer
	logger      *syslog.Writer
	nlri        packet.IPPrefix
	peerPathMap map[string]*Path
	locRibPath  *Path
	recalculate bool
}

func NewDestination(server *BGPServer, nlri packet.IPPrefix) *Destination {
	dest := &Destination{
		server:      server,
		logger:      server.logger,
		nlri:        nlri,
		peerPathMap: make(map[string]*Path),
	}

	return dest
}

func (d *Destination) AddOrUpdatePath(peerIp string, path *Path) bool {
	added := false
	oldPath, ok := d.peerPathMap[peerIp]
	if ok {
		if d.locRibPath == oldPath {
			d.locRibPath = path
			d.recalculate = true
		}
	} else {
		added = true
	}

	if d.locRibPath == nil || d.locRibPath.routeType >= path.routeType {
		d.recalculate = true
	}

	d.peerPathMap[peerIp] = path
	return added
}

func (d *Destination) RemovePath(peerIp string, path *Path) {
	if oldPath, ok := d.peerPathMap[peerIp]; ok {
		if d.locRibPath == oldPath {
			d.recalculate = true
			d.locRibPath = path
		}
		delete(d.peerPathMap, peerIp)
	} else {
		d.logger.Err(fmt.Sprintln("Can't remove path", d.nlri.Prefix.String(), "Destination not found in RIB"))
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

func (d *Destination) SelectRouteForLocRib() RouteSelectionAction {
	updatedPaths := make([]*Path, 0)
	maxPref := uint32(0)
	routeType := RouteTypeMax
	action := RouteSelectionNone

	if !d.recalculate {
		return action
	}
	d.recalculate = false

	if d.locRibPath != nil && !d.locRibPath.IsWithdrawn() && !d.locRibPath.IsUpdated() {
		maxPref = d.locRibPath.GetPreference()
		updatedPaths = append(updatedPaths, d.locRibPath)
	}

	for _, path := range d.peerPathMap {
		if path.IsUpdated() || (d.locRibPath != nil && (d.locRibPath.IsWithdrawn() || d.locRibPath.IsUpdated())) {
			if !path.IsLocal() && !path.IsReachable() {
				d.logger.Info(fmt.Sprintf("NEXT_HOP[%s] is not reachable", path.GetNextHop()))
				continue
			}

			if path.routeType > routeType {
				continue
			} else if path.routeType < routeType {
				if len(updatedPaths) > 0 {
					updatedPaths[0] = path
					updatedPaths = updatedPaths[:1]
				} else {
					updatedPaths = append(updatedPaths, path)
				}
				routeType = path.routeType
				maxPref = path.GetPreference()
				continue
			}

			currPref := path.GetPreference()
			if currPref > maxPref {
				if len(updatedPaths) > 0 {
					updatedPaths[0] = path
					updatedPaths = updatedPaths[:1]
				} else {
					updatedPaths = append(updatedPaths, path)
				}
				maxPref = currPref
			} else if currPref == maxPref {
				updatedPaths = append(updatedPaths, path)
			}
		}
	}

	// For garbage collection
	for i := len(updatedPaths); i < cap(updatedPaths); i++ {
		updatedPaths[i] = nil
	}

	if len(updatedPaths) > 0 {
		if len(updatedPaths) > 1 {
			updatedPaths = d.calculateBestPath(updatedPaths)
		}

		if len(updatedPaths) > 1 {
			d.logger.Err(fmt.Sprintf("Have more than one route after the tie breaking rules... using the first one, routes[%s]", updatedPaths))
		}

		selectedPath := updatedPaths[0]

		if d.locRibPath == nil {
			// Add route
			if !selectedPath.IsLocal() {
				d.logger.Info(fmt.Sprintf("Add route for ip=%s, mask=%s, next hop=%s", d.nlri.Prefix.String(),
					constructNetmaskFromLen(int(d.nlri.Length), 32).String(), selectedPath.NextHop))
				ret, err := d.server.ribdClient.CreateV4Route(d.nlri.Prefix.String(),
					constructNetmaskFromLen(int(d.nlri.Length), 32).String(),
					selectedPath.Metric, selectedPath.NextHop, selectedPath.NextHopIfType,
					selectedPath.NextHopIfIdx, 8)
				if err != nil {
					d.logger.Err(fmt.Sprintf("CreateV4Route failed with error: %s, retVal: %d", err, ret))
				}
			}
			action = RouteSelectionAdd
		} else if d.locRibPath != selectedPath || d.locRibPath.IsUpdated() {
			// Update path
			if !d.locRibPath.IsLocal() {
				d.logger.Info(fmt.Sprintf("Update route for ip=%s", d.nlri.Prefix.String()))
				err := d.server.ribdClient.UpdateV4Route(d.nlri.Prefix.String(),
					constructNetmaskFromLen(int(d.nlri.Length), 32).String(), 8,
					selectedPath.NextHop, selectedPath.NextHopIfIdx,
					selectedPath.Metric)
				if err != nil {
					d.logger.Err(fmt.Sprintf("UpdateV4Route failed with error: %s", err))
				}
			}
			action = RouteSelectionReplace
		}

		d.locRibPath = updatedPaths[0]
	} else {
		if d.locRibPath != nil {
			// Remove route
			if !d.locRibPath.IsLocal() {
				d.logger.Info(fmt.Sprintf("Remove route for ip=%s", d.nlri.Prefix.String()))
				ret, err := d.server.ribdClient.DeleteV4Route(d.nlri.Prefix.String(),
					constructNetmaskFromLen(int(d.nlri.Length), 32).String(), 8)
				if err != nil {
					d.logger.Err(fmt.Sprintf("DeleteV4Route failed with error: %s, retVal: %d", err, ret))
				}
			}
			action = RouteSelectionDelete
		}
	}

	return action
}

func (d *Destination) getRoutesWithSmallestAS(updatedPaths []*Path) []*Path {
	minASNums := uint32(4096)
	n := len(updatedPaths)
	idx := 0

	for i := 0; i < n; i++ {
		asNums := updatedPaths[i].GetNumASes()
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
		if path.peer.IsExternal() {
			return deleteIBGPRoutes(updatedPaths)
		}
	}

	return updatedPaths
}

func (d *Destination) getRoutesWithLowestBGPId(updatedPaths []*Path) []*Path {
	n := len(updatedPaths)
	lowestBGPId := uint32(math.MaxUint32)
	idx := 0

	for i := 0; i < n; i++ {
		bgpId := updatedPaths[i].peer.BGPId
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

func (d *Destination) calculateBestPath(updatedPaths []*Path) []*Path {
	if len(updatedPaths) > 1 {
		updatedPaths = d.getRoutesWithSmallestAS(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		updatedPaths = d.getRoutesWithLowestOrigin(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		updatedPaths = d.removeIBGPRoutesIfEBGPExist(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		updatedPaths = d.getRoutesWithLowestBGPId(updatedPaths)
	}

	if len(updatedPaths) > 1 {
		updatedPaths = d.getRoutesWithLowestPeerAddress(updatedPaths)
	}

	return updatedPaths
}
