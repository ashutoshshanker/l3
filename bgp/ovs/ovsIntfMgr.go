package ovsMgr

/*  Constructor for interface manager
 */
func NewOvsIntfMgr() *OvsIntfMgr {
	mgr := &OvsIntfMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func (mgr *OvsIntfMgr) Start() {

}

func (mgr *OvsIntfMgr) GetIPv4Information(ifIndex int32) (string, error) {
	return "", nil
}

func (mgr *OvsIntfMgr) GetIfIndex(ifIndex, ifType int) int32 {
	return 1
}

func (mgr *OvsIntfMgr) PortStateChange() {

}
