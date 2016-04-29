// ribdPolicyConditionApis.go
package rpc

import (
	"fmt"
	"ribd"
	"utils/policy"
)

func (m RIBDServicesHandler) CreatePolicyCondition(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyConditioncfg: ", cfg.Name))
	newPolicy := policy.PolicyConditionConfig{Name: cfg.Name, ConditionType: cfg.ConditionType, MatchProtocolConditionInfo: cfg.Protocol}
	matchPrefix := policy.PolicyPrefix{IpPrefix: cfg.IpPrefix, MasklengthRange: cfg.MaskLengthRange}
	newPolicy.MatchDstIpPrefixConditionInfo = policy.PolicyDstIpMatchPrefixSetCondition{Prefix: matchPrefix}
	err = m.server.GlobalPolicyEngineDB.ValidateConditionConfigCreate(newPolicy)
	if err != nil {
		logger.Err(fmt.Sprintln("PolicyEngine validation failed with err: ",err))
		return false,err
	}
	m.server.PolicyConditionCreateConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) DeletePolicyCondition(cfg *ribd.PolicyCondition) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyConditionConfig: ", cfg.Name))
	m.server.PolicyConditionDeleteConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) UpdatePolicyCondition(origconfig *ribd.PolicyCondition, newconfig *ribd.PolicyCondition, attrset []bool) (val bool, err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyConditionConfig:UpdatePolicyCondition: ", newconfig.Name))
	return true, err
}
func (m RIBDServicesHandler) GetPolicyConditionState(name string) (*ribd.PolicyConditionState, error) {
	logger.Info("Get state for Policy Condition")
	retState := ribd.NewPolicyConditionState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyConditionState(fromIndex ribd.Int, rcount ribd.Int) (policyConditions *ribd.PolicyConditionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyConditionState"))
	ret,err := m.server.GetBulkPolicyConditionState(fromIndex,rcount,m.server.GlobalPolicyEngineDB)
	return ret, err
}
