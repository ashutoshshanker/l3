package ovsMgr

import (
	"l3/bgp/utils"
)

/*  Constructor for policy manager
 */
func NewOvsPolicyMgr() *OvsPolicyMgr {
	mgr := &OvsPolicyMgr{
		plugin: "ovsdb",
	}

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
	switch info {
	case "connected":
		utils.Logger.Info("ADD connected route")
		mgr.connected = true
	case "static":
		utils.Logger.Info("ADD static route")
		mgr.static = true
	case "ospf":
		utils.Logger.Info("ADD ospf route")
		mgr.ospf = true
	}
}

func (mgr *OvsPolicyMgr) RemoveRedistributePolicy(info string) {
	switch info {
	case "connected":
		utils.Logger.Info("REMOVE connected route")
		mgr.connected = false
	case "static":
		utils.Logger.Info("REMOVE static route")
		mgr.static = false
	case "ospf":
		utils.Logger.Info("REMOVE ospf route")
		mgr.ospf = false
	}
}

func (mgr *OvsPolicyMgr) handleRedistribute() {

}
