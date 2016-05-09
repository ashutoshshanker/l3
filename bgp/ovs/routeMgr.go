package ovsMgr

import (
	"errors"
	"fmt"
	"github.com/socketplane/libovsdb"
	"l3/bgp/config"
	_ "net"
	"utils/logging"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)

func (mgr *OvsRouteMgr) initializePolicy() {
	mgr.PolicyEngineDB = policy.NewPolicyEngineDB(mgr.logger)
	mgr.redistributeFunc = mgr.SendRoute
	mgr.PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute,
		mgr.redistributeFunc)
}

/*  Constructor for route manager
 */
func NewOvsRouteMgr(logger *logging.Writer, db *BGPOvsdbHandler) *OvsRouteMgr {
	mgr := &OvsRouteMgr{
		plugin: "ovsdb",
		dbmgr:  db,
		logger: logger,
	}

	return mgr
}

func (mgr *OvsRouteMgr) Start() {
	mgr.initializePolicy()
}

/*  This is global next hop not bgp nexthop table
 *   type RouteConfig struct {
 *	    Cost              int32
 *	    IntfType          int32
 *	    Protocol          string
 *	    NextHopIp         string
 *	    NetworkMask       string
 *	    DestinationNw     string
 *	    OutgoingInterface string
 *   }
 */
func (mgr *OvsRouteMgr) insertRoute(cfg *config.RouteConfig) {

	vrfs, ok := mgr.dbmgr.cache["VRF"]
	if !ok {
		mgr.logger.Err("No vrf entry")
		return
	}

	var k string
	for k, _ = range vrfs {

	}
	vrfUuid := []libovsdb.UUID{libovsdb.UUID{k}}
	vrfSet, _ := libovsdb.NewOvsSet(vrfUuid)

	nextHop := make(map[string]interface{})
	nextHop["ip_address"] = cfg.NextHopIp
	nextHop["type"] = "unicast"
	nextHop["selected"] = true

	nextHopOp := libovsdb.Operation{
		Op:       "insert",
		Table:    "Nexthop",
		Row:      nextHop,
		UUIDName: "nexthop",
	}

	route := map[string]interface{}{
		"address_family": "ipv4",
		"distance":       20,
		"from":           "bgp",
	}
	route["prefix"] = "172.17.0.0/16"

	nextHopU := []libovsdb.UUID{libovsdb.UUID{GoUuid: "nexthop"}}
	nextHopSet, _ := libovsdb.NewOvsSet(nextHopU)
	route["nexthops"] = nextHopSet
	route["vrf"] = vrfSet

	routeOp := libovsdb.Operation{
		Op:       "insert",
		Table:    "Route",
		Row:      route,
		UUIDName: "route",
	}
	operations := []libovsdb.Operation{nextHopOp, routeOp}

	mgr.dbmgr.Transact(operations)
}

func (mgr *OvsRouteMgr) CreateRoute(cfg *config.RouteConfig) {
	fmt.Println("Create Route called in", mgr.plugin, "with configs", cfg)
	mgr.insertRoute(cfg)
}

func (mgr *OvsRouteMgr) DeleteRoute(cfg *config.RouteConfig) {

}

func (mgr *OvsRouteMgr) GetNextHopInfo(ipAddr string) (*config.NextHopInfo, error) {
	// @TODO: jgheewala this is hack just for the demo fix this properly
	routeEntries, exists := mgr.dbmgr.cache["Route"]
	if !exists {
		return nil, errors.New("No entries in Route table")
	}
	for _, value := range routeEntries {
		mgr.logger.Info(fmt.Sprintln(value))
	}
	reachInfo := &config.NextHopInfo{
		Ipaddr:      ipAddr,
		Mask:        "255.255.0.0",
		Metric:      20,
		IsReachable: true,
	}

	return reachInfo, nil
}

func (mgr *OvsRouteMgr) ApplyPolicy(protocol string, policyName string, action string,
	conditions []*config.ConditionInfo) {

	policyDB := mgr.PolicyEngineDB.PolicyDB

	nodeGet := policyDB.Get(patriciaDB.Prefix("RedistConnect_Policy"))
	if nodeGet == nil {
		mgr.logger.Err("Policy RedistConnect_Policy not defined")
		return
	}

	node := nodeGet.(policy.Policy)
	conditionNameList := make([]string, 0)

	redistributeActionInfo := policy.RedistributeActionInfo{true, "bgp"}
	policyAction := policy.PolicyAction{
		Name:       "Redistribution",
		ActionType: policyCommonDefs.PolicyActionTypeRouteRedistribute,
		ActionInfo: redistributeActionInfo,
	}
	mgr.PolicyEngineDB.UpdateApplyPolicy(policy.ApplyPolicyInfo{node, policyAction,
		conditionNameList}, true)
	return
}

func (mgr *OvsRouteMgr) GetRoutes() ([]*config.RouteInfo, []*config.RouteInfo) {
	return nil, nil
}

func (mgr *OvsRouteMgr) SendRoute(actionInfo interface{}, conditionInfo []interface{},
	params interface{}) {

}
