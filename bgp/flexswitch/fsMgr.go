package FSMgr

import (
	"fmt"
)

type FSRouteMgr struct {
	plugin string
}

type FSIntfMgr struct {
	plugin string
}

type FSPolicyMgr struct {
	plugin string
}

func NewFSIntfMgr() *FSIntfMgr {
	mgr := &FSIntfMgr{
		plugin: "ovsdb",
	}

	return mgr
}
func NewFSPolicyMgr() *FSPolicyMgr {
	mgr := &FSPolicyMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func NewFSRouteMgr() *FSRouteMgr {
	mgr := &FSRouteMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func (mgr *FSRouteMgr) CreateRoute() {
	fmt.Println("Create Route called in", mgr.plugin)
}

func (mgr *FSRouteMgr) DeleteRoute() {

}

func (mgr *FSPolicyMgr) AddPolicy() {

}

func (mgr *FSPolicyMgr) RemovePolicy() {

}

func (mgr *FSIntfMgr) PortStateChange() {

}
