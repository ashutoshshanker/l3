// bgp.go
package packet

import (
	"net"
)

func PrependAS(updateMsg *BGPMessage, AS uint32, asSize uint8) {
	body := updateMsg.Body.(*BGPUpdate)

	for _, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPathSegments := pa.(*BGPPathAttrASPath).Value
			var newASPathSegment BGPASPathSegment
			if len(asPathSegments) == 0 || asPathSegments[0].GetType() == BGPASPathSet || asPathSegments[0].GetLen() >= 255 {
				if asSize == 4 {
					newASPathSegment = NewBGPAS4PathSegmentSeq()
				} else {
					newASPathSegment = NewBGPAS2PathSegmentSeq()
				}

				pa.(*BGPPathAttrASPath).PrependASPathSegment(newASPathSegment)
			}

			asPathSegments = pa.(*BGPPathAttrASPath).Value
			asPathSegments[0].PrependAS(AS)
			pa.(*BGPPathAttrASPath).BGPPathAttrBase.Length += uint16(asSize)
			return
		}
	}
}

func addPathAttr(updateMsg *BGPMessage, code BGPPathAttrType, attr BGPPathAttr) {
	body := updateMsg.Body.(*BGPUpdate)
	addIdx := -1
	for idx, pa := range body.PathAttributes {
		if pa.GetCode() > code {
			addIdx = idx
		}
	}

	if addIdx == -1 {
		addIdx = len(body.PathAttributes)
	}

	body.PathAttributes = append(body.PathAttributes, attr)
	copy(body.PathAttributes[addIdx+1:], body.PathAttributes[addIdx:])
	body.PathAttributes[addIdx] = attr
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

func ConstructOptParams(as uint32) []BGPOptParam {
	optParams := make([]BGPOptParam, 0)
	capParams := make([]BGPCapability, 0)

	cap4ByteASPath := NewBGPCap4ByteASPath(as)
	capParams = append(capParams, cap4ByteASPath)

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

func GetNumASes(updateMsg *BGPMessage, asType BGPPathAttrType) uint32 {
	var total uint32 = 0

	if asType != BGPPathAttrTypeASPath && asType != BGPPathAttrTypeAS4Path {
		return total
	}

	body := updateMsg.Body.(*BGPUpdate)
	for _, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPaths := pa.(*BGPPathAttrASPath).Value
			for _, asPath := range asPaths {
				if asPath.GetType() == BGPASPathSet {
					total += 1
				} else if asPath.GetType() == BGPASPathSequence {
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
			newAS4Path := asPath.CloneAsAS4Path()
			newAS2Path := NewBGPPathAttrASPath()
			for _, seg := range asPath.Value {
				as4Seg := seg.(*BGPAS4PathSegment)
				as2Seg := as4Seg.CloneAsAS2PathSegment()
				newAS2Path.AppendASPathSegment(as2Seg)
			}
			body.PathAttributes[idx] = nil
			body.PathAttributes[idx] = newAS2Path
			addPathAttr(updateMsg, BGPPathAttrTypeAS4Path, newAS4Path)
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

	if asPath.ASSize == 2 {
		if asAggregator != nil && as4Aggregator != nil && asAggregator.AS != BGPASTrans {
			removePathAttr(updateMsg, BGPPathAttrTypeAS4Aggregator)
			removePathAttr(updateMsg, BGPPathAttrTypeAS4Path)
		} else {
			ConvertAS2ToAS4(updateMsg)
			if as4Path != nil {
				numASes := GetNumASes(updateMsg, BGPPathAttrTypeASPath)
				numAS4es := GetNumASes(updateMsg, BGPPathAttrTypeAS4Path)
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
