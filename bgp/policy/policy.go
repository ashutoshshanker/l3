// policy.go
package server

import (
	"fmt"
	bgprib "l3/bgp/rib"
	"utils/logging"
	utilspolicy "utils/policy"
)

var PolicyEngine *BGPPolicyEngine

type PolicyActionFunc struct {
	ApplyFunc utilspolicy.Policyfunc
	UndoFunc  utilspolicy.UndoActionfunc
}

type PolicyExtensions struct {
	HitCounter    int
	RouteList     []string
	RouteInfoList []*bgprib.Route
}

type BGPPolicyEngine struct {
	logger          *logging.Writer
	PolicyEngine    *utilspolicy.PolicyEngineDB
	ConditionCfgCh  chan utilspolicy.PolicyConditionConfig
	ActionCfgCh     chan utilspolicy.PolicyActionConfig
	StmtCfgCh       chan utilspolicy.PolicyStmtConfig
	DefinitionCfgCh chan utilspolicy.PolicyDefinitionConfig
	ConditionDelCh  chan string
	ActionDelCh     chan string
	StmtDelCh       chan string
	DefinitionDelCh chan string
}

func NewBGPPolicyEngine(logger *logging.Writer) *BGPPolicyEngine {
	if PolicyEngine == nil {
		bgpPE := &BGPPolicyEngine{}
		bgpPE.logger = logger
		bgpPE.PolicyEngine = utilspolicy.NewPolicyEngineDB(logger)
		bgpPE.ConditionCfgCh = make(chan utilspolicy.PolicyConditionConfig)
		bgpPE.ActionCfgCh = make(chan utilspolicy.PolicyActionConfig)
		bgpPE.StmtCfgCh = make(chan utilspolicy.PolicyStmtConfig)
		bgpPE.DefinitionCfgCh = make(chan utilspolicy.PolicyDefinitionConfig)
		bgpPE.ConditionDelCh = make(chan string)
		bgpPE.ActionDelCh = make(chan string)
		bgpPE.StmtDelCh = make(chan string)
		bgpPE.DefinitionDelCh = make(chan string)
		PolicyEngine = bgpPE
	}

	PolicyEngine.SetGetPolicyEntityMapIndexFunc(getPolicyEnityKey)
	return PolicyEngine
}

func (eng *BGPPolicyEngine) StartPolicyEngine() {
	for {
		select {
		case condCfg := <-eng.ConditionCfgCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - create condition", condCfg.Name))
			eng.PolicyEngine.CreatePolicyDstIpMatchPrefixSetCondition(condCfg)

		case actionCfg := <-eng.ActionCfgCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - create action", actionCfg.Name))
			eng.PolicyEngine.CreatePolicyAggregateAction(actionCfg)

		case stmtCfg := <-eng.StmtCfgCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - create statement", stmtCfg.Name))
			eng.PolicyEngine.CreatePolicyStatement(stmtCfg)

		case defCfg := <-eng.DefinitionCfgCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - create policy", defCfg.Name))
			defCfg.Extensions = PolicyExtensions{}
			eng.PolicyEngine.CreatePolicyDefinition(defCfg)

		case conditionName := <-eng.ConditionDelCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - delete condition", conditionName))
			conditionCfg := utilspolicy.PolicyConditionConfig{Name: conditionName}
			eng.PolicyEngine.DeletePolicyCondition(conditionCfg)

		case actionName := <-eng.ActionDelCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - delete action", actionName))
			actionCfg := utilspolicy.PolicyActionConfig{Name: actionName}
			eng.PolicyEngine.DeletePolicyAction(actionCfg)

		case stmtName := <-eng.StmtDelCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - delete statment", stmtName))
			stmtCfg := utilspolicy.PolicyStmtConfig{Name: stmtName}
			eng.PolicyEngine.DeletePolicyStatement(stmtCfg)

		case policyName := <-eng.DefinitionDelCh:
			eng.logger.Info(fmt.Sprintln("BGPPolicyEngine - delete statment", policyName))
			policyCfg := utilspolicy.PolicyDefinitionConfig{Name: policyName}
			eng.PolicyEngine.DeletePolicyDefinition(policyCfg)
		}
	}
}

func (eng *BGPPolicyEngine) SetTraverseFuncs(traverseApplyFunc utilspolicy.EntityTraverseAndApplyPolicyfunc,
	traverseReverseFunc utilspolicy.EntityTraverseAndReversePolicyfunc) {
	eng.logger.Info(fmt.Sprintln("BGPPolicyEngine:SetTraverseFunc traverse apply func %v", traverseApplyFunc))
	if traverseApplyFunc != nil {
		eng.PolicyEngine.SetTraverseAndApplyPolicyFunc(traverseApplyFunc)
	}
	eng.logger.Info(fmt.Sprintln("BGPPolicyEngine:SetTraverseFunc traverse reverse func %v", traverseReverseFunc))
	if traverseReverseFunc != nil {
		eng.PolicyEngine.SetTraverseAndReversePolicyFunc(traverseReverseFunc)
	}
}

func (eng *BGPPolicyEngine) SetActionFuncs(actionFuncMap map[int]PolicyActionFunc) {
	eng.logger.Info(fmt.Sprintf("BGPPolicyEngine:SetApplyActionFunc actionFuncMap %v", actionFuncMap))
	for actionType, actionFuncs := range actionFuncMap {
		eng.logger.Info(fmt.Sprintln("BGPPolicyEngine:SetApplyActionFunc set apply/undo callbacks for action", actionType))
		if actionFuncs.ApplyFunc != nil {
			eng.PolicyEngine.SetActionFunc(actionType, actionFuncs.ApplyFunc)
		}
		if actionFuncs.UndoFunc != nil {
			eng.PolicyEngine.SetUndoActionFunc(actionType, actionFuncs.UndoFunc)
		}
	}
}

func (eng *BGPPolicyEngine) SetEntityUpdateFunc(entityUpdateFunc utilspolicy.EntityUpdatefunc) {
	eng.logger.Info(fmt.Sprintln("BGPPolicyEngine:SetEntityUpdateFunc func %v", entityUpdateFunc))
	if entityUpdateFunc != nil {
		eng.PolicyEngine.SetEntityUpdateFunc(entityUpdateFunc)
	}
}

func (eng *BGPPolicyEngine) SetIsEntityPresentFunc(entityPresentFunc utilspolicy.PolicyCheckfunc) {
	eng.logger.Info(fmt.Sprintln("BGPPolicyEngine:SetIsEntityPresentFunc func %v", entityPresentFunc))
	if entityPresentFunc != nil {
		eng.PolicyEngine.SetIsEntityPresentFunc(entityPresentFunc)
	}
}

func (eng *BGPPolicyEngine) SetGetPolicyEntityMapIndexFunc(policyEntityKeyFunc utilspolicy.GetPolicyEnityMapIndexFunc) {
	eng.logger.Info(fmt.Sprintln("BGPPolicyEngine:SetGetPolicyEntityMapIndexFunc func %v", policyEntityKeyFunc))
	if policyEntityKeyFunc != nil {
		eng.PolicyEngine.SetGetPolicyEntityMapIndexFunc(policyEntityKeyFunc)
	}
}
