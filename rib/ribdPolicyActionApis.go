// ribdPolicyActionApis.go
package main

import (
	"fmt"
	"ribd"
	"ribdInt"
	"utils/policy"
)

func (m RIBDServicesHandler) CreatePolicyActionConfig(cfg *ribd.PolicyActionConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	newAction := policy.PolicyActionConfig{Name: cfg.Name, ActionType: cfg.ActionType, SetAdminDistanceValue: int(cfg.SetAdminDistanceValue), Accept: cfg.Accept, Reject: cfg.Reject, RedistributeAction: cfg.RedistributeAction, RedistributeTargetProtocol: cfg.RedistributeTargetProtocol, NetworkStatementTargetProtocol: cfg.NetworkStatementTargetProtocol}
	err = PolicyEngineDB.CreatePolicyAction(newAction)
	return val, err
}

func (m RIBDServicesHandler) DeletePolicyActionConfig(cfg *ribd.PolicyActionConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	newAction := policy.PolicyActionConfig{Name: cfg.Name}
	err = PolicyEngineDB.DeletePolicyAction(newAction)
	return val, err
}
func (m RIBDServicesHandler) UpdatePolicyActionConfig(origconfig *ribd.PolicyActionConfig , newconfig *ribd.PolicyActionConfig , attrset []bool) (val bool, err error) {
	return val,err
}
func (m RIBDServicesHandler) GetBulkPolicyActionState(fromIndex ribdInt.Int, rcount ribdInt.Int) (policyActions *ribdInt.PolicyActionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyActionState"))
	PolicyActionsDB := PolicyEngineDB.PolicyActionsDB
	localPolicyActionsDB := *PolicyEngineDB.LocalPolicyActionsDB
	var i, validCount, toIndex ribdInt.Int
	var tempNode []ribdInt.PolicyActionState = make([]ribdInt.PolicyActionState, rcount)
	var nextNode *ribdInt.PolicyActionState
	var returnNodes []*ribdInt.PolicyActionState
	var returnGetInfo ribdInt.PolicyActionStateGetInfo
	i = 0
	policyActions = &returnGetInfo
	more := true
	if localPolicyActionsDB == nil {
		logger.Info(fmt.Sprintln("PolicyDefinitionStmtMatchProtocolActionGetInfo not initialized"))
		return policyActions, err
	}
	for ; ; i++ {
		logger.Info(fmt.Sprintf("Fetching trie record for index %d\n", i+fromIndex))
		if i+fromIndex >= ribdInt.Int(len(localPolicyActionsDB)) {
			logger.Info(fmt.Sprintln("All the policy Actions fetched"))
			more = false
			break
		}
		if localPolicyActionsDB[i+fromIndex].IsValid == false {
			logger.Info(fmt.Sprintln("Invalid policy Action statement"))
			continue
		}
		if validCount == rcount {
			logger.Info(fmt.Sprintln("Enough policy Actions fetched"))
			break
		}
		logger.Info(fmt.Sprintf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyActionsDB[i+fromIndex].Prefix)))
		prefixNodeGet := PolicyActionsDB.Get(localPolicyActionsDB[i+fromIndex].Prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(policy.PolicyAction)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.Name
			nextNode.ActionInfo = prefixNode.ActionGetBulkInfo
			if prefixNode.PolicyStmtList != nil {
				nextNode.PolicyStmtList = make([]string, 0)
			}
			for idx := 0; idx < len(prefixNode.PolicyStmtList); idx++ {
				nextNode.PolicyStmtList = append(nextNode.PolicyStmtList, prefixNode.PolicyStmtList[idx])
			}
			toIndex = ribdInt.Int(prefixNode.LocalDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*ribdInt.PolicyActionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Info(fmt.Sprintf("Returning %d list of policyActions", validCount))
	policyActions.PolicyActionStateList = returnNodes
	policyActions.StartIdx = fromIndex
	policyActions.EndIdx = toIndex + 1
	policyActions.More = more
	policyActions.Count = validCount
	return policyActions, err
}
