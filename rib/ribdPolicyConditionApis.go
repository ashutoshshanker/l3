// ribdPolicyConditionApis.go
package main

import (
	"fmt"
	"ribd"
	"utils/policy"
)

func (m RouteServiceHandler) CreatePolicyPrefixSet(cfg *ribd.PolicyPrefixSet) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyPrefixSet"))
	return val, err
}

func (m RouteServiceHandler) CreatePolicyCondition(cfg *ribd.PolicyConditionConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyConditioncfg"))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name, ConditionType: cfg.ConditionType, MatchProtocolConditionInfo: cfg.MatchProtocolConditionInfo}
	if cfg.MatchDstIpPrefixConditionInfo != nil {
		matchPrefix := policy.PolicyPrefix{IpPrefix: cfg.MatchDstIpPrefixConditionInfo.Prefix.IpPrefix, MasklengthRange: cfg.MatchDstIpPrefixConditionInfo.Prefix.MasklengthRange}
		newPolicy.MatchDstIpPrefixConditionInfo = policy.PolicyDstIpMatchPrefixSetCondition{PrefixSet: cfg.MatchDstIpPrefixConditionInfo.PrefixSet, Prefix: matchPrefix}
	}
	err = PolicyEngineDB.CreatePolicyCondition(newPolicy)
	return val, err
}
func (m RouteServiceHandler) DeletePolicyCondition(cfg *ribd.PolicyConditionConfig) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyCondition"))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name}
	err = PolicyEngineDB.DeletePolicyCondition(newPolicy)
	return val, err
}
func (m RouteServiceHandler) GetBulkPolicyConditionState(fromIndex ribd.Int, rcount ribd.Int) (policyConditions *ribd.PolicyConditionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyConditionState"))
	PolicyConditionsDB := PolicyEngineDB.PolicyConditionsDB
	localPolicyConditionsDB := *PolicyEngineDB.LocalPolicyConditionsDB
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
