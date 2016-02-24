// ribdPolicyEngine.go
package main

import (
     "ribd"
	 "utils/patriciaDB"
	 "l3/rib/ribdCommonDefs"
	"utils/policy"
	  "utils/policy/policyCommonDefs"
	 "reflect"
	 "sort"
	 "strconv"
	"utils/commonDefs"
	"net"
	"asicdServices"
	"asicd/asicdConstDefs"
	"bytes"
    "database/sql"
)
func isPolicyTypeSame(oldPolicy Policy, policy Policy) (same bool){
	if oldPolicy.exportPolicy == policy.exportPolicy && oldPolicy.importPolicy==policy.importPolicy {
		same = true
	}
	return same
}
func actionListHasAction(actionList []string, actionType int, action string) (match bool) {
	logger.Println("actionListHasAction for action ", action)
	for i:=0;i<len(actionList);i++ {
	   logger.Println("action at index ", i, " is ", actionList[i])
	   actionInfoItem:=PolicyActionsDB.Get(patriciaDB.Prefix(actionList[i]))
	   if actionInfoItem == nil {
		logger.Println("nil action")
		return false
	   }
	   actionInfo:=actionInfoItem.(PolicyAction)
	   if actionType != actionInfo.actionType {
	      continue	
	   }
		switch actionInfo.actionType {
		   case policyCommonDefs.PolicyActionTypeRouteDisposition:
			  logger.Println("RouteDisposition action = ", actionInfo.actionInfo)
			  if actionInfo.actionInfo.(string) == action {
	             match = true
			  } 
			  break
		   case policyCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action ")
			  break
		   case policyCommonDefs.PoilcyActionTypeSetAdminDistance:
		       logger.Println("PoilcyActionTypeSetAdminDistance action")
			   match = true
		       break
		   default:
		      logger.Println("Unknown type of action")
			  break
		}
	}
	return match
}
func PolicyEngineCheck(route ribd.Routes, policyType int) (actionList []string){
    protocol := ReverseRouteProtoTypeMapDB[int(route.Prototype)]
	logger.Println("PolicyEngineTest to see if there are amny policies for match protocol ", protocol)
    policyStmtList := ProtocolPolicyListDB[protocol]
	if policyStmtList == nil || len (policyStmtList) == 0{
		logger.Println("no policy statements configured for this protocol")
		return nil
	}
	for i:=0;i<len(policyStmtList);i++ {
		logger.Println("Found policy stmt ", policyStmtList[i], " for this protocol")
        policyList := PolicyStmtPolicyMapDB[policyStmtList[i]]
		if policyList == nil || len(policyList) == 0 {
			logger.Println("No policies configured for this protocol")
			return nil
		}
		for j:=0;j<len(policyList);j++ {
			logger.Println("Found policy ", policyList[j], "for this statement")
			policyStmtInfo := PolicyStmtDB.Get(patriciaDB.Prefix(policyStmtList[i]))
			if policyStmtInfo == nil {
				logger.Println("Did not find this stmt in the DB")
				return nil
			}
			policyStmt := policyStmtInfo.(PolicyStmt)
			if conditionCheckValid(route, policyStmt.conditions) {
				logger.Println("All conditions valid for this route, so this policy will be potentially applied to this route")
                 return policyStmt.actions
			}
		}
	}
	return actionList
}
func conditionCheckValid(route ribd.Routes,conditionsList []string) (valid bool) {
   logger.Println("conditionCheckValid")	
   valid = true
   if conditionsList == nil {
      logger.Println("No conditions to match, so valid")
	  return true	
   }
   for i:=0;i<len(conditionsList);i++ {
	  logger.Printf("Find policy condition number %d name %s in the condition database\n", i,conditionsList[i])
	  conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(conditionsList[i]))
	  if conditionItem == nil {
	     logger.Println("Did not find condition ", conditionsList[i], " in the condition database")	
		 continue
	  }
	  conditionInfo := conditionItem.(PolicyCondition)
	  logger.Printf("policy condition number %d type %d\n", i, conditionInfo.conditionType)
      switch conditionInfo.conditionType {
		case policyCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
		  logger.Println("PolicyConditionTypeDstIpPrefixMatch case")
		  routePrefix,err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
		  if err != nil {
			logger.Println("Invalid routePrefix for the route ", route.Ipaddr," ", route.Mask)
			return false
		  }
          condition := conditionInfo.conditionInfo.(MatchPrefixConditionInfo)
          if condition.usePrefixSet {
		   logger.Println("Need to look up Prefix set to get the prefixes")
	     } else {
	     if condition.prefix.MasklengthRange == "exact" {
			logger.Println("exact prefix match conditiontype")
            ipPrefix, err := getNetworkPrefixFromCIDR(condition.prefix.IpPrefix)
	        if err != nil {
		      logger.Println("ipPrefix invalid ")
		      return  false
	        }
			if bytes.Equal(routePrefix, ipPrefix) == false {
			   logger.Println(" Route Prefix ", routePrefix, " does not match prefix condition ", ipPrefix)
			   return false
			}
	     } else {
		   logger.Println("Masklength= ", condition.prefix.MasklengthRange)
/*		   ip, _, err := net.ParseCIDR(condition.prefix.IpPrefix)
	       if err != nil {
		      return false 
	       }
	       ipAddrStr := ip.String()*/
	      }
        }
		break
		case policyCommonDefs.PolicyConditionTypeProtocolMatch:
		  logger.Println("PolicyConditionTypeProtocolMatch case")
		  matchProto := conditionInfo.conditionInfo.(string)
		  if matchProto == ReverseRouteProtoTypeMapDB[int(route.Prototype)] {
			logger.Println("Protocol condition matches")
		  } 
		break
		default:
		  logger.Println("Not a known condition type")
          break
	  }
   }
   logger.Println("returning valid= ", valid)
   return valid
}
func policyEngineActionRejectRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionRejectRoute for route ", route.Ipaddr, " ", route.Mask)
	routeInfo := params.(RouteParams)
  _, err := routeServiceHandler.DeleteV4Route(routeInfo.destNetIp, routeInfo.networkMask, ReverseRouteProtoTypeMapDB[int(routeInfo.routeType)],routeInfo.nextHopIp)// FIBAndRIB)//,policyCommonDefs.RoutePolicyStateChangetoInValid)
	  if err != nil {
		logger.Println("deleting v4 route failed with err ", err)
		return
	  }
	
}
/*func policyEngineActionRejectRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionRejectRoute for route ", route.Ipaddr, " ", route.Mask)
	routeInfo := params.(RouteParams)
	var delType ribd.Int
	//check if route is present
	ipPrefix, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Invalid prefix")
		return
	}
	routeRecordInfoListItem := RouteInfoMap.Get(ipPrefix)
	 //if not, create it with invalid policyBasedState
	 if routeRecordInfoListItem==nil {
		logger.Println("routeRecordInfoListItem nil route not present")
		if routeInfo.routeType == ribdCommonDefs.CONNECTED || routeInfo.routeType == ribdCommonDefs.STATIC {
		   logger.Println("Connected/Static Route not present for prefix ", ipPrefix, " install it")
	       _, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, route.Metric, route.NextHopIp, route.NextHopIfType, route.IfIndex, route.Prototype, RIBOnly, ribdCommonDefs.RoutePolicyStateChangetoInValid,routeInfo.sliceIdx)
	       if err != nil {
		     logger.Println("creating v4 route failed with err ", err)
		     return
	       }
	    } else {
			logger.Println("Route type ", ReverseRouteProtoTypeMapDB[int(route.Prototype)], " rejected because of reject policy")
			return
		}
     } else { //if yes, invalidate its policyBasedState and delete from FIBOnly
	  if routeInfo.routeType == ribdCommonDefs.CONNECTED || routeInfo.routeType == ribdCommonDefs.STATIC {
		delType = FIBOnly
	  } else {
		delType = FIBAndRIB
	  }
	  _, err = deleteV4Route(routeInfo.destNetIp, routeInfo.networkMask, routeInfo.routeType, delType,ribdCommonDefs.RoutePolicyStateChangetoInValid)
	  if err != nil {
		logger.Println("deleting v4 route failed with err ", err)
		return
	  }
	}
}*/
func policyEngineActionAcceptRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionAcceptRoute for ip ", route.Ipaddr, " and mask ", route.Mask)
	routeInfo := params.(RouteParams)
	_, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, route.Metric, route.NextHopIp, route.NextHopIfType, route.IfIndex, route.Prototype, routeInfo.createType, ribdCommonDefs.RoutePolicyStateChangetoValid,routeInfo.sliceIdx)
	if err != nil {
	   logger.Println("creating v4 route failed with err ", err)
	   return
	}
}
/*func policyEngineActionAcceptRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionAcceptRoute for ip ", route.Ipaddr, " and mask ", route.Mask)
	routeInfo := params.(RouteParams)
	//check if route is present
	ipPrefix, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Invalid prefix returned with err ", err)
		return
	}
	routeRecordInfoListItem := RouteInfoMap.Get(ipPrefix)
	//if not, create route correctly
	if routeRecordInfoListItem==nil {
		logger.Println("Route not present, install it")
	   _, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, route.Metric, route.NextHopIp, route.NextHopIfType, route.IfIndex, route.Prototype, routeInfo.createType, ribdCommonDefs.RoutePolicyStateChangetoValid,routeInfo.sliceIdx)
	   if err != nil {
		  logger.Println("creating v4 route failed with err ", err)
		  return
	   }
	} else {//if yes, validate its policyBasedState, call selectv4Route and install in ASICD
	   if routeRecordInfoListItem.(RouteInfoRecordList).isPolicyBasedStateValid == false && (routeInfo.routeType == ribdCommonDefs.CONNECTED || routeInfo.routeType == ribdCommonDefs.STATIC){
	     logger.Println("Route already present but invalid, validate and install it in FIB")
	     _, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, route.Metric, route.NextHopIp, route.NextHopIfType, route.IfIndex, route.Prototype, FIBOnly,ribdCommonDefs.RoutePolicyStateChangetoValid, routeInfo.sliceIdx)
	     if err != nil {
		    logger.Println("creating v4 route failed with err ", err)
		    return
	     }
	   } else {
		   logger.Println("Route present and valid and not a static/connected route - Install in FIB and RIB")
	       _, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, route.Metric, route.NextHopIp, route.NextHopIfType, route.IfIndex, route.Prototype, routeInfo.createType, ribdCommonDefs.RoutePolicyStateChangetoValid,routeInfo.sliceIdx)
	       if err != nil {
		     logger.Println("creating v4 route failed with err ", err)
		     return
	       }
	   }
	}
}*/
func policyEngineActionRedistribute(route ribd.Routes, redistributeActionInfo RedistributeActionInfo, params interface {}) {
	logger.Println("policyEngineActionRedistribute")
	//Send a event based on target protocol
    RouteInfo := params.(RouteParams) 
	if ((RouteInfo.createType != Invalid || RouteInfo.deleteType != Invalid ) && redistributeActionInfo.redistribute == false) {
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
		logger.Println("Create/Delete invalid, redistributeAction set to ", redistributeActionInfo.redistribute)
		if redistributeActionInfo.redistribute == true {
			logger.Println("evt = NOTIFY_ROUTE_CREATED")
			evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
		} else
		{
			logger.Println("evt = NOTIFY_ROUTE_DELETED")
			evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
		}
	}
/*	if evt == ribdCommonDefs.NOTIFY_ROUTE_CREATED && route.IsPolicyBasedStateValid == false {
		logger.Println("route.isPolicyBasedStateValid invalid, so cannot send NOTIFY_ROUTE_CREATED")
		return
	}*/
    switch redistributeActionInfo.redistributeTargetProtocol {
      case ribdCommonDefs.BGP:
        logger.Println("Redistribute to BGP")
        RouteNotificationSend(RIBD_BGPD_PUB, route, evt)
        break
      default:
        logger.Println("Unknown target protocol")	
    }
}
func policyEngineActionUndoRedistribute(route ribd.Routes, redistributeActionInfo RedistributeActionInfo, params interface {},conditionsList []string) {
	logger.Println("policyEngineActionUndoRedistribute")
	//Send a event based on target protocol
	var evt int
	logger.Println("redistributeAction set to ", redistributeActionInfo.redistribute)
	if redistributeActionInfo.redistribute == true {
	   logger.Println("evt = NOTIFY_ROUTE_DELETED")
	   evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
	} else {
		logger.Println("evt = NOTIFY_ROUTE_CREATED")
		evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
	}
    switch redistributeActionInfo.redistributeTargetProtocol {
      case ribdCommonDefs.BGP:
        logger.Println("Redistribute to BGP")
        RouteNotificationSend(RIBD_BGPD_PUB, route, evt)
        break
      default:
        logger.Println("Unknown target protocol")	
    }
}
func policyEngineActionUndoRejectRoute(route ribd.Routes,params interface {}, conditionsList []string)	{
	 logger.Println("policyEngineActionUndoRejectRoute - route: ", route.Ipaddr,":",route.Mask," type ",route.Prototype)
	 var tempRoute ribd.Routes
	 if route.Prototype == ribdCommonDefs.STATIC {
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
		
		   if !conditionCheckValid(tempRoute,conditionsList) {
			  logger.Println("This route does not qualify for reversing reject route")
			  continue
		   }
		  _,err = routeServiceHandler.CreateV4Route(tempRoute.Ipaddr, tempRoute.Mask, tempRoute.Metric, tempRoute.NextHopIp, tempRoute.NextHopIfType,tempRoute.IfIndex, "STATIC")//tempRoute.Prototype)
		  if(err != nil) {
			logger.Printf("Route create failed with err %s\n", err)
			return 
		  }
	    }
	} else if route.Prototype == ribdCommonDefs.CONNECTED {
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
		
		      if !conditionCheckValid(tempRoute,conditionsList) {
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
func policyEngineUndoActionsPolicyStmt(route ribd.Routes, policy Policy, policyStmt PolicyStmt, params interface{}, conditionsAndActionsList ConditionsAndActionsList) {
	logger.Println("policyEngineUndoActionsPolicyStmt")
	if conditionsAndActionsList.actionList == nil {
		logger.Println("No actions")
		return
	}
	var i int
	for i=0;i<len(conditionsAndActionsList.actionList);i++ {
	  logger.Printf("Find policy action number %d name %s in the action database\n", i, conditionsAndActionsList.actionList[i])
	  actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[i]))
	  if actionItem == nil {
	     logger.Println("Did not find action ", conditionsAndActionsList.actionList[i], " in the action database")	
		 continue
	  }
	  action := actionItem.(PolicyAction)
	  logger.Printf("policy action number %d type %d\n", i, action.actionType)
		switch action.actionType {
		   case policyCommonDefs.PolicyActionTypeRouteDisposition:
		      logger.Println("PolicyActionTypeRouteDisposition action to be applied")
			  logger.Println("RouteDisposition action = ", action.actionInfo)
			  if action.actionInfo.(string) == "Accept" {
                 logger.Println("Accept action - undoing it by deleting")
				 policyEngineActionRejectRoute(route, params)
			  } else if action.actionInfo.(string	) == "Reject" {
			     logger.Println("Reject action applied, undo reject")
				 policyEngineActionUndoRejectRoute(route,params,conditionsAndActionsList.conditionList)	
			  }
			  break
		   case policyCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action to be applied")
			  policyEngineActionUndoRedistribute(route, action.actionInfo.(RedistributeActionInfo), params,conditionsAndActionsList.conditionList)
			  break
		   default:
		      logger.Println("Unknown type of action")
			  return
		}
	}
}
func policyEngineUndoPolicyForRoute(route ribd.Routes, policy Policy, params interface{}) {
	logger.Println("policyEngineUndoPolicyForRoute - policy name ", policy.name, "  route: ", route.Ipaddr," ", route.Mask, " type:", route.Prototype)
    ipPrefix,err:=getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Invalid prefix, err= ", err)
		return
	}
    policyRouteIndex := PolicyRouteIndex{routeIP:route.Ipaddr,routeMask:route.Mask, policy:policy.name}
	policyStmtMap := PolicyRouteMap[policyRouteIndex]
	if policyStmtMap.policyStmtMap == nil{
		logger.Println("Unexpected:None of the policy statements of this policy have been applied on this route")
		return
	}
	for stmt,conditionsAndActionsList:=range policyStmtMap.policyStmtMap {
		logger.Println("Applied policyStmtName ",stmt)
		policyStmt := PolicyStmtDB.Get(patriciaDB.Prefix(stmt))
        if policyStmt == nil {
			logger.Println("Invalid policyStmt")
			continue
		}
		policyEngineUndoActionsPolicyStmt(route,policy,policyStmt.(PolicyStmt), params, conditionsAndActionsList)
		//check if the route still exists - it may have been deleted by the previous statement action
        routeInfoRecordList := RouteInfoMap.Get(ipPrefix)
		if routeInfoRecordList == nil {
			logger.Println("this route no longer exists")
			break
		}
	}
}
func policyEngineImplementActions(route ribd.Routes, policyStmt PolicyStmt, params interface {}) (actionList []string){
	logger.Println("policyEngineImplementActions")
	if policyStmt.actions == nil {
		logger.Println("No actions")
		return actionList
	}
	var i int
	createRoute := false
	addActionToList := false
	for i=0;i<len(policyStmt.actions);i++ {
	  addActionToList = false
	  logger.Printf("Find policy action number %d name %s in the action database\n", i, policyStmt.actions[i])
	  actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[i]))
	  if actionItem == nil {
	     logger.Println("Did not find action ", policyStmt.actions[i], " in the action database")	
		 continue
	  }
	  action := actionItem.(PolicyAction)
	  logger.Printf("policy action number %d type %d\n", i, action.actionType)
		switch action.actionType {
		   case policyCommonDefs.PolicyActionTypeRouteDisposition:
		      logger.Println("PolicyActionTypeRouteDisposition action to be applied")
			  logger.Println("RouteDisposition action = ", action.actionInfo)
			  if action.actionInfo.(string) == "Reject" {
                 logger.Println("Reject action")
				 policyEngineActionRejectRoute(route, params)
	             addActionToList = true
			  } else if action.actionInfo.(string) == "Accept"{
			     createRoute = true
	             addActionToList = true
			  }
			  break
		   case policyCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action to be applied")
			  policyEngineActionRedistribute(route, action.actionInfo.(RedistributeActionInfo), params)
	          addActionToList = true
			  break
		   default:
		      logger.Println("UnknownInvalid type of action")
			  break
		}
		if addActionToList == true {
		   if actionList == nil {
		      actionList = make([]string,0)
		   }
	       actionList = append(actionList,action.name)
		}
	}
	logger.Println("createRoute = ",createRoute)
	if createRoute {
		policyEngineActionAcceptRoute(route, params)
	}
	return actionList
}
func findPrefixMatch(ipAddr string, mask string, ipPrefix patriciaDB.Prefix, policyName string)(match bool){
	policyListItem := PrefixPolicyListDB.GetLongestPrefixNode(ipPrefix)
	if policyListItem == nil {
		logger.Println("intf stored at prefix ", ipPrefix, " is nil")
		return false
	}
    if policyListItem != nil && reflect.TypeOf(policyListItem).Kind() != reflect.Slice {
		logger.Println("Incorrect data type for this prefix ")
		 return false
	}
	policyListSlice := reflect.ValueOf(policyListItem)
	for idx :=0;idx < policyListSlice.Len();idx++ {
	   prefixPolicyListInfo := policyListSlice.Index(idx).Interface().(PrefixPolicyListInfo)
	   if prefixPolicyListInfo.policyName == policyName {
	      logger.Println("Found a potential match for this prefix")
	   }
	   if prefixPolicyListInfo.lowRange == -1 && prefixPolicyListInfo.highRange == -1 {
          logger.Println("Looking for exact match condition for prefix ", prefixPolicyListInfo.ipPrefix)
		  if bytes.Equal(ipPrefix, prefixPolicyListInfo.ipPrefix) {
			 logger.Println(" Matched the prefix")
	         return true
		  }	else {
			 logger.Println(" Did not match the exact prefix")
		     return false	
		  }
	   }
	   maskIP,err := getIP(mask)
	   if err != nil {
		 logger.Println("Error getting maskIP")
		 return false
	   }
	   logger.Println("maskIP = ", maskIP)
	   maskLen,err := getPrefixLen(maskIP)
	   if err != nil {
		  logger.Println("Error getting maskLen")
		  return false
	   }
	   logger.Println("Mask len = ", maskLen)
	   if maskLen < prefixPolicyListInfo.lowRange || maskLen > prefixPolicyListInfo.highRange {
	      logger.Println("Mask range of the route ", maskLen , " not within the required mask range:", prefixPolicyListInfo.lowRange,"..", prefixPolicyListInfo.highRange)	
		  return false
	   } else {
	      logger.Println("Mask range of the route ", maskLen , " within the required mask range:", prefixPolicyListInfo.lowRange,"..", prefixPolicyListInfo.highRange)	
		  return true
	   }
	} 
	return match
}
func PolicyEngineMatchConditions(route ribd.Routes, policyStmt PolicyStmt) (match bool, conditionsList []string){
    logger.Println("policyEngineMatchConditions")
	var i int
	allConditionsMatch := true
	anyConditionsMatch := false
	addConditiontoList := false
	for i=0;i<len(policyStmt.conditions);i++ {
	  addConditiontoList = false
	  logger.Printf("Find policy condition number %d name %s in the condition database\n", i, policyStmt.conditions[i])
	  conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[i]))
	  if conditionItem == nil {
	     logger.Println("Did not find condition ", policyStmt.conditions[i], " in the condition database")	
		 continue
	  }
	  condition := conditionItem.(PolicyCondition)
	  logger.Printf("policy condition number %d type %d\n", i, condition.conditionType)
      switch condition.conditionType {
		case policyCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
		  logger.Println("PolicyConditionTypeDstIpPrefixMatch case")
		  ipPrefix,err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
		  if err != nil {
			logger.Println("Invalid ipPrefix for the route ", route.Ipaddr," ", route.Mask)
		  }
		  match := findPrefixMatch(route.Ipaddr, route.Mask, ipPrefix,policyStmt.name)
		  if match {
		    logger.Println("Found a match for this prefix")
			anyConditionsMatch = true
			addConditiontoList = true
		  }
	      /*policyListItem:= PrefixPolicyListDB.Get(ipPrefix)
		  if policyListItem == nil {
			logger.Println("no policies configured for the prefix ", ipPrefix)
			break
		  }
	      if policyListItem != nil && reflect.TypeOf(policyListItem).Kind() != reflect.Slice {
		     logger.Println("Incorrect data type for this prefix ")
		     break
	      }
		  policyListSlice := reflect.ValueOf(policyListItem)
		  for idx :=0;idx < policyListSlice.Len();idx++ {
			if policyListSlice.Index(idx).Interface().(PrefixPolicyListInfo).policyName == policyStmt.name {
				logger.Println("Found a match for this prefix")
				anyConditionsMatch = true
				addConditiontoList = true
			}
		} */
		break
		case policyCommonDefs.PolicyConditionTypeProtocolMatch:
		  logger.Println("PolicyConditionTypeProtocolMatch case")
		  matchProto := condition.conditionInfo.(string)
		  if matchProto == ReverseRouteProtoTypeMapDB[int(route.Prototype)] {
			logger.Println("Protocol condition matches")
			anyConditionsMatch = true
			addConditiontoList = true
		  } 
		break
		default:
		  logger.Println("Not a known condition type")
          break
	  }
	  if addConditiontoList == true{
		if conditionsList == nil {
		   conditionsList = make([]string,0)
		}
		conditionsList = append(conditionsList,condition.name)
	  }
	}
   if policyStmt.matchConditions == "all" && allConditionsMatch == true {
	return true,conditionsList
   }
   if policyStmt.matchConditions == "any" && anyConditionsMatch == true {
	return true,conditionsList
   }
   return match,conditionsList
}

func policyEngineApplyPolicyStmt(route *ribd.Routes, policy Policy, policyStmt PolicyStmt, policyPath int, params interface{}, hit *bool, routeDeleted *bool) {
	logger.Println("policyEngineApplyPolicyStmt - ", policyStmt.name)
	var conditionList []string
	if policyStmt.conditions == nil {
		logger.Println("No policy conditions")
		*hit=true
	} else {
	   match,ret_conditionList := PolicyEngineMatchConditions(*route, policyStmt)
	   logger.Println("match = ", match)
	   *hit = match
	   if !match {
		   logger.Println("Conditions do not match")
		   return
	   }
	   if ret_conditionList != nil {
		 if conditionList == nil {
			conditionList = make([]string,0)
		 }
		 for j:=0;j<len(ret_conditionList);j++ {
			conditionList =append(conditionList,ret_conditionList[j])
		 }
	   }
	}
	actionList := policyEngineImplementActions(*route, policyStmt, params)
	if actionListHasAction(actionList, policyCommonDefs.PolicyActionTypeRouteDisposition,"Reject") {
		logger.Println("Reject action was applied for this route")
		*routeDeleted = true
	}
	//check if the route still exists - it may have been deleted by the previous statement action
	ipPrefix,err:=getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Error when getting ipPrefix, err= ", err)
		return
	}
    routeInfoRecordList := RouteInfoMap.Get(ipPrefix)
    if routeInfoRecordList == nil {
	   logger.Println("Route for this prefix no longer exists")
	   *routeDeleted = true
	} else {
		if routeInfoRecordList.(RouteInfoRecordList).selectedRouteProtocol != ReverseRouteProtoTypeMapDB[int(route.Prototype)] {
			logger.Println("this protocol is not the selected route anymore", err)
			*routeDeleted = true
		} else {
			routeInfoList := routeInfoRecordList.(RouteInfoRecordList).routeInfoProtocolMap[routeInfoRecordList.(RouteInfoRecordList).selectedRouteProtocol]
            if routeInfoList == nil {
				logger.Println("Route no longer exists for this protocol")
				*routeDeleted = true
			} else {
				routeFound := false
				for i:=0;i<len(routeInfoList);i++ {
                     testRoute := ribd.Routes{Ipaddr: routeInfoList[i].destNetIp.String(), Mask: routeInfoList[i].networkMask.String(), NextHopIp: routeInfoList[i].nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoList[i].nextHopIfType), IfIndex: routeInfoList[i].nextHopIfIndex, Metric: routeInfoList[i].metric, Prototype: ribd.Int(routeInfoList[i].protocol), IsPolicyBasedStateValid:routeInfoList[i].isPolicyBasedStateValid}
					if isSameRoute(testRoute,*route) {
						logger.Println("Route still exists")
						routeFound = true
					}
				}
				if !routeFound {
				   logger.Println("This specific route no longer exists")
				   *routeDeleted = true
				}
			}
		}
	}
    routeInfo := params.(RouteParams)
	var op int
	if routeInfo.deleteType != Invalid {
		op = del
	} else {
	    if *routeDeleted == false{
		  logger.Println("Reject action was not applied, so add this policy to the route")
		  op = add
	      updateRoutePolicyState(*route, op, policy.name, policyStmt.name)
        } 	 
        route.PolicyHitCounter++
	    addPolicyRouteMapEntry(route, policy.name, policyStmt.name, conditionList, actionList)
	}
	updatePolicyRouteMap(*route, policy, op)
}
func policyEngineApplyPolicy(route *ribd.Routes, policy Policy, policyPath int,params interface{}, hit *bool) {
	logger.Println("policyEngineApplyPolicy - ", policy.name)
     var policyStmtKeys []int
	routeDeleted := false
	 for k:=range policy.policyStmtPrecedenceMap {
		logger.Println("key k = ", k)
		policyStmtKeys = append(policyStmtKeys,k)
	}
	sort.Ints(policyStmtKeys)
	for i:=0;i<len(policyStmtKeys);i++ {
		logger.Println("Key: ", policyStmtKeys[i], " policyStmtName ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[policyStmtKeys[i]])))
        if policyStmt == nil {
			logger.Println("Invalid policyStmt")
			continue
		}
		policyEngineApplyPolicyStmt(route,policy,policyStmt.(PolicyStmt),policyPath, params, hit, &routeDeleted)
/*		//check if the route still exists - it may have been deleted by the previous statement action
		ipPrefix,err:=getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
		if err != nil {
			logger.Println("Error when getting ipPrefix, err= ", err)
			break
		}
        routeInfoRecordList := RouteInfoMap.Get(ipPrefix)
		if routeInfoRecordList == nil {
			logger.Println("this route no longer exists")
			break
		}*/
		if routeDeleted == true {
			logger.Println("Route was deleted as a part of the policyStmt ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
             break
		}
		if *hit == true {
			if policy.matchType == "any" {
				logger.Println("Match type for policy ", policy.name, " is any and the policy stmt ", (policyStmt.(PolicyStmt)).name, " is a hit, no more policy statements will be executed")
				break
			}
		}
	}
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
    var policyKeys []int
	var policyHit bool
	idx :=0
	var policyInfo interface{}
	if policyPath == policyCommonDefs.PolicyPath_Import{
	   for k:=range ImportPolicyPrecedenceMap {
	      policyKeys = append(policyKeys,k)
	   }
	} else if policyPath == policyCommonDefs.PolicyPath_Export{
	   for k:=range ExportPolicyPrecedenceMap {
	      policyKeys = append(policyKeys,k)
	   }
	}
	sort.Ints(policyKeys)
	for ;; {
		if routeInfo.deleteType != Invalid {
			if route.PolicyList != nil {
             if idx >= len(route.PolicyList) {
				break
			 } 		
		     logger.Println("getting policy ", idx, " from route.PolicyList")
	         policyInfo = 	PolicyDB.Get(patriciaDB.Prefix(route.PolicyList[idx]))
		     idx++
			 if policyInfo.(Policy).exportPolicy && policyPath == policyCommonDefs.PolicyPath_Import || policyInfo.(Policy).importPolicy && policyPath == policyCommonDefs.PolicyPath_Export {
				logger.Println("policy ", policyInfo.(Policy).name, " not the same type as the policypath -", policyPath_Str)
				continue
			 } 
	        } else if routeInfo.deleteType != Invalid {
		      logger.Println("route.PolicyList empty and this is a delete operation for the route, so break")
               break
	        }		
	    }  else {
			//case when no policies have been applied to the route
			//need to apply the default policy
			logger.Println("idx = ", idx, " len(policyKeys):", len(policyKeys))
            if idx >= len(policyKeys) {
				break
			}		
			policyName := ""
            if policyPath == policyCommonDefs.PolicyPath_Import {
               policyName = ImportPolicyPrecedenceMap[policyKeys[idx]]
			} else if policyPath == policyCommonDefs.PolicyPath_Export {
               policyName = ExportPolicyPrecedenceMap[policyKeys[idx]]
			}
		    logger.Println("getting policy  ", idx, " policyKeys[idx] = ", policyKeys[idx]," ", policyName," from PolicyDB")
             policyInfo = PolicyDB.Get((patriciaDB.Prefix(policyName)))
			idx++
	      }
	      if policyInfo == nil {
	        logger.Println("Nil policy")
		    continue
	      }
	      policy := policyInfo.(Policy)
	      if localPolicyDB != nil && localPolicyDB[policy.localDBSliceIdx].isValid == false {
	        logger.Println("Invalid policy at localDB slice idx ", policy.localDBSliceIdx)
		    continue	
	      }		
	      policyEngineApplyPolicy(&route, policy, policyPath, params, &policyHit)
	      if policyHit {
	         logger.Println("Policy ", policy.name, " applied to the route")	
		     break
	      }
	}
	if route.PolicyHitCounter == 0{
		logger.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == policyCommonDefs.PolicyPath_Import {
		   logger.Println("Applying default import policy")
		    //TO-DO: Need to add the default policy to policyList of the route
           policyEngineActionAcceptRoute(route , params ) 
		} else if policyPath == policyCommonDefs.PolicyPath_Export {
			logger.Println("Applying default export policy")
		}
	}
	var op int
	if routeInfo.deleteType != Invalid {
		op = delAll		//wipe out the policyList
	    updateRoutePolicyState(route, op, "", "")
	} 
}
/*
func PolicyEngineFilter(route ribd.Routes, policyPath int, params interface{}) {
	logger.Println("PolicyEngineFilter")
	var policyPath_Str string
	idx :=0
	var policyInfo interface{}
	if policyPath == policyCommonDefs.PolicyPath_Import {
	   policyPath_Str = "Import"
	} else {
	   policyPath_Str = "Export"
	}
//	policyEngineCheckPolicy(route, policyPath, funcName, params)
    routeInfo := params.(RouteParams)
	for ;; {
       if route.PolicyList != nil {
		  if idx >= len(route.PolicyList) {
			break
		  }
		  logger.Println("getting policy stmt ", idx, " from route.PolicyList")
	      policyInfo = 	PolicyStmtDB.Get(patriciaDB.Prefix(route.PolicyList[idx]))
		  idx++
	   } else if routeInfo.deleteType != Invalid {
		  logger.Println("route.PolicyList empty and this is a delete operation for the route, so break")
          break
	   } else if localPolicyStmtDB == nil {
		  logger.Println("localPolicyStmt nil")
			//case when no policies have been applied to the route
			//need to apply the default policy
		   break	   
		} else {
            if idx >= len(localPolicyStmtDB) {
				break
			}		
		    logger.Println("getting policy stmt ", idx, " from localPolicyStmtDB")
            policyInfo = PolicyStmtDB.Get(localPolicyStmtDB[idx].prefix)
			idx++
	   }
	   if policyInfo == nil {
	      logger.Println("Nil policy")
		  continue
	   }
	   policyStmt := policyInfo.(PolicyStmt)
	   if policyPath == policyCommonDefs.PolicyPath_Import && policyStmt.importPolicy == false || 
	      policyPath == policyCommonDefs.PolicyPath_Export && policyStmt.exportPolicy == false {
	         logger.Println("Cannot apply the policy ", policyStmt.name, " as ", policyPath_Str, " policy")
			 continue
	   }
	   policyEngineApplyPolicy(&route, policyStmt, params)
	}
/*	if localPolicyStmtDB == nil {
		logger.Println("No policies configured, so accept the route")
        //should be replaced by default import policy action
	} else {
		for idx :=0;idx < len(localPolicyStmtDB);idx++ {
		//for idx :=0;idx < len(policList);idx++ {
			if localPolicyStmtDB[idx].isValid == false {
				continue
			}
			policyInfo := PolicyDB.Get(localPolicyStmtDB[idx].prefix)
			if policyInfo == nil {
				logger.Println("Nil policy")
				continue
			}
			policyStmt := policyInfo.(PolicyStmt)
			if policyPath == policyCommonDefs.PolicyPath_Import {
				policyPath_Str = "Import"
			} else {
				policyPath_Str = "Export"
			}
			if policyPath == policyCommonDefs.PolicyPath_Import && policyStmt.importPolicy == false || 
			   policyPath == policyCommonDefs.PolicyPath_Export && policyStmt.exportPolicy == false {
				logger.Println("Cannot apply the policy ", policyStmt.name, " as ", policyPath_Str, " policy")
				continue
			}
		    policyEngineApplyPolicy(&route, policyStmt, params)
        }
	}*/
/*	logger.Println("After policyEngineApply policyCounter = ", route.PolicyHitCounter)
	if route.PolicyHitCounter == 0{
		logger.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == policyCommonDefs.PolicyPath_Import {
		   logger.Println("Applying default import policy")
		    //TO-DO: Need to add the default policy to policyList of the route
           policyEngineActionAcceptRoute(route , params ) 
		} else if policyPath == policyCommonDefs.PolicyPath_Export {
			logger.Println("Applying default export policy")
		}
	}
	var op int
	if routeInfo.deleteType != Invalid {
		op = delAll		//wipe out the policyList
	    updateRoutePolicyState(route, op, "")
	} 
}
*/
func policyEngineApplyForRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, handle patriciaDB.Item) (err error) {
   logger.Println("policyEngineApplyForRoute %v", prefix)	
   policy := handle.(Policy)
   rmapInfoRecordList := item.(RouteInfoRecordList)
   policyHit := false
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
     if len(rmapInfoRecordList.policyList) == 0 {
	  logger.Println("This route has no policy applied to it so far, just apply the new policy")
      policyEngineApplyPolicy(&policyRoute, policy, policyCommonDefs.PolicyPath_All,params, &policyHit)
     } else {
      logger.Println("This route already has policy applied to it - len(route.PolicyList) - ", len(rmapInfoRecordList.policyList))
    
	  for i:=0;i<len(rmapInfoRecordList.policyList);i++ {
		 logger.Println("policy at index ", i)
	     policyInfo := PolicyDB.Get(patriciaDB.Prefix(rmapInfoRecordList.policyList[i]))
	     if policyInfo == nil {
		    logger.Println("Unexpected: Invalid policy in the route policy list")
	     } else {
	       oldPolicy := policyInfo.(Policy)
		   if !isPolicyTypeSame(oldPolicy, policy) {
			 logger.Println("The policy type applied currently is not the same as new policy, so apply new policy")
              policyEngineApplyPolicy(&policyRoute, policy, policyCommonDefs.PolicyPath_All,params, &policyHit)
		   } else if oldPolicy.precedence < policy.precedence {
			 logger.Println("The policy types are same and precedence of the policy applied currently is lower than the new policy, so do nothing")
			 return err
		   } else {
			logger.Println("The new policy's precedence is lower, so undo old policy's actions and apply the new policy")
			policyEngineUndoPolicyForRoute(policyRoute, oldPolicy, params)
			policyEngineApplyPolicy(&policyRoute, policy, policyCommonDefs.PolicyPath_All,params, &policyHit)
		   }
		}
	  }	
    }
  }
   return err
}
func PolicyEngineTraverseAndApply(policy Policy) {
	logger.Println("PolicyEngineTraverseAndApply - traverse routing table and apply policy ", policy.name)
    RouteInfoMap.VisitAndUpdate(policyEngineApplyForRoute, policy)
}
func PolicyEngineTraverseAndApplyPolicy(policy Policy) {
	logger.Println("PolicyEngineTraverseAndApplyPolicy -  apply policy ", policy.name)
    if policy.exportPolicy || policy.importPolicy{
	   logger.Println("Applying import/export policy to all routes")
	   PolicyEngineTraverseAndApply(policy)
	} else if policy.globalPolicy {
		logger.Println("Need to apply global policy")
		policyEngineApplyGlobalPolicy(policy)
	}
}
func PolicyEngineTraverseAndReverse(policy Policy) {
	logger.Println("PolicyEngineTraverseAndReverse - traverse routing table and inverse policy actions", policy.name)
	if policy.routeList == nil {
		logger.Println("No route affected by this policy, so nothing to do")
		return
	}
	var policyRoute ribd.Routes
	var params RouteParams
	for idx :=0;idx<len(policy.routeInfoList);idx++ {
         policyRoute = policy.routeInfoList[idx]
         params = RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
		ipPrefix, err:=getNetowrkPrefixFromStrings(policy.routeInfoList[idx].Ipaddr, policy.routeInfoList[idx].Mask)
		if err != nil {
			logger.Println("Invalid route ", policy.routeList[idx])
			continue
		}
	//for idx:=0;idx<len(policy.routeList);idx++ {
	/*	ipPrefix, err:=getNetworkPrefixFromCIDR(policy.routeList[idx])
		if err != nil {
			logger.Println("Invalid route ", policy.routeList[idx])
			continue
		}
		routeInfoRecordListItem := RouteInfoMap.Get(ipPrefix)
		if routeInfoRecordListItem == nil {
			logger.Println("routeInfoRecordListItem nil for prefix ", policy.routeList[idx])
             policyRoute = policy.routeInfoList[idx]
             params = RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
		} else {
		 routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
         selectedRouteInfoRecord := routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
         policyRoute = ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol)}
         params = RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
        }*/
		policyEngineUndoPolicyForRoute(policyRoute, policy, params,)
		deleteRoutePolicyState(ipPrefix, policy.name)
        deletePolicyRouteMapEntry(policyRoute, policy.name)
	}
}
func PolicyEngineTraverseAndReversePolicy(policy Policy){
	logger.Println("PolicyEngineTraverseAndReversePolicy -  reverse policy ", policy.name)
    if policy.exportPolicy || policy.importPolicy{
	   logger.Println("Reversing import/export policy ")
	   PolicyEngineTraverseAndReverse(policy)
	} else if policy.globalPolicy {
		logger.Println("Need to reverse global policy")
		policyEngineReverseGlobalPolicy(policy)
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
func policyEngineApplyGlobalPolicyStmt(policy Policy, policyStmt PolicyStmt) {
	logger.Println("policyEngineApplyGlobalPolicyStmt - ", policyStmt.name)
    var conditionItem interface{}=nil
//global policies can only have statements with 1 condition and 1 action
	if policyStmt.actions == nil {
		logger.Println("No policy actions defined")
		return
	}
	if policyStmt.conditions == nil {
		logger.Println("No policy conditions")
	} else {
		if len(policyStmt.conditions) > 1 {
			logger.Println("only 1 condition allowed for global policy stmt")
			return
		}
		conditionItem = PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[0]))
		if conditionItem == nil {
			logger.Println("Condition ", policyStmt.conditions[0]," not found")
			return
		}
		actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[0]))
		if actionItem == nil {
			logger.Println("Action ", policyStmt.actions[0]," not found")
			return
		}
		actionInfo := actionItem.(PolicyAction)
		switch actionInfo.actionType {
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
			  conditionInfo := conditionItem.(PolicyCondition)
			  switch conditionInfo.conditionType {
			    case policyCommonDefs.PolicyConditionTypeProtocolMatch:
			      routeDistanceConfig := ProtocolAdminDistanceMapDB[conditionInfo.conditionInfo.(string)]
				  routeDistanceConfig.configuredDistance = int(actionInfo.actionInfo.(ribd.Int))
			      ProtocolAdminDistanceMapDB[conditionInfo.conditionInfo.(string)] =  routeDistanceConfig
			      logger.Println("Setting distance of prototype ", conditionInfo.conditionInfo.(string), " to value ", actionInfo.actionInfo.(ribd.Int))
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
}
func policyEngineApplyGlobalPolicy(policy Policy) {
	logger.Println("policyEngineApplyGlobalPolicy")
     var policyStmtKeys []int
	 for k:=range policy.policyStmtPrecedenceMap {
		logger.Println("key k = ", k)
		policyStmtKeys = append(policyStmtKeys,k)
	}
	sort.Ints(policyStmtKeys)
	for i:=0;i<len(policyStmtKeys);i++ {
		logger.Println("Key: ", policyStmtKeys[i], " policyStmtName ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[policyStmtKeys[i]])))
        if policyStmt == nil {
			logger.Println("Invalid policyStmt")
			continue
		}
		policyEngineApplyGlobalPolicyStmt(policy,policyStmt.(PolicyStmt))
	}
}
func policyEngineReverseGlobalPolicyStmt(policy Policy, policyStmt PolicyStmt) {
	logger.Println("policyEngineApplyGlobalPolicyStmt - ", policyStmt.name)
    var conditionItem interface{}=nil
//global policies can only have statements with 1 condition and 1 action
	if policyStmt.actions == nil {
		logger.Println("No policy actions defined")
		return
	}
	if policyStmt.conditions == nil {
		logger.Println("No policy conditions")
	} else {
		if len(policyStmt.conditions) > 1 {
			logger.Println("only 1 condition allowed for global policy stmt")
			return
		}
		conditionItem = PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[0]))
		if conditionItem == nil {
			logger.Println("Condition ", policyStmt.conditions[0]," not found")
			return
		}
		actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[0]))
		if actionItem == nil {
			logger.Println("Action ", policyStmt.actions[0]," not found")
			return
		}
		actionInfo := actionItem.(PolicyAction)
		switch actionInfo.actionType {
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
			  conditionInfo := conditionItem.(PolicyCondition)
			  switch conditionInfo.conditionType {
			    case policyCommonDefs.PolicyConditionTypeProtocolMatch:
			      routeDistanceConfig := ProtocolAdminDistanceMapDB[conditionInfo.conditionInfo.(string)]
				  routeDistanceConfig.configuredDistance = -1
			      ProtocolAdminDistanceMapDB[conditionInfo.conditionInfo.(string)] =  routeDistanceConfig
			      logger.Println("Setting configured distance of prototype ", conditionInfo.conditionInfo.(string), " to value ", 0, " default distance of this protocol is ", routeDistanceConfig.defaultDistance)
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
}
func policyEngineReverseGlobalPolicy(policy Policy) {
	logger.Println("policyEngineReverseGlobalPolicy")
     var policyStmtKeys []int
	 for k:=range policy.policyStmtPrecedenceMap {
		logger.Println("key k = ", k)
		policyStmtKeys = append(policyStmtKeys,k)
	}
	sort.Ints(policyStmtKeys)
	for i:=0;i<len(policyStmtKeys);i++ {
		logger.Println("Key: ", policyStmtKeys[i], " policyStmtName ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[policyStmtKeys[i]])))
        if policyStmt == nil {
			logger.Println("Invalid policyStmt")
			continue
		}
		policyEngineReverseGlobalPolicyStmt(policy,policyStmt.(PolicyStmt))
	}
}
