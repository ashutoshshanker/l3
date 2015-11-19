// rib.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"math"
	"net"
)

const BGP_INTERNAL_PREF = 100
const BGP_EXTERNAL_PREF = 50

type Destination struct {
	server *BGPServer
	logger *syslog.Writer
	nlri packet.IPPrefix
	peerPathMap map[string]*Path
	locRibPath *Path
}

func NewDestination(server *BGPServer, nlri packet.IPPrefix) *Destination {
	dest := &Destination{
		server: server,
		logger: server.logger,
		nlri: nlri,
		peerPathMap: make(map[string]*Path),
	}

	return dest
}

func (d *Destination) AddOrUpdatePath(peer *Peer, peerIp string, pa []packet.BGPPathAttr) (*Path, bool) {
	added := false
	path, ok := d.peerPathMap[peerIp]
	if ok {
		path.UpdatePath(pa)
	} else {
		added = true
		path = NewPath(peer, d.nlri, pa, false, true)
		d.peerPathMap[peerIp] = path
	}

	return path, added
}

func (d  *Destination) RemovePath(peerIp string) {
	if path, ok := d.peerPathMap[peerIp]; ok {
		path.SetWithdrawn(true)
	} else {
		d.logger.Err(fmt.Sprintln("Can't remove path", d.nlri.Prefix.String(), "Destination not found in RIB"))
	}
}

func (d *Destination) SelectRouteForLocRib() {
	updatedPaths := make([]*Path, 0)
	maxPref := int64(math.MinInt64)

	if d.locRibPath != nil && !d.locRibPath.IsWithdrawn() {
		maxPref = d.locRibPath.GetPreference()
		updatedPaths = append(updatedPaths, d.locRibPath)
	}

	for peerIp, path := range d.peerPathMap {
		if path.IsWithdrawn() {
			delete(d.peerPathMap, peerIp)
		} else if path.IsUpdated() || (d.locRibPath != nil && ((d.locRibPath.IsWithdrawn()) || (d.locRibPath.IsUpdated() &&
									  d.locRibPath.GetPreference() == path.GetPreference()))){
			reachabilityInfo, err := d.server.ribdClient.GetRouteReachabilityInfo(path.GetNextHop().String())
			if err != nil {
				d.logger.Info(fmt.Sprintf("NEXT_HOP[%s] is not reachable", d.nlri.Prefix))
				continue
			}

			path.SetReachabilityInfo(reachabilityInfo)
			currPref := path.GetPreference()
			if currPref > maxPref {
				if len(updatedPaths) > 0 {
					updatedPaths[0] = path
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
			d.logger.Info(fmt.Sprintf("Add route for ip=%s", d.nlri.Prefix.String()))
			d.server.ribdClient.CreateV4Route(d.nlri.Prefix.String(), net.CIDRMask(int(d.nlri.Length), 32).String(),
												selectedPath.Metric, selectedPath.NextHop, selectedPath.NextHopIfIdx, 8)
		} else if d.locRibPath != selectedPath || d.locRibPath.IsUpdated() {
			// Update path
			d.logger.Info(fmt.Sprintf("Update route for ip=%s", d.nlri.Prefix.String()))
			d.server.ribdClient.CreateV4Route(d.nlri.Prefix.String(), net.CIDRMask(int(d.nlri.Length), 32).String(),
												selectedPath.Metric, selectedPath.NextHop, selectedPath.NextHopIfIdx, 8)
		}

		d.locRibPath = updatedPaths[0]
	} else {
		if d.locRibPath != nil {
			// Remove route
			d.logger.Info(fmt.Sprintf("Remove route for ip=%s", d.nlri.Prefix.String()))
			d.server.ribdClient.DeleteV4Route(d.nlri.Prefix.String(), net.CIDRMask(int(d.nlri.Length), 32).String(), 8)
		}
	}
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

loop:
	for i <= n {
		if updatedPaths[i].peer.IsInternal() {
			updatedPaths[i] = updatedPaths[n]
			updatedPaths[n] = nil
			n--
			continue loop
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
		val, err := CompareNeighborAddress(path.peer.Peer.NeighborAddress, updatedPaths[0].peer.Peer.NeighborAddress)
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
