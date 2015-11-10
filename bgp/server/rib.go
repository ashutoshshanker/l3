// rib.go
package server

import (
	"fmt"
	"l3/bgp/packet"
	"log/syslog"
	_ "net"
)

const BGP_INTERNAL_PREF = 100
const BGP_EXTERNAL_PREF = 50

type AdjRib struct {
	server *BGPServer
	logger *syslog.Writer
	destPathMap map[string]map[string]*Path
}

func NewAdjRib(server *BGPServer) *AdjRib {
	rib := &AdjRib{
		server: server,
		logger: server.logger,
		destPathMap: make(map[string]map[string]*Path),
	}

	return rib
}

func isIpInList(ipPrefix []packet.IPPrefix, ip packet.IPPrefix) bool {
	for _, prefix := range ipPrefix {
		if prefix.Prefix.Equal(ip.Prefix) {
			return true
		}
	}
	return false
}

func (adjRib *AdjRib) AddOrUpdatePath(nlri packet.IPPrefix, peerIp string, pa []packet.BGPPathAttr) {
	newPath := NewPath(nlri, pa, false)
	if peerMap, ok  := adjRib.destPathMap[nlri.Prefix.String()]; ok {
		peerMap[peerIp] = newPath
	} else {
		adjRib.destPathMap[nlri.Prefix.String()] = make(map[string]*Path)
		adjRib.destPathMap[nlri.Prefix.String()][peerIp] = newPath
	}
}

func (adjRib *AdjRib) RemovePath(nlri packet.IPPrefix, peerIp string) {
	if peerMap, ok  := adjRib.destPathMap[nlri.Prefix.String()]; ok {
		if path, ok := peerMap[peerIp]; ok {
			path.SetWithdrawn(true)
		} else {
			adjRib.logger.Info(fmt.Sprintln("Can't remove path", nlri.Prefix.String(), "Peer not found in RIB"))
		}
	} else {
		adjRib.logger.Info(fmt.Sprintln("Can't remove path", nlri.Prefix.String(), "Destination not found in RIB"))
	}
}

func (adjRib *AdjRib) GetPreference(peerIp string, path *Path) uint32 {
	if adjRib.server.IsPeerLocal(peerIp) {
		for _, attr := range path.pathAttrs {
			if attr.GetCode() == packet.BGPPathAttrTypeLocalPref {
				return attr.(*packet.BGPPathAttrLocalPref).Value
			} else {
				return BGP_INTERNAL_PREF
			}
		}
	}

	return BGP_EXTERNAL_PREF
}

func (adjRib *AdjRib) SelectRouteForLocRib(nlri packet.IPPrefix) {
	var selectedPeer string = ""
	var withDrawn string = ""
	maxPref := uint32(0)
	for peerIp, path := range adjRib.destPathMap[nlri.Prefix.String()] {
		if path.GetWithdrawn() {
			withDrawn = peerIp
		} else {
			currPref := adjRib.GetPreference(peerIp, path)
			if currPref > maxPref {
				maxPref = currPref
				selectedPeer = peerIp
			}
		}
	}

	if withDrawn != "" {
		if selectedPeer != "" {
			adjRib.logger.Info(fmt.Sprintln("Update route with prefix", nlri, "in Loc RIB"))
		} else {
			adjRib.logger.Info(fmt.Sprintln("Remove route with prefix", nlri, "from Loc RIB"))
		}
	} else if selectedPeer != "" {
		adjRib.logger.Info(fmt.Sprintln("Add route with prefix", nlri, "to Loc RIB"))
	}
}

func (adjRib *AdjRib) ProcessUpdate(pktInfo *packet.BGPPktSrc) {
	adjRib.logger.Info(fmt.Sprintln("AdjRib:ProcessUpdate - start"))
	body := pktInfo.Msg.Body.(*packet.BGPUpdate)

	// process withdrawn routes
	for _, nlri := range body.WithdrawnRoutes {
		if !isIpInList(body.NLRI, nlri){
			adjRib.logger.Info(fmt.Sprintln("Processing withdraw destination", nlri.Prefix.String()))
			adjRib.RemovePath(nlri, pktInfo.Src)
			adjRib.SelectRouteForLocRib(nlri)
		} else {
			adjRib.logger.Info(fmt.Sprintln("Can't withdraw destination", nlri.Prefix.String(),
				"Destination is part of NLRI in the UDPATE"))
		}
	}

	for _, nlri := range body.NLRI {
		adjRib.AddOrUpdatePath(nlri, pktInfo.Src, body.PathAttributes)
		adjRib.SelectRouteForLocRib(nlri)
	}
}
