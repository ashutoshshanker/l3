// ribdPolicyActionApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
)

var PolicyActionsDB = patriciaDB.NewTrie()
type PolicyAction struct {
	name          string
	actionType int
	actionInfo interface {}
	localDBSliceIdx int
}
var localPolicyActionsDB []localDB

func (m RouteServiceHandler) CreatePolicyDefinitionStmtRedistributionAction(cfg *ribd.PolicyDefinitionStmtRedistributionAction) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStmtRedistributionAction")
	targetProtoType := -1

	policyAction := PolicyActionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyAction == nil) {
	   logger.Println("Defining a new policy action with name ", cfg.Name)
	   retProto,found := RouteProtocolTypeMapDB[cfg.RedistributeTargetProtocol]
	   if(found == false ) {
          logger.Println("Invalid target protocol type for redistribution %s ", cfg.RedistributeTargetProtocol)
		  return val,err
	   }
	   targetProtoType = retProto
	   logger.Printf("target protocol for RedistributeTargetProtocol %s is %d\n", cfg.RedistributeTargetProtocol, targetProtoType)
	   newPolicyAction := PolicyAction{name:cfg.Name,actionType:ribdCommonDefs.PolicyActionTypeRouteRedistribute,actionInfo:targetProtoType ,localDBSliceIdx:(len(localPolicyActionsDB))}
		if ok := PolicyActionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyAction); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyActionsDB == nil) {
			localPolicyActionsDB = make([]localDB, 0)
		} 
	    localPolicyActionsDB = append(localPolicyActionsDB, localDBRecord)
	} else {
		logger.Println("Duplicate action name")
		err = errors.New("Duplicate policy action definition")
		return val, err
	}
	return val, err
}

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
}