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

// bgp.go
package packet

import (
	"fmt"
	"l3/bgp/utils"
	"math"
	"net"
	"sort"
)

func PrependAS(updateMsg *BGPMessage, AS uint32, asSize uint8) {
	body := updateMsg.Body.(*BGPUpdate)

	for _, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPathSegments := pa.(*BGPPathAttrASPath).Value
			var newASPathSegment BGPASPathSegment
			if len(asPathSegments) == 0 || asPathSegments[0].GetType() == BGPASPathSegmentSet || asPathSegments[0].GetLen() >= 255 {
				if asSize == 4 {
					newASPathSegment = NewBGPAS4PathSegmentSeq()
				} else {
					newASPathSegment = NewBGPAS2PathSegmentSeq()
					if asSize == 2 {
						if AS > math.MaxUint16 {
							AS = uint32(BGPASTrans)
						}
					}
				}
				pa.(*BGPPathAttrASPath).PrependASPathSegment(newASPathSegment)
			}
			asPathSegments = pa.(*BGPPathAttrASPath).Value
			asPathSegments[0].PrependAS(AS)
			pa.(*BGPPathAttrASPath).BGPPathAttrBase.Length += uint16(asSize)
		} else if pa.GetCode() == BGPPathAttrTypeAS4Path {
			asPathSegments := pa.(*BGPPathAttrAS4Path).Value
			var newAS4PathSegment *BGPAS4PathSegment
			if len(asPathSegments) == 0 || asPathSegments[0].GetType() == BGPASPathSegmentSet || asPathSegments[0].GetLen() >= 255 {
				newAS4PathSegment = NewBGPAS4PathSegmentSeq()
				pa.(*BGPPathAttrAS4Path).AddASPathSegment(newAS4PathSegment)
			}
			asPathSegments = pa.(*BGPPathAttrAS4Path).Value
			asPathSegments[0].PrependAS(AS)
			pa.(*BGPPathAttrASPath).BGPPathAttrBase.Length += uint16(asSize)
		}
	}
}

func AppendASToAS4PathSeg(asPath *BGPPathAttrASPath, pathSeg BGPASPathSegment, asPathType BGPASPathSegmentType,
	asNum uint32) BGPASPathSegment {
	if pathSeg == nil {
		pathSeg = NewBGPAS4PathSegment(asPathType)
	} else if pathSeg.GetType() != asPathType {
		asPath.AppendASPathSegment(pathSeg)
	}

	if !pathSeg.AppendAS(asNum) {
		asPath.AppendASPathSegment(pathSeg)
		pathSeg = NewBGPAS4PathSegment(asPathType)
		pathSeg.AppendAS(asNum)
	}

	return pathSeg
}

func AddPathAttrToPathAttrs(pathAttrs []BGPPathAttr, code BGPPathAttrType, attr BGPPathAttr) {
	addIdx := -1
	for idx, pa := range pathAttrs {
		if pa.GetCode() > code {
			addIdx = idx
		}
	}

	if addIdx == -1 {
		addIdx = len(pathAttrs)
	}

	pathAttrs = append(pathAttrs, attr)
	copy(pathAttrs[addIdx+1:], pathAttrs[addIdx:])
	pathAttrs[addIdx] = attr
	return
}

func addPathAttr(updateMsg *BGPMessage, code BGPPathAttrType, attr BGPPathAttr) {
	body := updateMsg.Body.(*BGPUpdate)
	AddPathAttrToPathAttrs(body.PathAttributes, code, attr)
	return
}

func removePathAttr(updateMsg *BGPMessage, code BGPPathAttrType) {
	body := updateMsg.Body.(*BGPUpdate)

	for idx, pa := range body.PathAttributes {
		if pa.GetCode() == code {
			body.PathAttributes = append(body.PathAttributes[:idx], body.PathAttributes[idx+1:]...)
			return
		}
	}
}

func RemoveMultiExitDisc(updateMsg *BGPMessage) {
	removePathAttr(updateMsg, BGPPathAttrTypeMultiExitDisc)
}

func RemoveLocalPref(updateMsg *BGPMessage) {
	removePathAttr(updateMsg, BGPPathAttrTypeLocalPref)
}

func SetLocalPref(updateMsg *BGPMessage, pref uint32) {
	body := updateMsg.Body.(*BGPUpdate)

	var idx int
	var pa BGPPathAttr
	for idx, pa = range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeLocalPref {
			body.PathAttributes[idx].(*BGPPathAttrLocalPref).Value = pref
			return
		}
	}

	idx = -1
	for idx, pa = range body.PathAttributes {
		if pa.GetCode() > BGPPathAttrTypeLocalPref {
			break
		} else if idx == len(body.PathAttributes)-1 {
			idx += 1
		}
	}

	if idx >= 0 {
		paLocalPref := NewBGPPathAttrLocalPref()
		paLocalPref.Value = pref
		body.PathAttributes = append(body.PathAttributes, paLocalPref)
		if idx < len(body.PathAttributes)-1 {
			copy(body.PathAttributes[idx+1:], body.PathAttributes[idx:])
			body.PathAttributes[idx] = paLocalPref
		}
	}
}

func SetNextHop(updateMsg *BGPMessage, nextHop net.IP) {
	body := updateMsg.Body.(*BGPUpdate)
	SetNextHopPathAttrs(body.PathAttributes, nextHop)
}

func SetNextHopPathAttrs(pathAttrs []BGPPathAttr, nextHopIP net.IP) {
	for idx, pa := range pathAttrs {
		if pa.GetCode() == BGPPathAttrTypeNextHop {
			pathAttrs[idx].(*BGPPathAttrNextHop).Value = nextHopIP
		}
	}
}

func SetPathAttrAggregator(pathAttrs []BGPPathAttr, as uint32, ip net.IP) {
	for idx, pa := range pathAttrs {
		if pa.GetCode() == BGPPathAttrTypeAggregator {
			pathAttrs[idx].(*BGPPathAttrAggregator).AS = uint16(as)
			pathAttrs[idx].(*BGPPathAttrAggregator).IP = ip
		}
	}
}

func HasASLoop(pathAttrs []BGPPathAttr, localAS uint32) bool {
	for _, attr := range pathAttrs {
		if attr.GetCode() == BGPPathAttrTypeASPath {
			asPaths := attr.(*BGPPathAttrASPath).Value
			asSize := attr.(*BGPPathAttrASPath).ASSize
			for _, asSegment := range asPaths {
				if asSize == 4 {
					seg := asSegment.(*BGPAS4PathSegment)
					for _, as := range seg.AS {
						if as == localAS {
							return true
						}
					}
				} else {
					seg := asSegment.(*BGPAS2PathSegment)
					for _, as := range seg.AS {
						if as == uint16(localAS) {
							return true
						}
					}
				}
			}
			break
		}
	}

	return false
}

func GetNumASes(pathAttrs []BGPPathAttr) uint32 {
	var total uint32 = 0
	utils.Logger.Info(fmt.Sprintln("helpers:GetNumASes - path attrs =", pathAttrs))
	for _, attr := range pathAttrs {
		if attr.GetCode() == BGPPathAttrTypeASPath {
			asPaths := attr.(*BGPPathAttrASPath).Value
			for _, asPath := range asPaths {
				total += uint32(asPath.GetNumASes())
			}
			break
		}
	}

	return total
}

func GetOrigin(pathAttrs []BGPPathAttr) uint8 {
	for _, attr := range pathAttrs {
		if attr.GetCode() == BGPPathAttrTypeOrigin {
			return uint8(attr.(*BGPPathAttrOrigin).Value)
		}
	}

	return uint8(BGPPathAttrOriginMax)
}

func GetMED(pathAttrs []BGPPathAttr) (uint32, bool) {
	for _, attr := range pathAttrs {
		if attr.GetCode() == BGPPathAttrTypeMultiExitDisc {
			return attr.(*BGPPathAttrMultiExitDisc).Value, true
		}
	}

	return uint32(0), false
}

func GetNextHop(pathAttrs []BGPPathAttr) net.IP {
	for _, attr := range pathAttrs {
		if attr.GetCode() == BGPPathAttrTypeNextHop {
			return attr.(*BGPPathAttrNextHop).Value
		}
	}

	return net.IPv4zero
}

func GetNumClusters(pathAttrs []BGPPathAttr) uint16 {
	var total uint16 = 0
	for _, attr := range pathAttrs {
		if attr.GetCode() == BGPPathAttrTypeClusterList {
			length := attr.(*BGPPathAttrClusterList).Length
			total = length / 4
			break
		}
	}

	return total
}

var AggRoutesDefaultBGPPathAttr = map[BGPPathAttrType]BGPPathAttr{
	BGPPathAttrTypeOrigin:     NewBGPPathAttrOrigin(BGPPathAttrOriginIncomplete),
	BGPPathAttrTypeASPath:     NewBGPPathAttrASPath(),
	BGPPathAttrTypeNextHop:    NewBGPPathAttrNextHop(),
	BGPPathAttrTypeAggregator: NewBGPPathAttrAggregator(),
	//BGPPathAttrTypeAtomicAggregate: NewBGPPathAttrAtomicAggregate(),
}

func AggregateASPaths(asPathList []*BGPPathAttrASPath) *BGPPathAttrASPath {
	aggASPath := NewBGPPathAttrASPath()
	if len(asPathList) > 0 {
		asNumMap := make(map[uint32]bool, 10)
		asPathIterList := make([]*ASPathIter, 0, len(asPathList))
		for i := 0; i < len(asPathList); i++ {
			asPathIterList = append(asPathIterList, NewASPathIter(asPathList[i]))
		}

		var asPathSeg BGPASPathSegment
		asSeqDone := false
		var asPathVal, iterASPathVal uint32
		var asPathType, iterASPathType BGPASPathSegmentType
		var flag, iterFlag bool
		var idx int
		for {
			idx = 0
			asPathVal, asPathType, flag = asPathIterList[idx].Next()
			if !flag {
				break
			}

			for idx = 1; idx < len(asPathIterList); idx++ {
				iterASPathVal, iterASPathType, iterFlag = asPathIterList[idx].Next()
				if !iterFlag || iterASPathType != asPathType || iterASPathVal != asPathVal {
					asSeqDone = true
					break
				}
			}

			if asSeqDone {
				break
			}

			asPathSeg = AppendASToAS4PathSeg(aggASPath, asPathSeg, asPathType, asPathVal)
			asNumMap[asPathVal] = true
		}
		if asPathSeg != nil && asPathSeg.GetNumASes() > 0 {
			aggASPath.AppendASPathSegment(asPathSeg)
		}
		if !flag || !iterFlag {
			asPathIterList[idx] = nil
		}
		asPathSeg = NewBGPAS4PathSegmentSet()

		if flag {
			if !asNumMap[asPathVal] {
				asPathSeg = AppendASToAS4PathSeg(aggASPath, asPathSeg, asPathType, asPathVal)
				asNumMap[asPathVal] = true
			}
			if iterFlag {
				if !asNumMap[iterASPathVal] {
					asPathSeg = AppendASToAS4PathSeg(aggASPath, asPathSeg, iterASPathType, iterASPathVal)
					asNumMap[iterASPathVal] = true
				}
			}
			for idx = idx + 1; idx < len(asPathIterList); idx++ {
				asPathVal, asPathType, flag = asPathIterList[idx].Next()
				if flag {
					if !asNumMap[asPathVal] {
						asPathSeg = AppendASToAS4PathSeg(aggASPath, asPathSeg, asPathType, asPathVal)
						asNumMap[asPathVal] = true
					}
				} else {
					asPathIterList[idx] = nil
				}
			}
		}
		asPathIterList = RemoveNilItemsFromList(asPathIterList)
		for idx = 0; idx < len(asPathIterList); idx++ {
			for asPathVal, asPathType, flag = asPathIterList[idx].Next(); flag; {
				if !asNumMap[asPathVal] {
					asPathSeg = AppendASToAS4PathSeg(aggASPath, asPathSeg, asPathType, asPathVal)
					asNumMap[asPathVal] = true
				} else {
					asPathIterList[idx] = nil
					break
				}
			}
		}
	}
	return aggASPath
}

func ConstructPathAttrForAggRoutes(pathAttrs []BGPPathAttr, generateASSet bool) []BGPPathAttr {
	newPathAttrs := make([]BGPPathAttr, 0)
	reqAttrs := []BGPPathAttrType{BGPPathAttrTypeOrigin, BGPPathAttrTypeASPath, BGPPathAttrTypeNextHop,
		BGPPathAttrTypeAtomicAggregate, BGPPathAttrTypeAggregator}

	for _, pa := range pathAttrs {
		if pa.GetCode() == BGPPathAttrTypeNextHop || pa.GetCode() == BGPPathAttrTypeOrigin ||
			pa.GetCode() == BGPPathAttrTypeASPath || pa.GetCode() == BGPPathAttrTypeAtomicAggregate ||
			pa.GetCode() == BGPPathAttrTypeAggregator || pa.GetCode() == BGPPathAttrTypeMultiExitDisc {
			if pa.GetCode() == BGPPathAttrTypeASPath && !generateASSet {
				asPath := NewBGPPathAttrASPath()
				newPathAttrs = append(newPathAttrs, asPath)
			} else {
				newPathAttrs = append(newPathAttrs, pa.Clone())
			}
		}
	}

	sort.Sort(PathAttrs(newPathAttrs))

	total := len(newPathAttrs)
	idx := 0
	for _, pa := range newPathAttrs {
		for i, paType := range reqAttrs[idx:total] {
			if paType < pa.GetCode() && (paType != BGPPathAttrTypeASPath || generateASSet) {
				pathAttr := AggRoutesDefaultBGPPathAttr[paType]
				newPathAttrs = append(newPathAttrs, pathAttr)
			} else {
				if paType == pa.GetCode() {
					idx += (i + 1)
				} else {
					idx += i
				}
				break
			}
		}
	}
	return newPathAttrs
}

func ConstructPathAttrForConnRoutes(ip net.IP, as uint32) []BGPPathAttr {
	pathAttrs := make([]BGPPathAttr, 0)

	origin := NewBGPPathAttrOrigin(BGPPathAttrOriginIncomplete)
	pathAttrs = append(pathAttrs, origin)

	asPath := NewBGPPathAttrASPath()
	pathAttrs = append(pathAttrs, asPath)

	nextHop := NewBGPPathAttrNextHop()
	nextHop.Value = ip
	pathAttrs = append(pathAttrs, nextHop)

	return pathAttrs
}

func ConstructIPPrefix(ipStr string, maskStr string) *IPPrefix {
	ip := net.ParseIP(ipStr)
	mask := net.IPMask(net.ParseIP(maskStr).To4())
	ones, _ := mask.Size()
	return NewIPPrefix(ip.Mask(mask), uint8(ones))
}

func ConstructIPPrefixFromCIDR(cidr string) (*IPPrefix, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		utils.Logger.Info(fmt.Sprintln("ConstructIPPrefixFromCIDR: ParseCIDR for IPPrefix", cidr, "failed with err", err))
		return nil, err
	}

	ones, _ := ipNet.Mask.Size()
	return NewIPPrefix(ipNet.IP, uint8(ones)), nil
}

func AddOriginatorId(updateMsg *BGPMessage, id net.IP) bool {
	body := updateMsg.Body.(*BGPUpdate)
	var pa BGPPathAttr

	for _, pa = range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeOriginatorId {
			return false
		}
	}

	idx := -1
	for idx, pa = range body.PathAttributes {
		if pa.GetCode() > BGPPathAttrTypeOriginatorId {
			break
		} else if idx == len(body.PathAttributes)-1 {
			idx += 1
		}
	}

	if idx >= 0 {
		paOriginatorId := NewBGPPathAttrOriginatorId(id)
		body.PathAttributes = append(body.PathAttributes[:idx], paOriginatorId)
		copy(body.PathAttributes[idx+1:], body.PathAttributes[idx:])
		body.PathAttributes[idx] = paOriginatorId
	}

	return true
}

func AddClusterId(updateMsg *BGPMessage, id uint32) bool {
	body := updateMsg.Body.(*BGPUpdate)
	var pa BGPPathAttr
	var i int
	found := false
	idx := -1

	for i, pa = range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeClusterList {
			idx = i
			found = true
			break
		} else if idx == -1 {
			if pa.GetCode() > BGPPathAttrTypeClusterList {
				idx = i
			} else if i == len(body.PathAttributes)-1 {
				idx = i + 1
			}
		}
	}

	if !found && idx >= 0 {
		clusterList := NewBGPPathAttrClusterList()
		body.PathAttributes = append(body.PathAttributes[:idx], clusterList)
		copy(body.PathAttributes[idx+1:], body.PathAttributes[idx:])
		body.PathAttributes[idx] = clusterList
	}

	if idx >= 0 {
		body.PathAttributes[idx].(*BGPPathAttrClusterList).PrependId(id)
		return true
	}

	return false
}

func ConvertIPBytesToUint(bytes []byte) uint32 {
	return uint32(bytes[0])<<24 | uint32(bytes[1]<<16) | uint32(bytes[2]<<8) | uint32(bytes[3])
}

func ConstructOptParams(as uint32, afiSAfiMap map[uint32]bool, addPathsRx bool, addPathsMaxTx uint8) []BGPOptParam {
	optParams := make([]BGPOptParam, 0)
	capParams := make([]BGPCapability, 0)

	cap4ByteASPath := NewBGPCap4ByteASPath(as)
	capParams = append(capParams, cap4ByteASPath)
	capAddPaths := NewBGPCapAddPath()
	addPathFlags := uint8(0)
	if addPathsRx {
		addPathFlags |= BGPCapAddPathRx
	}
	if addPathsMaxTx > 0 {
		addPathFlags |= BGPCapAddPathTx
	}

	for protoFamily, _ := range afiSAfiMap {
		afi, safi := GetAfiSafi(protoFamily)
		utils.Logger.Info(fmt.Sprintf("Advertising capability for afi %d safi %d\n", afi, safi))
		capAfiSafi := NewBGPCapMPExt(afi, safi)
		capParams = append(capParams, capAfiSafi)

		addPathAfiSafi := NewAddPathAFISAFI(afi, safi, addPathFlags)
		capAddPaths.AddAddPathAFISAFI(addPathAfiSafi)
	}

	if addPathFlags != 0 {
		utils.Logger.Info(fmt.Sprintf("Advertising capability for addPaths %+v\n", capAddPaths.Value))
		capParams = append(capParams, capAddPaths)
	}

	optCapability := NewBGPOptParamCapability(capParams)
	optParams = append(optParams, optCapability)

	return optParams
}

func GetASSize(openMsg *BGPOpen) uint8 {
	for _, optParam := range openMsg.OptParams {
		if optParam.GetCode() == BGPOptParamTypeCapability {
			capabilities := optParam.(*BGPOptParamCapability)
			for _, capability := range capabilities.Value {
				if capability.GetCode() == BGPCapTypeAS4Path {
					return 4
				}
			}
		}
	}

	return 2
}

func GetAddPathFamily(openMsg *BGPOpen) map[AFI]map[SAFI]uint8 {
	addPathFamily := make(map[AFI]map[SAFI]uint8)
	for _, optParam := range openMsg.OptParams {
		if capabilities, ok := optParam.(*BGPOptParamCapability); ok {
			for _, capability := range capabilities.Value {
				if addPathCap, ok := capability.(*BGPCapAddPath); ok {
					utils.Logger.Info(fmt.Sprintf("add path capability = %+v\n", addPathCap))
					for _, val := range addPathCap.Value {
						if _, ok := addPathFamily[val.AFI]; !ok {
							addPathFamily[val.AFI] = make(map[SAFI]uint8)
						}
						if _, ok := addPathFamily[val.AFI][val.SAFI]; !ok {
							addPathFamily[val.AFI][val.SAFI] = val.Flags
						}
					}
					return addPathFamily
				}
			}
		}
	}
	return addPathFamily
}

func IsAddPathsTxEnabledForIPv4(addPathFamily map[AFI]map[SAFI]uint8) bool {
	enabled := false
	if _, ok := addPathFamily[AfiIP]; ok {
		for safi, flags := range addPathFamily[AfiIP] {
			if (safi == SafiUnicast || safi == SafiMulticast) && (flags&BGPCapAddPathTx != 0) {
				utils.Logger.Info(fmt.Sprintf("isAddPathsTxEnabledForIPv4 - add path Tx enabled for IPv4"))
				enabled = true
			}
		}
	}
	return enabled
}

func GetNumASesByASType(updateMsg *BGPMessage, asType BGPPathAttrType) uint32 {
	var total uint32 = 0

	if asType != BGPPathAttrTypeASPath && asType != BGPPathAttrTypeAS4Path {
		return total
	}

	body := updateMsg.Body.(*BGPUpdate)
	for _, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPaths := pa.(*BGPPathAttrASPath).Value
			for _, asPath := range asPaths {
				if asPath.GetType() == BGPASPathSegmentSet {
					total += 1
				} else if asPath.GetType() == BGPASPathSegmentSequence {
					total += uint32(asPath.GetLen())
				}
			}
			break
		}
	}

	return total
}

func ConvertAS2ToAS4(updateMsg *BGPMessage) {
	body := updateMsg.Body.(*BGPUpdate)
	for idx, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPath := pa.(*BGPPathAttrASPath)
			newASPath := NewBGPPathAttrASPath()
			for _, seg := range asPath.Value {
				as2Seg := seg.(*BGPAS2PathSegment)
				as4Seg := NewBGPAS4PathSegmentSeq()
				as4Seg.Type = as2Seg.Type
				as4Seg.Length = as2Seg.GetLen()
				as4Seg.BGPASPathSegmentLen += (uint16(as2Seg.GetLen()) * 4)
				for i, as := range as2Seg.AS {
					as4Seg.AS[i] = uint32(as)
				}
				newASPath.AppendASPathSegment(as4Seg)
			}
			body.PathAttributes[idx] = nil
			body.PathAttributes[idx] = newASPath
			break
		}
	}
}

func ConstructASPathFromAS4Path(asPath *BGPPathAttrASPath, as4Path *BGPPathAttrAS4Path, skip uint16) *BGPPathAttrASPath {
	var segIdx int
	var segment BGPASPathSegment
	var asNum uint16 = 0
	newASPath := NewBGPPathAttrASPath()
	for segIdx, segment = range asPath.Value {
		if (uint16(segment.GetNumASes()) + asNum) > skip {
			break
		}
		seg := segment.(*BGPAS2PathSegment)
		newSeg := NewBGPAS4PathSegmentSeq()
		newSeg.Type = seg.Type
		newSeg.Length = seg.Length
		newSeg.BGPASPathSegmentLen += (uint16(newSeg.Length) * 4)
		for asIdx, as := range seg.AS {
			newSeg.AS[asIdx] = uint32(as)
		}
		newASPath.AppendASPathSegment(newSeg)
		asNum += uint16(seg.GetNumASes())
	}

	for idx, segment := range as4Path.Value {
		seg4 := segment.Clone()
		if idx == 0 {
			seg := asPath.Value[segIdx].(*BGPAS2PathSegment)
			for asNum < skip {
				seg4.PrependAS(uint32(seg.AS[skip-asNum-1]))
			}
		}
		newASPath.AppendASPathSegment(seg4)
	}

	return newASPath
}

func Convert4ByteTo2ByteASPath(updateMsg *BGPMessage) {
	body := updateMsg.Body.(*BGPUpdate)
	for idx, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPath := pa.(*BGPPathAttrASPath)
			addAS4Path := false
			newAS4Path := asPath.CloneAsAS4Path()
			newAS2Path := NewBGPPathAttrASPath()
			for _, seg := range asPath.Value {
				as4Seg := seg.(*BGPAS4PathSegment)
				as2Seg, mappable := as4Seg.CloneAsAS2PathSegment()
				if !mappable {
					addAS4Path = true
				}
				newAS2Path.AppendASPathSegment(as2Seg)
			}
			body.PathAttributes[idx] = nil
			body.PathAttributes[idx] = newAS2Path
			if addAS4Path {
				addPathAttr(updateMsg, BGPPathAttrTypeAS4Path, newAS4Path)
			}
			break
		}
	}
}

func NormalizeASPath(updateMsg *BGPMessage, data interface{}) {
	var asPath *BGPPathAttrASPath
	var as4Path *BGPPathAttrAS4Path
	var asAggregator *BGPPathAttrAggregator
	var as4Aggregator *BGPPathAttrAS4Aggregator

	body := updateMsg.Body.(*BGPUpdate)
	if body.TotalPathAttrLen == 0 {
		return
	}

	for _, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPath = pa.(*BGPPathAttrASPath)
		} else if pa.GetCode() == BGPPathAttrTypeAS4Path {
			as4Path = pa.(*BGPPathAttrAS4Path)
		} else if pa.GetCode() == BGPPathAttrTypeAggregator {
			asAggregator = pa.(*BGPPathAttrAggregator)
		} else if pa.GetCode() == BGPPathAttrTypeAS4Aggregator {
			as4Aggregator = pa.(*BGPPathAttrAS4Aggregator)
		}
	}

	if asPath == nil {
		utils.Logger.Err("***** BGP update message does not have AS path *****")
		return
	}

	if asPath.ASSize == 2 {
		if asAggregator != nil && as4Aggregator != nil && asAggregator.AS != BGPASTrans {
			removePathAttr(updateMsg, BGPPathAttrTypeAS4Aggregator)
			removePathAttr(updateMsg, BGPPathAttrTypeAS4Path)
		} else {
			ConvertAS2ToAS4(updateMsg)
			if as4Path != nil {
				numASes := GetNumASesByASType(updateMsg, BGPPathAttrTypeASPath)
				numAS4es := GetNumASesByASType(updateMsg, BGPPathAttrTypeAS4Path)
				if numASes >= numAS4es {
					newASPath := ConstructASPathFromAS4Path(asPath, as4Path, uint16(numASes-numAS4es))
					removePathAttr(updateMsg, BGPPathAttrTypeAS4Path)
					removePathAttr(updateMsg, BGPPathAttrTypeASPath)
					addPathAttr(updateMsg, BGPPathAttrTypeASPath, newASPath)
				}
			}
		}
	} else if asPath.ASSize == 4 {
		if as4Aggregator != nil {
			removePathAttr(updateMsg, BGPPathAttrTypeAS4Aggregator)
		}
		if as4Path != nil {
			removePathAttr(updateMsg, BGPPathAttrTypeAS4Path)
		}
	}
}

func ConstructMaxSizedUpdatePackets(bgpMsg *BGPMessage) []*BGPMessage {
	var withdrawnRoutes []NLRI
	newUpdateMsgs := make([]*BGPMessage, 0)
	pktLen := uint32(BGPUpdateMsgMinLen)
	startIdx := 0
	lastIdx := 0
	updateMsg := bgpMsg.Body.(*BGPUpdate)

	if updateMsg.WithdrawnRoutes != nil {
		for lastIdx = 0; lastIdx < len(updateMsg.WithdrawnRoutes); lastIdx++ {
			nlriLen := updateMsg.WithdrawnRoutes[lastIdx].Len()
			if nlriLen+pktLen > BGPMsgMaxLen {
				newMsg := NewBGPUpdateMessage(updateMsg.WithdrawnRoutes[startIdx:lastIdx], nil, nil)
				newUpdateMsgs = append(newUpdateMsgs, newMsg)
				startIdx = lastIdx
				pktLen = uint32(BGPUpdateMsgMinLen)
			}
			pktLen += nlriLen
		}
	}

	if lastIdx > startIdx {
		withdrawnRoutes = updateMsg.WithdrawnRoutes[startIdx:lastIdx]
	}

	paLen := uint32(0)
	for i := 0; i < len(updateMsg.PathAttributes); i++ {
		paLen += updateMsg.PathAttributes[i].TotalLen()
	}
	if pktLen+paLen > BGPMsgMaxLen {
		newMsg := NewBGPUpdateMessage(withdrawnRoutes, nil, nil)
		withdrawnRoutes = nil
		newUpdateMsgs = append(newUpdateMsgs, newMsg)
		pktLen = BGPUpdateMsgMinLen
	}

	startIdx = 0
	lastIdx = 0
	for lastIdx = 0; lastIdx < len(updateMsg.NLRI); lastIdx++ {
		nlriLen := updateMsg.NLRI[lastIdx].Len()
		if nlriLen+pktLen+paLen > BGPMsgMaxLen {
			newMsg := NewBGPUpdateMessage(withdrawnRoutes, updateMsg.PathAttributes, updateMsg.NLRI[startIdx:lastIdx])
			newUpdateMsgs = append(newUpdateMsgs, newMsg)
			if withdrawnRoutes != nil {
				withdrawnRoutes = nil
			}
			startIdx = lastIdx
			pktLen = uint32(BGPUpdateMsgMinLen)
		}
		pktLen += nlriLen
	}

	if (withdrawnRoutes != nil && len(withdrawnRoutes) > 0) || (lastIdx > startIdx) {
		newMsg := NewBGPUpdateMessage(withdrawnRoutes, updateMsg.PathAttributes, updateMsg.NLRI[startIdx:lastIdx])
		newUpdateMsgs = append(newUpdateMsgs, newMsg)
	}

	return newUpdateMsgs
}
