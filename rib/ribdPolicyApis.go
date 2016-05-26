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

// policyApis.go
package main

import (
	"fmt"
	"ribd"
	"ribdInt"
	"utils/policy"
)

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

func (m RIBDServicesHandler) CreatePolicyStmt(cfg *ribd.PolicyStmt) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyStatement"))
	m.PolicyStmtCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) ProcessPolicyStmtConfigCreate(cfg *ribd.PolicyStmt) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyStatementCreate:CreatePolicyStatement"))
	newPolicyStmt := policy.PolicyStmtConfig{Name: cfg.Name, MatchConditions: cfg.MatchConditions}
	newPolicyStmt.Actions = make([]string, 0)
	newPolicyStmt.Actions = append(newPolicyStmt.Actions,cfg.Action)
	newPolicyStmt.Conditions = make([]string, 0)
	for i := 0; i < len(cfg.Conditions); i++ {
		newPolicyStmt.Conditions = append(newPolicyStmt.Conditions, cfg.Conditions[i])
	}
	err = PolicyEngineDB.CreatePolicyStatement(newPolicyStmt)
	return err
}

func (m RIBDServicesHandler) DeletePolicyStmt(cfg *ribd.PolicyStmt) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyStatement for name ", cfg.Name))
	m.PolicyStmtDeleteConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) ProcessPolicyStmtConfigDelete(cfg *ribd.PolicyStmt) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyStatementDelete:DeletePolicyStatement for name ", cfg.Name))
	stmt := policy.PolicyStmtConfig{Name: cfg.Name}
	err = PolicyEngineDB.DeletePolicyStatement(stmt)
	return err
}
func (m RIBDServicesHandler) UpdatePolicyStmt(origconfig *ribd.PolicyStmt, newconfig *ribd.PolicyStmt, attrset []bool) (val bool, err error) {
	return true, err
}
func (m RIBDServicesHandler) GetPolicyStmtState(name string) (*ribd.PolicyStmtState, error) {
	logger.Info("Get state for Policy Stmt")
	retState := ribd.NewPolicyStmtState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyStmtState(fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyStmtStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyStmtState"))
	PolicyStmtDB := PolicyEngineDB.PolicyStmtDB
	localPolicyStmtDB := *PolicyEngineDB.LocalPolicyStmtDB
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

func (m RIBDServicesHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyDefinition"))
	m.PolicyDefinitionCreateConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) ProcessPolicyDefinitionConfigCreate(cfg *ribd.PolicyDefinition) (err error) {
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
	err = PolicyEngineDB.CreatePolicyDefinition(newPolicy)
	return err
}

func (m RIBDServicesHandler) DeletePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyDefinition for name ", cfg.Name))
	m.PolicyDefinitionDeleteConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) ProcessPolicyDefinitionConfigDelete(cfg *ribd.PolicyDefinition) (err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyDefinitionDelete:DeletePolicyDefinition for name ", cfg.Name))
	policy := policy.PolicyDefinitionConfig{Name: cfg.Name}
	err = PolicyEngineDB.DeletePolicyDefinition(policy)
	return err
}
func (m RIBDServicesHandler) UpdatePolicyDefinition(origconfig *ribd.PolicyDefinition, newconfig *ribd.PolicyDefinition, attrset []bool) (val bool, err error) {
	return true, err
}
func (m RIBDServicesHandler) GetPolicyDefinitionState(name string) (*ribd.PolicyDefinitionState, error) {
	logger.Info("Get state for Policy Definition")
	retState := ribd.NewPolicyDefinitionState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyDefinitionState(fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyDefinitionState"))
	PolicyDB := PolicyEngineDB.PolicyDB
	localPolicyDB := *PolicyEngineDB.LocalPolicyDB
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
