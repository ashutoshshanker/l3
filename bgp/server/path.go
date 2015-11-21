// path.go
package server

import (
	"fmt"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"ribd"
)

type Path struct {
	logger *syslog.Writer
	peer *Peer
	nlri packet.IPPrefix
	pathAttrs []packet.BGPPathAttr
	withdrawn bool
	updated bool
	Pref int64

	NextHop string
	NextHopIfIdx ribd.Int
	Metric ribd.Int
}

func NewPath(logger *syslog.Writer, peer *Peer, nlri packet.IPPrefix, pa []packet.BGPPathAttr, withdrawn bool, updated bool) *Path {
	path := &Path{
		logger: logger,
		peer: peer,
		nlri: nlri,
		pathAttrs: pa,
		withdrawn: withdrawn,
		updated: updated,
	}

	path.Pref = path.calculatePref()
	return path
}

func (p *Path) calculatePref() int64 {
	var pref int64
	if p.peer.IsInternal() {
		pref = BGP_INTERNAL_PREF
		for _, attr := range p.pathAttrs {
			if attr.GetCode() == packet.BGPPathAttrTypeLocalPref {
				pref = int64(attr.(*packet.BGPPathAttrLocalPref).Value)
				break
			}
		}
	} else {
		pref = BGP_EXTERNAL_PREF
	}

	return pref
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

func (p *Path) GetPreference() int64 {
	return p.Pref
}

func (p *Path) GetNumASes() uint32 {
	var total uint32 = 0
	for _, attr := range p.pathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeASPath {
			asPaths := attr.(*packet.BGPPathAttrASPath).Value
			for _, asPath := range asPaths {
				if asPath.Type == packet.BGPASPathSet {
					total += 1
				} else if asPath.Type == packet.BGPASPathSequence {
					total += uint32(asPath.Length)
				}
			}
		}
	}

	return total
}

func (p *Path) GetOrigin() uint8 {
	for _, attr := range p.pathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeOrigin {
			return uint8(attr.(*packet.BGPPathAttrOrigin).Value)
		}
	}

	return uint8(packet.BGPPathAttrOriginMax)
}

func (p *Path) GetNextHop() net.IP {
	for _, attr := range p.pathAttrs {
		if attr.GetCode() == packet.BGPPathAttrTypeNextHop {
			return attr.(*packet.BGPPathAttrNextHop).Value
		}
	}

	return net.IPv4zero
}

func (p *Path) SetReachabilityInfo(nhInfo *ribd.NextHopInfo) {
	p.NextHop = nhInfo.NextHopIp
	if p.NextHop == "" || p.NextHop[0] == '0' {
		p.logger.Info(fmt.Sprintf("Next hop for %s is %s. Using %s as the next hop", p.nlri.Prefix.String(),
									p.NextHop, p.GetNextHop().String()))
		p.NextHop = p.GetNextHop().String()
	}
	p.NextHopIfIdx = nhInfo.NextHopIfIndex
	p.Metric = nhInfo.Metric
}
