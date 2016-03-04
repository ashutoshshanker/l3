// bgpdPolicyActionApis.go
package policy

import (
	"bgpd"
	"errors"
	"fmt"
	"utils/policy/policyCommonDefs"
	"utils/patriciaDB"
)

var PolicyActionsDB = patriciaDB.NewTrie()

type PolicyAggregateActionInfo struct {
	GenerateASSet   bool
	SendSummaryOnly bool
}

type PolicyAction struct {
	name              string
	actionType        int
	actionInfo        interface{}
	policyStmtList    []string
	actionGetBulkInfo string
	localDBSliceIdx   int
}

var localPolicyActionsDB []localDB

func updateLocalActionsDB(prefix patriciaDB.Prefix) {
	localDBRecord := localDB{prefix: prefix, isValid: true}
	if localPolicyActionsDB == nil {
		localPolicyActionsDB = make([]localDB, 0)
	}
	localPolicyActionsDB = append(localPolicyActionsDB, localDBRecord)
}

func CreatePolicyAggregateAction(cfg *bgpd.BGPPolicyActionConfig) (val bool, err error) {
	fmt.Println("CreatePolicyAggregateAction")

	policyAction := PolicyActionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if policyAction == nil {
		fmt.Println("Defining a new policy action with name ", cfg.Name)
		aggregateActionInfo := PolicyAggregateActionInfo{GenerateASSet: cfg.AggregateActionInfo.GenerateASSet, SendSummaryOnly: cfg.AggregateActionInfo.SendSummaryOnly}
		newPolicyAction := PolicyAction{name: cfg.Name, actionType: policyCommonDefs.PolicyActionTypeAggregate, actionInfo: aggregateActionInfo, localDBSliceIdx: (len(localPolicyActionsDB))}
		var generateASSet, sendSummaryOnly string
		if cfg.AggregateActionInfo.GenerateASSet == true {
			generateASSet = "true"
		} else {
			generateASSet = "false"
		}
		if cfg.AggregateActionInfo.SendSummaryOnly == true {
			sendSummaryOnly = "true"
		} else {
			sendSummaryOnly = "false"
		}
		newPolicyAction.actionGetBulkInfo = "Aggregate action set GenerateASSet to " + generateASSet + " set SendSummaryOnly to " + sendSummaryOnly
		if ok := PolicyActionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyAction); ok != true {
			fmt.Println(" return value not ok")
			return val, err
		}
		updateLocalActionsDB(patriciaDB.Prefix(cfg.Name))
	} else {
		fmt.Println("Duplicate action name")
		err = errors.New("Duplicate policy action definition")
		return val, err
	}
	return val, err
}

func GetBulkBGPPolicyActionState(fromIndex bgpd.Int, rcount bgpd.Int) (policyActions *bgpd.BGPPolicyActionStateGetInfo, err error) { //(routes []*bgpd.Routes, err error) {
	fmt.Println("GetBulkPolicyActionState")
	var i, validCount, toIndex bgpd.Int
	var tempNode []bgpd.BGPPolicyActionState = make([]bgpd.BGPPolicyActionState, rcount)
	var nextNode *bgpd.BGPPolicyActionState
	var returnNodes []*bgpd.BGPPolicyActionState
	var returnGetInfo bgpd.BGPPolicyActionStateGetInfo
	policyActions = &returnGetInfo
	more := true
	if localPolicyActionsDB == nil {
		fmt.Println("PolicyDefinitionStmtMatchProtocolActionGetInfo not initialized")
		return policyActions, err
	}
	for ; ; i++ {
		fmt.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if i+fromIndex >= bgpd.Int(len(localPolicyActionsDB)) {
			fmt.Println("All the policy Actions fetched")
			more = false
			break
		}
		if localPolicyActionsDB[i+fromIndex].isValid == false {
			fmt.Println("Invalid policy Action statement")
			continue
		}
		if validCount == rcount {
			fmt.Println("Enough policy Actions fetched")
			break
		}
		fmt.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyActionsDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyActionsDB.Get(localPolicyActionsDB[i+fromIndex].prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(PolicyAction)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.name
			nextNode.ActionInfo = prefixNode.actionGetBulkInfo
			if prefixNode.policyStmtList != nil {
				nextNode.PolicyStmtList = make([]string, 0)
			}
			for idx := 0; idx < len(prefixNode.policyStmtList); idx++ {
				nextNode.PolicyStmtList = append(nextNode.PolicyStmtList, prefixNode.policyStmtList[idx])
			}
			toIndex = bgpd.Int(prefixNode.localDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*bgpd.BGPPolicyActionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	fmt.Printf("Returning %d list of policyActions", validCount)
	policyActions.PolicyActionStateList = returnNodes
	policyActions.StartIdx = fromIndex
	policyActions.EndIdx = toIndex + 1
	policyActions.More = more
	policyActions.Count = validCount
	return policyActions, err
}
