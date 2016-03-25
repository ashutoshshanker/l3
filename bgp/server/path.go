// path.go
package server

import (
	"encoding/binary"
	"fmt"
	"l3/bgp/packet"
	"net"
	"ribd"
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
	server          *BGPServer
	logger          *logging.Writer
	peer            *Peer
	pathAttrs       []packet.BGPPathAttr
	withdrawn       bool
	updated         bool
	Pref            uint32
	NextHop         string
	NextHopIfType   ribd.Int
	NextHopIfIdx    ribd.Int
	Metric          ribd.Int
	routeType       uint8
	MED             uint32
	LocalPref       uint32
	AggregatedPaths map[string]*Path
}

func NewPath(server *BGPServer, peer *Peer, pa []packet.BGPPathAttr, withdrawn bool, updated bool, routeType uint8) *Path {
	path := &Path{
		server:          server,
		logger:          server.logger,
		peer:            peer,
		pathAttrs:       pa,
		withdrawn:       withdrawn,
		updated:         updated,
		routeType:       routeType,
		AggregatedPaths: make(map[string]*Path),
	}

	path.logger.Info(fmt.Sprintln("Path:NewPath - path attr =", pa, "path.path attrs =", path.pathAttrs))
	path.Pref = path.calculatePref()
	return path
}

func (p *Path) Clone() *Path {
	path := &Path{
		server:       p.server,
		logger:       p.server.logger,
		peer:         p.peer,
		pathAttrs:    p.pathAttrs,
		withdrawn:    p.withdrawn,
		updated:      p.updated,
		Pref:         p.Pref,
		NextHop:      p.NextHop,
		NextHopIfIdx: p.NextHopIfIdx,
		Metric:       p.Metric,
		routeType:    p.routeType,
		MED:          p.MED,
		LocalPref:    p.LocalPref,
	}

	return path
}

func (p *Path) calculatePref() uint32 {
	var pref uint32

	pref = BGP_INTERNAL_PREF

	for _, attr := range p.pathAttrs {
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
	for _, attr := range p.pathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeOriginatorId {
			if p.peer.Global.RouterId.Equal(attr.(*packet.BGPPathAttrOriginatorId).Value) {
				return false
			}
		}

		if attr.GetCode() == packet.BGPPathAttrTypeClusterList {
			clusters := attr.(*packet.BGPPathAttrClusterList).Value
			for _, clusterId := range clusters {
				if clusterId == p.peer.PeerConf.RouteReflectorClusterId {
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
	p.pathAttrs = pa
	p.Pref = p.calculatePref()
	p.updated = true
}

func (p *Path) SetUpdate(status bool) {
	p.updated = status
}

func (p *Path) IsUpdated() bool {
	return p.updated
}

func (p *Path) GetPreference() uint32 {
	return p.Pref
}

func (p *Path) GetAS4ByteList() []string {
	asList := make([]string, 0)
	for _, attr := range p.pathAttrs {
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
	if p.peer == nil {
		return false
	}
	return packet.HasASLoop(p.pathAttrs, p.peer.PeerConf.LocalAS)
}

func (p *Path) IsLocal() bool {
	return getRouteSource(p.routeType) == RouteSrcLocal
}

func (p *Path) IsAggregate() bool {
	return p.routeType == RouteTypeAgg
}

func (p *Path) IsExternal() bool {
	return p.peer != nil && p.peer.IsExternal()
}

func (p *Path) IsInternal() bool {
	return p.peer != nil && p.peer.IsInternal()
}

func (p *Path) GetNumASes() uint32 {
	p.logger.Info(fmt.Sprintln("Path:GetNumASes - path attrs =", p.pathAttrs))
	return packet.GetNumASes(p.pathAttrs)
}

func (p *Path) GetOrigin() uint8 {
	return packet.GetOrigin(p.pathAttrs)
}

func (p *Path) GetNextHop() net.IP {
	return packet.GetNextHop(p.pathAttrs)
}

func (p *Path) GetBGPId() uint32 {
	for _, attr := range p.pathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeOriginatorId {
			return binary.BigEndian.Uint32(attr.(*packet.BGPPathAttrOriginatorId).Value.To4())
		}
	}

	return binary.BigEndian.Uint32(p.peer.BGPId.To4())
}

func (p *Path) GetNumClusters() uint16 {
	return packet.GetNumClusters(p.pathAttrs)
}

func (p *Path) GetReachabilityInfo() {
	ipStr := p.GetNextHop().String()
	reachabilityInfo, err := p.server.ribdClient.GetRouteReachabilityInfo(ipStr)
	if err != nil {
		p.logger.Info(fmt.Sprintf("NEXT_HOP[%s] is not reachable", ipStr))
		p.NextHop = ""
		return
	}
	p.NextHop = reachabilityInfo.NextHopIp
	if p.NextHop == "" || p.NextHop[0] == '0' {
		p.logger.Info(fmt.Sprintf("Next hop for %s is %s. Using %s as the next hop", ipStr, p.NextHop, ipStr))
		p.NextHop = ipStr
	}
	p.NextHopIfType = ribd.Int(reachabilityInfo.NextHopIfType)
	p.NextHopIfIdx = ribd.Int(reachabilityInfo.NextHopIfIndex)
	p.Metric = ribd.Int(reachabilityInfo.Metric)
}

func (p *Path) IsReachable() bool {
	if p.NextHop != "" {
		return true
	}

	p.GetReachabilityInfo()
	if p.NextHop != "" {
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
	aggMED, aggOK := packet.GetMED(p.pathAttrs)
	med, ok := packet.GetMED(path.pathAttrs)
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
	for idx = 0; idx < len(p.pathAttrs); idx++ {
		for i = pathIdx; i < len(path.pathAttrs) && path.pathAttrs[i].GetCode() < p.pathAttrs[idx].GetCode(); i++ {
			if path.pathAttrs[i].GetCode() == packet.BGPPathAttrTypeAtomicAggregate {
				atomicAggregate := packet.NewBGPPathAttrAtomicAggregate()
				packet.AddPathAttrToPathAttrs(p.pathAttrs, packet.BGPPathAttrTypeAtomicAggregate, atomicAggregate)
			}
		}

		if path.pathAttrs[i].GetCode() == p.pathAttrs[idx].GetCode() {
			if p.pathAttrs[idx].GetCode() == packet.BGPPathAttrTypeOrigin {
				if path.pathAttrs[i].(*packet.BGPPathAttrOrigin).Value > p.pathAttrs[idx].(*packet.BGPPathAttrOrigin).Value {
					p.pathAttrs[idx].(*packet.BGPPathAttrOrigin).Value = path.pathAttrs[i].(*packet.BGPPathAttrOrigin).Value
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

func (p *Path) isAggregatePath() bool {
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
		for _, pathAttr := range individualPath.pathAttrs {
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

	for idx, pathAttr := range p.pathAttrs {
		if pathAttr.GetCode() == packet.BGPPathAttrTypeOrigin && origin != nil {
			p.pathAttrs[idx] = origin
		}

		if pathAttr.GetCode() == packet.BGPPathAttrTypeASPath && aggASPath != nil {
			p.pathAttrs[idx] = aggASPath
		}

		if pathAttr.GetCode() == packet.BGPPathAttrTypeAtomicAggregate && atomicAggregate != nil {
			p.pathAttrs[idx] = atomicAggregate
		}
	}
}
