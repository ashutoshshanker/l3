package ovsMgr

import (
	"sync"
	"utils/logging"
	"utils/policy"
)

type OvsIntfMgr struct {
	plugin string
}

type OvsRouteMgr struct {
	plugin           string
	logger           *logging.Writer
	dbmgr            *BGPOvsdbHandler
	PolicyEngineDB   *policy.PolicyEngineDB
	redistributeFunc policy.Policyfunc
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
