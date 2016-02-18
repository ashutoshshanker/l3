package vrrpServer

import (
	"log/syslog"
)

/*
	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                    IPv4 Fields or IPv6 Fields                 |
	...                                                             ...
	|                                                               |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|Version| Type  | Virtual Rtr ID|   Priority    |Count IPvX Addr|
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|(rsvd) |     Max Adver Int     |          Checksum             |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                                                               |
	+                                                               +
	|                       IPvX Address(es)                        |
	+                                                               +
	+                                                               +
	+                                                               +
	+                                                               +
	|                                                               |
	+                                                               +
	|                                                               |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

type VrrpServiceHandler struct {
}

/*
	IfIndex int32
	// no default for VRID
	VRID int32
	// default value is 100
	Priority int32
	// No Default for IPv4 addr.. Can support one or more IPv4 addresses
	IPv4Addr []string
	// IPv6Addr... will add later when we decide to support IPv6

	// Default is 100 centiseconds which is 1 SEC
	AdvertisementInterval int32
	// False to prohibit preemption. Default is True.
	PreemptMode bool
	// The default is False.
	AcceptMode bool
	// MAC address used for the source MAC address in VRRP advertisements
	VirtualRouterMACAddress string
*/
type VrrpGlobalInfo struct {
	// The initial value is the same as Advertisement_Interval.
	MasterAdverInterval int32
	// (((256 - priority) * Master_Adver_Interval) / 256)
	SkewTime int32
	// (3 * Master_Adver_Interval) + Skew_time
	MasterDownInterval int32
}

var (
	logger      *syslog.Writer
	vrrpGblInfo map[int32]VrrpGlobalInfo
)
