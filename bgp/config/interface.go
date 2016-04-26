package config

/*  Port/Interface state change manager.
 */
type IntfStateMgrIntf interface {
	Init()
	PortStateChange()
	GetIPv4Information(ifIndex int32) (string, error)
	GetIfIndex(int, int) int32
}

/*  Adding routes to rib/switch/linux interface
 */
type RouteMgrIntf interface {
	Init()
	GetNextHopInfo(ipAddr string) (*NextHopInfo, error)
	CreateRoute()
	DeleteRoute()
}

/*  Interface for handling policy related operations
 */
type PolicyMgrIntf interface {
	AddPolicy()
	RemovePolicy()
}

/*  Interface for handling bfd state notifications
 */
type BfdMgrIntf interface {
	Init()
	CreateBfdSession(ipAddr string) (bool, error)
	DeleteBfdSession(ipAddr string) (bool, error)
}
