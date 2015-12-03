// bgp.go
package packet

import (
	"fmt"
	"net"
)

func PrependAS(updateMsg *BGPMessage, AS uint32, setAS bool) {
	body := updateMsg.Body.(*BGPUpdate)

	for _, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeASPath {
			asPathSegments := pa.(*BGPPathAttrASPath).Value
			if setAS && (len(asPathSegments) == 0 || (asPathSegments[0].Type == BGPASPathSet || asPathSegments[0].Length >= 255)) {
				newASPathSegment := NewBGPASPathSegmentSeq()
//				var dummyASPathSegment BGPASPathSegment
//				pa.(*BGPPathAttrASPath).Value = append(pa.(*BGPPathAttrASPath).Value, dummyASPathSegment)
//				copy(pa.(*BGPPathAttrASPath).Value[1:], pa.(*BGPPathAttrASPath).Value[0:])
//				pa.(*BGPPathAttrASPath).Value[0] = *newASPathSegment
//				pa.(*BGPPathAttrASPath).BGPPathAttrBase.Length += 2
				pa.(*BGPPathAttrASPath).AddASPathSegment(newASPathSegment)
			}

			if setAS {
				asPathSegments[0].PrependAS(uint16(AS))
//				asPathSegments[0].AS = append(asPathSegments[0].AS, 0)
//				copy(asPathSegments[0].AS[1:], asPathSegments[0].AS[0:])
//				asPathSegments[0].AS[0] = uint16(AS)
//				asPathSegments[0].Length += 1
				pa.(*BGPPathAttrASPath).BGPPathAttrBase.Length += 2
			}
			return
		}
	}
}

func removePathAttr(updateMsg *BGPMessage, code BGPPathAttrType) {
	body := updateMsg.Body.(*BGPUpdate)

	for idx, pa := range body.PathAttributes {
		if pa.GetCode() == code {
			body.PathAttributes = append(body.PathAttributes[:idx], body.PathAttributes[idx + 1:]...)
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
		} else if idx == len(body.PathAttributes) - 1 {
			idx += 1
		}
	}

	if idx >= 0 {
		paLocalPref := NewBGPPathAttrLocalPref()
		paLocalPref.Value = pref
		body.PathAttributes = append(body.PathAttributes, paLocalPref)
		if idx < len(body.PathAttributes) - 1 {
			copy(body.PathAttributes[idx + 1:], body.PathAttributes[idx:])
			body.PathAttributes[idx] = paLocalPref
		}
	}
}

func SetNextHop(updateMsg *BGPMessage, nextHop net.IP) {
	body := updateMsg.Body.(*BGPUpdate)

	for idx, pa := range body.PathAttributes {
		if pa.GetCode() == BGPPathAttrTypeNextHop {
			body.PathAttributes[idx].(*BGPPathAttrNextHop).Value = nextHop
		}
	}
}

func ConstructPathAttrForConnRoutes() []BGPPathAttr {
	pathAttrs := make([]BGPPathAttr, 0)

	origin := NewBGPPathAttrOrigin(BGPPathAttrOriginIGP)
	pathAttrs = append(pathAttrs, origin)

	asPath := NewBGPPathAttrASPath()
	//asPathSeq := NewBGPASPathSegmentSeq()
	//asPath.AddASPathSegment(asPathSeq)
	pathAttrs = append(pathAttrs, asPath)

	nextHop := NewBGPPathAttrNextHop()
	pathAttrs = append(pathAttrs, nextHop)

	return pathAttrs
}

func ConstructIPPrefix(ipStr string, maskStr string) *IPPrefix {
	fmt.Println("helpers:ConstructIPPrefix - ip str =", ipStr, "mask str =", maskStr)
	ip := net.ParseIP(ipStr)
	mask := net.IPMask(net.ParseIP(maskStr).To4())
	fmt.Println("helpers:ConstructIPPrefix - ip =", ip, "mask =", mask)
	ones, bits := mask.Size()
	fmt.Println("helpers:ConstructIPPrefix - ones =", ones, "bits =", bits)
	return NewIPPrefix(ip.Mask(mask), uint8(ones))
}
