package ovsMgr

/*  Constructor for bfd manager
 */
func NewOvsBfdMgr() *OvsBfdMgr {
	mgr := &OvsBfdMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func (mgr *OvsBfdMgr) Start() {

}

func (mgr *OvsBfdMgr) CreateBfdSession(ipAddr string, sessionParam string) (bool, error) {
	return true, nil
}

func (mgr *OvsBfdMgr) DeleteBfdSession(ipAddr string) (bool, error) {
	return true, nil
}
