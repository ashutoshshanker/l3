// ribdPolicyEngine.go
package main

import (
     "ribd"
	 "utils/patriciaDB"
	 "l3/rib/ribdCommonDefs"
)
func policyEngineApplyPolicy(route ribd.Routes, name string) {
	logger.Println("PolicyEngineApplyPolicy - ", name)
	var i int
	var allConditionsMatch = true
	var anyConditionsMatch = false
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
	for i=0;i<len(policyStmt.conditions);i++ {
	  logger.Printf("policy condition number %d type %d\n", i, policyStmt.conditions[i].conditionType)
      switch policyStmt.conditions[i].conditionType {
		case ribdCommonDefs.PolicyConditionTypePrefixMatch:
		  logger.Println("PolicyConditionTypePrefixMatch case")
		break
		case ribdCommonDefs.PolicyConditionTypeProtocolMatch:
		  logger.Println("PolicyConditionTypeProtocolMatch case")
		  matchProto := policyStmt.conditions[i].conditionInfo.(int)
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
		  return
	  }
	}
	if allConditionsMatch {
		logger.Println("All conditions match")
		//use this later to control actions
	}
	if anyConditionsMatch {
		logger.Println("Some conditions match")
	}
	if policyStmt.actions == nil {
		logger.Println("No actions")
		return
	}
	for i=0;i<len(policyStmt.actions);i++ {
	  logger.Printf("policy action number %d type %d\n", i, policyStmt.actions[i].actionType)
		switch policyStmt.actions[i].actionType {
		   case ribdCommonDefs.PolicyActionTypeRouteDisposition:
		      logger.Println("PolicyActionTypeRouteDisposition action to be applied")
			  break
		   case ribdCommonDefs.PolicyActionTypeRouteRedistribute:
		      logger.Println("PolicyActionTypeRouteRedistribute action to be applied")
			  break
		   default:
		      logger.Println("Unknown type of action")
			  return
		}
	}
}
func policyEngineCheckConditions( route ribd.Routes) {
	logger.Println("policyEngineCheckConditions")
	
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
	policyEngineCheckConditions(route)
}
