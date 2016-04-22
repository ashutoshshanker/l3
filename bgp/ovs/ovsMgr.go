package ovsdbHandler

import (
	"fmt"
	"l3/bgp/server"
)

type OvsIntfMgr struct {
	IntfMgr   server.IntfMgrIntf
	PolicyMgr server.PolicyMgrIntf
	RouteMgr  server.RouteMgrIntf
}

func NewOvsIntfMgr() *OvsIntfMgr {
	mgr := new(OvsIntfMgr)
	return mgr
}

func (mgr *OvsIntfMgr) CreateRoute() {
	fmt.Println("Create Route called in ovsdb manager")
}

func (mgr *OvsIntfMgr) DeleteRoute() {

}

func (mgr *OvsIntfMgr) AddPolicy() {

}

func (mgr *OvsIntfMgr) RemovePolicy() {

}
