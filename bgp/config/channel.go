// conn.go
package config

type ReachabilityInfo struct {
	IP          string
	ReachableCh chan bool
}

type Operation int

const (
	NOTIFY_ROUTE_CREATED Operation = iota + 1
	NOTIFY_ROUTE_DELETED
	BFD_STATE_VALID
	BFD_STATE_INVALID
	INTF_CREATED
	INTF_DELETED
	INTF_STATE_DOWN
	INTF_STATE_UP
	NOTIFY_POLICY_CONDITION_CREATED
	NOTIFY_POLICY_CONDITION_DELETED
	NOTIFY_POLICY_CONDITION_UPDATED
	NOTIFY_POLICY_STMT_CREATED
	NOTIFY_POLICY_STMT_DELETED
	NOTIFY_POLICY_STMT_UPDATED
	NOTIFY_POLICY_DEFINITION_CREATED
	NOTIFY_POLICY_DEFINITION_DELETED
	NOTIFY_POLICY_DEFINITION_UPDATED
)

type BfdInfo struct {
	Oper   Operation
	DestIp string
	State  bool
}

type IntfStateInfo struct {
	Idx    int32
	IPAddr string
	State  Operation
}

func NewIntfStateInfo(idx int32, ipAddr string, state Operation) *IntfStateInfo {
	return &IntfStateInfo{
		Idx:    idx,
		IPAddr: ipAddr,
		State:  state,
	}
}

/*  This is mimic of ribd object...@TODO: need to change this to bgp server object
 */
type RouteInfo struct {
	IPAddr           string
	Mask             string
	NextHopIp        string
	Prototype        int
	NetworkStatement bool
	RouteOrigin      string
}

type RouteCh struct {
	Add    []*RouteInfo
	Remove []*RouteInfo
}

type NextHopInfo struct {
	IPAddr         string
	Mask           string
	Metric         int32
	NextHopIp      string
	IsReachable    bool
	NextHopIfType  int32
	NextHopIfIndex int32
}
