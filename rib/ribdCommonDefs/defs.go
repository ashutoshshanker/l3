package ribdCommonDefs
import "ribdInt"

const (
      CONNECTED  = 0
      STATIC     = 1
      OSPF       = 89
      EBGP        = 8
      IBGP        = 9
	  BGP         = 17
	  PUB_SOCKET_ADDR = "ipc:///tmp/ribd.ipc"	
	  PUB_SOCKET_BGPD_ADDR = "ipc:///tmp/ribd_bgpd.ipc"
	  PUB_SOCKET_BFDD_ADDR = "ipc:///tmp/ribd_bfdd.ipc"
	  NOTIFY_ROUTE_CREATED = 1
	  NOTIFY_ROUTE_DELETED = 2
	  NOTIFY_ROUTE_INVALIDATED = 3
	  NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE = 4
	  DEFAULT_NOTIFICATION_SIZE = 128
	  RoutePolicyStateChangetoValid=1
	  RoutePolicyStateChangetoInValid = 2
	  RoutePolicyStateChangeNoChange=3
)

type RibdNotifyMsg struct {
    MsgType uint16
    MsgBuf []byte
}

type RoutelistInfo struct {
    RouteInfo ribdInt.Routes
}
type RouteReachabilityStatusMsgInfo struct {
	Network string
	IsReachable bool
	NextHopIntf ribdInt.NextHopInfo
}
