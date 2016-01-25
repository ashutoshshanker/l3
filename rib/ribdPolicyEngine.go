// ribdPolicyEngine.go
package main

import (
     "ribd"
	 "utils/patriciaDB"
	 "l3/rib/ribdCommonDefs"
)
func policyEngineActionRedistribute(route ribd.Routes, targetProtocol int) {
	logger.Println("policyEngineActionRedistribute")
	//Send a event based on target protocol
    switch targetProtocol {
      case ribdCommonDefs.BGP:
        logger.Println("Redistribute to BGP")
        RouteNotificationSend(RIBD_BGPD_PUB, route)
        break
      default:
        logger.Println("Unknown target protocol")	
    }
}

func policyEngineImplementActions(route ribd.Routes, policyStmt PolicyStmt) {
	logger.Println("policyEngineImplementActions")
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
			  break
		   case ribdCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action to be applied")
			  logger.Println("Redistribute target protocol = %d %s ", action.actionInfo, ReverseRouteProtoTypeMapDB[action.actionInfo.(int)])
	
	          //Send a event
			  policyEngineActionRedistribute(route, action.actionInfo.(int))
			  break
		   default:
		      logger.Println("Unknown type of action")
			  return
		}
	}
}
func policyEngineMatchConditions(route ribd.Routes, policyStmt PolicyStmt) (allConditionsMatch bool , anyConditionsMatch bool){
    logger.Println("policyEngineMatchConditions")
	var i int
	allConditionsMatch = true
	anyConditionsMatch = false
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
          return allConditionsMatch, anyConditionsMatch
	  }
	}
   return allConditionsMatch, anyConditionsMatch
}

func policyEngineApplyPolicy(route ribd.Routes, name string) {
	logger.Println("PolicyEngineApplyPolicy - ", name)
	policyStmtGet := PolicyDB.Get(patriciaDB.Prefix(name))
	if(policyStmtGet == nil) {
		logger.Printf("Didnt find the policy %s in the policy DB\n", name)
		return
	}
	policyStmt := policyStmtGet.(PolicyStmt)
	if policyStmt.name != name {
		logger.Println("Mismatch in the policy names")
		return
	}
	if policyStmt.conditions == nil {
		logger.Println("No policy conditions")
		return
	}
	allConditionsMatch, anyConditionsMatch := policyEngineMatchConditions(route, policyStmt)
	if allConditionsMatch {
		logger.Println("All conditions match")
		//use this later to control actions
	}
	if anyConditionsMatch {
		logger.Println("Some conditions match")
	}
	policyEngineImplementActions(route, policyStmt)
}
func policyEngineCheckPolicy( route ribd.Routes) {
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
		policyEngineApplyPolicy(route, policyList[policyNum])
	}
	
	//Prefix based policy checks
}
func PolicyEngineFilter(route ribd.Routes) {
	logger.Println("PolicyEngineFilter")
	policyEngineCheckPolicy(route)
}

func policyEngineCheck(prefix patriciaDB.Prefix, item patriciaDB.Item, handle patriciaDB.Item) (err error) {
   logger.Println("policyEngineCheck")	
   policy := handle.(PolicyStmt)
   rmapInfoRecordList := item.(RouteInfoRecordList)
   if len(rmapInfoRecordList.routeInfoList) == 0 {
      logger.Println("len(rmapInfoRecordList.routeInfoList) == 0")
	  return err	
   }
   logger.Println("Selected route index = ", rmapInfoRecordList.selectedRouteIdx)
   selectedRouteInfoRecord := rmapInfoRecordList.routeInfoList[rmapInfoRecordList.selectedRouteIdx]
   policyRoute := ribd.Routes{Ipaddr: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol)}
   policyEngineApplyPolicy(policyRoute, policy.name)
   return err
}
func PolicyEngineTraverseAndApply(policy PolicyStmt) {
	logger.Println("PolicyEngineTraverseAndApply - traverse routing table and apply policy ", policy.name)
    RouteInfoMap.VisitAndUpdate(policyEngineCheck, policy)
}
