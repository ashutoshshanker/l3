// policy.go
package policy

import (
	"bgpd"
	"fmt"
	"log/syslog"
)

var PolicyEngine *BGPPolicyEngine

type BGPPolicyEngine struct {
	logger          *syslog.Writer
	ConditionCfgCh  chan *bgpd.BGPPolicyConditionConfig
	ActionCfgCh     chan *bgpd.BGPPolicyActionConfig
	StmtCfgCh       chan *bgpd.BGPPolicyStmtConfig
	DefinitionCfgCh chan *bgpd.BGPPolicyDefinitionConfig
	TraverseFunc    TraverseFunc
	ActionFuncMap   map[int][2]ApplyActionFunc
}

func NewBGPPolicyEngine(logger *syslog.Writer) *BGPPolicyEngine {
	if PolicyEngine == nil {
		bgpPolicy := &BGPPolicyEngine{}
		bgpPolicy.logger = logger
		bgpPolicy.ConditionCfgCh = make(chan *bgpd.BGPPolicyConditionConfig)
		bgpPolicy.ActionCfgCh = make(chan *bgpd.BGPPolicyActionConfig)
		bgpPolicy.StmtCfgCh = make(chan *bgpd.BGPPolicyStmtConfig)
		bgpPolicy.DefinitionCfgCh = make(chan *bgpd.BGPPolicyDefinitionConfig)
		bgpPolicy.ActionFuncMap = make(map[int][2]ApplyActionFunc)
		PolicyEngine = bgpPolicy
	}
	return PolicyEngine
}

func (eng *BGPPolicyEngine) StartPolicyEngine() {
	for {
		select {
		case condCfg := <-eng.ConditionCfgCh:
			CreatePolicyDstIpMatchPrefixSetCondition(condCfg)

		case actionCfg := <-eng.ActionCfgCh:
			CreatePolicyAggregateAction(actionCfg)

		case stmtCfg := <-eng.StmtCfgCh:
			CreateBGPPolicyStmtConfig(stmtCfg)

		case defCfg := <-eng.DefinitionCfgCh:
			CreateBGPPolicyDefinitionConfig(defCfg)
		}
	}
}

func (eng *BGPPolicyEngine) SetTraverseFunc(traverse TraverseFunc) {
	eng.TraverseFunc = traverse
}

func (eng *BGPPolicyEngine) SetApplyActionFunc(actionFuncMap map[int][2]ApplyActionFunc) {
	fmt.Sprintf("BGPPolicyEngine: SetApplyActionFunc actionFuncMap rx = %v", actionFuncMap)
	eng.ActionFuncMap = actionFuncMap
	fmt.Sprintf("BGPPolicyEngine: SetApplyActionFunc ActionFuncMap set = %v", eng.ActionFuncMap)
}
