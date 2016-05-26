//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

// path.go
package server

import (
	"encoding/binary"
	"fmt"
	"l3/bgp/baseobjects"
	"l3/bgp/packet"
	"net"
	_ "ribd"
	"strconv"
	"strings"
	"utils/logging"
)

const (
	RouteTypeAgg uint8 = 1 << iota
	RouteTypeConnected
	RouteTypeStatic
	RouteTypeIGP
	RouteTypeEGP
	RouteTypeMax
)

const (
	RouteSrcLocal uint8 = 1 << iota
	RouteSrcExternal
	RouteSrcUnknown
)

var RouteTypeToSource = map[uint8]uint8{
	RouteTypeAgg:       RouteSrcLocal,
	RouteTypeConnected: RouteSrcLocal,
	RouteTypeStatic:    RouteSrcLocal,
	RouteTypeIGP:       RouteSrcLocal,
	RouteTypeEGP:       RouteSrcExternal,
}

func getRouteSource(routeType uint8) uint8 {
	if routeSource, ok := RouteTypeToSource[routeType]; ok {
		return routeSource
	}

	return RouteSrcUnknown
}

type Path struct {
	rib              *AdjRib
	logger           *logging.Writer
	NeighborConf     *base.NeighborConf
	PathAttrs        []packet.BGPPathAttr
	withdrawn        bool
	updated          bool
	Pref             uint32
	reachabilityInfo *ReachabilityInfo
	routeType        uint8
	MED              uint32
	LocalPref        uint32
	AggregatedPaths  map[string]*Path
}

func NewPath(adjRib *AdjRib, peer *base.NeighborConf, pa []packet.BGPPathAttr, withdrawn bool, updated bool, routeType uint8) *Path {
	path := &Path{
		rib:             adjRib,
		logger:          adjRib.logger,
		NeighborConf:    peer,
		PathAttrs:       pa,
		withdrawn:       withdrawn,
		updated:         updated,
		routeType:       routeType,
		AggregatedPaths: make(map[string]*Path),
	}

	path.logger.Info(fmt.Sprintln("Path:NewPath - path attr =", pa, "path.path attrs =", path.PathAttrs))
	path.Pref = path.calculatePref()
	return path
}

func (p *Path) Clone() *Path {
	path := &Path{
		rib:              p.rib,
		logger:           p.rib.logger,
		NeighborConf:     p.NeighborConf,
		PathAttrs:        p.PathAttrs,
		withdrawn:        p.withdrawn,
		updated:          p.updated,
		Pref:             p.Pref,
		reachabilityInfo: p.reachabilityInfo,
		routeType:        p.routeType,
		MED:              p.MED,
		LocalPref:        p.LocalPref,
	}

	return path
}

func (p *Path) calculatePref() uint32 {
	var pref uint32

	pref = BGP_INTERNAL_PREF

	for _, attr := range p.PathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeLocalPref {
			p.LocalPref = attr.(*packet.BGPPathAttrLocalPref).Value
			pref = p.LocalPref
		} else if attr.GetCode() == packet.BGPPathAttrTypeMultiExitDisc {
			p.MED = attr.(*packet.BGPPathAttrMultiExitDisc).Value
		}
	}

	if p.IsExternal() {
		pref = BGP_EXTERNAL_PREF
	}

	return pref
}

func (p *Path) IsValid() bool {
	for _, attr := range p.PathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeOriginatorId {
			if p.NeighborConf.Global.RouterId.Equal(attr.(*packet.BGPPathAttrOriginatorId).Value) {
				return false
			}
		}

		if attr.GetCode() == packet.BGPPathAttrTypeClusterList {
			clusters := attr.(*packet.BGPPathAttrClusterList).Value
			for _, clusterId := range clusters {
				if clusterId == p.NeighborConf.RunningConf.RouteReflectorClusterId {
					return false
				}
			}
		}
	}

	return true
}

func (p *Path) SetWithdrawn(status bool) {
	p.withdrawn = status
}

func (p *Path) IsWithdrawn() bool {
	return p.withdrawn
}

func (p *Path) UpdatePath(pa []packet.BGPPathAttr) {
	p.PathAttrs = pa
	p.Pref = p.calculatePref()
	p.updated = true
}

func (p *Path) SetUpdate(status bool) {
	p.updated = status
}

func (p *Path) IsUpdated() bool {
	return p.updated
}

func (p *Path) GetNeighborConf() *base.NeighborConf {
	return p.NeighborConf
}

func (p *Path) GetPeerIP() string {
	if p.NeighborConf != nil {
		return p.NeighborConf.Neighbor.NeighborAddress.String()
	}
	return ""
}

func (p *Path) GetPreference() uint32 {
	return p.Pref
}

func (p *Path) GetAS4ByteList() []string {
	asList := make([]string, 0)
	for _, attr := range p.PathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeASPath {
			asPaths := attr.(*packet.BGPPathAttrASPath).Value
			asSize := attr.(*packet.BGPPathAttrASPath).ASSize
			for _, asSegment := range asPaths {
				if asSize == 4 {
					seg := asSegment.(*packet.BGPAS4PathSegment)
					if seg.Type == packet.BGPASPathSegmentSet {
						asSetList := make([]string, 0, len(seg.AS))
						for _, as := range seg.AS {
							asSetList = append(asSetList, strconv.Itoa(int(as)))
						}
						asSetStr := strings.Join(asSetList, ", ")
						asSetStr = "{ " + asSetStr + " }"
						//asSetStr = append(asSetStr, "}")
						asList = append(asList, asSetStr)
					} else if seg.Type == packet.BGPASPathSegmentSequence {
						//asSeqList := make([]string, 0, len(seg.AS))
						for _, as := range seg.AS {
							//asSeq := make([]int32, 1)
							//asSeq[0] = int32(as)
							//asSeqList = append(asSeqList, asSeq)
							asList = append(asList, strconv.Itoa(int(as)))
						}
						//asList = append(asList, asSeqList...)
					}
				} else {
					seg := asSegment.(*packet.BGPAS2PathSegment)
					if seg.Type == packet.BGPASPathSegmentSet {
						asSetList := make([]string, 0, len(seg.AS))
						for _, as := range seg.AS {
							asSetList = append(asSetList, strconv.Itoa(int(as)))
						}
						asSetStr := strings.Join(asSetList, ", ")
						asSetStr = "{ " + asSetStr + " }"
						//asSetStr = append("{", asSetStr...)
						//asSetStr = append(asSetStr, "}")
						asList = append(asList, asSetStr)
					} else if seg.Type == packet.BGPASPathSegmentSequence {
						//asSeqList := make([][]int32, 0, len(seg.AS))
						for _, as := range seg.AS {
							//asSeq := make([]int32, 1)
							//asSeq[0] = int32(as)
							//asSeqList = append(asSeqList, asSeq)
							asList = append(asList, strconv.Itoa(int(as)))
						}
						//asList = append(asList, asSeqList...)
					}
				}
			}
			break
		}
	}

	return asList
}

func (p *Path) HasASLoop() bool {
	if p.NeighborConf == nil {
		return false
	}
	return packet.HasASLoop(p.PathAttrs, p.NeighborConf.RunningConf.LocalAS)
}

func (p *Path) IsLocal() bool {
	return getRouteSource(p.routeType) == RouteSrcLocal
}

func (p *Path) IsAggregate() bool {
	return p.routeType == RouteTypeAgg
}

func (p *Path) IsExternal() bool {
	return p.NeighborConf != nil && p.NeighborConf.IsExternal()
}

func (p *Path) IsInternal() bool {
	return p.NeighborConf != nil && p.NeighborConf.IsInternal()
}

func (p *Path) GetNumASes() uint32 {
	p.logger.Info(fmt.Sprintln("Path:GetNumASes - path attrs =", p.PathAttrs))
	return packet.GetNumASes(p.PathAttrs)
}

func (p *Path) GetOrigin() uint8 {
	return packet.GetOrigin(p.PathAttrs)
}

func (p *Path) GetNextHop() net.IP {
	return packet.GetNextHop(p.PathAttrs)
}

func (p *Path) GetBGPId() uint32 {
	for _, attr := range p.PathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeOriginatorId {
			return binary.BigEndian.Uint32(attr.(*packet.BGPPathAttrOriginatorId).Value.To4())
		}
	}

	return binary.BigEndian.Uint32(p.NeighborConf.BGPId.To4())
}

func (p *Path) GetNumClusters() uint16 {
	return packet.GetNumClusters(p.PathAttrs)
}

func (p *Path) SetReachabilityInfo(reachabilityInfo *ReachabilityInfo) {
	p.reachabilityInfo = reachabilityInfo
}

func (p *Path) IsReachable() bool {
	if p.IsLocal() || (p.reachabilityInfo != nil && p.reachabilityInfo.NextHop != "") {
		return true
	}
	return false
}

func (p *Path) setAggregatedPath(destIP string, path *Path) {
	if _, ok := p.AggregatedPaths[destIP]; ok {
		p.logger.Err(fmt.Sprintf("Path from %s is already added to the aggregated paths %v", destIP, p.AggregatedPaths))
	}
	p.AggregatedPaths[destIP] = path
}

func (p *Path) checkMEDForAggregation(path *Path) (uint32, uint32, bool) {
	aggMED, aggOK := packet.GetMED(p.PathAttrs)
	med, ok := packet.GetMED(path.PathAttrs)
	if aggOK == ok && aggMED == med {
		return aggMED, med, true
	}

	return aggMED, med, false
}

func (p *Path) addPathToAggregate(destIP string, path *Path, generateASSet bool) bool {
	aggMED, med, isMEDEqual := p.checkMEDForAggregation(path)

	if _, ok := p.AggregatedPaths[destIP]; ok {
		if !isMEDEqual {
			p.logger.Info(fmt.Sprintln("addPathToAggregate: MED", med, "in the new path", path, "is not the same as the MED",
				aggMED, "in the agg path, remove the old path..."))
			delete(p.AggregatedPaths, destIP)
			p.removePathFromAggregate(destIP, generateASSet)
		} else {
			p.logger.Info(fmt.Sprintf("addPathToAggregatePath from %s is already aggregated, replace it...", destIP))
			p.AggregatedPaths[destIP] = path
			p.aggregateAllPaths(generateASSet)
		}
		p.updated = true

		return true
	}

	if !isMEDEqual {
		p.logger.Info(fmt.Sprintln("addPathToAggregate: Can't aggregate new path MEDs not equal, new path MED =", med,
			"Agg path MED =", aggMED))
		return false
	}

	p.updated = true
	idx, i := 0, 0
	pathIdx := 0
	for idx = 0; idx < len(p.PathAttrs); idx++ {
		for i = pathIdx; i < len(path.PathAttrs) && path.PathAttrs[i].GetCode() < p.PathAttrs[idx].GetCode(); i++ {
			if path.PathAttrs[i].GetCode() == packet.BGPPathAttrTypeAtomicAggregate {
				atomicAggregate := packet.NewBGPPathAttrAtomicAggregate()
				packet.AddPathAttrToPathAttrs(p.PathAttrs, packet.BGPPathAttrTypeAtomicAggregate, atomicAggregate)
			}
		}

		if path.PathAttrs[i].GetCode() == p.PathAttrs[idx].GetCode() {
			if p.PathAttrs[idx].GetCode() == packet.BGPPathAttrTypeOrigin {
				if path.PathAttrs[i].(*packet.BGPPathAttrOrigin).Value > p.PathAttrs[idx].(*packet.BGPPathAttrOrigin).Value {
					p.PathAttrs[idx].(*packet.BGPPathAttrOrigin).Value = path.PathAttrs[i].(*packet.BGPPathAttrOrigin).Value
				}
			}
		}
	}
	p.AggregatedPaths[destIP] = path
	return true
}

func (p *Path) removePathFromAggregate(destIP string, generateASSet bool) {
	p.updated = true
	delete(p.AggregatedPaths, destIP)
	p.aggregateAllPaths(generateASSet)
}

func (p *Path) IsAggregatePath() bool {
	return (len(p.AggregatedPaths) > 0)
}

func (p *Path) isAggregatePathEmpty() bool {
	return (len(p.AggregatedPaths) == 0)
}

func (p *Path) aggregateAllPaths(generateASSet bool) {
	var origin, atomicAggregate packet.BGPPathAttr
	asPathList := make([]*packet.BGPPathAttrASPath, 0, len(p.AggregatedPaths))
	var aggASPath *packet.BGPPathAttrASPath
	for _, individualPath := range p.AggregatedPaths {
		for _, pathAttr := range individualPath.PathAttrs {
			if pathAttr.GetCode() == packet.BGPPathAttrTypeOrigin {
				if origin == nil || pathAttr.(*packet.BGPPathAttrOrigin).Value > origin.(*packet.BGPPathAttrOrigin).Value {
					origin = pathAttr
				}
			}

			if pathAttr.GetCode() == packet.BGPPathAttrTypeAtomicAggregate {
				if atomicAggregate == nil {
					atomicAggregate = pathAttr
				}
			}

			if pathAttr.GetCode() == packet.BGPPathAttrTypeASPath {
				asPathList = append(asPathList, pathAttr.(*packet.BGPPathAttrASPath))
			}
		}
	}

	if generateASSet {
		aggASPath = packet.AggregateASPaths(asPathList)
	}

	for idx, pathAttr := range p.PathAttrs {
		if pathAttr.GetCode() == packet.BGPPathAttrTypeOrigin && origin != nil {
			p.PathAttrs[idx] = origin
		}

		if pathAttr.GetCode() == packet.BGPPathAttrTypeASPath && aggASPath != nil {
			p.PathAttrs[idx] = aggASPath
		}

		if pathAttr.GetCode() == packet.BGPPathAttrTypeAtomicAggregate && atomicAggregate != nil {
			p.PathAttrs[idx] = atomicAggregate
		}
	}
}
