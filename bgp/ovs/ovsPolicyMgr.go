package ovsMgr

import (
	"fmt"
	"l3/bgp/utils"
	"sync"
)

const (
	ROUTE_TABLE = "Route"
)

/*  Constructor for policy manager
 */
func NewOvsPolicyMgr(db *BGPOvsdbHandler) *OvsPolicyMgr {
	mgr := &OvsPolicyMgr{
		plugin: "ovsdb",
		dbmgr:  db,
	}
	mgr.redistributeLock = sync.RWMutex{}

	return mgr
}

func (mgr *OvsPolicyMgr) AddPolicy() {

}

func (mgr *OvsPolicyMgr) RemovePolicy() {

}

func (mgr *OvsPolicyMgr) Start() {
	go mgr.handleRedistribute()
}

func (mgr *OvsPolicyMgr) handleRedistribute() {
	for {
		routeEntries, exists :=
			mgr.dbmgr.cache[ROUTE_TABLE]
		if !exists {
			continue
		}
		mgr.redistributeLock.RLock()
		select {
		case conn := <-mgr.connected:
			if conn {
				utils.Logger.Info(fmt.Sprintln("Send Connected Route Entries:",
					routeEntries))
			}
		case static := <-mgr.static:
			if static {
				utils.Logger.Info(fmt.Sprintln("Send Static Route Entries:",
					routeEntries))
			}
		case ospf := <-mgr.ospf:
			if ospf {
				utils.Logger.Info(fmt.Sprintln("Send Ospf Route Entries:",
					routeEntries))
			}
		}
		mgr.redistributeLock.RUnlock()
	}
}
