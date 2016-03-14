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
	bgpRoute     *bgpd.BGPRoute
	dest         *Destination
	path         *Path
	routeListIdx int
	time         time.Time
	action       RouteAction
	outPathId    uint32
}

func NewRoute(dest *Destination, path *Path, action RouteAction, inPathId, outPathId uint32) *Route {
	bgpRoute := &bgpd.BGPRoute{
		Network:   dest.ipPrefix.Prefix.String(),
		CIDRLen:   int16(dest.ipPrefix.Length),
		NextHop:   path.NextHop,
		Metric:    int32(path.MED),
		LocalPref: int32(path.LocalPref),
		Path:      path.GetAS4ByteList(),
		PathId:    int32(inPathId),
	}
	return &Route{
		bgpRoute:     bgpRoute,
		dest:         dest,
		path:         path,
		routeListIdx: -1,
		time:         time.Now(),
		action:       action,
		outPathId:    outPathId,
	}
}

func (r *Route) GetBGPRoute() *bgpd.BGPRoute {
	if r.bgpRoute != nil {
		r.bgpRoute.Updated = time.Now().Sub(r.time).String()
	}
	return r.bgpRoute
}

func (r *Route) update() {
	r.time = time.Now()
}

func (r *Route) setAction(action RouteAction) {
	r.action = action
}

func (r *Route) setIdx(idx int) {
	r.routeListIdx = idx
}
