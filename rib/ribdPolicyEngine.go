// ribdPolicyEngine.go
package main

import (
     "ribd"
	 "utils/policy"
	 "utils/netUtils"
	 "utils/policy/policyCommonDefs"
	 "l3/rib/ribdCommonDefs"
)

func policyEngineActionRejectRoute(params interface{}) {
	routeInfo := params.(RouteParams)
    logger.Println("policyEngineActionRejectRoute for route ", routeInfo.destNetIp, " ", routeInfo.networkMask)
  _, err := routeServiceHandler.DeleteV4Route(routeInfo.destNetIp, routeInfo.networkMask, ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)],routeInfo.nextHopIp)// FIBAndRIB)//,ribdCommonDefs.RoutePolicyStateChangetoInValid)
	  if err != nil {
		logger.Println("deleting v4 route failed with err ", err)
		return
	  }
	
}
func policyEngineActionAcceptRoute(params interface{}) {
	routeInfo := params.(RouteParams)
    logger.Println("policyEngineActionAcceptRoute for ip ", routeInfo.destNetIp, " and mask ", routeInfo.networkMask)
	_, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, routeInfo.metric, routeInfo.nextHopIp, routeInfo.nextHopIfType, routeInfo.nextHopIfIndex, routeInfo.routeType, routeInfo.createType, ribdCommonDefs.RoutePolicyStateChangetoValid,routeInfo.sliceIdx)
	if err != nil {
	   logger.Println("creating v4 route failed with err ", err)
	   return
	}
}
func policyEngineRouteDispositionAction(action interface {}, params interface {}) {
	logger.Println("policyEngineRouteDispositionAction")
	logger.Println("RouteDisposition action = ", action.(string))
    if action.(string) == "Reject" {
        logger.Println("Reject action")
		policyEngineActionRejectRoute(params)
	    } else if action.(string) == "Accept"{
            policyEngineActionAcceptRoute(params)
		}
}
func defaultImportPolicyEngineActionFunc(actionInfo interface{},params interface{}){
	logger.Println("defaultImportPolicyEngineAction")
	policyEngineActionAcceptRoute(params)
}

func defaultExportPolicyEngineActionFunc(actionInfo interface{},params interface{}){
	logger.Println("defaultExportPolicyEngineActionFunc")
}
func policyEngineActionRedistribute( actionInfo interface{}, params interface {}) {
	logger.Println("policyEngineActionRedistribute")
	redistributeActionInfo := actionInfo.(policy.RedistributeActionInfo)
	//Send a event based on target protocol
    RouteInfo := params.(RouteParams) 
	if ((RouteInfo.createType != Invalid || RouteInfo.deleteType != Invalid ) && redistributeActionInfo.Redistribute == false) {
		logger.Println("Don't redistribute action set for a route create/delete, return")
		return
	}
	var evt int
	if RouteInfo.createType != Invalid {
		logger.Println("Create type not invalid")
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
	} else if RouteInfo.deleteType != Invalid {
		logger.Println("Delete type not invalid")
		evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	} else {
		logger.Println("Create/Delete invalid, redistributeAction set to ", redistributeActionInfo.Redistribute)
		if redistributeActionInfo.Redistribute == true {
			logger.Println("evt = NOTIFY_ROUTE_CREATED")
			evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
		} else
		{
			logger.Println("evt = NOTIFY_ROUTE_DELETED")
			evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
		}
	}
    switch RouteProtocolTypeMapDB[redistributeActionInfo.RedistributeTargetProtocol] {
      case ribdCommonDefs.BGP:
        logger.Println("Redistribute to BGP")
		route := ribd.Routes{Ipaddr: RouteInfo.destNetIp, Mask: RouteInfo.networkMask, NextHopIp: RouteInfo.nextHopIp, NextHopIfType: ribd.Int(RouteInfo.nextHopIfType), IfIndex: RouteInfo.nextHopIfIndex, Metric: RouteInfo.metric, Prototype: ribd.Int(RouteInfo.routeType)}
        RouteNotificationSend(RIBD_BGPD_PUB, route, evt)
        break
      default:
        logger.Println("Unknown target protocol")	
    }
}

func UpdateRouteAndPolicyDB(policyDetails policy.PolicyDetails, params interface{}){
	routeInfo := params.(RouteParams)
    route := ribd.Routes{Ipaddr: routeInfo.destNetIp, Mask: routeInfo.networkMask, NextHopIp: routeInfo.nextHopIp, NextHopIfType: ribd.Int(routeInfo.nextHopIfType), IfIndex: routeInfo.nextHopIfIndex, Metric: routeInfo.metric, Prototype: ribd.Int(routeInfo.routeType)}
	var op int
	if routeInfo.deleteType != Invalid {
		op = del
	} else {
	    if policyDetails.EntityDeleted == false{
		  logger.Println("Reject action was not applied, so add this policy to the route")
		  op = add
	      updateRoutePolicyState(route, op, policyDetails.Policy, policyDetails.PolicyStmt)
        } 	 
        route.PolicyHitCounter++
	    addPolicyRouteMapEntry(&route, policyDetails.Policy, policyDetails.PolicyStmt, policyDetails.ConditionList, policyDetails.ActionList)
	}
	updatePolicyRouteMap(route, policyDetails.Policy, op)

}
func DoesRouteExist(params interface{}) (exists bool) {
		//check if the route still exists - it may have been deleted by the previous statement action
	routeDeleted :=false
	routeInfo := params.(RouteParams)
	ipPrefix,err:=getNetowrkPrefixFromStrings(routeInfo.destNetIp, routeInfo.networkMask)
	if err != nil {
		logger.Println("Error when getting ipPrefix, err= ", err)
		return
	}
    routeInfoRecordList := RouteInfoMap.Get(ipPrefix)
    if routeInfoRecordList == nil {
	   logger.Println("Route for this prefix no longer exists")
	   routeDeleted = true
	} else {
		if routeInfoRecordList.(RouteInfoRecordList).selectedRouteProtocol != ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)] {
			logger.Println("this protocol is not the selected route anymore", err)
			routeDeleted = true
		} else {
			routeInfoList := routeInfoRecordList.(RouteInfoRecordList).routeInfoProtocolMap[routeInfoRecordList.(RouteInfoRecordList).selectedRouteProtocol]
            if routeInfoList == nil {
				logger.Println("Route no longer exists for this protocol")
				routeDeleted = true
			} else {
				routeFound := false
				route := ribd.Routes{Ipaddr: routeInfo.destNetIp, Mask: routeInfo.networkMask, NextHopIp: routeInfo.nextHopIp, NextHopIfType: ribd.Int(routeInfo.nextHopIfType), IfIndex: routeInfo.nextHopIfIndex, Metric: routeInfo.metric, Prototype: ribd.Int(routeInfo.routeType)}
				for i:=0;i<len(routeInfoList);i++ {
                     testRoute := ribd.Routes{Ipaddr: routeInfoList[i].destNetIp.String(), Mask: routeInfoList[i].networkMask.String(), NextHopIp: routeInfoList[i].nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoList[i].nextHopIfType), IfIndex: routeInfoList[i].nextHopIfIndex, Metric: routeInfoList[i].metric, Prototype: ribd.Int(routeInfoList[i].protocol), IsPolicyBasedStateValid:routeInfoList[i].isPolicyBasedStateValid}
					if isSameRoute(testRoute,route) {
						logger.Println("Route still exists")
						routeFound = true
					}
				}
				if !routeFound {
				   logger.Println("This specific route no longer exists")
				   routeDeleted = true
				}
			}
		}
	}
	exists = !routeDeleted
	return exists
}
func PolicyEngineFilter(route ribd.Routes, policyPath int, params interface{}) {
	logger.Println("PolicyEngineFilter")
	var policyPath_Str string
	if policyPath == policyCommonDefs.PolicyPath_Import {
	   policyPath_Str = "Import"
	} else if policyPath == policyCommonDefs.PolicyPath_Export {
	   policyPath_Str = "Export"
	} else if policyPath == policyCommonDefs.PolicyPath_All {
		policyPath_Str = "ALL"
		logger.Println("policy path ", policyPath_Str, " unexpected in this function")
		return
	}
    routeInfo := params.(RouteParams)
	logger.Println("PolicyEngineFilter for policypath ", policyPath_Str, "createType = ", routeInfo.createType, " deleteType = ", routeInfo.deleteType, " route: ", route.Ipaddr,":",route.Mask, " protocol type: ", route.Prototype)
    var entity policy.PolicyEngineFilterEntityParams
	destNetIp, err := netUtils.GetCIDR(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("error getting CIDR address for ", route.Ipaddr,":", route.Mask)
		return
	}
	entity.DestNetIp = destNetIp
	route.DestNetIp = destNetIp
	entity.NextHopIp = route.NextHopIp
	entity.RouteProtocol = ReverseRouteProtoTypeMapDB[int(route.Prototype)]
	if routeInfo.createType != Invalid {
		entity.CreatePath = true
	}
	if routeInfo.deleteType != Invalid {
		entity.DeletePath = true
	}
	PolicyEngineDB.PolicyEngineFilter(entity,policyPath,params)
	var op int
	if routeInfo.deleteType != Invalid {
		op = delAll		//wipe out the policyList
	    updateRoutePolicyState(route, op, "", "")
	} 
}