// route.go
package server

import (
	"bgpd"
	"net"
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
	dest         *Destination
	path         *Path
	routeListIdx int
	time         time.Time
	action       RouteAction
}

func NewRoute(dest *Destination, path *Path, action RouteAction) *Route {
	return &Route{
		dest:         dest,
		path:         path,
		routeListIdx: -1,
		time:         time.Now(),
		action:       action,
	}
}

func (r *Route) GetBGPRoute() *bgpd.BGPRoute {
	if r.dest != nil {
		return &bgpd.BGPRoute{
			Network:   r.dest.nlri.Prefix.String(),
			Mask:      net.CIDRMask(int(r.dest.nlri.Length), 32).String(),
			NextHop:   r.path.NextHop,
			Metric:    int32(r.path.MED),
			LocalPref: int32(r.path.LocalPref),
			Path:      r.path.GetAS4ByteList(),
			Updated:   time.Now().Sub(r.time).String(),
		}
	}

	return nil
}

func (r *Route) update() {
	r.time = time.Now()
}

func (r *Route) setAction(action RouteAction) {
	r.action = action
}
