package ribdCommonDefs
import "ribd"

const (
      CONNECTED  = 0
      STATIC     = 1
      OSPF       = 89
      BGP        = 8
	  PUB_SOCKET_ADDR = "ipc:///tmp/ribd.ipc"	
	  NOTIFY_ROUTE_DELETED = 1
	  NOTIFY_ROUTE_INVALIDATED = 2
	  DEFAULT_NOTIFICATION_SIZE = 128
)

type RibdNotifyMsg struct {
    MsgType uint16
    MsgBuf []byte
}

type RoutelistInfo struct {
    RouteInfo ribd.Routes
}
