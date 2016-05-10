package ovsMgr

import (
	"errors"
	"fmt"
	"github.com/socketplane/libovsdb"
	"l3/bgp/api"
	"l3/bgp/config"
	"l3/bgp/utils"
	"net"
	"strings"
	"utils/logging"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)

func (mgr *OvsRouteMgr) hackPolicyDB() {
	cfg := policy.PolicyStmtConfig{
		Name:            "RedistConnect",
		MatchConditions: "all",
	}
	cfg.Actions = append(cfg.Actions, "permit")
	err := mgr.PolicyEngineDB.CreatePolicyStatement(cfg)
	if err != nil {
		mgr.logger.Err(fmt.Sprintln("Creating Policy, failed, error", err))
	}

	dcfg := policy.PolicyDefinitionConfig{
		Name: "RedistConnect_Policy",
	}

	pstmt := policy.PolicyDefinitionStmtPrecedence{1, cfg.Name}
	dcfg.PolicyDefinitionStatements = append(dcfg.PolicyDefinitionStatements, pstmt)

	err = mgr.PolicyEngineDB.CreatePolicyDefinition(dcfg)
	if err != nil {
		mgr.logger.Err(fmt.Sprintln("Creating definition, failed error", err))
	}
}

func (mgr *OvsRouteMgr) initializePolicy() {
	mgr.PolicyEngineDB = policy.NewPolicyEngineDB(mgr.logger)
	mgr.redistributeFunc = mgr.SendRoute
	mgr.PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute,
		mgr.redistributeFunc)
	mgr.PolicyEngineDB.SetTraverseAndApplyPolicyFunc(mgr.TraverseAndApply)

	mgr.hackPolicyDB()
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
	mgr.logger.Info(fmt.Sprintln("OVS Route Manager Apply Policy Called:", protocol,
		policyName, action, conditions))
	policyDB := mgr.PolicyEngineDB.PolicyDB

	nodeGet := policyDB.Get(patriciaDB.Prefix(policyName))
	if nodeGet == nil {
		mgr.logger.Err("Policy " + policyName + " not defined")
		return
	}

	node := nodeGet.(policy.Policy)
	conditionNameList := make([]string, 0)

	redistributeActionInfo := policy.RedistributeActionInfo{true, protocol}
	policyAction := policy.PolicyAction{
		Name:       "Redistribution",
		ActionType: policyCommonDefs.PolicyActionTypeRouteRedistribute,
		ActionInfo: redistributeActionInfo,
	}
	mgr.logger.Info(fmt.Sprintln("OVS Route Manager Apply Policy:", protocol, policyName, action,
		conditions))
	mgr.PolicyEngineDB.UpdateApplyPolicy(policy.ApplyPolicyInfo{node, policyAction,
		conditionNameList}, true)
	return
}

func (mgr *OvsRouteMgr) GetRoutes() ([]*config.RouteInfo, []*config.RouteInfo) {
	return nil, nil
}

func (mgr *OvsRouteMgr) SendRoute(actionInfo interface{}, conditionInfo []interface{},
	params interface{}) {
	mgr.logger.Info(fmt.Sprintln("Send route", params))

	routes := make([]*config.RouteInfo, 0)
	routes = append(routes, params.(*config.RouteInfo))
	mgr.logger.Info(fmt.Sprintln("Routes:", routes))
	api.SendRouteNotification(routes, make([]*config.RouteInfo, 0))
}

/*
type PolicyEngineFilterEntityParams struct {
	DestNetIp        string //CIDR format
	NextHopIp        string
	RouteProtocol    string
	CreatePath       bool
	DeletePath       bool
	PolicyList       []string
	PolicyHitCounter int
}
type RouteInfo struct {
	Ipaddr           string
	Mask             string
	NextHopIp        string
	Prototype        int
	NetworkStatement bool
	RouteOrigin      string
}
*/

func uitoa(val uint) string {
	var buf [32]byte // big enough for int64
	i := len(buf) - 1
	for val >= 10 {
		buf[i] = byte(val%10 + '0')
		i--
		val /= 10
	}
	buf[i] = byte(val + '0')
	return string(buf[i:])
}

func (mgr *OvsRouteMgr) TraverseAndApply(data interface{}, updatefunc policy.PolicyApplyfunc) {
	mgr.logger.Info("Traverse route")

	// entity is for policyDB, params is for the sendRoute
	routeEntries, exists :=
		mgr.dbmgr.cache[ROUTE_TABLE]
	if !exists {
		return
	}
	for _, value := range routeEntries {
		entity := policy.PolicyEngineFilterEntityParams{}
		dstIp, ok := value.Fields["prefix"].(string)
		if !ok {
			utils.Logger.Err("No prefix configured")
			continue
		}
		entity.DestNetIp = dstIp
		entity.RouteProtocol = strings.ToUpper(value.Fields["from"].(string))
		entity.NextHopIp = "0.0.0.0"
		ip, ipnet, _ := net.ParseCIDR(dstIp)
		p4 := ipnet.Mask
		mask := uitoa(uint(p4[0])) + "." +
			uitoa(uint(p4[1])) + "." +
			uitoa(uint(p4[2])) + "." +
			uitoa(uint(p4[3]))
		params := &config.RouteInfo{
			Ipaddr:    ip.String(),
			Mask:      mask,
			NextHopIp: entity.NextHopIp,
		}
		mgr.logger.Info(fmt.Sprintln("entity:", entity, "params:", params))
		updatefunc(entity, data, params)
		/*
					if value.Fields["from"] == "connected" {
			utils.Logger.Info(fmt.Sprintln("Key:", key))
			utils.Logger.Info(fmt.Sprintln("Value:", value))
			nhId, ok = value.Fields["nexthops"].(libovsdb.UUID)
			if !ok {
				utils.Logger.Err(fmt.Sprintln("No next hop configured for",
					value.Fields["prefix"]))
				continue
			}
			utils.Logger.Info("nh uuid: " + nhId.GoUuid)
			nhs, exists := mgr.dbmgr.cache["Nexthop"]
			if len(nhs) < 1 {
				utils.Logger.Err(fmt.Sprintln("No next hop configured for",
					value.Fields["prefix"]))
				continue
			}
			utils.Logger.Info(fmt.Sprintln("nhs:", nhs))
			nh, exists := nhs[nhId.GoUuid]
			utils.Logger.Info(fmt.Sprintln("nh:", nh))
			if !exists {
				utils.Logger.Err(fmt.Sprintln("No next hop configured for",
					value.Fields["prefix"]))
				continue
			}
			portId, ok := nh.Fields["ports"].(libovsdb.UUID)
			if !ok {
				utils.Logger.Err(fmt.Sprintln("No port information for",
					value.Fields["prefix"]))
				continue
			}
			utils.Logger.Info(fmt.Sprintln("PortID information is", portId.GoUuid))
			ports, exists := mgr.dbmgr.cache["Port"]
			if len(ports) < 1 {
				utils.Logger.Err(fmt.Sprintln("No entry for", portId.GoUuid,
					"in Port Table"))
				continue
			}
			port, exists := ports[portId.GoUuid]
			if !exists {
				utils.Logger.Err(fmt.Sprintln("No entry for", portId.GoUuid,
					"in Port Table"))
				continue
			}
			ip := port.Fields["ip4_address"]
			utils.Logger.Info("Ip address for the port is " + ip.(string))
					}
		*/
	}

}
