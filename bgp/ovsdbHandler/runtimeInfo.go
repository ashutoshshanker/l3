package ovsdbHandler

import (
	"bgpd"
)

type UUID string

type BGPFlexSwitch struct {
	neighbor bgpd.BGPNeighbor
	global   bgpd.BGPGlobal
}

var (
	bgpCachedOvsdb map[UUID]BGPFlexSwitch
)
