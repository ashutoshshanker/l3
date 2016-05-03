package ovsMgr

import (
	"utils/logging"
)

type OvsIntfMgr struct {
	plugin string
}

type OvsRouteMgr struct {
	plugin string
	logger *logging.LogFile
	dbHdl  *BGPOvsdbHandler
}

type OvsPolicyMgr struct {
	plugin    string
	Ospf      bool
	Static    bool
	Connected bool
}

type OvsBfdMgr struct {
	plugin string
}
