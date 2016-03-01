// policy.go
package policy

import (
	"bgpd"
	"log/syslog"
)

type BGPPolicyEngine struct {
	logger          *syslog.Writer
	ConditionCfgCh  chan *bgpd.BGPPolicyConditionConfig
	ActionCfgCh     chan *bgpd.BGPPolicyActionConfig
	StmtCfgCh       chan *bgpd.BGPPolicyStmtConfig
	DefinitionCfgCh chan *bgpd.BGPPolicyDefinitionConfig
}

func NewBGPPolicyEngine(logger *syslog.Writer) *BGPPolicyEngine {
	bgpPolicy := &BGPPolicyEngine{}
	bgpPolicy.logger = logger
	bgpPolicy.ConditionCfgCh = make(chan *bgpd.BGPPolicyConditionConfig)
	bgpPolicy.ActionCfgCh = make(chan *bgpd.BGPPolicyActionConfig)
	bgpPolicy.StmtCfgCh = make(chan *bgpd.BGPPolicyStmtConfig)
	bgpPolicy.DefinitionCfgCh = make(chan *bgpd.BGPPolicyDefinitionConfig)
	return bgpPolicy
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
