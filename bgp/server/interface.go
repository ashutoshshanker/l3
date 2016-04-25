package server

/*  Port/Interface state change manager.
 */
type IntfStateMgrIntf interface {
	Init(server *BGPServer)
	PortStateChange()
	GetIPv4Information(ifIndex int32) (string, error)
	GetIfIndex(int, int) int32
}

/*  Adding routes to rib/switch/linux interface
 */
type RouteMgrIntf interface {
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
	Init(server *BGPServer)
	ProcessBfd(peer *Peer)
}
