package FSMgr

import (
	"utils/logging"
)

/*  Init policy manager with specific needs
 */
func NewFSPolicyMgr(logger *logging.Writer, fileName string) *FSPolicyMgr {
	mgr := &FSPolicyMgr{
		plugin: "ovsdb",
		logger: logger,
	}

	return mgr
}

func (mgr *FSPolicyMgr) Start() {

}

func (mgr *FSPolicyMgr) AddPolicy() {

}

func (mgr *FSPolicyMgr) RemovePolicy() {

}

func (mgr *FSPolicyMgr) AddRedistributePolicy(info string) {

}

func (mgr *FSPolicyMgr) RemoveRedistributePolicy(info string) {

}
