// ribdPolicyEngine.go
package server

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"database/sql"
	"fmt"
	"l3/rib/ribdCommonDefs"
	"net"
	"ribd"
	"ribdInt"
	"strconv"
	"strings"
	"utils/commonDefs"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)

type TraverseAndApplyPolicyData struct {
	data       interface{}
	updatefunc policy.PolicyApplyfunc
}

func policyEngineActionRejectRoute(params interface{}) {
	routeInfo := params.(RouteParams)
	logger.Info(fmt.Sprintln("policyEngineActionRejectRoute for route ", routeInfo.destNetIp, " ", routeInfo.networkMask))
	nextHopIfTypeStr := ""
	switch routeInfo.nextHopIfType {
	case commonDefs.IfTypePort:
		nextHopIfTypeStr = "PHY"
		break
	case commonDefs.IfTypeVlan:
		nextHopIfTypeStr = "VLAN"
		break
	case commonDefs.IfTypeNull:
		nextHopIfTypeStr = "NULL"
		break
	}
	cfg := ribd.IPv4Route{
		DestinationNw:     routeInfo.destNetIp,
		Protocol:          ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)],
		OutgoingInterface: strconv.Itoa(int(routeInfo.nextHopIfIndex)),
		OutgoingIntfType:  nextHopIfTypeStr,
		Cost:              int32(routeInfo.metric),
		NetworkMask:       routeInfo.networkMask,
		NextHopIp:         routeInfo.nextHopIp}

	_, err := routeServiceHandler.ProcessRouteDeleteConfig(&cfg) //routeInfo.destNetIp, routeInfo.networkMask, ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)], routeInfo.nextHopIp) // FIBAndRIB)//,ribdCommonDefs.RoutePolicyStateChangetoInValid)
	if err != nil {
		logger.Info(fmt.Sprintln("deleting v4 route failed with err ", err))
		return
	}
}
func policyEngineActionUndoRejectRoute(conditionsList []string, params interface{}, policyStmt policy.PolicyStmt) {
	routeInfo := params.(RouteParams)
	logger.Info(fmt.Sprintln("policyEngineActionUndoRejectRoute - route: ", routeInfo.destNetIp, ":", routeInfo.networkMask, " type ", routeInfo.routeType))
	var tempRoute ribdInt.Routes
	if routeInfo.routeType == ribdCommonDefs.STATIC {
		logger.Info(fmt.Sprintln("this is a static route, fetch it from the DB"))
		DbName := PARAMSDIR + "/UsrConfDb.db"
		logger.Info(fmt.Sprintln("DB Location: ", DbName))
		dbHdl, err := sql.Open("sqlite3", DbName)
		if err != nil {
			logger.Info(fmt.Sprintln("Failed to create the handle with err ", err))
			return
		}

		if err = dbHdl.Ping(); err != nil {
			logger.Info(fmt.Sprintln("Failed to keep DB connection alive"))
			return
		}
		dbCmd := "select * from IPV4Route"
		rows, err := dbHdl.Query(dbCmd)
		if err != nil {
			logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
			return
		}
		var ipRoute IPRoute
		for rows.Next() {
			if err = rows.Scan(&ipRoute.DestinationNw, &ipRoute.NetworkMask, &ipRoute.Cost, &ipRoute.NextHopIp, &ipRoute.OutgoingIntfType, &ipRoute.OutgoingInterface, &ipRoute.Protocol); err != nil {
				logger.Info(fmt.Sprintf("DB Scan failed when iterating over IPV4Route rows with error %s\n", err))
				return
			}
			outIntf, _ := strconv.Atoi(ipRoute.OutgoingInterface)
			var outIntfType ribd.Int
			if ipRoute.OutgoingIntfType == "VLAN" {
				outIntfType = commonDefs.IfTypeVlan
			} else {
				outIntfType = commonDefs.IfTypePort
			}
			proto, _ := strconv.Atoi(ipRoute.Protocol)
			tempRoute.Ipaddr = ipRoute.DestinationNw
			tempRoute.Mask = ipRoute.NetworkMask
			tempRoute.NextHopIp = ipRoute.NextHopIp
			tempRoute.NextHopIfType = ribdInt.Int(outIntfType)
			tempRoute.IfIndex = ribdInt.Int(outIntf)
			tempRoute.Prototype = ribdInt.Int(proto)
			tempRoute.Metric = ribdInt.Int(ipRoute.Cost)

			entity, err := buildPolicyEntityFromRoute(tempRoute, params)
			if err != nil {
				logger.Err(fmt.Sprintln("Error builiding policy entity params"))
				return
			}
			if !PolicyEngineDB.ConditionCheckValid(entity, conditionsList, policyStmt) {
				logger.Info(fmt.Sprintln("This route does not qualify for reversing reject route"))
				continue
			}
			cfg := ribd.IPv4Route{
				DestinationNw:     tempRoute.Ipaddr,
				Protocol:          "STATIC",
				OutgoingInterface: ipRoute.OutgoingInterface,
				OutgoingIntfType:  ipRoute.OutgoingIntfType,
				Cost:              int32(tempRoute.Metric),
				NetworkMask:       tempRoute.Mask,
				NextHopIp:         tempRoute.NextHopIp}

			_, err = routeServiceHandler.ProcessRouteCreateConfig(&cfg) //tempRoute.Ipaddr, tempRoute.Mask, tempRoute.Metric, tempRoute.NextHopIp, tempRoute.NextHopIfType, tempRoute.IfIndex, "STATIC") //tempRoute.Prototype)
			if err != nil {
				logger.Info(fmt.Sprintf("Route create failed with err %s\n", err))
				return
			}
		}
	} else if routeInfo.routeType == ribdCommonDefs.CONNECTED {
		logger.Info(fmt.Sprintln("this is a connected route, fetch it from ASICD"))
		if !asicdclnt.IsConnected {
			logger.Info(fmt.Sprintln("Not connected to ASICD"))
			return
		}
		var currMarker asicdServices.Int
		var count asicdServices.Int
		count = 100
		for {
			logger.Info(fmt.Sprintf("Getting %d objects from currMarker %d\n", count, currMarker))
			IPIntfBulk, err := asicdclnt.ClientHdl.GetBulkIPv4IntfState(currMarker, count)
			if err != nil {
				logger.Info(fmt.Sprintln("GetBulkIPv4IntfState with err ", err))
				return
			}
			if IPIntfBulk.Count == 0 {
				logger.Info(fmt.Sprintln("0 objects returned from GetBulkIPv4IntfState"))
				return
			}
			logger.Info(fmt.Sprintf("len(IPIntfBulk.IPv4IntfStateList)  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfStateList), IPIntfBulk.Count))
			for i := 0; i < int(IPIntfBulk.Count); i++ {
				var ipMask net.IP
				ip, ipNet, err := net.ParseCIDR(IPIntfBulk.IPv4IntfStateList[i].IpAddr)
				if err != nil {
					return
				}
				ipMask = make(net.IP, 4)
				copy(ipMask, ipNet.Mask)
				ipAddrStr := ip.String()
				ipMaskStr := net.IP(ipMask).String()
				tempRoute.Ipaddr = ipAddrStr
				tempRoute.Mask = ipMaskStr
				tempRoute.NextHopIp = "0.0.0.0"
				tempRoute.NextHopIfType = ribdInt.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex))
				nextHopIfTypeStr := ""
				switch tempRoute.NextHopIfType {
				case commonDefs.IfTypePort:
					nextHopIfTypeStr = "PHY"
					break
				case commonDefs.IfTypeVlan:
					nextHopIfTypeStr = "VLAN"
					break
				case commonDefs.IfTypeNull:
					nextHopIfTypeStr = "NULL"
					break
				}
				tempRoute.IfIndex = ribdInt.Int(asicdCommonDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex))
				tempRoute.Prototype = ribdCommonDefs.CONNECTED
				tempRoute.Metric = 0
				entity, err := buildPolicyEntityFromRoute(tempRoute, params)
				if err != nil {
					logger.Err(fmt.Sprintln("Error builiding policy entity params"))
					return
				}
				if !PolicyEngineDB.ConditionCheckValid(entity, conditionsList, policyStmt) {
					logger.Info(fmt.Sprintln("This route does not qualify for reversing reject route"))
					continue
				}
				logger.Info(fmt.Sprintf("Calling createv4Route with ipaddr %s mask %s\n", ipAddrStr, ipMaskStr))
				cfg := ribd.IPv4Route{
					DestinationNw:     tempRoute.Ipaddr,
					Protocol:          "CONNECTED",
					OutgoingInterface: strconv.Itoa(int(tempRoute.IfIndex)),
					OutgoingIntfType:  nextHopIfTypeStr,
					Cost:              0,
					NetworkMask:       tempRoute.Mask,
					NextHopIp:         "0.0.0.0"}
				_, err = routeServiceHandler.ProcessRouteCreateConfig(&cfg) //ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), "CONNECTED") // FIBAndRIB, ribd.Int(len(destNetSlice)))
				if err != nil {
					logger.Info(fmt.Sprintf("Failed to create connected route for ip Addr %s/%s intfType %d intfId %d\n", ipAddrStr, ipMaskStr, ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex))))
				}
			}
			if IPIntfBulk.More == false {
				logger.Info(fmt.Sprintln("more returned as false, so no more get bulks"))
				return
			}
			currMarker = asicdServices.Int(IPIntfBulk.EndIdx)
		}
	}
}
func policyEngineUndoRouteDispositionAction(action interface{}, conditionList []interface{}, params interface{}, policyStmt policy.PolicyStmt) {
	logger.Info(fmt.Sprintln("policyEngineUndoRouteDispositionAction"))
	logger.Info(fmt.Sprintln("RouteDisposition action = ", action.(string)))
	if action.(string) == "Reject" {
		logger.Info(fmt.Sprintln("Reject action"))
		conditionNameList := make([]string, len(conditionList))
		for i := 0; i < len(conditionList); i++ {
			condition := conditionList[i].(policy.PolicyCondition)
			conditionNameList[i] = condition.Name
		}
		policyEngineActionUndoRejectRoute(conditionNameList, params, policyStmt)
	} else if action.(string) == "Accept" {
		policyEngineActionRejectRoute(params)
	}
}
func policyEngineActionUndoNetworkStatemenAdvertiseAction(actionItem interface{}, conditionsList []interface{}, params interface{}, policyStmt policy.PolicyStmt) {
	logger.Info(fmt.Sprintln("policyEngineActionUndoNetworkStatemenAdvertiseAction"))
	RouteInfo := params.(RouteParams)
	var route ribdInt.Routes
	networkStatementTargetProtocol := actionItem.(string)
	//Send a event based on target protocol
	var evt int
	evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	switch RouteProtocolTypeMapDB[networkStatementTargetProtocol] {
	case ribdCommonDefs.BGP:
		logger.Info(fmt.Sprintln("Undo network statement advertise to BGP"))
		route = ribdInt.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribdInt.Int(RouteInfo.nextHopIfType), IfIndex: ribdInt.Int(RouteInfo.nextHopIfIndex), Metric: ribdInt.Int(RouteInfo.metric), Prototype: ribdInt.Int(RouteInfo.routeType)}
		route.NetworkStatement = true
		publisherInfo, ok := PublisherInfoMap["BGP"]
		if ok {
			RedistributionNotificationSend(publisherInfo.pub_socket, route, evt)
		}
		break
	default:
		logger.Info(fmt.Sprintln("Unknown target protocol"))
	}
	UpdateRedistributeTargetMap(evt, networkStatementTargetProtocol, route)
}
func policyEngineActionUndoRedistribute(actionItem interface{}, conditionsList []interface{}, params interface{}, policyStmt policy.PolicyStmt) {
	logger.Info(fmt.Sprintln("policyEngineActionUndoRedistribute"))
	RouteInfo := params.(RouteParams)
	var route ribdInt.Routes
	redistributeActionInfo := actionItem.(policy.RedistributeActionInfo)
	//Send a event based on target protocol
	var evt int
	logger.Info(fmt.Sprintln("redistributeAction set to ", redistributeActionInfo.Redistribute))
	if redistributeActionInfo.Redistribute == true {
		logger.Info(fmt.Sprintln("evt = NOTIFY_ROUTE_DELETED"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	} else {
		logger.Info(fmt.Sprintln("evt = NOTIFY_ROUTE_CREATED"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
	}
	route = ribdInt.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribdInt.Int(RouteInfo.nextHopIfType), IfIndex: ribdInt.Int(RouteInfo.nextHopIfIndex), Metric: ribdInt.Int(RouteInfo.metric), Prototype: ribdInt.Int(RouteInfo.routeType)}
	route.RouteOrigin = ReverseRouteProtoTypeMapDB[int(RouteInfo.routeType)]
	publisherInfo, ok := PublisherInfoMap[redistributeActionInfo.RedistributeTargetProtocol]
	if ok {
		logger.Info(fmt.Sprintln("ReditributeNotificationSend event called for target protocol - ", redistributeActionInfo.RedistributeTargetProtocol))
		RedistributionNotificationSend(publisherInfo.pub_socket, route, evt)
	} else {
		logger.Info("Unknown target protocol")
	}
	/*	switch RouteProtocolTypeMapDB[redistributeActionInfo.RedistributeTargetProtocol] {
		case ribdCommonDefs.BGP:
			logger.Info(fmt.Sprintln("Redistribute to BGP"))
			route = ribdInt.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribdInt.Int(RouteInfo.nextHopIfType), IfIndex: ribdInt.Int(RouteInfo.nextHopIfIndex), Metric: ribdInt.Int(RouteInfo.metric), Prototype: ribdInt.Int(RouteInfo.routeType)}
			route.RouteOrigin = ReverseRouteProtoTypeMapDB[int(RouteInfo.routeType)]
			publisherInfo, ok := PublisherInfoMap["BGP"]
			if ok {
				RedistributionNotificationSend(publisherInfo.pub_socket, route, evt)
			}
			break
		default:
			logger.Info(fmt.Sprintln("Unknown target protocol"))
		}*/
	UpdateRedistributeTargetMap(evt, redistributeActionInfo.RedistributeTargetProtocol, route)
}
func policyEngineUpdateRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, handle patriciaDB.Item) (err error) {
	logger.Info(fmt.Sprintln("policyEngineUpdateRoute for ", prefix))

	rmapInfoRecordList := item.(RouteInfoRecordList)
	if rmapInfoRecordList.routeInfoProtocolMap == nil {
		logger.Info(fmt.Sprintln("No routes configured for this prefix"))
		return err
	}
	routeInfoList := rmapInfoRecordList.routeInfoProtocolMap[rmapInfoRecordList.selectedRouteProtocol]
	if len(routeInfoList) == 0 {
		logger.Info(fmt.Sprintln("len(routeInfoList) == 0"))
		return err
	}
	logger.Info(fmt.Sprintln("Selected route protocol = ", rmapInfoRecordList.selectedRouteProtocol))
	selectedRouteInfoRecord := routeInfoList[rmapInfoRecordList.selectedRouteIdx]
	//route := ribdInt.Routes{Ipaddr:selectedRouteInfoRecord.destNetIp.String() , Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribdInt.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: ribdInt.Int(selectedRouteInfoRecord.nextHopIfIndex), Metric: ribdInt.Int(selectedRouteInfoRecord.metric), Prototype: ribdInt.Int(selectedRouteInfoRecord.protocol), IsPolicyBasedStateValid: rmapInfoRecordList.isPolicyBasedStateValid}
	nextHopIfTypeStr := ""
	switch selectedRouteInfoRecord.nextHopIfType {
	case commonDefs.IfTypePort:
		nextHopIfTypeStr = "PHY"
		break
	case commonDefs.IfTypeVlan:
		nextHopIfTypeStr = "VLAN"
		break
	case commonDefs.IfTypeNull:
		nextHopIfTypeStr = "NULL"
		break
	}
	nextHopIf := strconv.Itoa(int(selectedRouteInfoRecord.nextHopIfIndex))
	cfg := ribd.IPv4Route{
		DestinationNw:     selectedRouteInfoRecord.destNetIp.String(),
		Protocol:          ReverseRouteProtoTypeMapDB[int(selectedRouteInfoRecord.protocol)],
		OutgoingInterface: nextHopIf,
		OutgoingIntfType:  nextHopIfTypeStr,
		Cost:              int32(selectedRouteInfoRecord.metric),
		NetworkMask:       selectedRouteInfoRecord.networkMask.String(),
		NextHopIp:         selectedRouteInfoRecord.nextHopIp.String()}
	//Even though we could potentially have multiple selected routes, calling update once for this prefix should suffice
	//routeServiceHandler.UpdateIPv4Route(&cfg, nil, nil)
	routeServiceHandler.ProcessRouteUpdateConfig(&cfg, &cfg, nil)
	return err
}
func policyEngineTraverseAndUpdate() {
	logger.Info(fmt.Sprintln("policyEngineTraverseAndUpdate"))
	RouteInfoMap.VisitAndUpdate(policyEngineUpdateRoute, nil)
}
func policyEngineActionAcceptRoute(params interface{}) {
	routeInfo := params.(RouteParams)
	logger.Info(fmt.Sprintln("policyEngineActionAcceptRoute for ip ", routeInfo.destNetIp, " and mask ", routeInfo.networkMask))
	_, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, routeInfo.metric, routeInfo.nextHopIp, routeInfo.nextHopIfType, routeInfo.nextHopIfIndex, routeInfo.routeType, routeInfo.createType, ribdCommonDefs.RoutePolicyStateChangetoValid, routeInfo.sliceIdx)
	//_, err := routeServiceHandler.InstallRoute(routeInfo)
	if err != nil {
		logger.Info(fmt.Sprintln("creating v4 route failed with err ", err))
		return
	}
}
func policyEngineActionUndoSetAdminDistance(actionItem interface{}, conditionsList []interface{}, conditionItem interface{}, policyStmt policy.PolicyStmt) {
	logger.Info(fmt.Sprintln("policyEngineActionUndoSetAdminDistance"))
	logger.Info(fmt.Sprintln("PoilcyActionTypeSetAdminDistance action to be undone"))
	if ProtocolAdminDistanceMapDB == nil {
		logger.Info(fmt.Sprintln("ProtocolAdminDistanceMap nil"))
		return
	}
	if conditionItem == nil {
		logger.Info(fmt.Sprintln("No valid condition provided for set admin distance action"))
		return
	}
	conditionInfo := conditionItem.(policy.PolicyCondition).ConditionInfo
	conditionProtocol := conditionInfo.(string)
	//case policyCommonDefs.PolicyConditionTypeProtocolMatch:
	routeDistanceConfig, ok := ProtocolAdminDistanceMapDB[conditionProtocol]
	if !ok {
		logger.Info(fmt.Sprintln("Invalid protocol provided for undo set admin distance"))
		return
	}
	routeDistanceConfig.configuredDistance = -1
	ProtocolAdminDistanceMapDB[conditionProtocol] = routeDistanceConfig
	logger.Info(fmt.Sprintln("Setting configured distance of prototype ", conditionProtocol, " to value ", 0, " default distance of this protocol is ", routeDistanceConfig.defaultDistance))
	policyEngineTraverseAndUpdate()
}
func policyEngineActionSetAdminDistance(actionItem interface{}, conditionList []interface{}, params interface{}) {
	logger.Info(fmt.Sprintln("policyEngipolicyEngineActionSetAdminDistance"))
	actionInfo := actionItem.(int)
	logger.Info(fmt.Sprintln("PoilcyActionTypeSetAdminDistance action to be applied"))
	if ProtocolAdminDistanceMapDB == nil {
		logger.Info(fmt.Sprintln("ProtocolAdminDistanceMap nil"))
		return
	}
	if conditionList == nil {
		logger.Info(fmt.Sprintln("No valid condition provided for set admin distance action"))
		return
	}
	for i := 0; i < len(conditionList); i++ {
		//case policyCommonDefs.PolicyConditionTypeProtocolMatch:
		conditionProtocol := conditionList[i].(string)
		routeDistanceConfig, ok := ProtocolAdminDistanceMapDB[conditionProtocol]
		if !ok {
			logger.Info(fmt.Sprintln("Invalid protocol provided for set admin distance"))
			return
		}
		routeDistanceConfig.configuredDistance = actionInfo
		ProtocolAdminDistanceMapDB[conditionProtocol] = routeDistanceConfig
		logger.Info(fmt.Sprintln("Setting distance of prototype ", conditionProtocol, " to value ", actionInfo))
	}
	policyEngineTraverseAndUpdate()
	return
}
func policyEngineRouteDispositionAction(action interface{}, conditionInfo []interface{}, params interface{}) {
	logger.Info(fmt.Sprintln("policyEngineRouteDispositionAction"))
	logger.Info(fmt.Sprintln("RouteDisposition action = ", action.(string)))
	if action.(string) == "Reject" {
		logger.Info(fmt.Sprintln("Reject action"))
		policyEngineActionRejectRoute(params)
	} else if action.(string) == "Accept" {
		policyEngineActionAcceptRoute(params)
	}
}
func defaultImportPolicyEngineActionFunc(actionInfo interface{}, conditionInfo []interface{}, params interface{}) {
	logger.Info(fmt.Sprintln("defaultImportPolicyEngineAction"))
	policyEngineActionAcceptRoute(params)
}

func defaultExportPolicyEngineActionFunc(actionInfo interface{}, conditionInfo []interface{}, params interface{}) {
	logger.Info(fmt.Sprintln("defaultExportPolicyEngineActionFunc"))
}
func policyEngineActionNetworkStatementAdvertise(actionInfo interface{}, conditionInfo []interface{}, params interface{}) {
	logger.Info(fmt.Sprintln("policyEngineActionNetworkStatementAdvertise"))
	var route ribdInt.Routes
	networkStatementAdvertiseTargetProtocol := actionInfo.(string)
	//Send a event based on target protocol
	RouteInfo := params.(RouteParams)
	var evt int
	if RouteInfo.createType != Invalid {
		logger.Info(fmt.Sprintln("Create type not invalid"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
	} else if RouteInfo.deleteType != Invalid {
		logger.Info(fmt.Sprintln("Delete type not invalid"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	} else {
		logger.Info(fmt.Sprintln("Create/Delete invalid,  so evt = NOTIFY_ROUTE_CREATED"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
	}
	switch RouteProtocolTypeMapDB[networkStatementAdvertiseTargetProtocol] {
	case ribdCommonDefs.BGP:
		logger.Info(fmt.Sprintln("NetworkStatemtnAdvertise to BGP"))
		route = ribdInt.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribdInt.Int(RouteInfo.nextHopIfType), IfIndex: ribdInt.Int(RouteInfo.nextHopIfIndex), Metric: ribdInt.Int(RouteInfo.metric), Prototype: ribdInt.Int(RouteInfo.routeType)}
		route.NetworkStatement = true
		publisherInfo, ok := PublisherInfoMap["BGP"]
		if ok {
			RedistributionNotificationSend(publisherInfo.pub_socket, route, evt)
		}
		break
	default:
		logger.Info(fmt.Sprintln("Unknown target protocol"))
	}
	UpdateRedistributeTargetMap(evt, networkStatementAdvertiseTargetProtocol, route)
}
func policyEngineActionRedistribute(actionInfo interface{}, conditionInfo []interface{}, params interface{}) {
	logger.Info(fmt.Sprintln("policyEngineActionRedistribute"))
	var route ribdInt.Routes
	redistributeActionInfo := actionInfo.(policy.RedistributeActionInfo)
	//Send a event based on target protocol
	RouteInfo := params.(RouteParams)
	if (RouteInfo.createType != Invalid || RouteInfo.deleteType != Invalid) && redistributeActionInfo.Redistribute == false {
		logger.Info(fmt.Sprintln("Don't redistribute action set for a route create/delete, return"))
		return
	}
	var evt int
	if RouteInfo.createType != Invalid {
		logger.Info(fmt.Sprintln("Create type not invalid"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
	} else if RouteInfo.deleteType != Invalid {
		logger.Info(fmt.Sprintln("Delete type not invalid"))
		evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	} else {
		logger.Info(fmt.Sprintln("Create/Delete invalid, redistributeAction set to ", redistributeActionInfo.Redistribute))
		if redistributeActionInfo.Redistribute == true {
			logger.Info(fmt.Sprintln("evt = NOTIFY_ROUTE_CREATED"))
			evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
		} else {
			logger.Info(fmt.Sprintln("evt = NOTIFY_ROUTE_DELETED"))
			evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
		}
	}
	if strings.Contains(ReverseRouteProtoTypeMapDB[int(RouteInfo.routeType)], redistributeActionInfo.RedistributeTargetProtocol) {
		logger.Info("Redistribute target protocol same as route source, do nothing more here")
		return
	}
	route = ribdInt.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribdInt.Int(RouteInfo.nextHopIfType), IfIndex: ribdInt.Int(RouteInfo.nextHopIfIndex), Metric: ribdInt.Int(RouteInfo.metric), Prototype: ribdInt.Int(RouteInfo.routeType)}
	route.RouteOrigin = ReverseRouteProtoTypeMapDB[int(RouteInfo.routeType)]
	publisherInfo, ok := PublisherInfoMap[redistributeActionInfo.RedistributeTargetProtocol]
	if ok {
		logger.Info(fmt.Sprintln("ReditributeNotificationSend event called for target protocol - ", redistributeActionInfo.RedistributeTargetProtocol))
		RedistributionNotificationSend(publisherInfo.pub_socket, route, evt)
	} else {
		logger.Info("Unknown target protocol")
	}

	/*	switch RouteProtocolTypeMapDB[redistributeActionInfo.RedistributeTargetProtocol] {
		case ribdCommonDefs.BGP:
			logger.Info(fmt.Sprintln("Redistribute target Protocol BGP"))
			route = ribdInt.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribdInt.Int(RouteInfo.nextHopIfType), IfIndex: ribdInt.Int(RouteInfo.nextHopIfIndex), Metric: ribdInt.Int(RouteInfo.metric), Prototype: ribdInt.Int(RouteInfo.routeType)}
			route.RouteOrigin = ReverseRouteProtoTypeMapDB[int(RouteInfo.routeType)]
			publisherInfo, ok := PublisherInfoMap["BGP"]
			if ok {
				RedistributionNotificationSend(publisherInfo.pub_socket, route, evt)
			}
			break
		default:
			logger.Info(fmt.Sprintln("Unknown target protocol"))
		}*/
	UpdateRedistributeTargetMap(evt, redistributeActionInfo.RedistributeTargetProtocol, route)
}

func UpdateRouteAndPolicyDB(policyDetails policy.PolicyDetails, params interface{}) {
	routeInfo := params.(RouteParams)
	route := ribdInt.Routes{Ipaddr: routeInfo.destNetIp, Mask: routeInfo.networkMask, NextHopIp: routeInfo.nextHopIp, NextHopIfType: ribdInt.Int(routeInfo.nextHopIfType), IfIndex: ribdInt.Int(routeInfo.nextHopIfIndex), Metric: ribdInt.Int(routeInfo.metric), Prototype: ribdInt.Int(routeInfo.routeType)}
	var op int
	if routeInfo.deleteType != Invalid {
		op = del
	} else {
		if policyDetails.EntityDeleted == false {
			logger.Info(fmt.Sprintln("Reject action was not applied, so add this policy to the route"))
			op = add
			updateRoutePolicyState(route, op, policyDetails.Policy, policyDetails.PolicyStmt)
		}
		route.PolicyHitCounter++
	}
	updatePolicyRouteMap(route, policyDetails.Policy, op)

}
func DoesRouteExist(params interface{}) (exists bool) {
	//check if the route still exists - it may have been deleted by the previous statement action
	routeDeleted := false
	routeInfo := params.(RouteParams)
	ipPrefix, err := getNetowrkPrefixFromStrings(routeInfo.destNetIp, routeInfo.networkMask)
	if err != nil {
		logger.Info(fmt.Sprintln("Error when getting ipPrefix, err= ", err))
		return
	}
	routeInfoRecordList := RouteInfoMap.Get(ipPrefix)
	if routeInfoRecordList == nil {
		logger.Info(fmt.Sprintln("Route for this prefix no longer exists"))
		routeDeleted = true
	} else {
		if routeInfoRecordList.(RouteInfoRecordList).selectedRouteProtocol != ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)] {
			logger.Info(fmt.Sprintln("this protocol is not the selected route anymore", err))
			routeDeleted = true
		} else {
			routeInfoList := routeInfoRecordList.(RouteInfoRecordList).routeInfoProtocolMap[routeInfoRecordList.(RouteInfoRecordList).selectedRouteProtocol]
			if routeInfoList == nil {
				logger.Info(fmt.Sprintln("Route no longer exists for this protocol"))
				routeDeleted = true
			} else {
				routeFound := false
				route := ribdInt.Routes{Ipaddr: routeInfo.destNetIp, Mask: routeInfo.networkMask, NextHopIp: routeInfo.nextHopIp, NextHopIfType: ribdInt.Int(routeInfo.nextHopIfType), IfIndex: ribdInt.Int(routeInfo.nextHopIfIndex), Metric: ribdInt.Int(routeInfo.metric), Prototype: ribdInt.Int(routeInfo.routeType)}
				for i := 0; i < len(routeInfoList); i++ {
					testRoute := ribdInt.Routes{Ipaddr: routeInfoList[i].destNetIp.String(), Mask: routeInfoList[i].networkMask.String(), NextHopIp: routeInfoList[i].nextHopIp.String(), NextHopIfType: ribdInt.Int(routeInfoList[i].nextHopIfType), IfIndex: ribdInt.Int(routeInfoList[i].nextHopIfIndex), Metric: ribdInt.Int(routeInfoList[i].metric), Prototype: ribdInt.Int(routeInfoList[i].protocol), IsPolicyBasedStateValid: routeInfoList[i].isPolicyBasedStateValid}
					if isSameRoute(testRoute, route) {
						logger.Info(fmt.Sprintln("Route still exists"))
						routeFound = true
					}
				}
				if !routeFound {
					logger.Info(fmt.Sprintln("This specific route no longer exists"))
					routeDeleted = true
				}
			}
		}
	}
	exists = !routeDeleted
	return exists
}
func PolicyEngineFilter(route ribdInt.Routes, policyPath int, params interface{}) {
	logger.Info(fmt.Sprintln("PolicyEngineFilter"))
	var policyPath_Str string
	if policyPath == policyCommonDefs.PolicyPath_Import {
		policyPath_Str = "Import"
	} else if policyPath == policyCommonDefs.PolicyPath_Export {
		policyPath_Str = "Export"
	} else if policyPath == policyCommonDefs.PolicyPath_All {
		policyPath_Str = "ALL"
		logger.Info(fmt.Sprintln("policy path ", policyPath_Str, " unexpected in this function"))
		return
	}
	routeInfo := params.(RouteParams)
	logger.Info(fmt.Sprintln("PolicyEngineFilter for policypath ", policyPath_Str, "createType = ", routeInfo.createType, " deleteType = ", routeInfo.deleteType, " route: ", route.Ipaddr, ":", route.Mask, " protocol type: ", route.Prototype))
	entity, err := buildPolicyEntityFromRoute(route, params)
	if err != nil {
		logger.Info(fmt.Sprintln("Error building policy params"))
		return
	}
	entity.PolicyList = make([]string, 0)
	for j := 0; j < len(route.PolicyList); j++ {
		entity.PolicyList = append(entity.PolicyList, route.PolicyList[j])
	}
	PolicyEngineDB.PolicyEngineFilter(entity, policyPath, params)
	var op int
	if routeInfo.deleteType != Invalid {
		op = delAll //wipe out the policyList
		updateRoutePolicyState(route, op, "", "")
	}
}

func policyEngineApplyForRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, traverseAndApplyPolicyDataInfo patriciaDB.Item) (err error) {
	logger.Info(fmt.Sprintln("policyEngineApplyForRoute"))
	traverseAndApplyPolicyData := traverseAndApplyPolicyDataInfo.(TraverseAndApplyPolicyData)
	rmapInfoRecordList := item.(RouteInfoRecordList)
	if rmapInfoRecordList.routeInfoProtocolMap == nil {
		logger.Info(fmt.Sprintln("rmapInfoRecordList.routeInfoProtocolMap) = nil"))
		return err
	}
	logger.Info(fmt.Sprintln("Selected route protocol = ", rmapInfoRecordList.selectedRouteProtocol))
	selectedRouteList := rmapInfoRecordList.routeInfoProtocolMap[rmapInfoRecordList.selectedRouteProtocol]
	if len(selectedRouteList) == 0 {
		logger.Info(fmt.Sprintln("len(selectedRouteList) == 0"))
		return err
	}
	for i := 0; i < len(selectedRouteList); i++ {
		selectedRouteInfoRecord := selectedRouteList[i]
		if destNetSlice[selectedRouteInfoRecord.sliceIdx].isValid == false {
			continue
		}
		policyRoute := ribdInt.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribdInt.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: ribdInt.Int(selectedRouteInfoRecord.nextHopIfIndex), Metric: ribdInt.Int(selectedRouteInfoRecord.metric), Prototype: ribdInt.Int(selectedRouteInfoRecord.protocol), IsPolicyBasedStateValid: rmapInfoRecordList.isPolicyBasedStateValid}
		params := RouteParams{destNetIp: policyRoute.Ipaddr, networkMask: policyRoute.Mask, routeType: ribd.Int(policyRoute.Prototype), nextHopIp: selectedRouteInfoRecord.nextHopIp.String(), sliceIdx: ribd.Int(policyRoute.SliceIdx), createType: Invalid, deleteType: Invalid}
		entity, err := buildPolicyEntityFromRoute(policyRoute, params)
		if err != nil {
			logger.Err(fmt.Sprintln("Error builiding policy entity params"))
			return err
		}
		entity.PolicyList = make([]string, 0)
		for j := 0; j < len(rmapInfoRecordList.policyList); j++ {
			entity.PolicyList = append(entity.PolicyList, rmapInfoRecordList.policyList[j])
		}
		traverseAndApplyPolicyData.updatefunc(entity, traverseAndApplyPolicyData.data, params)
	}
	return err
}
func policyEngineTraverseAndApply(data interface{}, updatefunc policy.PolicyApplyfunc) {
	logger.Info(fmt.Sprintln("PolicyEngineTraverseAndApply - traverse routing table and apply policy "))
	traverseAndApplyPolicyData := TraverseAndApplyPolicyData{data: data, updatefunc: updatefunc}
	RouteInfoMap.VisitAndUpdate(policyEngineApplyForRoute, traverseAndApplyPolicyData)
}
func policyEngineTraverseAndReverse(policyItem interface{}) {
	policy := policyItem.(policy.Policy)
	logger.Info(fmt.Sprintln("PolicyEngineTraverseAndReverse - traverse routing table and inverse policy actions", policy.Name))
	ext := policy.Extensions.(PolicyExtensions)
	if ext.routeList == nil {
		logger.Info(fmt.Sprintln("No route affected by this policy, so nothing to do"))
		return
	}
	var policyRoute ribdInt.Routes
	var params RouteParams
	for idx := 0; idx < len(ext.routeInfoList); idx++ {
		policyRoute = ext.routeInfoList[idx]
		params = RouteParams{destNetIp: policyRoute.Ipaddr, networkMask: policyRoute.Mask, routeType: ribd.Int(policyRoute.Prototype), sliceIdx: ribd.Int(policyRoute.SliceIdx), createType: Invalid, deleteType: Invalid}
		ipPrefix, err := getNetowrkPrefixFromStrings(ext.routeInfoList[idx].Ipaddr, ext.routeInfoList[idx].Mask)
		if err != nil {
			logger.Info(fmt.Sprintln("Invalid route ", ext.routeList[idx]))
			continue
		}
		entity, err := buildPolicyEntityFromRoute(policyRoute, params)
		if err != nil {
			logger.Err(fmt.Sprintln("Error builiding policy entity params"))
			return
		}
		PolicyEngineDB.PolicyEngineUndoPolicyForEntity(entity, policy, params)
		deleteRoutePolicyState(ipPrefix, policy.Name)
		PolicyEngineDB.DeletePolicyEntityMapEntry(entity, policy.Name)
	}
}
