// path.go
package server

import (
	"sort"
)

type PathSortIface struct {
	paths Paths
	iface sort.Interface
}

type Paths []*Path

func ClonePaths(paths Paths) Paths {
	newPaths := make(Paths, 0, len(paths))
	for i := 0; i < len(paths); i++ {
		newPaths[i] = paths[i]
	}

	return newPaths
}

func (p Paths) Len() int {
	return len(p)
}

func (p Paths) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Paths) getPaths() Paths {
	return p
}

type ByPref struct {
	Paths
}

func (b ByPref) Less(i, j int) bool {
	return b.Paths[i].Pref < b.Paths[i].Pref
}

type BySmallestAS struct {
	Paths
}

func (b BySmallestAS) Less(i, j int) bool {
	return b.Paths[i].GetNumASes() < b.Paths[i].GetNumASes()
}

type ByLowestOrigin struct {
	Paths
}

func (b ByLowestOrigin) Less(i, j int) bool {
	return b.Paths[i].GetOrigin() < b.Paths[i].GetOrigin()
}

type ByIBGPOrEBGPRoutes struct {
	Paths
}

func (b ByIBGPOrEBGPRoutes) Less(i, j int) bool {
	return true
}

type ByLowestBGPId struct {
	Paths
}

func (b ByLowestBGPId) Less(i, j int) bool {
	return b.Paths[i].GetBGPId() < b.Paths[j].GetBGPId()
}

type ByShorterClusterLen struct {
	Paths
}

func (b ByShorterClusterLen) Less(i, j int) bool {
	return b.Paths[i].GetNumClusters() < b.Paths[j].GetNumClusters()
}

type ByLowestPeerAddress struct {
	Paths
}

func (b ByLowestPeerAddress) Less(i, j int) bool {
	if b.Paths[i].peer == nil {
		return true
	} else if b.Paths[j].peer == nil {
		return false
	}

	iNetIP := b.Paths[i].peer.Neighbor.NeighborAddress
	jNetIP := b.Paths[j].peer.Neighbor.NeighborAddress

	if len(iNetIP) < len(jNetIP) {
		return true
	} else if len(jNetIP) < len(iNetIP) {
		return false
	}

	for i, val := range iNetIP {
		if val < jNetIP[i] {
			return true
		} else if val > jNetIP[i] {
			return false
		}
	}

	return false
}
