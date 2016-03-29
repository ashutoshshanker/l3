// ribdPolicyConditionApis.go
package main

import (
	"fmt"
	"ribd"
	"utils/policy"
)
func (m RIBDServicesHandler) CreatePolicyCondition(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyConditioncfg: ",cfg.Name))
	//m.PolicyConditionCreateConfCh <- cfg
	val,err = m.ProcessPolicyConditionConfigCreate(cfg)
	return val,err
}
func (m RIBDServicesHandler) ProcessPolicyConditionConfigCreate(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyConditionConfigCreate:CreatePolicyConditioncfg: ",cfg.Name))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name, ConditionType: cfg.ConditionType, MatchProtocolConditionInfo: cfg.MatchProtocol}
	matchPrefix := policy.PolicyPrefix{IpPrefix: cfg.IpPrefix, MasklengthRange: cfg.MaskLengthRange}
	newPolicy.MatchDstIpPrefixConditionInfo = policy.PolicyDstIpMatchPrefixSetCondition{ Prefix: matchPrefix}
/*	if cfg.MatchDstIpPrefixConditionInfo != nil {
		matchPrefix := policy.PolicyPrefix{IpPrefix: cfg.MatchDstIpPrefixConditionInfo.Prefix.IpPrefix, MasklengthRange: cfg.MatchDstIpPrefixConditionInfo.Prefix.MasklengthRange}
		newPolicy.MatchDstIpPrefixConditionInfo = policy.PolicyDstIpMatchPrefixSetCondition{PrefixSet: cfg.MatchDstIpPrefixConditionInfo.PrefixSet, Prefix: matchPrefix}
	}*/
	val,err = PolicyEngineDB.CreatePolicyCondition(newPolicy)
	return val,err
}
func (m RIBDServicesHandler) DeletePolicyCondition(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyConditionConfig: ",cfg.Name))
	//m.PolicyConditionDeleteConfCh <- cfg
	val,err = m.ProcessPolicyConditionConfigDelete(cfg)
	return val,err
}
func (m RIBDServicesHandler) ProcessPolicyConditionConfigDelete(cfg *ribd.PolicyCondition) (val bool,  err error) {
	logger.Info(fmt.Sprintln("ProcessPolicyConditionConfigDelete:DeletePolicyCondition: ",cfg.Name))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name}
	val,err = PolicyEngineDB.DeletePolicyCondition(newPolicy)
	return val,err
}
func (m RIBDServicesHandler) UpdatePolicyCondition(origconfig *ribd.PolicyCondition , newconfig *ribd.PolicyCondition , attrset []bool) (val bool, err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyConditionConfig:UpdatePolicyCondition: ",newconfig.Name))
	return true,err
}
func (m RIBDServicesHandler) GetBulkPolicyConditionState(fromIndex ribd.Int, rcount ribd.Int) (policyConditions *ribd.PolicyConditionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
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
