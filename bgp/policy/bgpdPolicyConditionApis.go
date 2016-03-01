// bgpdPolicyConditionApis.go
package policy

import (
	"bgpd"
	"errors"
	"fmt"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
)

var PolicyConditionsDB = patriciaDB.NewTrie()

type MatchPrefixConditionInfo struct {
	usePrefixSet bool
	prefixSet    string
	dstIpMatch   bool
	srcIpMatch   bool
	Prefix       bgpd.BGPPolicyPrefix
}
type PolicyCondition struct {
	Name                 string
	ConditionType        int
	ConditionInfo        interface{}
	PolicyStmtList       []string
	ConditionGetBulkInfo string
	localDBSliceIdx      int
}

var localPolicyConditionsDB []localDB

func updateLocalConditionsDB(prefix patriciaDB.Prefix) {
	localDBRecord := localDB{prefix: prefix, isValid: true}
	if localPolicyConditionsDB == nil {
		localPolicyConditionsDB = make([]localDB, 0)
	}
	localPolicyConditionsDB = append(localPolicyConditionsDB, localDBRecord)

}
func CreatePolicyDstIpMatchPrefixSetCondition(inCfg *bgpd.BGPPolicyConditionConfig) (val bool, err error) {
	fmt.Println("CreatePolicyDstIpMatchPrefixSetCondition")
	cfg := inCfg.MatchDstIpPrefixConditionInfo
	var conditionInfo MatchPrefixConditionInfo
	var conditionGetBulkInfo string
	if len(cfg.PrefixSet) == 0 && cfg.Prefix == nil {
		fmt.Println("Empty prefix set")
		err = errors.New("Empty prefix set")
		return val, err
	}
	if len(cfg.PrefixSet) != 0 && cfg.Prefix != nil {
		fmt.Println("Cannot provide both prefix set and individual prefix")
		err = errors.New("Cannot provide both prefix set and individual prefix")
		return val, err
	}
	if cfg.Prefix != nil {
		conditionInfo.usePrefixSet = false
		conditionInfo.Prefix.IpPrefix = cfg.Prefix.IpPrefix
		conditionInfo.Prefix.MasklengthRange = cfg.Prefix.MasklengthRange
		conditionGetBulkInfo = "match destination Prefix " + cfg.Prefix.IpPrefix + "MasklengthRange " + cfg.Prefix.MasklengthRange
	} else if len(cfg.PrefixSet) != 0 {
		conditionInfo.usePrefixSet = true
		conditionInfo.prefixSet = cfg.PrefixSet
		conditionGetBulkInfo = "match destination Prefix " + cfg.PrefixSet
	}
	conditionInfo.dstIpMatch = true
	policyCondition := PolicyConditionsDB.Get(patriciaDB.Prefix(inCfg.Name))
	if policyCondition == nil {
		fmt.Println("Defining a new policy condition with name ", inCfg.Name)
		newPolicyCondition := PolicyCondition{Name: inCfg.Name, ConditionType: ribdCommonDefs.PolicyConditionTypeDstIpPrefixMatch, ConditionInfo: conditionInfo, localDBSliceIdx: (len(localPolicyConditionsDB))}
		newPolicyCondition.ConditionGetBulkInfo = conditionGetBulkInfo
		if ok := PolicyConditionsDB.Insert(patriciaDB.Prefix(inCfg.Name), newPolicyCondition); ok != true {
			fmt.Println(" return value not ok")
			return val, err
		}
		updateLocalConditionsDB(patriciaDB.Prefix(inCfg.Name))
	} else {
		fmt.Println("Duplicate Condition name")
		err = errors.New("Duplicate policy condition definition")
		return val, err
	}
	return val, err
}

func GetBulkBGPPolicyConditionState(fromIndex bgpd.Int, rcount bgpd.Int) (policyConditions *bgpd.BGPPolicyConditionStateGetInfo, err error) {
	fmt.Println("GetBulkPolicyConditionState")
	var i, validCount, toIndex bgpd.Int
	var tempNode []bgpd.BGPPolicyConditionState = make([]bgpd.BGPPolicyConditionState, rcount)
	var nextNode *bgpd.BGPPolicyConditionState
	var returnNodes []*bgpd.BGPPolicyConditionState
	var returnGetInfo bgpd.BGPPolicyConditionStateGetInfo
	i = 0
	policyConditions = &returnGetInfo
	more := true
	if localPolicyConditionsDB == nil {
		fmt.Println("PolicyDefinitionStmtMatchProtocolConditionGetInfo not initialized")
		return policyConditions, err
	}
	for ; ; i++ {
		fmt.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if i+fromIndex >= bgpd.Int(len(localPolicyConditionsDB)) {
			fmt.Println("All the policy conditions fetched")
			more = false
			break
		}
		if localPolicyConditionsDB[i+fromIndex].isValid == false {
			fmt.Println("Invalid policy condition statement")
			continue
		}
		if validCount == rcount {
			fmt.Println("Enough policy conditions fetched")
			break
		}
		fmt.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyConditionsDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyConditionsDB.Get(localPolicyConditionsDB[i+fromIndex].prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(PolicyCondition)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.Name
			nextNode.ConditionInfo = prefixNode.ConditionGetBulkInfo
			if prefixNode.PolicyStmtList != nil {
				nextNode.PolicyStmtList = make([]string, 0)
			}
			for idx := 0; idx < len(prefixNode.PolicyStmtList); idx++ {
				nextNode.PolicyStmtList = append(nextNode.PolicyStmtList, prefixNode.PolicyStmtList[idx])
			}
			toIndex = bgpd.Int(prefixNode.localDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*bgpd.BGPPolicyConditionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	fmt.Printf("Returning %d list of policyConditions", validCount)
	policyConditions.PolicyConditionStateList = returnNodes
	policyConditions.StartIdx = fromIndex
	policyConditions.EndIdx = toIndex + 1
	policyConditions.More = more
	policyConditions.Count = validCount
	return policyConditions, err
}
