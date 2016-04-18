// route.go
package server

import (
	"bgpd"
	"time"
)

type RouteAction uint8

const (
	RouteActionNone RouteAction = iota
	RouteActionAdd
	RouteActionReplace
	RouteActionDelete
)

type Route struct {
	BGPRouteState    *bgpd.BGPRouteState
	dest             *Destination
	path             *Path
	routeListIdx     int
	time             time.Time
	action           RouteAction
	OutPathId        uint32
	PolicyList       []string
	PolicyHitCounter int
}

func NewRoute(dest *Destination, path *Path, action RouteAction, inPathId, outPathId uint32) *Route {
	currTime := time.Now()
	bgpRoute := &bgpd.BGPRouteState{
		Network:     dest.IPPrefix.Prefix.String(),
		CIDRLen:     int16(dest.IPPrefix.Length),
		NextHop:     path.GetNextHop().String(),
		Metric:      int32(path.MED),
		LocalPref:   int32(path.LocalPref),
		Path:        path.GetAS4ByteList(),
		PathId:      int32(inPathId),
		UpdatedTime: currTime.String(),
	}
	return &Route{
		BGPRouteState:    bgpRoute,
		dest:             dest,
		path:             path,
		routeListIdx:     -1,
		time:             currTime,
		action:           action,
		OutPathId:        outPathId,
		PolicyList:       make([]string, 0),
		PolicyHitCounter: 0,
	}
}

func (r *Route) GetBGPRoute() *bgpd.BGPRouteState {
	if r.BGPRouteState != nil {
		r.BGPRouteState.UpdatedDuration = time.Now().Sub(r.time).String()
	}
	return r.BGPRouteState
}

func (r *Route) update() {
	r.time = time.Now()
	r.BGPRouteState.UpdatedTime = r.time.String()
}

func (r *Route) setAction(action RouteAction) {
	r.action = action
}

func (r *Route) setIdx(idx int) {
	r.routeListIdx = idx
}
