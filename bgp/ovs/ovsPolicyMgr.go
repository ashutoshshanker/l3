package ovsMgr

/*  Constructor for policy manager
 */
func NewOvsPolicyMgr() *OvsPolicyMgr {
	mgr := &OvsPolicyMgr{
		plugin: "ovsdb",
	}

	return mgr
}

func (mgr *OvsPolicyMgr) AddPolicy() {

}

func (mgr *OvsPolicyMgr) RemovePolicy() {

}
