// bgpdPolicyEngine.go
package policy

import (
	"bgpd"
	"bytes"
	"fmt"
	"utils/policy/policyCommonDefs"
	"reflect"
	"sort"
	"strconv"
	"utils/patriciaDB"
)

func isPolicyTypeSame(oldPolicy Policy, policy Policy) (same bool) {
	if oldPolicy.exportPolicy == policy.exportPolicy && oldPolicy.importPolicy == policy.importPolicy {
		same = true
	}
	return same
}
func actionListHasAction(actionList []string, actionType int, action string) (match bool) {
	fmt.Println("actionListHasAction for action ", action)
	for i := 0; i < len(actionList); i++ {
		fmt.Println("action at index ", i, " is ", actionList[i])
		actionInfoItem := PolicyActionsDB.Get(patriciaDB.Prefix(actionList[i]))
		if actionInfoItem == nil {
			fmt.Println("nil action")
			return false
		}
		actionInfo := actionInfoItem.(PolicyAction)
		if actionType != actionInfo.actionType {
			continue
		}
		switch actionInfo.actionType {
		case policyCommonDefs.PolicyActionTypeRouteDisposition:
			fmt.Println("RouteDisposition action = ", actionInfo.actionInfo)
			if actionInfo.actionInfo.(string) == action {
				match = true
			}
			break
		case policyCommonDefs.PolicyActionTypeRouteRedistribute:
			fmt.Println("PolicyActionTypeRouteRedistribute action ")
			break
		case policyCommonDefs.PoilcyActionTypeSetAdminDistance:
			fmt.Println("PoilcyActionTypeSetAdminDistance action")
			match = true
			break
		default:
			fmt.Println("Unknown type of action")
			break
		}
	}
	return match
}
func conditionCheckValid(route bgpd.BGPRoute, conditionsList []string) (valid bool) {
	fmt.Println("conditionCheckValid")
	valid = true
	if conditionsList == nil {
		fmt.Println("No conditions to match, so valid")
		return true
	}
	for i := 0; i < len(conditionsList); i++ {
		fmt.Printf("Find policy condition number %d name %s in the condition database\n", i, conditionsList[i])
		conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(conditionsList[i]))
		if conditionItem == nil {
			fmt.Println("Did not find condition ", conditionsList[i], " in the condition database")
			continue
		}
		conditionInfo := conditionItem.(PolicyCondition)
		fmt.Printf("policy condition number %d type %d\n", i, conditionInfo.ConditionType)
		switch conditionInfo.ConditionType {
		case policyCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
			fmt.Println("PolicyConditionTypeDstIpPrefixMatch case")
			routePrefix, err := getNetworkPrefixFromCIDR(route.Network + "/" + strconv.Itoa(int(route.CIDRLen)))
			if err != nil {
				fmt.Println("Invalid routePrefix for the route ", route.Network, "/", route.CIDRLen)
				return false
			}
			condition := conditionInfo.ConditionInfo.(MatchPrefixConditionInfo)
			if condition.usePrefixSet {
				fmt.Println("Need to look up Prefix set to get the prefixes")
			} else {
				if condition.Prefix.MasklengthRange == "exact" {
					fmt.Println("exact prefix match conditiontype")
					ipPrefix, err := getNetworkPrefixFromCIDR(condition.Prefix.IpPrefix)
					if err != nil {
						fmt.Println("ipPrefix invalid ")
						return false
					}
					if bytes.Equal(routePrefix, ipPrefix) == false {
						fmt.Println(" Route Prefix ", routePrefix, " does not match prefix condition ", ipPrefix)
						return false
					}
				} else {
					fmt.Println("Masklength= ", condition.Prefix.MasklengthRange)
					/*		   ip, _, err := net.ParseCIDR(condition.prefix.IpPrefix)
					       if err != nil {
						      return false
					       }
					       ipAddrStr := ip.String()*/
				}
			}
			break
		default:
			fmt.Println("Not a known condition type")
			break
		}
	}
	fmt.Println("returning valid= ", valid)
	return valid
}
func policyEngineActionRejectRoute(route *bgpd.BGPRoute, params interface{}) {
	fmt.Println("policyEngineActionRejectRoute for route ", route.Network, "/", route.CIDRLen)
}
func policyEngineActionAcceptRoute(route *bgpd.BGPRoute, params interface{}) {
	fmt.Println("policyEngineActionAcceptRoute for ip ", route.Network, "/", route.CIDRLen)
}
func policyEngineActionAggregate(route bgpd.BGPRoute, aggregateActionInfo PolicyAggregateActionInfo, params interface{}) {
	fmt.Println("policyEngineActionAggregate")
	//Send a event based on target protocol
	/*    RouteInfo := params.(RouteParams)
	if ((RouteInfo.createType != Invalid || RouteInfo.deleteType != Invalid ) && redistributeActionInfo.redistribute == false) {
		fmt.Println("Don't redistribute action set for a route create/delete, return")
		return
	}*/
}
func policyEngineActionUndoAggregate(route *bgpd.BGPRoute, aggregateActionInfo PolicyAggregateActionInfo, params interface{}, conditionsList []string) {
	fmt.Println("policyEngineActionUndoAggregate")
}
func policyEngineActionUndoRejectRoute(route bgpd.BGPRoute, params interface{}, conditionsList []string) {
	fmt.Println("policyEngineActionUndoRejectRoute - route: ", route.Network, "/", route.CIDRLen, " type ")
}
func policyEngineUndoActionsPolicyStmt(route *bgpd.BGPRoute, policy Policy, policyStmt PolicyStmt, params interface{}, conditionsAndActionsList ConditionsAndActionsList) {
	fmt.Println("policyEngineUndoActionsPolicyStmt")
	if conditionsAndActionsList.actionList == nil {
		fmt.Println("No actions")
		return
	}
	var i int
	for i = 0; i < len(conditionsAndActionsList.actionList); i++ {
		fmt.Printf("Find policy action number %d name %s in the action database\n", i, conditionsAndActionsList.actionList[i])
		actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[i]))
		if actionItem == nil {
			fmt.Println("Did not find action ", conditionsAndActionsList.actionList[i], " in the action database")
			continue
		}
		action := actionItem.(PolicyAction)
		fmt.Printf("policy action number %d type %d\n", i, action.actionType)
		switch action.actionType {
		case policyCommonDefs.PolicyActionTypeAggregate:
			fmt.Println("PolicyActionTypeAggregate action to be applied")
			policyEngineActionUndoAggregate(route, action.actionInfo.(PolicyAggregateActionInfo), params, conditionsAndActionsList.conditionList)
			break
		default:
			fmt.Println("Unknown type of action")
			return
		}
	}
}
func policyEngineUndoPolicyForRoute(route *bgpd.BGPRoute, policy Policy, params interface{}) {
	fmt.Println("policyEngineUndoPolicyForRoute - policy name ", policy.name, "  route: ", route.Network, "/", route.CIDRLen)
	policyRouteIndex := PolicyRouteIndex{routeIP: route.Network, prefixLen: uint16(route.CIDRLen), policy: policy.name}
	policyStmtMap := PolicyRouteMap[policyRouteIndex]
	if policyStmtMap.policyStmtMap == nil {
		fmt.Println("Unexpected:None of the policy statements of this policy have been applied on this route")
		return
	}
	for stmt, conditionsAndActionsList := range policyStmtMap.policyStmtMap {
		fmt.Println("Applied policyStmtName ", stmt)
		policyStmt := PolicyStmtDB.Get(patriciaDB.Prefix(stmt))
		if policyStmt == nil {
			fmt.Println("Invalid policyStmt")
			continue
		}
		policyEngineUndoActionsPolicyStmt(route, policy, policyStmt.(PolicyStmt), params, conditionsAndActionsList)
		//check if the route still exists - it may have been deleted by the previous statement action
	}
}
func policyEngineImplementActions(route *bgpd.BGPRoute, policyStmt PolicyStmt, conditionList []string, params interface{}, ctx interface{}) (actionList []string) {
	fmt.Println("policyEngineImplementActions")
	if policyStmt.actions == nil {
		fmt.Println("No actions")
		return actionList
	}
	var i int
	var callbackFunc ApplyActionFunc
	var ok bool
	createRoute := false
	addActionToList := false
	routeParams := params.(RouteParams)

	for i = 0; i < len(policyStmt.actions); i++ {
		addActionToList = false
		fmt.Printf("Find policy action number %d name %s in the action database\n", i, policyStmt.actions[i])
		actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[i]))
		if actionItem == nil {
			fmt.Println("Did not find action ", policyStmt.actions[i], " in the action database")
			continue
		}
		action := actionItem.(PolicyAction)
		fmt.Printf("policy action number %d type %d\n", i, action.actionType)
		switch action.actionType {
		case policyCommonDefs.PolicyActionTypeAggregate:
			fmt.Println("PolicyActionTypeAggregate action to be applied")
			if callbackFunc, ok = routeParams.ActionFuncMap[action.actionType]; !ok {
				fmt.Println("Callback function NOT found for action PolicyActionTypeAggregate")
				break
			}
			callbackFunc(route, conditionList, action.actionInfo.(PolicyAggregateActionInfo), params, ctx)
			addActionToList = true
			break
		default:
			fmt.Println("UnknownInvalid type of action")
			break
		}
		if addActionToList == true {
			if actionList == nil {
				actionList = make([]string, 0)
			}
			actionList = append(actionList, action.name)
		}
	}
	fmt.Println("createRoute = ", createRoute)
	if createRoute {
		policyEngineActionAcceptRoute(route, params)
	}
	return actionList
}
func findPrefixMatch(ipAddr string, prefixLen uint16, ipPrefix patriciaDB.Prefix, policyName string) (match bool) {
	policyListItem := PrefixPolicyListDB.GetLongestPrefixNode(ipPrefix)
	if policyListItem == nil {
		fmt.Println("intf stored at prefix ", ipPrefix, " is nil")
		return false
	}
	if policyListItem != nil && reflect.TypeOf(policyListItem).Kind() != reflect.Slice {
		fmt.Println("Incorrect data type for this prefix ")
		return false
	}
	policyListSlice := reflect.ValueOf(policyListItem)
	for idx := 0; idx < policyListSlice.Len(); idx++ {
		prefixPolicyListInfo := policyListSlice.Index(idx).Interface().(PrefixPolicyListInfo)
		if prefixPolicyListInfo.policyName == policyName {
			fmt.Println("Found a potential match for this prefix")
		}
		if prefixPolicyListInfo.lowRange == -1 && prefixPolicyListInfo.highRange == -1 {
			fmt.Println("Looking for exact match condition for prefix ", prefixPolicyListInfo.ipPrefix)
			if bytes.Equal(ipPrefix, prefixPolicyListInfo.ipPrefix) {
				fmt.Println(" Matched the prefix")
				return true
			} else {
				fmt.Println(" Did not match the exact prefix")
				return false
			}
		}
		fmt.Println("Prefix len = ", prefixLen)
		if int(prefixLen) < prefixPolicyListInfo.lowRange || int(prefixLen) > prefixPolicyListInfo.highRange {
			fmt.Println("Mask range of the route ", prefixLen, " not within the required mask range:", prefixPolicyListInfo.lowRange, "..", prefixPolicyListInfo.highRange)
			return false
		} else {
			fmt.Println("Mask range of the route ", prefixLen, " within the required mask range:", prefixPolicyListInfo.lowRange, "..", prefixPolicyListInfo.highRange)
			return true
		}
	}
	return match
}
func PolicyEngineMatchConditions(route bgpd.BGPRoute, policyStmt PolicyStmt) (match bool, conditionsList []string) {
	fmt.Println("policyEngineMatchConditions")
	var i int
	allConditionsMatch := true
	anyConditionsMatch := false
	addConditiontoList := false
	for i = 0; i < len(policyStmt.conditions); i++ {
		addConditiontoList = false
		fmt.Printf("Find policy condition number %d name %s in the condition database\n", i, policyStmt.conditions[i])
		conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[i]))
		if conditionItem == nil {
			fmt.Println("Did not find condition ", policyStmt.conditions[i], " in the condition database")
			continue
		}
		condition := conditionItem.(PolicyCondition)
		fmt.Printf("policy condition number %d type %d\n", i, condition.ConditionType)
		switch condition.ConditionType {
		case policyCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
			fmt.Println("PolicyConditionTypeDstIpPrefixMatch case")
			ipPrefix, err := getNetworkPrefixFromCIDR(route.Network + "/" + strconv.Itoa(int(route.CIDRLen)))
			if err != nil {
				fmt.Println("Invalid ipPrefix for the route ", route.Network, "/", route.CIDRLen)
			}
			match := findPrefixMatch(route.Network, uint16(route.CIDRLen), ipPrefix, policyStmt.name)
			if match {
				fmt.Println("Found a match for this prefix")
				anyConditionsMatch = true
				addConditiontoList = true
			}
			break
		default:
			fmt.Println("Not a known condition type")
			break
		}
		if addConditiontoList == true {
			if conditionsList == nil {
				conditionsList = make([]string, 0)
			}
			conditionsList = append(conditionsList, condition.Name)
		}
	}
	if policyStmt.matchConditions == "all" && allConditionsMatch == true {
		return true, conditionsList
	}
	if policyStmt.matchConditions == "any" && anyConditionsMatch == true {
		return true, conditionsList
	}
	return match, conditionsList
}

func policyEngineApplyPolicyStmt(route *bgpd.BGPRoute, policy Policy, policyStmt PolicyStmt, policyPath int, params interface{}, ctx interface{}, hit *bool, routeDeleted *bool) {
	fmt.Println("policyEngineApplyPolicyStmt - ", policyStmt.name)
	var conditionList []string
	if policyStmt.conditions == nil {
		fmt.Println("No policy conditions")
		*hit = true
	} else {
		match, ret_conditionList := PolicyEngineMatchConditions(*route, policyStmt)
		fmt.Println("match = ", match)
		*hit = match
		if !match {
			fmt.Println("Conditions do not match")
			return
		}
		if ret_conditionList != nil {
			if conditionList == nil {
				conditionList = make([]string, 0)
			}
			for j := 0; j < len(ret_conditionList); j++ {
				conditionList = append(conditionList, ret_conditionList[j])
			}
		}
	}
	actionList := policyEngineImplementActions(route, policyStmt, conditionList, params, ctx)
	if actionListHasAction(actionList, policyCommonDefs.PolicyActionTypeRouteDisposition, "Reject") {
		fmt.Println("Reject action was applied for this route")
		*routeDeleted = true
	}
	//check if the route still exists - it may have been deleted by the previous statement action
	//TO-DO: need to check if route still exists
	routeInfo := params.(RouteParams)
	var op int
	if routeInfo.DeleteType != Invalid {
		op = del
	} else {
		if *routeDeleted == false {
			fmt.Println("Reject action was not applied, so add this policy to the route")
			op = add
			updateRoutePolicyState(route, op, policy.name, policyStmt.name)
		}
		addPolicyRouteMapEntry(route, policy.name, policyStmt.name, conditionList, actionList)
	}
	updatePolicyRouteMap(route, policy, op)
}
func policyEngineApplyPolicy(route *bgpd.BGPRoute, policy Policy, policyPath int, params interface{}, ctx interface{}, hit *bool) {
	fmt.Println("policyEngineApplyPolicy - ", policy.name)
	var policyStmtKeys []int
	routeDeleted := false
	for k := range policy.policyStmtPrecedenceMap {
		fmt.Println("key k = ", k)
		policyStmtKeys = append(policyStmtKeys, k)
	}
	sort.Ints(policyStmtKeys)
	for i := 0; i < len(policyStmtKeys); i++ {
		fmt.Println("Key: ", policyStmtKeys[i], " policyStmtName ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[policyStmtKeys[i]])))
		if policyStmt == nil {
			fmt.Println("Invalid policyStmt")
			continue
		}
		policyEngineApplyPolicyStmt(route, policy, policyStmt.(PolicyStmt), policyPath, params, ctx, hit, &routeDeleted)
		//check if the route still exists - it may have been deleted by the previous statement action

		if routeDeleted == true {
			fmt.Println("Route was deleted as a part of the policyStmt ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
			break
		}
		if *hit == true {
			if policy.matchType == "any" {
				fmt.Println("Match type for policy ", policy.name, " is any and the policy stmt ", (policyStmt.(PolicyStmt)).name, " is a hit, no more policy statements will be executed")
				break
			}
		}
	}
}
func PolicyEngineFilter(route *bgpd.BGPRoute, policyPath int, params interface{}, ctx interface{}) {
	fmt.Println("PolicyEngineFilter")
	var policyPath_Str string
	if policyPath == policyCommonDefs.PolicyPath_Import {
		policyPath_Str = "Import"
	} else if policyPath == policyCommonDefs.PolicyPath_Export {
		policyPath_Str = "Export"
	} else if policyPath == policyCommonDefs.PolicyPath_All {
		policyPath_Str = "ALL"
		fmt.Println("policy path ", policyPath_Str, " unexpected in this function")
		return
	}
	routeInfo := params.(RouteParams)
	fmt.Println("PolicyEngineFilter for policypath ", policyPath_Str, "createType = ", routeInfo.CreateType, " deleteType = ", routeInfo.DeleteType, " route: ", route.Network, "/", route.CIDRLen)
	var policyKeys []int
	var policyHit bool
	idx := 0
	var policyInfo interface{}
	if policyPath == policyCommonDefs.PolicyPath_Import {
		for k := range ImportPolicyPrecedenceMap {
			policyKeys = append(policyKeys, k)
		}
	} else if policyPath == policyCommonDefs.PolicyPath_Export {
		for k := range ExportPolicyPrecedenceMap {
			policyKeys = append(policyKeys, k)
		}
	}
	sort.Ints(policyKeys)
	for {
		if routeInfo.DeleteType != Invalid {
			if route.PolicyList != nil {
				if idx >= len(route.PolicyList) {
					break
				}
				fmt.Println("getting policy ", idx, " from route.PolicyList")
				policyInfo = PolicyDB.Get(patriciaDB.Prefix(route.PolicyList[idx]))
				idx++
				if policyInfo.(Policy).exportPolicy && policyPath == policyCommonDefs.PolicyPath_Import || policyInfo.(Policy).importPolicy && policyPath == policyCommonDefs.PolicyPath_Export {
					fmt.Println("policy ", policyInfo.(Policy).name, " not the same type as the policypath -", policyPath_Str)
					continue
				}
			} else if routeInfo.DeleteType != Invalid {
				fmt.Println("route.PolicyList empty and this is a delete operation for the route, so break")
				break
			}
		} else {
			//case when no policies have been applied to the route
			//need to apply the default policy
			fmt.Println("idx = ", idx, " len(policyKeys):", len(policyKeys))
			if idx >= len(policyKeys) {
				break
			}
			policyName := ""
			if policyPath == policyCommonDefs.PolicyPath_Import {
				policyName = ImportPolicyPrecedenceMap[policyKeys[idx]]
			} else if policyPath == policyCommonDefs.PolicyPath_Export {
				policyName = ExportPolicyPrecedenceMap[policyKeys[idx]]
			}
			fmt.Println("getting policy  ", idx, " policyKeys[idx] = ", policyKeys[idx], " ", policyName, " from PolicyDB")
			policyInfo = PolicyDB.Get((patriciaDB.Prefix(policyName)))
			idx++
		}
		if policyInfo == nil {
			fmt.Println("Nil policy")
			continue
		}
		policy := policyInfo.(Policy)
		if localPolicyDB != nil && localPolicyDB[policy.localDBSliceIdx].isValid == false {
			fmt.Println("Invalid policy at localDB slice idx ", policy.localDBSliceIdx)
			continue
		}
		policyEngineApplyPolicy(route, policy, policyPath, params, ctx, &policyHit)
		if policyHit {
			fmt.Println("Policy ", policy.name, " applied to the route")
			break
		}
	}
	if route.PolicyHitCounter == 0 {
		fmt.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == policyCommonDefs.PolicyPath_Import {
			fmt.Println("Applying default import policy")
			//TO-DO: Need to add the default policy to policyList of the route
			policyEngineActionAcceptRoute(route, params)
		} else if policyPath == policyCommonDefs.PolicyPath_Export {
			fmt.Println("Applying default export policy")
		}
	}
	var op int
	if routeInfo.DeleteType != Invalid {
		op = delAll //wipe out the policyList
		updateRoutePolicyState(route, op, "", "")
	}
}

/*
func PolicyEngineFilter(route bgpd.BGPRoute, policyPath int, params interface{}) {
	fmt.Println("PolicyEngineFilter")
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
		  fmt.Println("getting policy stmt ", idx, " from route.PolicyList")
	      policyInfo = 	PolicyStmtDB.Get(patriciaDB.Prefix(route.PolicyList[idx]))
		  idx++
	   } else if routeInfo.deleteType != Invalid {
		  fmt.Println("route.PolicyList empty and this is a delete operation for the route, so break")
          break
	   } else if localPolicyStmtDB == nil {
		  fmt.Println("localPolicyStmt nil")
			//case when no policies have been applied to the route
			//need to apply the default policy
		   break
		} else {
            if idx >= len(localPolicyStmtDB) {
				break
			}
		    fmt.Println("getting policy stmt ", idx, " from localPolicyStmtDB")
            policyInfo = PolicyStmtDB.Get(localPolicyStmtDB[idx].prefix)
			idx++
	   }
	   if policyInfo == nil {
	      fmt.Println("Nil policy")
		  continue
	   }
	   policyStmt := policyInfo.(PolicyStmt)
	   if policyPath == policyCommonDefs.PolicyPath_Import && policyStmt.importPolicy == false ||
	      policyPath == policyCommonDefs.PolicyPath_Export && policyStmt.exportPolicy == false {
	         fmt.Println("Cannot apply the policy ", policyStmt.name, " as ", policyPath_Str, " policy")
			 continue
	   }
	   policyEngineApplyPolicy(&route, policyStmt, params)
	}
/*	if localPolicyStmtDB == nil {
		fmt.Println("No policies configured, so accept the route")
        //should be replaced by default import policy action
	} else {
		for idx :=0;idx < len(localPolicyStmtDB);idx++ {
		//for idx :=0;idx < len(policList);idx++ {
			if localPolicyStmtDB[idx].isValid == false {
				continue
			}
			policyInfo := PolicyDB.Get(localPolicyStmtDB[idx].prefix)
			if policyInfo == nil {
				fmt.Println("Nil policy")
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
				fmt.Println("Cannot apply the policy ", policyStmt.name, " as ", policyPath_Str, " policy")
				continue
			}
		    policyEngineApplyPolicy(&route, policyStmt, params)
        }
	}*/
/*	fmt.Println("After policyEngineApply policyCounter = ", route.PolicyHitCounter)
	if route.PolicyHitCounter == 0{
		fmt.Println("Need to apply default policy, policyPath = ", policyPath, "policyPath_Str= ", policyPath_Str)
		if policyPath == policyCommonDefs.PolicyPath_Import {
		   fmt.Println("Applying default import policy")
		    //TO-DO: Need to add the default policy to policyList of the route
           policyEngineActionAcceptRoute(route , params )
		} else if policyPath == policyCommonDefs.PolicyPath_Export {
			fmt.Println("Applying default export policy")
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
	fmt.Println("policyEngineApplyForRoute %v", prefix)
	/*   policy := handle.(Policy)
	   rmapInfoRecordList := item.(RouteInfoRecordList)
	   policyHit := false
	   if rmapInfoRecordList.routeInfoProtocolMap == nil {
	      fmt.Println("rmapInfoRecordList.routeInfoProtocolMap) = nil")
		  return err
	   }
	   fmt.Println("Selected route protocol = ", rmapInfoRecordList.selectedRouteProtocol)
	   selectedRouteList := rmapInfoRecordList.routeInfoProtocolMap[rmapInfoRecordList.selectedRouteProtocol]
	   if len(selectedRouteList) == 0 {
	      fmt.Println("len(selectedRouteList) == 0")
		  return err
	  }
	  for i:=0;i<len(selectedRouteList);i++ {
	     selectedRouteInfoRecord := selectedRouteList[i]
	     policyRoute := bgpd.BGPRoute{Network: selectedRouteInfoRecord.destNetIp.String(), Mask: selectedRouteInfoRecord.networkMask.String(), NextHopIp: selectedRouteInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(selectedRouteInfoRecord.nextHopIfType), IfIndex: selectedRouteInfoRecord.nextHopIfIndex, Metric: selectedRouteInfoRecord.metric, Prototype: ribd.Int(selectedRouteInfoRecord.protocol), IsPolicyBasedStateValid:rmapInfoRecordList.isPolicyBasedStateValid}
	     params := RouteParams{destNetIp:policyRoute.Network, networkMask:policyRoute.Mask, routeType:policyRoute.Prototype, sliceIdx:policyRoute.SliceIdx, createType:Invalid, deleteType:Invalid}
	     if len(rmapInfoRecordList.policyList) == 0 {
		  fmt.Println("This route has no policy applied to it so far, just apply the new policy")
	      policyEngineApplyPolicy(&policyRoute, policy, policyCommonDefs.PolicyPath_All,params, &policyHit)
	     } else {
	      fmt.Println("This route already has policy applied to it - len(route.PolicyList) - ", len(rmapInfoRecordList.policyList))

		  for i:=0;i<len(rmapInfoRecordList.policyList);i++ {
			 fmt.Println("policy at index ", i)
		     policyInfo := PolicyDB.Get(patriciaDB.Prefix(rmapInfoRecordList.policyList[i]))
		     if policyInfo == nil {
			    fmt.Println("Unexpected: Invalid policy in the route policy list")
		     } else {
		       oldPolicy := policyInfo.(Policy)
			   if !isPolicyTypeSame(oldPolicy, policy) {
				 fmt.Println("The policy type applied currently is not the same as new policy, so apply new policy")
	              policyEngineApplyPolicy(&policyRoute, policy, policyCommonDefs.PolicyPath_All,params, &policyHit)
			   } else if oldPolicy.precedence < policy.precedence {
				 fmt.Println("The policy types are same and precedence of the policy applied currently is lower than the new policy, so do nothing")
				 return err
			   } else {
				fmt.Println("The new policy's precedence is lower, so undo old policy's actions and apply the new policy")
				policyEngineUndoPolicyForRoute(policyRoute, oldPolicy, params)
				policyEngineApplyPolicy(&policyRoute, policy, policyCommonDefs.PolicyPath_All,params, &policyHit)
			   }
			}
		  }
	    }
	  }*/
	return err
}
func PolicyEngineTraverseAndApply(policy Policy) {
	fmt.Println("PolicyEngineTraverseAndApply - traverse routing table and apply policy ", policy.name)
	//RouteInfoMap.VisitAndUpdate(policyEngineApplyForRoute, policy)
	//TO-DO_Write your traver function here
}
func PolicyEngineTraverseAndApplyPolicy(policy Policy) {
	fmt.Println("PolicyEngineTraverseAndApplyPolicy -  apply policy ", policy.name)
	if policy.exportPolicy || policy.importPolicy {
		fmt.Println("Applying import/export policy to all routes")
		PolicyEngineTraverseAndApply(policy)
	} else if policy.globalPolicy {
		fmt.Println("Need to apply global policy")
		policyEngineApplyGlobalPolicy(policy)
	}
}
func PolicyEngineTraverseAndReverse(policy Policy) {
	fmt.Println("PolicyEngineTraverseAndReverse - traverse routing table and inverse policy actions", policy.name)
	if policy.routeList == nil {
		fmt.Println("No route affected by this policy, so nothing to do")
		return
	}
	var policyRoute bgpd.BGPRoute
	var params RouteParams
	for idx := 0; idx < len(policy.routeInfoList); idx++ {
		policyRoute = policy.routeInfoList[idx]
		params = RouteParams{DestNetIp: policyRoute.Network, PrefixLen: uint16(policyRoute.CIDRLen), CreateType: Invalid, DeleteType: Invalid}
		ipPrefix, err := getNetworkPrefixFromCIDR(policy.routeInfoList[idx].Network + "/" + strconv.Itoa(int(policy.routeInfoList[idx].CIDRLen)))
		if err != nil {
			fmt.Println("Invalid route ", policy.routeList[idx])
			continue
		}
		policyEngineUndoPolicyForRoute(&policyRoute, policy, params)
		deleteRoutePolicyState(ipPrefix, policy.name)
		deletePolicyRouteMapEntry(&policyRoute, policy.name)
	}
}
func PolicyEngineTraverseAndReversePolicy(policy Policy) {
	fmt.Println("PolicyEngineTraverseAndReversePolicy -  reverse policy ", policy.name)
	if policy.exportPolicy || policy.importPolicy {
		fmt.Println("Reversing import/export policy ")
		PolicyEngineTraverseAndReverse(policy)
	} else if policy.globalPolicy {
		fmt.Println("Need to reverse global policy")
		policyEngineReverseGlobalPolicy(policy)
	}

}

func policyEngineUpdateRoute(prefix patriciaDB.Prefix, item patriciaDB.Item, handle patriciaDB.Item) (err error) {
	fmt.Println("policyEngineUpdateRoute for ", prefix)

	//routeServiceHandler.UpdateIPV4Route(&route, nil, nil)
	return err
}
func policyEngineTraverseAndUpdate() {
	fmt.Println("policyEngineTraverseAndUpdate")
	//RouteInfoMap.VisitAndUpdate(policyEngineUpdateRoute, nil)
	//TO-DO: need to visit all routes and re-evaluate
}
func policyEngineApplyGlobalPolicyStmt(policy Policy, policyStmt PolicyStmt) {
	fmt.Println("policyEngineApplyGlobalPolicyStmt - ", policyStmt.name)
	var conditionItem interface{} = nil
	//global policies can only have statements with 1 condition and 1 action
	if policyStmt.actions == nil {
		fmt.Println("No policy actions defined")
		return
	}
	if policyStmt.conditions == nil {
		fmt.Println("No policy conditions")
	} else {
		if len(policyStmt.conditions) > 1 {
			fmt.Println("only 1 condition allowed for global policy stmt")
			return
		}
		conditionItem = PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[0]))
		if conditionItem == nil {
			fmt.Println("Condition ", policyStmt.conditions[0], " not found")
			return
		}
		actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[0]))
		if actionItem == nil {
			fmt.Println("Action ", policyStmt.actions[0], " not found")
			return
		}
		actionInfo := actionItem.(PolicyAction)
		switch actionInfo.actionType {
		default:
			fmt.Println("Invalid global policy action")
			return
		}
	}
}
func policyEngineApplyGlobalPolicy(policy Policy) {
	fmt.Println("policyEngineApplyGlobalPolicy")
	var policyStmtKeys []int
	for k := range policy.policyStmtPrecedenceMap {
		fmt.Println("key k = ", k)
		policyStmtKeys = append(policyStmtKeys, k)
	}
	sort.Ints(policyStmtKeys)
	for i := 0; i < len(policyStmtKeys); i++ {
		fmt.Println("Key: ", policyStmtKeys[i], " policyStmtName ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[policyStmtKeys[i]])))
		if policyStmt == nil {
			fmt.Println("Invalid policyStmt")
			continue
		}
		policyEngineApplyGlobalPolicyStmt(policy, policyStmt.(PolicyStmt))
	}
}
func policyEngineReverseGlobalPolicyStmt(policy Policy, policyStmt PolicyStmt) {
	fmt.Println("policyEngineApplyGlobalPolicyStmt - ", policyStmt.name)
	var conditionItem interface{} = nil
	//global policies can only have statements with 1 condition and 1 action
	if policyStmt.actions == nil {
		fmt.Println("No policy actions defined")
		return
	}
	if policyStmt.conditions == nil {
		fmt.Println("No policy conditions")
	} else {
		if len(policyStmt.conditions) > 1 {
			fmt.Println("only 1 condition allowed for global policy stmt")
			return
		}
		conditionItem = PolicyConditionsDB.Get(patriciaDB.Prefix(policyStmt.conditions[0]))
		if conditionItem == nil {
			fmt.Println("Condition ", policyStmt.conditions[0], " not found")
			return
		}
		actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(policyStmt.actions[0]))
		if actionItem == nil {
			fmt.Println("Action ", policyStmt.actions[0], " not found")
			return
		}
		actionInfo := actionItem.(PolicyAction)
		switch actionInfo.actionType {
		default:
			fmt.Println("Invalid global policy action")
			return
		}
	}
}
func policyEngineReverseGlobalPolicy(policy Policy) {
	fmt.Println("policyEngineReverseGlobalPolicy")
	var policyStmtKeys []int
	for k := range policy.policyStmtPrecedenceMap {
		fmt.Println("key k = ", k)
		policyStmtKeys = append(policyStmtKeys, k)
	}
	sort.Ints(policyStmtKeys)
	for i := 0; i < len(policyStmtKeys); i++ {
		fmt.Println("Key: ", policyStmtKeys[i], " policyStmtName ", policy.policyStmtPrecedenceMap[policyStmtKeys[i]])
		policyStmt := PolicyStmtDB.Get((patriciaDB.Prefix(policy.policyStmtPrecedenceMap[policyStmtKeys[i]])))
		if policyStmt == nil {
			fmt.Println("Invalid policyStmt")
			continue
		}
		policyEngineReverseGlobalPolicyStmt(policy, policyStmt.(PolicyStmt))
	}
}
