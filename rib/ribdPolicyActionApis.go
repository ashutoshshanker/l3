// ribdPolicyActionApis.go
package main

import (
	"fmt"
	"ribd"
	"utils/policy"
)
func (m RIBDServicesHandler) CreatePolicyAction(cfg *ribd.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	//m.PolicyActionCreateConfCh <- cfg
	val,err=m.ProcessPolicyActionConfigCreate(cfg)
	return val, err
}

func (m RIBDServicesHandler) ProcessPolicyActionConfigCreate(cfg *ribd.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyActionConfigCreate:CreatePolicyAction"))
	newAction := policy.PolicyActionConfig{Name: cfg.Name, ActionType: cfg.ActionType, SetAdminDistanceValue: int(cfg.SetAdminDistanceValue), Accept: cfg.Accept, Reject: cfg.Reject, RedistributeAction: cfg.RedistributeAction, RedistributeTargetProtocol: cfg.RedistributeTargetProtocol, NetworkStatementTargetProtocol: cfg.NetworkStatementTargetProtocol}
	val,err = PolicyEngineDB.CreatePolicyAction(newAction)
	return val,err
}

func (m RIBDServicesHandler) DeletePolicyAction(cfg *ribd.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	//m.PolicyActionDeleteConfCh <- cfg
	val,err=m.ProcessPolicyActionConfigDelete(cfg)
	return val, err
}

func (m RIBDServicesHandler) ProcessPolicyActionConfigDelete(cfg *ribd.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyActionConfigDelete:CreatePolicyAction"))
	newAction := policy.PolicyActionConfig{Name: cfg.Name}
	val,err = PolicyEngineDB.DeletePolicyAction(newAction)
	return val,err
}
func (m RIBDServicesHandler) UpdatePolicyAction(origconfig *ribd.PolicyAction , newconfig *ribd.PolicyAction , attrset []bool) (val bool, err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyAction"))
	return true,err
}
func (m RIBDServicesHandler) GetBulkPolicyActionState(fromIndex ribd.Int, rcount ribd.Int) (policyActions *ribd.PolicyActionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyActionState"))
	PolicyActionsDB := PolicyEngineDB.PolicyActionsDB
	localPolicyActionsDB := *PolicyEngineDB.LocalPolicyActionsDB
	var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyActionState = make([]ribd.PolicyActionState, rcount)
	var nextNode *ribd.PolicyActionState
	var returnNodes []*ribd.PolicyActionState
	var returnGetInfo ribd.PolicyActionStateGetInfo
	i = 0
	policyActions = &returnGetInfo
	more := true
	if localPolicyActionsDB == nil {
		logger.Info(fmt.Sprintln("PolicyDefinitionStmtMatchProtocolActionGetInfo not initialized"))
		return policyActions, err
	}
	for ; ; i++ {
		logger.Info(fmt.Sprintf("Fetching trie record for index %d\n", i+fromIndex))
		if i+fromIndex >= ribd.Int(len(localPolicyActionsDB)) {
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
			toIndex = ribd.Int(prefixNode.LocalDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*ribd.PolicyActionState, 0)
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
