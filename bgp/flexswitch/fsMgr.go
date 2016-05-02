package FSMgr

import (
	"asicdServices"
	"bfdd"
	nanomsg "github.com/op/go-nanomsg"
	"ribd"
	"utils/logging"
)

/*  Router manager will handle all the communication with ribd
 */
type FSRouteMgr struct {
	plugin          string
	logger          *logging.Writer
	ribdClient      *ribd.RIBDServicesClient
	ribSubSocket    *nanomsg.SubSocket
	ribSubBGPSocket *nanomsg.SubSocket
}

/*  Interface manager will handle all the communication with asicd
 */
type FSIntfMgr struct {
	plugin               string
	logger               *logging.Writer
	AsicdClient          *asicdServices.ASICDServicesClient
	asicdL3IntfSubSocket *nanomsg.SubSocket
}

/*  @FUTURE: this will be using in future if FlexSwitch is planning to support
 *	     daemon which is handling policy statments
 */
type FSPolicyMgr struct {
	plugin string
	logger *logging.Writer
}

/*  BFD manager will handle all the communication with bfd daemon
 */
type FSBfdMgr struct {
	plugin       string
	logger       *logging.Writer
	bfddClient   *bfdd.BFDDServicesClient
	bfdSubSocket *nanomsg.SubSocket
}

/*  Init policy manager with specific needs
 */
func NewFSPolicyMgr(logger *logging.Writer, fileName string) *FSPolicyMgr {
	mgr := &FSPolicyMgr{
		plugin: "ovsdb",
		logger: logger,
	}

	return mgr
}

func (mgr *FSPolicyMgr) AddPolicy() {

}

func (mgr *FSPolicyMgr) RemovePolicy() {

}

func (mgr *FSIntfMgr) PortStateChange() {

}
