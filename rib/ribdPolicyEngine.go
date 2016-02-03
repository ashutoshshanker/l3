// ribdPolicyEngine.go
package main

import (
     "ribd"
	 "utils/patriciaDB"
	 "l3/rib/ribdCommonDefs"
	 "reflect"
	 "sort"
)
func policyEngineActionRejectRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionRejectRoute for route ", route.Ipaddr, " ", route.Mask)
	routeInfo := params.(RouteParams)
  _, err := routeServiceHandler.DeleteV4Route(routeInfo.destNetIp, routeInfo.networkMask, routeInfo.routeType,)// FIBAndRIB)//,ribdCommonDefs.RoutePolicyStateChangetoInValid)
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
func policyEngineActionUndoRedistribute(route ribd.Routes, redistributeActionInfo RedistributeActionInfo, params interface {}) {
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

func policyEngineUndoActionsPolicyStmt(route ribd.Routes, policy Policy, policyStmt PolicyStmt, params interface{}) {
	logger.Println("policyEngineUndoActionsPolicyStmt")
	if policyStmt.actions == nil {
		logger.Println("No actions")
		return
	}
	var i int
	for i=0;i<len(policyStmt.actions);i++ {
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
			  if action.actionInfo.(string) == "Accept" {
                 logger.Println("Accept action - undoing it by deleting")
				 policyEngineActionRejectRoute(route, params)
				 return
			  }
			  break
		   case ribdCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action to be applied")
			  policyEngineActionUndoRedistribute(route, action.actionInfo.(RedistributeActionInfo), params)
			  break
		   default:
		      logger.Println("Unknown type of action")
			  return
		}
	}
}
func policyEngineUndoActionsPolicy(route ribd.Routes, policy Policy, params interface{}) {
	logger.Println("policyEngineUndoActionsPolicy")
}
func policyEngineImplementActions(route ribd.Routes, policyStmt PolicyStmt, params interface {}) {
	logger.Println("policyEngineImplementActions")
	if policyStmt.actions == nil {
		logger.Println("No actions")
		return
	}
	var i int
	createRoute := false
	for i=0;i<len(policyStmt.actions);i++ {
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
				 return
			  }
			  createRoute = true
			  break
		   case ribdCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action to be applied")
			  policyEngineActionRedistribute(route, action.actionInfo.(RedistributeActionInfo), params)
			  break
		   default:
		      logger.Println("Unknown type of action")
			  return
		}
	}
	logger.Println("createRoute = ",createRoute)
	if createRoute {
		policyEngineActionAcceptRoute(route, params)
	}
}
func PolicyEngineMatchConditions(route ribd.Routes, policyStmt PolicyStmt) (match bool){
    logger.Println("policyEngineMatchConditions")
	var i int
	allConditionsMatch := true
	anyConditionsMatch := false
	for i=0;i<len(policyStmt.conditions);i++ {
	  logger.Printf("Find policy condition number %d name %s in the condition database\n", i, policyStmt.conditions[i])
	  conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[i]))
	  if conditionItem == nil {
	     logger.Println("Did not find condition ", policyStmt.conditions[i], " in the condition database")	
		 continue
	  }
	  condition := conditionItem.(PolicyCondition)
	  logger.Printf("policy condition number %d type %d\n", i, condition.conditionType)
      switch condition.conditionType {
		case ribdCommonDefs.PolicyConditionTypePrefixMatch:
		  logger.Println("PolicyConditionTypePrefixMatch case")
		  ipPrefix,err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
		  if err != nil {
			logger.Println("Invalid ipPrefix for the route ", route.Ipaddr," ", route.Mask)
		  }
	      policyListItem:= PrefixPolicyListDB.Get(ipPrefix)
		  if policyListItem == nil {
			logger.Println("no policies configured for the prefix ", ipPrefix)
			return match
		  }
	      if policyListItem != nil && reflect.TypeOf(policyListItem).Kind() != reflect.Slice {
		     logger.Println("Incorrect data type for this prefix ")
		     return match
	      }
		  policyListSlice := reflect.ValueOf(policyListItem)
		  for idx :=0;idx < policyListSlice.Len();idx++ {
			if policyListSlice.Index(idx).Interface().(string) == policyStmt.name {
				logger.Println("Found a match for this prefix")
				anyConditionsMatch = true
			}
		} 
		break
		case ribdCommonDefs.PolicyConditionTypeProtocolMatch:
		  logger.Println("PolicyConditionTypeProtocolMatch case")
		  matchProto := condition.conditionInfo.(int)
		  if matchProto == int(route.Prototype) {
			logger.Println("Protocol condition matches")
			anyConditionsMatch = true
		  } 
		break
		default:
		  logger.Println("Not a known condition type")
          return match
	  }
	}
   if policyStmt.matchConditions == "all" && allConditionsMatch == true {
	return true
   }
   if policyStmt.matchConditions == "any" && anyConditionsMatch == true {
	return true
   }
   return match
}

func policyEngineApplyPolicyStmt(route *ribd.Routes, policy Policy, policyStmt PolicyStmt, policyPath int, params interface{}, hit *bool) {
	logger.Println("policyEngineApplyPolicyStmt - ", policyStmt.name)
	var policyPath_Str string
	if policyPath == ribdCommonDefs.PolicyPath_Import {
	   policyPath_Str = "Import"
	} else if policyPath == ribdCommonDefs.PolicyPath_Export {
	   policyPath_Str = "Export"
	} else if policyPath == ribdCommonDefs.PolicyPath_All {
		policyPath_Str = "ALL"
	}
	if policyPath == ribdCommonDefs.PolicyPath_Import && policyStmt.importPolicy == false || 
	   policyPath == ribdCommonDefs.PolicyPath_Export && policyStmt.exportPolicy == false {
	   logger.Println("Cannot apply the policy ", policyStmt.name, " as ", policyPath_Str, " policy")
	   return
	}
	if policyStmt.conditions == nil {
		logger.Println("No policy conditions")
		return
	}
	match := PolicyEngineMatchConditions(*route, policyStmt)
	logger.Println("match = ", match)
	if !match {
		logger.Println("Conditions do not match")
		return
	}
	policyEngineImplementActions(*route, policyStmt, params)
    routeInfo := params.(RouteParams)
	var op int
	if routeInfo.deleteType != Invalid {
		op = del
	} else {
		op = add
	    route.PolicyHitCounter++
	    updateRoutePolicyState(*route, op, policy.name, policyStmt.name)
	}
	updatePolicyRouteMap(*route, policy, op)
	*hit = match
}
func policyEngineApplyPolicy(route *ribd.Routes, policy Policy, policyPath int,params interface{}, hit *bool) {
	logger.Println("policyEngineApplyPolicy - ", policy.name)
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
		policyEngineApplyPolicyStmt(route,policy,policyStmt.(PolicyStmt),policyPath, params, hit)
		//check if the route still exists - it may have been deleted by the previous statement action
		ipPrefix,err:=getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
		if err != nil {
			logger.Println("Error when getting ipPrefix, err= ", err)
			break
		}
        routeInfoRecordList := RouteInfoMap.Get(ipPrefix)
		if routeInfoRecordList == nil {
			logger.Println("this route no longer exists")
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
func policyEngineCheckPolicy(route *ribd.Routes, params interface {}) {
	logger.Println("policyEngineCheckPolicy")
	
	//Protocol based policy checks
	policyList := ProtocolPolicyListDB[int(route.Prototype)]
	if(policyList == nil) {
		logger.Println("No policy configured for this route type ", route.Prototype)
		//return 0, err
	}
	logger.Printf("Number of policies configured for this route type %d is %d\n", route.Prototype, len(policyList))
	for policyNum :=0;policyNum < len(policyList);policyNum++ {
		logger.Printf("policy number %d name %s\n", policyNum, policyList[policyNum])
//		policyEngineApplyPolicy(route, policyList[policyNum], params)
	}
	
	//Prefix based policy checks
}
func PolicyEngineFilter(route ribd.Routes, policyPath int, params interface{}) {
	logger.Println("PolicyEngineFilter")
    routeInfo := params.(RouteParams)
    var policyKeys []int
	var policyHit bool
	idx :=0
	var policyInfo interface{}
	for k:=range PolicyPrecedenceMap {
	   policyKeys = append(policyKeys,k)
	}
	sort.Ints(policyKeys)
	for ;; {
       if route.PolicyList != nil {
          var routePolicyKeys []string
	      for k:=range route.PolicyList {
	        routePolicyKeys = append(routePolicyKeys,k)
	      }
	      sort.Strings(routePolicyKeys)
		  if idx >= len(routePolicyKeys) {
			break
		  }
		  logger.Println("getting policy ", idx, " from route.PolicyList")
	      policyInfo = 	PolicyDB.Get(patriciaDB.Prefix(routePolicyKeys[idx]))
		  idx++
	   } else if routeInfo.deleteType != Invalid {
		  logger.Println("route.PolicyList empty and this is a delete operation for the route, so break")
          break
	   } else if localPolicyDB == nil {
		  logger.Println("localPolicy nil")
			//case when no policies have been applied to the route
			//need to apply the default policy
		   break	   
		} else {
            if idx >= len(policyKeys) {
				break
			}		
		    logger.Println("getting policy  ", idx, " policyKeys[idx] = ", policyKeys[idx]," ", PolicyPrecedenceMap[policyKeys[idx]]," from PolicyDB")
            policyInfo = PolicyDB.Get((patriciaDB.Prefix(PolicyPrecedenceMap[policyKeys[idx]])))
			idx++
	   }
	   if policyInfo == nil {
	      logger.Println("Nil policy")
		  continue
	   }
	   policy := policyInfo.(Policy)
	   policyEngineApplyPolicy(&route, policy, policyPath, params, &policyHit)
	   if policyHit {
	      logger.Println("Policy ", policy.name, " applied to the route")	
		  break
	   }
	}
	var policyPath_Str string
	if policyPath == ribdCommonDefs.PolicyPath_Import {
	   policyPath_Str = "Import"
	} else if policyPath == ribdCommonDefs.PolicyPath_Export {
	   policyPath_Str = "Export"
	} else if policyPath == ribdCommonDefs.PolicyPath_All {
		policyPath_Str = "ALL"
	}
	if route.PolicyHitCounter == 0{
		logger.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == ribdCommonDefs.PolicyPath_Import {
		   logger.Println("Applying default import policy")
		    //TO-DO: Need to add the default policy to policyList of the route
           policyEngineActionAcceptRoute(route , params ) 
		} else if policyPath == ribdCommonDefs.PolicyPath_Export {
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
	if policyPath == ribdCommonDefs.PolicyPath_Import {
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
	   if policyPath == ribdCommonDefs.PolicyPath_Import && policyStmt.importPolicy == false || 
	      policyPath == ribdCommonDefs.PolicyPath_Export && policyStmt.exportPolicy == false {
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
			if policyPath == ribdCommonDefs.PolicyPath_Import {
				policyPath_Str = "Import"
			} else {
				policyPath_Str = "Export"
			}
			if policyPath == ribdCommonDefs.PolicyPath_Import && policyStmt.importPolicy == false || 
			   policyPath == ribdCommonDefs.PolicyPath_Export && policyStmt.exportPolicy == false {
				logger.Println("Cannot apply the policy ", policyStmt.name, " as ", policyPath_Str, " policy")
				continue
			}
		    policyEngineApplyPolicy(&route, policyStmt, params)
        }
	}*/
/*	logger.Println("After policyEngineApply policyCounter = ", route.PolicyHitCounter)
	if route.PolicyHitCounter == 0{
		logger.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == ribdCommonDefs.PolicyPath_Import {
		   logger.Println("Applying default import policy")
		    //TO-DO: Need to add the default policy to policyList of the route
           policyEngineActionAcceptRoute(route , params ) 
		} else if policyPath == ribdCommonDefs.PolicyPath_Export {
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
   if len(rmapInfoRecordList.routeInfoList) == 0 {
      logger.Println("len(rmapInfoRecordList.routeInfoList) == 0")
	  return err	
   }
   logger.Println("Selected route index = ", rmapInfoRecordList.selectedRouteIdx)
   selectedRouteInfoRecord := rmapInfoRecordList.routeInfoList[rmapInfoRecordList.selectedRouteIdx]
   policyRoute := ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol), IsPolicyBasedStateValid:rmapInfoRecordList.isPolicyBasedStateValid}
   params := RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
   if rmapInfoRecordList.policyList == nil {
	  logger.Println("This route has no policy applied to it so far, just apply the new policy")
      policyEngineApplyPolicy(&policyRoute, policy, ribdCommonDefs.PolicyPath_All,params, &policyHit)
   } else {
      logger.Println("This route already has policy applied to it - len(route.PolicyList) - ", len(rmapInfoRecordList.policyList))
    
      var routePolicyKeys []string
	  for k:=range rmapInfoRecordList.policyList {
	    routePolicyKeys = append(routePolicyKeys,k)
	  }
	  sort.Strings(routePolicyKeys)
	  for i:=0;i<len(routePolicyKeys);i++ {
		 logger.Println("policy at index ", i)
	     policyInfo := PolicyDB.Get(patriciaDB.Prefix(routePolicyKeys[i]))
	     if policyInfo == nil {
		    logger.Println("Unexpected: Invalid policy in the route policy list")
	     } else {
	       oldPolicy := policyInfo.(Policy)
		   if 	oldPolicy.precedence < policy.precedence {
			 logger.Println("The precedence of the policy applied currently is lower than the new policy, so do nothing")
			 return err
		   } else {
			logger.Println("The new policy's precedence is lower, so undo old policy's actions and apply the new policy")
			policyEngineUndoActionsPolicy(policyRoute, oldPolicy, params)
			policyEngineApplyPolicy(&policyRoute, policy, ribdCommonDefs.PolicyPath_All,params, &policyHit)
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
	PolicyEngineTraverseAndApply(policy)
/*     var policyStmtKeys []int
	 for k:=range policy.policyStmtPrecedenceMap {
		policyStmtKeys = append(policyStmtKeys,k)
	}
	sort.Ints(policyStmtKeys)
	for k:=range policyStmtKeys {
		logger.Println("Key: ", k, " policyStmtName ", policy.policyStmtPrecedenceMap[k])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[k])))
        if policyStmt == nil {
			logger.Println("Invalid policyStmt")
			continue
		}
		PolicyEngineTraverseAndApply(policyStmt.(PolicyStmt))
	}*/
}
func PolicyEngineTraverseAndReverse(policy Policy) {
	logger.Println("PolicyEngineTraverseAndReverse - traverse routing table and inverse policy actions", policy.name)
	if policy.routeList == nil {
		logger.Println("No route affected by this policy, so nothing to do")
		return
	}
	for idx:=0;idx<len(policy.routeList);idx++ {
		routeInfoRecordListItem := RouteInfoMap.Get(patriciaDB.Prefix(policy.routeList[idx]))
		if routeInfoRecordListItem == nil {
			logger.Println("routeInfoRecordListItem nil for prefix ", policy.routeList[idx])
			continue
		}
		routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
        selectedRouteInfoRecord := routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
        policyRoute := ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol)}
        params := RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
		policyEngineUndoActionsPolicy(policyRoute, policy, params)
		deleteRoutePolicyState(patriciaDB.Prefix(policy.routeList[idx]), policy.name)
	}
}
