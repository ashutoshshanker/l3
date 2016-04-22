// policyApis.go
package rpc

import (
	"fmt"
	"ribd"
	"ribdInt"
)

func (m RIBDServicesHandler) CreatePolicyStmt(cfg *ribd.PolicyStmt) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyStatement"))
	m.server.PolicyStmtCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) DeletePolicyStmt(cfg *ribd.PolicyStmt) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyStatement for name ", cfg.Name))
	m.server.PolicyStmtDeleteConfCh <- cfg
	return true, err
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
	policyStmts,err = m.server.GetBulkPolicyStmtState(fromIndex,rcount,m.server.GlobalPolicyEngineDB)
	return policyStmts, err
}

func (m RIBDServicesHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Info(fmt.Sprintln("CreatePolicyDefinition"))
	m.server.PolicyDefinitionCreateConfCh <- cfg
	return true, err
}

func (m RIBDServicesHandler) DeletePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeletePolicyDefinition for name ", cfg.Name))
	m.server.PolicyDefinitionDeleteConfCh <- cfg
	return true, err
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
	policyStmts,err = m.server.GetBulkPolicyDefinitionState(fromIndex,rcount,m.server.GlobalPolicyEngineDB)
	return policyStmts, err
}

func (m RIBDServicesHandler) ApplyPolicy(source string ,policy string, action string , conditions []*ribdInt.ConditionInfo) (err error) {
	logger.Info(fmt.Sprintln("ApplyPolicy source:", source, " policy:", policy, " action:", action," number of extra conditions:",len(conditions)))
	return nil
}
