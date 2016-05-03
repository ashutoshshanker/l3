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

func (mgr *OvsPolicyMgr) AddRedistributePolicy(info string) {
	mgr.redistributeLock.Lock()
	switch info {
	case "connected":
		utils.Logger.Info("ADD connected route")
		mgr.connected <- true
	case "static":
		utils.Logger.Info("ADD static route")
		mgr.static <- true
	case "ospf":
		utils.Logger.Info("ADD ospf route")
		mgr.ospf <- true
	}
	mgr.redistributeLock.Unlock()
}

func (mgr *OvsPolicyMgr) RemoveRedistributePolicy(info string) {
	mgr.redistributeLock.Lock()
	switch info {
	case "connected":
		utils.Logger.Info("REMOVE connected route")
		mgr.connected <- false
	case "static":
		utils.Logger.Info("REMOVE static route")
		mgr.static <- false
	case "ospf":
		utils.Logger.Info("REMOVE ospf route")
		mgr.ospf <- false
	}
	mgr.redistributeLock.Unlock()
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
