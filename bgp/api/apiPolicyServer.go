//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

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
