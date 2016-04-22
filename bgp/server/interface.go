package server

/*  Port/Interface state change manager.
 */
type IntfMgrIntf interface {
	PortStateChange()
}

/*  Adding routes to rib/switch/linux interface
 */
type RouteMgrIntf interface {
	CreateRoute()
	DeleteRoute()
}

/*  Inteface for handling policy related operations
 */
type PolicyMgrIntf interface {
	AddPolicy()
	RemovePolicy()
}
