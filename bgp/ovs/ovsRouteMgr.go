package ovsMgr

import (
	"fmt"
	"l3/bgp/config"
)

/*  Constructor for route manager
 */
func NewOvsRouteMgr() *OvsRouteMgr {
	mgr := &OvsRouteMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func (mgr *OvsRouteMgr) Start() {

}

func (mgr *OvsRouteMgr) CreateRoute(cfg *config.RouteConfig) {
	fmt.Println("Create Route called in", mgr.plugin, "with configs", cfg)
}

func (mgr *OvsRouteMgr) DeleteRoute(cfg *config.RouteConfig) {

}

func (mgr *OvsRouteMgr) GetNextHopInfo(ipAddr string) (*config.NextHopInfo, error) {
	return nil, nil
}
