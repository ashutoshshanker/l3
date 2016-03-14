// policyApis.go
package main

import (
	"fmt"
	"ribd"
	"utils/policy"
)

type PolicyExtensions struct {
	hitCounter    int
	routeList     []string
	routeInfoList []ribd.Routes
}
type Policy struct {
	*policy.Policy
	hitCounter    int
	routeList     []string
	routeInfoList []ribd.Routes
}

func (m RouteServiceHandler) CreatePolicyStatement(cfg *ribd.PolicyStmtConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyStatement"))
	newPolicyStmt := policy.PolicyStmtConfig{Name: cfg.Name, MatchConditions: cfg.MatchConditions}
	newPolicyStmt.Conditions = make([]string, 0)
	for i := 0; i < len(cfg.Conditions); i++ {
		newPolicyStmt.Conditions = append(newPolicyStmt.Conditions, cfg.Conditions[i])
	}
	for i := 0; i < len(cfg.Actions); i++ {
		newPolicyStmt.Actions = append(newPolicyStmt.Actions, cfg.Actions[i])
	}
	err = PolicyEngineDB.CreatePolicyStatement(newPolicyStmt)
	return val, err
}

func (m RouteServiceHandler) DeletePolicyStatement(cfg *ribd.PolicyStmtConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyStatement for name ", cfg.Name))
	stmt := policy.PolicyStmtConfig{Name: cfg.Name}
	err = PolicyEngineDB.DeletePolicyStatement(stmt)
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyStmtState(fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyStmtStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
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
			nextNode.Actions = prefixNode.Actions
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

func (m RouteServiceHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinitionConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyDefinition"))
	newPolicy := policy.PolicyDefinitionConfig{Name: cfg.Name, Precedence: int(cfg.Precedence), MatchType: cfg.MatchType}
	newPolicy.PolicyDefinitionStatements = make([]policy.PolicyDefinitionStmtPrecedence, 0)
	var policyDefinitionStatement policy.PolicyDefinitionStmtPrecedence
	for i := 0; i < len(cfg.PolicyDefinitionStatements); i++ {
		policyDefinitionStatement.Precedence = int(cfg.PolicyDefinitionStatements[i].Precedence)
		policyDefinitionStatement.Statement = cfg.PolicyDefinitionStatements[i].Statement
		newPolicy.PolicyDefinitionStatements = append(newPolicy.PolicyDefinitionStatements, policyDefinitionStatement)
	}
	newPolicy.Extensions = PolicyExtensions{}
	err = PolicyEngineDB.CreatePolicyDefinition(newPolicy)
	return val, err
}

func (m RouteServiceHandler) DeletePolicyDefinition(cfg *ribd.PolicyDefinitionConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyDefinition for name ", cfg.Name))
	policy := policy.PolicyDefinitionConfig{Name: cfg.Name}
	err = PolicyEngineDB.DeletePolicyDefinition(policy)
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyDefinitionState(fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
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
			nextNode.HitCounter = ribd.Int(extensions.hitCounter)
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
