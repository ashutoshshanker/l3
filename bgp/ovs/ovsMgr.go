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
	dbmgr          *BGPOvsdbHandler
	plugin         string
	logger         *logging.Writer
	applyPolicyCh  chan PolicyInfo
	PolicyEngineDB *policy.PolicyEngineDB
}

type OvsPolicyMgr struct {
	plugin           string
	dbmgr            *BGPOvsdbHandler
	ospf             chan bool
	static           chan bool
	connected        chan bool
	redistributeLock sync.RWMutex
}

type OvsBfdMgr struct {
	plugin string
}
