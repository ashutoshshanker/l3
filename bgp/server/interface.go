package server

/*  Port/Interface state change manager.
 */
type IntfStateMgrIntf interface {
	PortStateChange()
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
	Init()
	ProcessBfd(peer *Peer)
}
