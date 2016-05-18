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
package rpc

import (
	"fmt"
	"l3/rib/server"
	"ribd"
	"ribdInt"
	"utils/policy"
)

func (m RIBDServicesHandler) CreatePolicyStmt(cfg *ribd.PolicyStmt) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyStatement"))
	newPolicyStmt := policy.PolicyStmtConfig{Name: cfg.Name, MatchConditions: cfg.MatchConditions}
	newPolicyStmt.Conditions = make([]string, 0)
	for i := 0; i < len(cfg.Conditions); i++ {
		newPolicyStmt.Conditions = append(newPolicyStmt.Conditions, cfg.Conditions[i])
	}
	newPolicyStmt.Actions = make([]string, 0)
	newPolicyStmt.Actions = append(newPolicyStmt.Actions, cfg.Action)
	err = m.server.GlobalPolicyEngineDB.ValidatePolicyStatementCreate(newPolicyStmt)
	if err != nil {
		logger.Err(fmt.Sprintln("PolicyEngine validation failed with err: ", err))
		return false, err
	}
	m.server.PolicyStmtCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) DeletePolicyStmt(cfg *ribd.PolicyStmt) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyStatement for name ", cfg.Name))
	err = m.server.GlobalPolicyEngineDB.ValidatePolicyStatementDelete(policy.PolicyStmtConfig{Name: cfg.Name})
	if err != nil {
		logger.Err(fmt.Sprintln("PolicyEngine validation failed with err: ", err))
		return false, err
	}
	m.server.PolicyStmtDeleteConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) UpdatePolicyStmt(origconfig *ribd.PolicyStmt, newconfig *ribd.PolicyStmt, attrset []bool, op string) (val bool, err error) {
	return true, err
}
func (m RIBDServicesHandler) GetPolicyStmtState(name string) (*ribd.PolicyStmtState, error) {
	logger.Info("Get state for Policy Stmt")
	retState := ribd.NewPolicyStmtState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyStmtState(fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyStmtStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyStmtState"))
	policyStmts, err = m.server.GetBulkPolicyStmtState(fromIndex, rcount, m.server.GlobalPolicyEngineDB)
	return policyStmts, err
}

func (m RIBDServicesHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyDefinition"))
	newPolicy := policy.PolicyDefinitionConfig{Name: cfg.Name, Precedence: int(cfg.Priority), MatchType: cfg.MatchType, PolicyType: cfg.PolicyType}
	newPolicy.PolicyDefinitionStatements = make([]policy.PolicyDefinitionStmtPrecedence, 0)
	var policyDefinitionStatement policy.PolicyDefinitionStmtPrecedence
	for i := 0; i < len(cfg.StatementList); i++ {
		policyDefinitionStatement.Precedence = int(cfg.StatementList[i].Priority)
		policyDefinitionStatement.Statement = cfg.StatementList[i].Statement
		newPolicy.PolicyDefinitionStatements = append(newPolicy.PolicyDefinitionStatements, policyDefinitionStatement)
	}
	newPolicy.Extensions = server.PolicyExtensions{}
	err = m.server.GlobalPolicyEngineDB.ValidatePolicyDefinitionCreate(newPolicy)
	if err != nil {
		logger.Err(fmt.Sprintln("validation failed with err ", err))
		return false, err
	}
	m.server.PolicyDefinitionCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) DeletePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyDefinition for name ", cfg.Name))
	newPolicy := policy.PolicyDefinitionConfig{Name: cfg.Name}
	err = m.server.GlobalPolicyEngineDB.ValidatePolicyDefinitionDelete(newPolicy)
	if err != nil {
		logger.Err(fmt.Sprintln("validation failed with err ", err))
		return false, err
	}
	m.server.PolicyDefinitionDeleteConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) UpdatePolicyDefinition(origconfig *ribd.PolicyDefinition, newconfig *ribd.PolicyDefinition, attrset []bool, op string) (val bool, err error) {
	return true, err
}
func (m RIBDServicesHandler) GetPolicyDefinitionState(name string) (*ribd.PolicyDefinitionState, error) {
	logger.Info("Get state for Policy Definition")
	retState := ribd.NewPolicyDefinitionState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyDefinitionState(fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyDefinitionState"))
	policyStmts, err = m.server.GetBulkPolicyDefinitionState(fromIndex, rcount, m.server.GlobalPolicyEngineDB)
	return policyStmts, err
}

//this API is called by applications when user applies a policy to a entity and RIBD applies the policy/runs the policyEngine
func (m RIBDServicesHandler) ApplyPolicy(source string, policy string, action string, conditions []*ribdInt.ConditionInfo) (err error) {
	logger.Info(fmt.Sprintln("RIB handler ApplyPolicy source:", source, " policy:", policy, " action:", action, " conditions: "))
	for j := 0; j < len(conditions); j++ {
		logger.Info(fmt.Sprintf("ConditionType = %s :", conditions[j].ConditionType))
		switch conditions[j].ConditionType {
		case "MatchProtocol":
			logger.Info(fmt.Sprintln(conditions[j].Protocol))
		case "MatchDstIpPrefix":
		case "MatchSrcIpPrefix":
			logger.Info(fmt.Sprintln("IpPrefix:", conditions[j].IpPrefix, "MasklengthRange:", conditions[j].MasklengthRange))
		}
	}
	m.server.PolicyApplyCh <- server.ApplyPolicyInfo{source, policy, action, conditions}
	return nil
}

//this API is called when an external application has applied a policy and wants to update the application map for the policy in the global policy DB
func (m RIBDServicesHandler) UpdateApplyPolicy(source string, policy string, action string, conditions []*ribdInt.ConditionInfo) (err error) {
	logger.Info(fmt.Sprintln("RIB handler UpdateApplyPolicy source:", source, " policy:", policy, " action:", action, " conditions: "))
	for j := 0; j < len(conditions); j++ {
		logger.Info(fmt.Sprintf("ConditionType = %s :", conditions[j].ConditionType))
		switch conditions[j].ConditionType {
		case "MatchProtocol":
			logger.Info(fmt.Sprintln(conditions[j].Protocol))
		case "MatchDstIpPrefix":
		case "MatchSrcIpPrefix":
			logger.Info(fmt.Sprintln("IpPrefix:", conditions[j].IpPrefix, "MasklengthRange:", conditions[j].MasklengthRange))
		}
	}
	m.server.PolicyUpdateApplyCh <- server.ApplyPolicyInfo{source, policy, action, conditions}
	return nil
}
