package ovsMgr

import (
	"fmt"
	"l3/bgp/server"
)

type OvsIntfMgr struct {
	plugin string
}

type OvsRouteMgr struct {
	plugin string
}

type OvsPolicyMgr struct {
	plugin string
}

type OvsBfdMgr struct {
	plugin string
}

func NewOvsIntfMgr() *OvsIntfMgr {
	mgr := &OvsIntfMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func NewOvsPolicyMgr() *OvsPolicyMgr {
	mgr := &OvsPolicyMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func NewOvsRouteMgr() *OvsRouteMgr {
	mgr := &OvsRouteMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func NewOvsBfdMgr() *OvsBfdMgr {
	mgr := &OvsBfdMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func (mgr *OvsRouteMgr) CreateRoute() {
	fmt.Println("Create Route called in", mgr.plugin)
}

func (mgr *OvsRouteMgr) DeleteRoute() {

}

func (mgr *OvsRouteMgr) Init(server *server.BGPServer) {

}

/*
func (mgr *OvsRouteMgr) GetConnectedRoutes() {

}
*/

func (mgr *OvsPolicyMgr) AddPolicy() {

}

func (mgr *OvsPolicyMgr) RemovePolicy() {

}

func (mgr *OvsIntfMgr) PortStateChange() {

}

func (mgr *OvsBfdMgr) ProcessBfd(peer *server.Peer) {

}

func (mgr *OvsBfdMgr) Init(server *server.BGPServer) {

}

func (mgr *OvsIntfMgr) Init(server *server.BGPServer) {

}

func (mgr *OvsIntfMgr) GetIPv4Information(ifIndex int32) (string, error) {
	return "", nil
}

func (mgr *OvsIntfMgr) GetIfIndex(ifIndex, ifType int) int32 {
	return 1
}
