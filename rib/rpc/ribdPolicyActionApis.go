// ribdPolicyActionApis.go
package rpc

import (
	"fmt"
	"ribd"
)

func (m RIBDServicesHandler) CreatePolicyAction(cfg *ribd.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	m.server.PolicyActionCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) DeletePolicyAction(cfg *ribd.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	m.server.PolicyActionDeleteConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) UpdatePolicyAction(origconfig *ribd.PolicyAction, newconfig *ribd.PolicyAction, attrset []bool) (val bool, err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyAction"))
	return true, err
}
func (m RIBDServicesHandler) GetPolicyActionState(name string) (*ribd.PolicyActionState, error) {
	logger.Info("Get state for Policy Action")
	retState := ribd.NewPolicyActionState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyActionState(fromIndex ribd.Int, rcount ribd.Int) (policyActions *ribd.PolicyActionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyActionState"))
	policyActions,err = m.server.GetBulkPolicyActionState(fromIndex,rcount)
	return policyActions, err
}
