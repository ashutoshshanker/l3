// ribdPolicyActionApis.go
package rpc

import (
	"fmt"
	"ribdInt"
)

func (m RIBDServicesHandler) CreatePolicyAction(cfg *ribdInt.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	m.server.PolicyActionCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) DeletePolicyAction(cfg *ribdInt.PolicyAction) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyAction"))
	m.server.PolicyActionDeleteConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) UpdatePolicyAction(origconfig *ribdInt.PolicyAction, newconfig *ribdInt.PolicyAction, attrset []bool) (val bool, err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyAction"))
	return true, err
}

/*func (m RIBDServicesHandler) GetPolicyActionState(name string) (*ribdInt.PolicyActionState, error) {
	logger.Info("Get state for Policy Action")
	retState := ribd.NewPolicyActionState()
	return retState, nil
}
func (m RIBDServicesHandler) GetBulkPolicyActionState(fromIndex ribd.Int, rcount ribd.Int) (policyActions *ribdInt.PolicyActionStateGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Info(fmt.Sprintln("GetBulkPolicyActionState"))
	policyActions,err = m.server.GetBulkPolicyActionState(fromIndex,rcount)
	return policyActions, err
}*/
