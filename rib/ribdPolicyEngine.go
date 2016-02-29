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
		   case ribdCommonDefs.PolicyActionTypeRouteDisposition:
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
		   case ribdCommonDefs.PolicyActionTypeRouteRedistribute:
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
    logger.Println("Prefix match policy ", policyName)
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
	   if prefixPolicyListInfo.policyName != policyName {
	      logger.Println("Found a potential match for this prefix but the policy ", policyName, " is not what we are looking for")
		  continue
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
		case ribdCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
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
		case ribdCommonDefs.PolicyConditionTypeProtocolMatch:
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
	if actionListHasAction(actionList, ribdCommonDefs.PolicyActionTypeRouteDisposition,"Reject") {
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
