package ovsMgr

import (
	"sync"
	"utils/logging"
)

type OvsIntfMgr struct {
	plugin string
}

type OvsRouteMgr struct {
	plugin string
	logger *logging.LogFile
	dbmgr  *BGPOvsdbHandler
}

type OvsPolicyMgr struct {
	plugin    string
	dbmgr     *BGPOvsdbHandler
	ospf      chan bool
	static    chan bool
	connected chan bool
	/*
		ospf             bool
		static           bool
		connected        bool
	*/
	redistributeLock sync.RWMutex
}

type OvsBfdMgr struct {
	plugin string
}
