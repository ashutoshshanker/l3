// bgp.go
package packet

import (
	"fmt"
	"l3/bgp/utils"
)

type ASPathIter struct {
	asPath     *BGPPathAttrASPath
	segments   []*BGPAS4PathSegment
	segmentLen int
	segIdx     int
	asValIdx   int
}

func NewASPathIter(asPath *BGPPathAttrASPath) *ASPathIter {
	iter := ASPathIter{
		asPath:     asPath,
		segmentLen: len(asPath.Value),
	}

	iter.segments = make([]*BGPAS4PathSegment, 0, len(asPath.Value))
	for idx := 0; idx < len(asPath.Value); idx++ {
		var as4Seg *BGPAS4PathSegment
		var ok bool
		if as4Seg, ok = asPath.Value[idx].(*BGPAS4PathSegment); !ok {
			utils.Logger.Err(fmt.Sprintln("AS path segment", idx, "is not AS4PathSegment"))
			return nil
		}
		iter.segments = append(iter.segments, as4Seg)
	}
	return &iter
}

func RemoveNilItemsFromList(iterList []*ASPathIter) []*ASPathIter {
	lastIdx := len(iterList) - 1
	var modIdx, idx int
	for idx = 0; idx < len(iterList); idx++ {
		if iterList[idx] == nil {
			for modIdx = lastIdx; modIdx > idx && iterList[modIdx] == nil; modIdx-- {
			}
			if modIdx <= idx {
				break
			}
			iterList[idx] = iterList[modIdx]
			iterList[modIdx] = nil
			lastIdx = modIdx
		}
	}

	return iterList[:idx]
}

func (a *ASPathIter) Next() (val uint32, segType BGPASPathSegmentType, flag bool) {
	if a.segIdx >= a.segmentLen {
		return val, segType, flag
	}

	val = a.segments[a.segIdx].AS[a.asValIdx]
	segType = a.segments[a.segIdx].Type
	flag = true

	a.asValIdx++
	if a.asValIdx >= len(a.segments[a.segIdx].AS) {
		a.segIdx++
		a.asValIdx = 0
	}

	return val, segType, flag
}
