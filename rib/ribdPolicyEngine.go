// ribdPolicyEngine.go
package main

import (
     "ribd"
	"utils/patriciaDB"
	 "utils/policy"
	 "utils/policy/policyCommonDefs"
	 "l3/rib/ribdCommonDefs"
	 "strconv"
	"utils/commonDefs"
	"net"
    "database/sql"
	"asicdServices"
	"asicd/asicdConstDefs"
)

type TraverseAndApplyPolicyData struct {
	data interface{}
	updatefunc policy.PolicyApplyfunc
}
func policyEngineActionRejectRoute(params interface{}) {
	routeInfo := params.(RouteParams)
    logger.Println("policyEngineActionRejectRoute for route ", routeInfo.destNetIp, " ", routeInfo.networkMask)
  _, err := routeServiceHandler.DeleteV4Route(routeInfo.destNetIp, routeInfo.networkMask, ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)],routeInfo.nextHopIp)// FIBAndRIB)//,ribdCommonDefs.RoutePolicyStateChangetoInValid)
	  if err != nil {
		logger.Println("deleting v4 route failed with err ", err)
		return
	  }
}
func policyEngineActionUndoRejectRoute(conditionsList []string, params interface {}, policyStmt policy.PolicyStmt)	{
	 routeInfo := params.(RouteParams)
	 logger.Println("policyEngineActionUndoRejectRoute - route: ", routeInfo.destNetIp,":",routeInfo.networkMask," type ",routeInfo.routeType)
	 var tempRoute ribd.Routes
	 if routeInfo.routeType == ribdCommonDefs.STATIC {
		logger.Println("this is a static route, fetch it from the DB")
	    DbName := PARAMSDIR + "/UsrConfDb.db"
        logger.Println("DB Location: ", DbName)
        dbHdl, err := sql.Open("sqlite3", DbName)
        if err != nil {
          logger.Println("Failed to create the handle with err ", err)
          return 
        }

        if err = dbHdl.Ping(); err != nil {
           logger.Println("Failed to keep DB connection alive")
           return 
        }
        dbCmd := "select * from IPV4Route"
	    rows, err := dbHdl.Query(dbCmd)
	    if(err != nil) {
		   logger.Printf("DB Query failed for %s with err %s\n", dbCmd, err)
		   return 
	    }
		var ipRoute IPRoute
	    for rows.Next() {
		if err = rows.Scan(&ipRoute.DestinationNw, &ipRoute.NetworkMask,&ipRoute.Cost, &ipRoute.NextHopIp, &ipRoute.OutgoingIntfType, &ipRoute.OutgoingInterface, &ipRoute.Protocol); err != nil {
			  logger.Printf("DB Scan failed when iterating over IPV4Route rows with error %s\n", err)
			  return 
		   }
		   outIntf, _ := strconv.Atoi(ipRoute.OutgoingInterface)
		   var outIntfType ribd.Int
		   if ipRoute.OutgoingIntfType == "VLAN" {
			  outIntfType = commonDefs.L2RefTypeVlan
		   } else {
			  outIntfType = commonDefs.L2RefTypePort
		   }
		   proto, _ := strconv.Atoi(ipRoute.Protocol)
		   tempRoute.Ipaddr = ipRoute.DestinationNw
		   tempRoute.Mask = ipRoute.NetworkMask
		   tempRoute.NextHopIp = ipRoute.NextHopIp
		   tempRoute.NextHopIfType = outIntfType
		   tempRoute.IfIndex = ribd.Int(outIntf)
		   tempRoute.Prototype = ribd.Int(proto)
		   tempRoute.Metric = ribd.Int(ipRoute.Cost)
		
		   entity := buildPolicyEntityFromRoute(tempRoute, params)
		   if !PolicyEngineDB.ConditionCheckValid(entity,conditionsList,policyStmt) {
			  logger.Println("This route does not qualify for reversing reject route")
			  continue
		   }
		  _,err = routeServiceHandler.CreateV4Route(tempRoute.Ipaddr, tempRoute.Mask, tempRoute.Metric, tempRoute.NextHopIp, tempRoute.NextHopIfType,tempRoute.IfIndex, "STATIC")//tempRoute.Prototype)
		  if(err != nil) {
			logger.Printf("Route create failed with err %s\n", err)
			return 
		  }
	    }
	} else if routeInfo.routeType == ribdCommonDefs.CONNECTED {
		logger.Println("this is a connected route, fetch it from ASICD")
		if !asicdclnt.IsConnected {
			logger.Println("Not connected to ASICD")
			return
		}
	    var currMarker asicdServices.Int
	    var count asicdServices.Int
	    count = 100
	    for {
		   logger.Printf("Getting %d objects from currMarker %d\n", count, currMarker)
		   IPIntfBulk, err := asicdclnt.ClientHdl.GetBulkIPv4Intf(currMarker, count)
		   if err != nil {
			 logger.Println("GetBulkIPv4Intf with err ", err)
			 return
		   }
		   if IPIntfBulk.Count == 0 {
			logger.Println("0 objects returned from GetBulkIPv4Intf")
			return
		   }
		   logger.Printf("len(IPIntfBulk.IPv4IntfList)  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfList), IPIntfBulk.Count)
		   for i := 0; i < int(IPIntfBulk.Count); i++ {
		      var ipMask net.IP
		      ip, ipNet, err := net.ParseCIDR(IPIntfBulk.IPv4IntfList[i].IpAddr)
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
		      tempRoute.NextHopIfType = ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex))
		      tempRoute.IfIndex = ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex))
		      tempRoute.Prototype = ribdCommonDefs.CONNECTED
		      tempRoute.Metric = 0
		      entity := buildPolicyEntityFromRoute(tempRoute, params)
		      if !PolicyEngineDB.ConditionCheckValid(entity,conditionsList,policyStmt) {
			    logger.Println("This route does not qualify for reversing reject route")
			    continue
		      }
			  logger.Printf("Calling createv4Route with ipaddr %s mask %s\n", ipAddrStr, ipMaskStr)
			  _, err = routeServiceHandler.CreateV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), "CONNECTED") // FIBAndRIB, ribd.Int(len(destNetSlice)))
			  if err != nil {
				logger.Printf("Failed to create connected route for ip Addr %s/%s intfType %d intfId %d\n", ipAddrStr, ipMaskStr, ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)))
			  }
		   }
		   if IPIntfBulk.More == false {
			 logger.Println("more returned as false, so no more get bulks")
			 return
		   }
		   currMarker = asicdServices.Int(IPIntfBulk.EndIdx)
	    }
	}
}
func policyEngineUndoRouteDispositionAction(actionItem interface {}, conditionList []string, params interface {}, policyStmt policy.PolicyStmt) {
	logger.Println("policyEngineUndoRouteDispositionAction")
	action := actionItem.(policy.PolicyAction).ActionInfo
	logger.Println("RouteDisposition action = ", action.(string))
    if action.(string) == "Reject" {
        logger.Println("Reject action")
		policyEngineActionUndoRejectRoute(conditionList, params,policyStmt)
	    } else if action.(string) == "Accept"{
            policyEngineActionRejectRoute(params)
		}
}
func policyEngineActionUndoRedistribute(actionItem interface{},conditionsList []string, params interface {}, policyStmt policy.PolicyStmt) {
	logger.Println("policyEngineActionUndoRedistribute")
	RouteInfo := params.(RouteParams)
	action := actionItem.(policy.PolicyAction).ActionInfo
	redistributeActionInfo := action.(policy.RedistributeActionInfo)
	//Send a event based on target protocol
	var evt int
	logger.Println("redistributeAction set to ", redistributeActionInfo.Redistribute)
	if redistributeActionInfo.Redistribute == true {
	   logger.Println("evt = NOTIFY_ROUTE_DELETED")
	   evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	} else {
		logger.Println("evt = NOTIFY_ROUTE_CREATED")
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
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
func policyEngineUpdateRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, handle patriciaDB.Item) (err error) {
	logger.Println("policyEngineUpdateRoute for ", prefix)
	
   rmapInfoRecordList := item.(RouteInfoRecordList)
   if rmapInfoRecordList.routeInfoProtocolMap == nil {
      logger.Println("No routes configured for this prefix")
	  return err	
   }
   routeInfoList := rmapInfoRecordList.routeInfoProtocolMap[rmapInfoRecordList.selectedRouteProtocol]
   if len(routeInfoList) == 0 {
      logger.Println("len(routeInfoList) == 0")
	  return err	
   }
   logger.Println("Selected route protocol = ", rmapInfoRecordList.selectedRouteProtocol)
   selectedRouteInfoRecord := routeInfoList[rmapInfoRecordList.selectedRouteIdx]
   route := ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol), IsPolicyBasedStateValid:rmapInfoRecordList.isPolicyBasedStateValid}
   //Even though we could potentially have multiple selected routes, calling update once for this prefix should suffice
   routeServiceHandler.UpdateIPV4Route(&route, nil, nil)
   return err
}
func  policyEngineTraverseAndUpdate() {
	logger.Println("policyEngineTraverseAndUpdate")
	RouteInfoMap.VisitAndUpdate(policyEngineUpdateRoute, nil)
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
func policyEngineActionUndoSetAdminDistance(actionItem interface{},conditionsList []string, conditionItem interface {}, policyStmt policy.PolicyStmt) {
	logger.Println("policyEngineActionUndoSetAdminDistance")
	actionInfo := actionItem.(policy.PolicyAction)
	switch actionInfo.ActionType {
		case policyCommonDefs.PoilcyActionTypeSetAdminDistance:
		  logger.Println("PoilcyActionTypeSetAdminDistance action to be undone")
		  if ProtocolAdminDistanceMapDB == nil {
			   logger.Println("ProtocolAdminDistanceMap nil")
			   break	
		  }
		  if conditionItem == nil {
			 logger.Println("No valid condition provided for set admin distance action")
			 return	
		  }
	      conditionInfo := conditionItem.(policy.PolicyCondition)
		  switch conditionInfo.ConditionType {
			case policyCommonDefs.PolicyConditionTypeProtocolMatch:
			      routeDistanceConfig := ProtocolAdminDistanceMapDB[conditionInfo.ConditionInfo.(string)]
				  routeDistanceConfig.configuredDistance = -1
			      ProtocolAdminDistanceMapDB[conditionInfo.ConditionInfo.(string)] =  routeDistanceConfig
			      logger.Println("Setting configured distance of prototype ", conditionInfo.ConditionInfo.(string), " to value ", 0, " default distance of this protocol is ", routeDistanceConfig.defaultDistance)
				break
		    default:
		       logger.Println("Invalid condition type provided for undo set admin distance")
			   return	
		  }
            policyEngineTraverseAndUpdate()
	        break
		    default:
			   logger.Println("Invalid global policy action")
			   return
		}
}
func policyEngineActionSetAdminDistance(actionItem interface {}, conditionItem interface {}, params interface {}) {
	logger.Println("policyEngipolicyEngineActionSetAdminDistance")
	actionInfo := actionItem.(policy.PolicyAction)
	switch actionInfo.ActionType {
		case policyCommonDefs.PoilcyActionTypeSetAdminDistance:
		  logger.Println("PoilcyActionTypeSetAdminDistance action to be applied")
		  if ProtocolAdminDistanceMapDB == nil {
			   logger.Println("ProtocolAdminDistanceMap nil")
			   break	
		  }
		  if conditionItem == nil {
			 logger.Println("No valid condition provided for set admin distance action")
			 return	
		  }
	      conditionInfo := conditionItem.(policy.PolicyCondition)
		  switch conditionInfo.ConditionType {
			case policyCommonDefs.PolicyConditionTypeProtocolMatch:
			    routeDistanceConfig := ProtocolAdminDistanceMapDB[conditionInfo.ConditionInfo.(string)]
				routeDistanceConfig.configuredDistance = int(actionInfo.ActionInfo.(int))
			    ProtocolAdminDistanceMapDB[conditionInfo.ConditionInfo.(string)] =  routeDistanceConfig
			    logger.Println("Setting distance of prototype ", conditionInfo.ConditionInfo.(string), " to value ", actionInfo.ActionInfo.(int))
				break
		    default:
		       logger.Println("Invalid condition type provided for set admin distance")
			   return	
		  }
            policyEngineTraverseAndUpdate()
	        break
		    default:
			   logger.Println("Invalid global policy action")
			   return
		}
}
func policyEngineRouteDispositionAction(action interface {}, conditionInfo interface {}, params interface {}) {
	logger.Println("policyEngineRouteDispositionAction")
	logger.Println("RouteDisposition action = ", action.(string))
    if action.(string) == "Reject" {
        logger.Println("Reject action")
		policyEngineActionRejectRoute(params)
	    } else if action.(string) == "Accept"{
            policyEngineActionAcceptRoute(params)
		}
}
func defaultImportPolicyEngineActionFunc(actionInfo interface{},conditionInfo interface {},params interface{}){
	logger.Println("defaultImportPolicyEngineAction")
	policyEngineActionAcceptRoute(params)
}

func defaultExportPolicyEngineActionFunc(actionInfo interface{},conditionInfo interface {},params interface{}){
	logger.Println("defaultExportPolicyEngineActionFunc")
}
func policyEngineActionRedistribute( actionInfo interface{}, conditionInfo interface {}, params interface {}) {
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
    entity := buildPolicyEntityFromRoute(route, params)
	PolicyEngineDB.PolicyEngineFilter(entity,policyPath,params)
	var op int
	if routeInfo.deleteType != Invalid {
		op = delAll		//wipe out the policyList
	    updateRoutePolicyState(route, op, "", "")
	} 
}

func policyEngineApplyForRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, traverseAndApplyPolicyDataInfo patriciaDB.Item) (err error) {
   logger.Println("policyEngineApplyForRoute")	
   traverseAndApplyPolicyData := traverseAndApplyPolicyDataInfo.(TraverseAndApplyPolicyData)
   rmapInfoRecordList := item.(RouteInfoRecordList)
   if rmapInfoRecordList.routeInfoProtocolMap == nil {
      logger.Println("rmapInfoRecordList.routeInfoProtocolMap) = nil")
	  return err	
   }
   logger.Println("Selected route protocol = ", rmapInfoRecordList.selectedRouteProtocol)
   selectedRouteList := rmapInfoRecordList.routeInfoProtocolMap[rmapInfoRecordList.selectedRouteProtocol]
   if len(selectedRouteList) == 0 {
      logger.Println("len(selectedRouteList) == 0")
	  return err	
  }
  for i:=0;i<len(selectedRouteList);i++ {
     selectedRouteInfoRecord := selectedRouteList[i]	
     policyRoute := ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol), IsPolicyBasedStateValid:rmapInfoRecordList.isPolicyBasedStateValid}
     params := RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, nextHopIp: selectedRouteInfoRecord.nextHopIp.String(),sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
     entity := buildPolicyEntityFromRoute(policyRoute,params)
	 entity.PolicyList = make([]string,0)
	 for j:=0; j<len(rmapInfoRecordList.policyList) ;j++ {
		entity.PolicyList = append(entity.PolicyList,rmapInfoRecordList.policyList[j])
	 }
	 traverseAndApplyPolicyData.updatefunc(entity, traverseAndApplyPolicyData.data, params)
  }
 return err
}
func policyEngineTraverseAndApply(data interface{}, updatefunc policy.PolicyApplyfunc) {
	logger.Println("PolicyEngineTraverseAndApply - traverse routing table and apply policy ")
	traverseAndApplyPolicyData := TraverseAndApplyPolicyData{data:data, updatefunc:updatefunc}
    RouteInfoMap.VisitAndUpdate(policyEngineApplyForRoute, traverseAndApplyPolicyData)
}
func policyEngineTraverseAndReverse(policyItem interface{}) {
    policy := policyItem.(policy.Policy)
	logger.Println("PolicyEngineTraverseAndReverse - traverse routing table and inverse policy actions", policy.Name)
	ext := policy.Extensions.(PolicyExtensions)
	if ext.routeList == nil {
		logger.Println("No route affected by this policy, so nothing to do")
		return
	}
	var policyRoute ribd.Routes
	var params RouteParams
	for idx :=0;idx<len(ext.routeInfoList);idx++ {
         policyRoute = ext.routeInfoList[idx]
         params = RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
		ipPrefix, err:=getNetowrkPrefixFromStrings(ext.routeInfoList[idx].Ipaddr, ext.routeInfoList[idx].Mask)
		if err != nil {
			logger.Println("Invalid route ", ext.routeList[idx])
			continue
		}
		entity := buildPolicyEntityFromRoute(policyRoute, params)
		PolicyEngineDB.PolicyEngineUndoPolicyForEntity(entity, policy, params,)
		deleteRoutePolicyState(ipPrefix, policy.Name)
        PolicyEngineDB.DeletePolicyEntityMapEntry(entity, policy.Name)
	}
}
