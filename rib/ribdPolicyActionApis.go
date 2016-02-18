// ribdPolicyActionApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
	"strconv"
)

var PolicyActionsDB = patriciaDB.NewTrie()
type RedistributeActionInfo struct {
	redistribute bool
	redistributeTargetProtocol int
}
type PolicyAction struct {
	name          string
	actionType int
	actionInfo interface {}
	policyStmtList []string
	actionGetBulkInfo string
	localDBSliceIdx int
}
var localPolicyActionsDB []localDB
func updateLocalActionsDB(prefix patriciaDB.Prefix) {
    localDBRecord := localDB{prefix:prefix, isValid:true}
    if(localPolicyActionsDB == nil) {
		localPolicyActionsDB = make([]localDB, 0)
	} 
	localPolicyActionsDB = append(localPolicyActionsDB, localDBRecord)
}
func CreatePolicyRouteDispositionAction(cfg *ribd.PolicyActionConfig )(val bool, err error) {
	logger.Println("CreateRouteDispositionAction")
	policyAction := PolicyActionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyAction == nil) {
	   logger.Println("Defining a new policy action with name ", cfg.Name)
	   routeDispositionAction := ""
	   if cfg.Accept == true {
	      routeDispositionAction = "Accept"	
	   } else if cfg.Reject == true {
	      routeDispositionAction = "Reject"	
	   } else {
	      logger.Println("User should set either one of accept/reject to true for this action type")
		  err = errors.New("User should set either one of accept/reject to true for this action type")
		  return val,err	
	   }
	   newPolicyAction := PolicyAction{name:cfg.Name,actionType:ribdCommonDefs.PolicyActionTypeRouteDisposition,actionInfo:routeDispositionAction ,localDBSliceIdx:(len(localPolicyActionsDB))}
       newPolicyAction.actionGetBulkInfo =   routeDispositionAction
		if ok := PolicyActionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyAction); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
	  updateLocalActionsDB(patriciaDB.Prefix(cfg.Name))
	} else {
		logger.Println("Duplicate action name")
		err = errors.New("Duplicate policy action definition")
		return val, err
	}
	return val, err
}

func CreatePolicyAdminDistanceAction(cfg *ribd.PolicyActionConfig) (val bool, err error) {
	logger.Println("CreatePolicyAdminDistanceAction")
	policyAction := PolicyActionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyAction == nil) {
	   logger.Println("Defining a new policy action with name ", cfg.Name, "Setting admin distance value to ", cfg.SetAdminDistanceValue)
	   newPolicyAction := PolicyAction{name:cfg.Name,actionType:ribdCommonDefs.PoilcyActionTypeSetAdminDistance,actionInfo:cfg.SetAdminDistanceValue ,localDBSliceIdx:(len(localPolicyActionsDB))}
       newPolicyAction.actionGetBulkInfo =  "Set admin distance to value "+strconv.Itoa(int(cfg.SetAdminDistanceValue))
		if ok := PolicyActionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyAction); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
	  updateLocalActionsDB(patriciaDB.Prefix(cfg.Name))
	} else {
		logger.Println("Duplicate action name")
		err = errors.New("Duplicate policy action definition")
		return val, err
	}
	return val, err
}

func CreatePolicyRedistributionAction(cfg *ribd.PolicyActionConfig) (val bool, err error) {
	logger.Println("CreatePolicyRedistributionAction")
	targetProtoType := -1

	policyAction := PolicyActionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyAction == nil) {
	   logger.Println("Defining a new policy action with name ", cfg.Name)
	   retProto,found := RouteProtocolTypeMapDB[cfg.RedistributeActionInfo.RedistributeTargetProtocol]
	   if(found == false ) {
          logger.Println("Invalid target protocol type for redistribution %s ", cfg.RedistributeActionInfo.RedistributeTargetProtocol)
		  return val,err
	   }
	   targetProtoType = retProto
	   logger.Printf("target protocol for RedistributeTargetProtocol %s is %d\n", cfg.RedistributeActionInfo.RedistributeTargetProtocol, targetProtoType)
	   redistributeActionInfo := RedistributeActionInfo{ redistributeTargetProtocol:targetProtoType}
       if cfg.RedistributeActionInfo.Redistribute == "Allow" {
	      redistributeActionInfo.redistribute = true	
	   } else if cfg.RedistributeActionInfo.Redistribute == "Block" {
	      redistributeActionInfo.redistribute = false	
	   } else {
	      logger.Println("Invalid redistribute option ",cfg.RedistributeActionInfo.Redistribute," - should be either Allow/Block")	
          err = errors.New("Invalid redistribute option")
		  return val,err
	   }
	   newPolicyAction := PolicyAction{name:cfg.Name,actionType:ribdCommonDefs.PolicyActionTypeRouteRedistribute,actionInfo:redistributeActionInfo ,localDBSliceIdx:(len(localPolicyActionsDB))}
       newPolicyAction.actionGetBulkInfo = cfg.RedistributeActionInfo.Redistribute + " Redistribute to Target Protocol " + cfg.RedistributeActionInfo.RedistributeTargetProtocol
		if ok := PolicyActionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyAction); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
	    updateLocalActionsDB(patriciaDB.Prefix(cfg.Name))
	} else {
		logger.Println("Duplicate action name")
		err = errors.New("Duplicate policy action definition")
		return val, err
	}
	return val, err
}
func (m RouteServiceHandler) CreatePolicyAction(cfg *ribd.PolicyActionConfig) (val bool, err error) {
	logger.Println("CreatePolicyAction")
	switch cfg.ActionType {
		case "RouteDisposition":
		   CreatePolicyRouteDispositionAction(cfg)
		   break
		case "Redistribution":
		   CreatePolicyRedistributionAction(cfg)
		   break
        case "SetAdminDistance":
		   CreatePolicyAdminDistanceAction(cfg)
		   break
		default:
		   logger.Println("Unknown action type ", cfg.ActionType)
		   err = errors.New("Unknown action type")
	}
	return val,err
}

/*
func (m RouteServiceHandler) GetBulkPolicyDefinitionStmtRedistributionActions( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStmtRedistributionActionsGetInfo, err error){
	logger.Println("getBulkPolicyDefinitionStmtRedistributionActions")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionStmtRedistributionAction = make ([]ribd.PolicyDefinitionStmtRedistributionAction, rcount)
	var nextNode *ribd.PolicyDefinitionStmtRedistributionAction
    var returnNodes []*ribd.PolicyDefinitionStmtRedistributionAction
	var returnGetInfo ribd.PolicyDefinitionStmtRedistributionActionsGetInfo
	i = 0
	policyActions := &returnGetInfo
	more := true
    if(localPolicyActionsDB == nil) {
		logger.Println("localPolicyActionsDB not initialized")
		return policyActions, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyActionsDB))) {
			logger.Println("All the policy actions fetched")
			more = false
			break
		}
		if(localPolicyActionsDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy action statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy actions fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyActionsDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyActionsDB.Get(localPolicyActionsDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyAction)
			if prefixNode.actionType != ribdCommonDefs.PolicyActionTypeRouteRedistribute {
				continue
			}
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.RedistributeTargetProtocol = ReverseRouteProtoTypeMapDB[prefixNode.actionInfo.(int)]
			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyDefinitionStmtRedistributionAction, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policyActions", validCount)
	policyActions.PolicyDefinitionStmtRedistributionActionList = returnNodes
	policyActions.StartIdx = fromIndex
	policyActions.EndIdx = toIndex+1
	policyActions.More = more
	policyActions.Count = validCount
	return policyActions, err
}*/

func (m RouteServiceHandler) GetBulkPolicyActionState( fromIndex ribd.Int, rcount ribd.Int) (policyActions *ribd.PolicyActionStateGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkPolicyActionState")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyActionState = make ([]ribd.PolicyActionState, rcount)
	var nextNode *ribd.PolicyActionState
    var returnNodes []*ribd.PolicyActionState
	var returnGetInfo ribd.PolicyActionStateGetInfo
	i = 0
	policyActions = &returnGetInfo
	more := true
    if(localPolicyActionsDB == nil) {
		logger.Println("PolicyDefinitionStmtMatchProtocolActionGetInfo not initialized")
		return policyActions, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyActionsDB))) {
			logger.Println("All the policy Actions fetched")
			more = false
			break
		}
		if(localPolicyActionsDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy Action statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy Actions fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyActionsDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyActionsDB.Get(localPolicyActionsDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyAction)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.ActionInfo = prefixNode.actionGetBulkInfo
            if prefixNode.policyStmtList != nil {
				nextNode.PolicyStmtList = make([]string,0)
			}
			for idx := 0;idx < len(prefixNode.policyStmtList);idx++ {
				nextNode.PolicyStmtList = append(nextNode.PolicyStmtList, prefixNode.policyStmtList[idx])
			}
 			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyActionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policyActions", validCount)
	policyActions.PolicyActionStateList = returnNodes
	policyActions.StartIdx = fromIndex
	policyActions.EndIdx = toIndex+1
	policyActions.More = more
	policyActions.Count = validCount
	return policyActions, err
}
