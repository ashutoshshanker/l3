//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

package server

import (
	"fmt"
	"ribd"
	"ribdInt"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)
/*
   This structure can be used along with policyDefinitionConfig object to pass on any application specific
   info to policy engine
*/
type PolicyExtensions struct {
	hitCounter    int
	routeList     []string
	routeInfoList []ribdInt.Routes
}
type Policy struct {
	*policy.Policy
	hitCounter    int
	routeList     []string
	routeInfoList []ribdInt.Routes
}

func (m RIBDServer) ProcessPolicyConditionConfigCreate(cfg *ribd.PolicyCondition, db *policy.PolicyEngineDB) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyConditionConfigCreate:CreatePolicyConditioncfg: ", cfg.Name))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name, ConditionType: cfg.ConditionType, MatchProtocolConditionInfo: cfg.Protocol}
	matchPrefix := policy.PolicyPrefix{IpPrefix: cfg.IpPrefix, MasklengthRange: cfg.MaskLengthRange}
	newPolicy.MatchDstIpPrefixConditionInfo = policy.PolicyDstIpMatchPrefixSetCondition{Prefix: matchPrefix}
	val, err = db.CreatePolicyCondition(newPolicy)
	return val, err
}

func (m RIBDServer) ProcessPolicyConditionConfigDelete(cfg *ribd.PolicyCondition, db *policy.PolicyEngineDB) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyConditionConfigDelete:DeletePolicyCondition: ", cfg.Name))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name}
	val, err = db.DeletePolicyCondition(newPolicy)
	return val, err
}

func (m RIBDServer) ProcessPolicyActionConfigCreate(cfg *ribdInt.PolicyAction, db *policy.PolicyEngineDB) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyActionConfigCreate:CreatePolicyAction"))
	newAction := policy.PolicyActionConfig{Name: cfg.Name, ActionType: cfg.ActionType, SetAdminDistanceValue: int(cfg.SetAdminDistanceValue), Accept: cfg.Accept, Reject: cfg.Reject, RedistributeAction: cfg.RedistributeAction, RedistributeTargetProtocol: cfg.RedistributeTargetProtocol, NetworkStatementTargetProtocol: cfg.NetworkStatementTargetProtocol}
	val, err = db.CreatePolicyAction(newAction)
	return val, err
}

func (m RIBDServer) ProcessPolicyActionConfigDelete(cfg *ribdInt.PolicyAction, db *policy.PolicyEngineDB) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyActionConfigDelete:CreatePolicyAction"))
	newAction := policy.PolicyActionConfig{Name: cfg.Name}
	val, err = db.DeletePolicyAction(newAction)
	return val, err
}

func (m RIBDServer) ProcessPolicyStmtConfigCreate(cfg *ribd.PolicyStmt, db *policy.PolicyEngineDB) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyStatementCreate:CreatePolicyStatement"))
	newPolicyStmt := policy.PolicyStmtConfig{Name: cfg.Name, MatchConditions: cfg.MatchConditions}
	newPolicyStmt.Conditions = make([]string, 0)
	for i := 0; i < len(cfg.Conditions); i++ {
		newPolicyStmt.Conditions = append(newPolicyStmt.Conditions, cfg.Conditions[i])
	}
	newPolicyStmt.Actions = make([]string, 0)
	newPolicyStmt.Actions = append(newPolicyStmt.Actions, cfg.Action)
	err = db.CreatePolicyStatement(newPolicyStmt)
	return err
}

func (m RIBDServer) ProcessPolicyStmtConfigDelete(cfg *ribd.PolicyStmt, db *policy.PolicyEngineDB) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyStatementDelete:DeletePolicyStatement for name ", cfg.Name))
	stmt := policy.PolicyStmtConfig{Name: cfg.Name}
	err = db.DeletePolicyStatement(stmt)
	return err
}

func (m RIBDServer) ProcessPolicyDefinitionConfigCreate(cfg *ribd.PolicyDefinition, db *policy.PolicyEngineDB) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyDefinitionCreate:CreatePolicyDefinition"))
	newPolicy := policy.PolicyDefinitionConfig{Name: cfg.Name, Precedence: int(cfg.Priority), MatchType: cfg.MatchType}
	newPolicy.PolicyDefinitionStatements = make([]policy.PolicyDefinitionStmtPrecedence, 0)
	var policyDefinitionStatement policy.PolicyDefinitionStmtPrecedence
	for i := 0; i < len(cfg.StatementList); i++ {
		policyDefinitionStatement.Precedence = int(cfg.StatementList[i].Priority)
		policyDefinitionStatement.Statement = cfg.StatementList[i].Statement
		newPolicy.PolicyDefinitionStatements = append(newPolicy.PolicyDefinitionStatements, policyDefinitionStatement)
	}
	newPolicy.Extensions = PolicyExtensions{}
	err = db.CreatePolicyDefinition(newPolicy)
	return err
}

func (m RIBDServer) ProcessPolicyDefinitionConfigDelete(cfg *ribd.PolicyDefinition, db *policy.PolicyEngineDB) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyDefinitionDelete:DeletePolicyDefinition for name ", cfg.Name))
	policy := policy.PolicyDefinitionConfig{Name: cfg.Name}
	err = db.DeletePolicyDefinition(policy)
	return err
}

func (m RIBDServer) GetBulkPolicyConditionState(fromIndex ribd.Int, rcount ribd.Int, db *policy.PolicyEngineDB) (policyConditions *ribd.PolicyConditionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyConditionState"))
	PolicyConditionsDB := db.PolicyConditionsDB
	localPolicyConditionsDB := *db.LocalPolicyConditionsDB
	var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyConditionState = make([]ribd.PolicyConditionState, rcount)
	var nextNode *ribd.PolicyConditionState
	var returnNodes []*ribd.PolicyConditionState
	var returnGetInfo ribd.PolicyConditionStateGetInfo
	i = 0
	policyConditions = &returnGetInfo
	more := true
	if localPolicyConditionsDB == nil {
		logger.Info(fmt.Sprintln("PolicyDefinitionStmtMatchProtocolConditionGetInfo not initialized"))
		return policyConditions, err
	}
	for ; ; i++ {
		logger.Info(fmt.Sprintf("Fetching trie record for index %d\n", i+fromIndex))
		if i+fromIndex >= ribd.Int(len(localPolicyConditionsDB)) {
			logger.Info(fmt.Sprintln("All the policy conditions fetched"))
			more = false
			break
		}
		if localPolicyConditionsDB[i+fromIndex].IsValid == false {
			logger.Info(fmt.Sprintln("Invalid policy condition statement"))
			continue
		}
		if validCount == rcount {
			logger.Info(fmt.Sprintln("Enough policy conditions fetched"))
			break
		}
		logger.Info(fmt.Sprintf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyConditionsDB[i+fromIndex].Prefix)))
		prefixNodeGet := PolicyConditionsDB.Get(localPolicyConditionsDB[i+fromIndex].Prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(policy.PolicyCondition)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.Name
			nextNode.ConditionInfo = prefixNode.ConditionGetBulkInfo
			if prefixNode.PolicyStmtList != nil {
				nextNode.PolicyStmtList = make([]string, 0)
			}
			for idx := 0; idx < len(prefixNode.PolicyStmtList); idx++ {
				nextNode.PolicyStmtList = append(nextNode.PolicyStmtList, prefixNode.PolicyStmtList[idx])
			}
			toIndex = ribd.Int(prefixNode.LocalDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*ribd.PolicyConditionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Info(fmt.Sprintf("Returning %d list of policyConditions", validCount))
	policyConditions.PolicyConditionStateList = returnNodes
	policyConditions.StartIdx = fromIndex
	policyConditions.EndIdx = toIndex + 1
	policyConditions.More = more
	policyConditions.Count = validCount
	return policyConditions, err
}

/*
func (m RIBDServer) GetBulkPolicyActionState(fromIndex ribd.Int, rcount ribd.Int) (policyActions *ribdInt.PolicyActionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyActionState"))
	PolicyActionsDB := PolicyEngineDB.PolicyActionsDB
	localPolicyActionsDB := *PolicyEngineDB.LocalPolicyActionsDB
	var i, validCount, toIndex ribd.Int
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
*/
func (m RIBDServer) GetBulkPolicyStmtState(fromIndex ribd.Int, rcount ribd.Int, db *policy.PolicyEngineDB) (policyStmts *ribd.PolicyStmtStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyStmtState"))
	PolicyStmtDB := db.PolicyStmtDB
	localPolicyStmtDB := *db.LocalPolicyStmtDB
	var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyStmtState = make([]ribd.PolicyStmtState, rcount)
	var nextNode *ribd.PolicyStmtState
	var returnNodes []*ribd.PolicyStmtState
	var returnGetInfo ribd.PolicyStmtStateGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
	if localPolicyStmtDB == nil {
		logger.Info(fmt.Sprintln("destNetSlice not initialized"))
		return policyStmts, err
	}
	for ; ; i++ {
		logger.Info(fmt.Sprintf("Fetching trie record for index %d\n", i+fromIndex))
		if i+fromIndex >= ribd.Int(len(localPolicyStmtDB)) {
			logger.Info(fmt.Sprintln("All the policy statements fetched"))
			more = false
			break
		}
		if localPolicyStmtDB[i+fromIndex].IsValid == false {
			logger.Info(fmt.Sprintln("Invalid policy statement"))
			continue
		}
		if validCount == rcount {
			logger.Info(fmt.Sprintln("Enough policy statements fetched"))
			break
		}
		logger.Info(fmt.Sprintf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyStmtDB[i+fromIndex].Prefix)))
		prefixNodeGet := PolicyStmtDB.Get(localPolicyStmtDB[i+fromIndex].Prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(policy.PolicyStmt)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.Name
			nextNode.Conditions = prefixNode.Conditions
			nextNode.Action = prefixNode.Actions[0]
			if prefixNode.PolicyList != nil {
				nextNode.PolicyList = make([]string, 0)
			}
			for idx := 0; idx < len(prefixNode.PolicyList); idx++ {
				nextNode.PolicyList = append(nextNode.PolicyList, prefixNode.PolicyList[idx])
			}
			toIndex = ribd.Int(prefixNode.LocalDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*ribd.PolicyStmtState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Info(fmt.Sprintf("Returning %d list of policyStmts", validCount))
	policyStmts.PolicyStmtStateList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex + 1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}

func (m RIBDServer) GetBulkPolicyDefinitionState(fromIndex ribd.Int, rcount ribd.Int, db *policy.PolicyEngineDB) (policyStmts *ribd.PolicyDefinitionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyDefinitionState"))
	PolicyDB := db.PolicyDB
	localPolicyDB := *db.LocalPolicyDB
	var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionState = make([]ribd.PolicyDefinitionState, rcount)
	var nextNode *ribd.PolicyDefinitionState
	var returnNodes []*ribd.PolicyDefinitionState
	var returnGetInfo ribd.PolicyDefinitionStateGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
	if localPolicyDB == nil {
		logger.Info(fmt.Sprintln("LocalPolicyDB not initialized"))
		return policyStmts, err
	}
	for ; ; i++ {
		logger.Info(fmt.Sprintf("Fetching trie record for index %d\n", i+fromIndex))
		if i+fromIndex >= ribd.Int(len(localPolicyDB)) {
			logger.Info(fmt.Sprintln("All the policies fetched"))
			more = false
			break
		}
		if localPolicyDB[i+fromIndex].IsValid == false {
			logger.Info(fmt.Sprintln("Invalid policy"))
			continue
		}
		if validCount == rcount {
			logger.Info(fmt.Sprintln("Enough policies fetched"))
			break
		}
		logger.Info(fmt.Sprintf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyDB[i+fromIndex].Prefix)))
		prefixNodeGet := PolicyDB.Get(localPolicyDB[i+fromIndex].Prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(policy.Policy)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.Name
			extensions := prefixNode.Extensions.(PolicyExtensions)
			nextNode.IpPrefixList = make([]string, 0)
			for k := 0; k < len(extensions.routeList); k++ {
				nextNode.IpPrefixList = append(nextNode.IpPrefixList, extensions.routeList[k])
			}
			toIndex = ribd.Int(prefixNode.LocalDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*ribd.PolicyDefinitionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Info(fmt.Sprintf("Returning %d list of policies", validCount))
	policyStmts.PolicyDefinitionStateList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex + 1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}

func (m RIBDServer) UpdateApplyPolicy(info ApplyPolicyInfo, apply bool, db *policy.PolicyEngineDB) {
	var err error
	conditionName := ""
	source := info.Source
	policyName := info.Policy
	action := info.Action
	var policyAction policy.PolicyAction
	conditionNameList := make([]string, 0)

	policyDB := db.PolicyDB
	policyConditionsDB := db.PolicyConditionsDB

	nodeGet := policyDB.Get(patriciaDB.Prefix(policyName))
	if nodeGet == nil {
		logger.Err(fmt.Sprintln("Policy ", policyName, " not defined"))
		return
	}
	node := nodeGet.(policy.Policy)
	conditions := make([]ribdInt.ConditionInfo, 0)
	for i := 0; i < len(info.Conditions); i++ {
		conditions = append(conditions, *info.Conditions[i])
	}
	logger.Info(fmt.Sprintln("RIB handler UpdateApplyPolicy source:", source, " policy:", policyName, " action:", action, " apply:", apply, "conditions: "))
	for j := 0; j < len(conditions); j++ {
		logger.Info(fmt.Sprintf("ConditionType = %s :", conditions[j].ConditionType))
		switch conditions[j].ConditionType {
		case "MatchProtocol":
			logger.Info(fmt.Sprintln(conditions[j].Protocol))
			conditionName := "Match" + conditions[j].Protocol
			ok := policyConditionsDB.Match(patriciaDB.Prefix(conditionName))
			if !ok {
				logger.Info(fmt.Sprintln("Define condition ", conditionName))
				policyCondition := ribd.PolicyCondition{Name: conditionName, ConditionType: conditions[j].ConditionType, Protocol: conditions[j].Protocol}
				_, err = m.ProcessPolicyConditionConfigCreate(&policyCondition, db)
			}
		case "MatchDstIpPrefix":
		case "MatchSrcIpPrefix":
			logger.Info(fmt.Sprintln("IpPrefix:", conditions[j].IpPrefix, "MasklengthRange:", conditions[j].MasklengthRange))
		}
		if err == nil {
			conditionNameList = append(conditionNameList, conditionName)
		}
	}
	//define Action
	switch action {
	case "Redistribution":
		logger.Info("Setting up Redistribution action map")
		redistributeActionInfo := policy.RedistributeActionInfo{true, source}
		policyAction = policy.PolicyAction{Name: action, ActionType: policyCommonDefs.PolicyActionTypeRouteRedistribute, ActionInfo: redistributeActionInfo}
		break
	default:
		logger.Info(fmt.Sprintln("Action ", action, "currently a no-op"))
	}
	db.UpdateApplyPolicy(policy.ApplyPolicyInfo{node, policyAction, conditionNameList}, apply)
	return
}
