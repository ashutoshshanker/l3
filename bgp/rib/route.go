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
	BGPRoute         *bgpd.BGPRoute
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
	bgpRoute := &bgpd.BGPRoute{
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
		BGPRoute:         bgpRoute,
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

func (r *Route) GetBGPRoute() *bgpd.BGPRoute {
	if r.BGPRoute != nil {
		r.BGPRoute.UpdatedDuration = time.Now().Sub(r.time).String()
	}
	return r.BGPRoute
}

func (r *Route) update() {
	r.time = time.Now()
	r.BGPRoute.UpdatedTime = r.time.String()
}

func (r *Route) setAction(action RouteAction) {
	r.action = action
}

func (r *Route) setIdx(idx int) {
	r.routeListIdx = idx
}
