// ribdPolicyEngine.go
package main

import (
     "ribd"
	 "utils/patriciaDB"
	 "l3/rib/ribdCommonDefs"
)
func policyEngineActionRejectRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionRejectRoute")
	routeInfo := params.(RouteParams)
	_, err := routeServiceHandler.DeleteV4Route(routeInfo.destNetIp, routeInfo.networkMask, routeInfo.routeType)
	if err != nil {
		logger.Println("deleting v4 route failed with err ", err)
		return
	}
}
func policyEngineActionAcceptRoute(route ribd.Routes, params interface{}) {
    logger.Println("policyEngineActionAcceptRoute")
	routeInfo := params.(RouteParams)
	_, err := createV4Route(routeInfo.destNetIp, routeInfo.networkMask, routeInfo.metric, routeInfo.nextHopIp, routeInfo.nextHopIfType, routeInfo.nextHopIfIndex, routeInfo.routeType, routeInfo.createType, routeInfo.sliceIdx)
	if err != nil {
		logger.Println("creating v4 route failed with err ", err)
		return
	}
}
func policyEngineActionRedistribute(route ribd.Routes, targetProtocol int, params interface {}) {
	logger.Println("policyEngineActionRedistribute")
	//Send a event based on target protocol
    RouteInfo := params.(RouteParams) 
	var evt int
    switch targetProtocol {
      case ribdCommonDefs.BGP:
        logger.Println("Redistribute to BGP")
		if RouteInfo.createType != Invalid {
			logger.Println("Create type not invalid")
			evt = ribdCommonDefs.NOTIFY_ROUTE_CREATED
		} else if RouteInfo.deleteType != Invalid {
			logger.Println("Delete type not invalid")
			evt = ribdCommonDefs.NOTIFY_ROUTE_DELETED
		}
        RouteNotificationSend(RIBD_BGPD_PUB, route, evt)
        break
      default:
        logger.Println("Unknown target protocol")	
    }
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
			  logger.Println("Redistribute target protocol = %d %s ", action.actionInfo, ReverseRouteProtoTypeMapDB[action.actionInfo.(int)])
			  policyEngineActionRedistribute(route, action.actionInfo.(int), params)
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
		break
		case ribdCommonDefs.PolicyConditionTypeProtocolMatch:
		  logger.Println("PolicyConditionTypeProtocolMatch case")
		  matchProto := condition.conditionInfo.(int)
		  if matchProto == int(route.Prototype) {
			logger.Println("Protocol condition matches")
			anyConditionsMatch = true
		  } else {
			logger.Println("Protocol condition does not match")
			allConditionsMatch = false
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

func policyEngineApplyPolicy(route *ribd.Routes, policyStmt PolicyStmt, params interface{}) {
	logger.Println("PolicyEngineApplyPolicy - ", policyStmt.name)
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
	route.PolicyCounter++
	policyEngineImplementActions(*route, policyStmt, params)
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
	var policyPath_Str string
//	policyEngineCheckPolicy(route, policyPath, funcName, params)
    routeInfo := params.(RouteParams)
	logger.Println("Beginning createType = ", routeInfo.createType, " deleteType = ", routeInfo.deleteType)
	if localPolicyStmtDB == nil {
		logger.Println("No policies configured, so accept the route")
        //should be replaced by default import policy action
	} else {
		for idx :=0;idx < len(localPolicyStmtDB);idx++ {
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
	}
	logger.Println("After policyEngineApply createType = ", routeInfo.createType, " deleteType = ", routeInfo.deleteType)
	if route.PolicyCounter == 0{
		logger.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == ribdCommonDefs.PolicyPath_Import {
		   logger.Println("Applying default import policy")
           policyEngineActionAcceptRoute(route , params ) 
		} else if policyPath == ribdCommonDefs.PolicyPath_Export {
			logger.Println("Applying default export policy")
		}
	}
}

func policyEngineApplyForRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, handle patriciaDB.Item) (err error) {
   logger.Println("policyEngpolicyEngineApplyForRouteineCheckAndApply")	
   policy := handle.(PolicyStmt)
   rmapInfoRecordList := item.(RouteInfoRecordList)
   if len(rmapInfoRecordList.routeInfoList) == 0 {
      logger.Println("len(rmapInfoRecordList.routeInfoList) == 0")
	  return err	
   }
   logger.Println("Selected route index = ", rmapInfoRecordList.selectedRouteIdx)
   selectedRouteInfoRecord := rmapInfoRecordList.routeInfoList[rmapInfoRecordList.selectedRouteIdx]
   policyRoute := ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol)}
   params := RouteParams{destNetIp:policyRoute.Ipaddr, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:FIBAndRIB}
   policyEngineApplyPolicy(&policyRoute, policy, params)
   return err
}
func PolicyEngineTraverseAndApply(policy PolicyStmt) {
	logger.Println("PolicyEngineTraverseAndApply - traverse routing table and apply policy ", policy.name)
    RouteInfoMap.VisitAndUpdate(policyEngineApplyForRoute, policy)
}
/*func PolicyEngineTraverseAndReverse(policy PolicyStmt) {
	logger.Println("PolicyEngineTraverseAndReverse - traverse routing table and inverse policy actions", policy.name)
    RouteInfoMap.VisitAndUpdate(policyEngineCheck, policy)
}*/
