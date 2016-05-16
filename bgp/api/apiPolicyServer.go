package api

import (
	"models"
	utilspolicy "utils/policy"
	"sync"
)

type PolicyApiLayer struct {
	conditionAddCh chan utilspolicy.PolicyConditionConfig
	conditionDelCh chan string
}

var bgppolicyapi *PolicyApiLayer = nil
var policyOnce sync.Once

/*  Singleton instance should be accesible only within api
 */
func getPolicyInstance() *PolicyApiLayer {
	policyOnce.Do(func() {
		bgppolicyapi = &PolicyApiLayer{}
	})
	return bgppolicyapi
}

/*  Initialize bgp api layer with the channels that will be used for communicating
 *  with the policy engine server
 */
func InitPolicy(conditionAddCh chan utilspolicy.PolicyConditionConfig,
                conditionDelCh chan string) {
	bgppolicyapi = getPolicyInstance()
	bgppolicyapi.conditionAddCh = conditionAddCh
	bgppolicyapi.conditionDelCh = conditionDelCh
}
func convertModelsToPolicyConditionConfig(
	cfg *models.PolicyCondition) utilspolicy.PolicyConditionConfig {
	condition := utilspolicy.PolicyConditionConfig{}
	if cfg == nil {
		return  condition
	}
	destIPMatch := utilspolicy.PolicyDstIpMatchPrefixSetCondition{
		Prefix: utilspolicy.PolicyPrefix{
			IpPrefix:        cfg.IpPrefix,
			MasklengthRange: cfg.MaskLengthRange,
		},
	}
	return utilspolicy.PolicyConditionConfig{
		Name:                          cfg.Name,
		ConditionType:                 cfg.ConditionType,
		MatchDstIpPrefixConditionInfo: destIPMatch,
	}
}

func SendPolicyConditionNotification(add *models.PolicyCondition, remove *models.PolicyCondition, update *models.PolicyCondition) {
    if add != nil {  //conditionAdd
		bgppolicyapi.conditionAddCh <- convertModelsToPolicyConditionConfig(add)
	} else if remove != nil {
		bgppolicyapi.conditionDelCh <- (convertModelsToPolicyConditionConfig(remove)).Name
	}
}
