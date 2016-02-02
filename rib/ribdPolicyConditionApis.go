// ribdPolicyConditionApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
)

var PolicyConditionsDB = patriciaDB.NewTrie()
type MatchPrefixConditionInfo struct {
	usePrefixSet bool
	prefixSet string
	prefix ribd.PolicyDefinitionSetsPrefix
}
type PolicyCondition struct {
	name          string
	conditionType int
	conditionInfo interface {}
	policyList    [] string
	conditionGetBulkInfo string
	localDBSliceIdx int
}
var localPolicyConditionsDB []localDB
func updateLocalConditionsDB(prefix patriciaDB.Prefix) {
	localDBRecord := localDB{prefix:prefix, isValid:true}
	if(localPolicyConditionsDB == nil) {
		localPolicyConditionsDB = make([]localDB, 0)
	} 
	localPolicyConditionsDB = append(localPolicyConditionsDB, localDBRecord)

}
func (m RouteServiceHandler) CreatePolicyDefinitionStmtMatchPrefixSetCondition(cfg *ribd.PolicyDefinitionStmtMatchPrefixSetCondition) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStmtMatchPrefixSetCondition")
	var conditionInfo MatchPrefixConditionInfo
	var conditionGetBulkInfo string
    if len(cfg.PrefixSet) == 0 && cfg.Prefix == nil {
		logger.Println("Empty prefix set")
		err = errors.New("Empty prefix set")
		return val, err
	}
    if len(cfg.PrefixSet) != 0 && cfg.Prefix != nil {
		logger.Println("Cannot provide both prefix set and individual prefix")
		err = errors.New("Cannot provide both prefix set and individual prefix")
		return val, err
	}
    if cfg.Prefix != nil {
	   conditionInfo.usePrefixSet = false
       conditionInfo.prefix.IpPrefix = cfg.Prefix.IpPrefix
	   conditionInfo.prefix.MasklengthRange = cfg.Prefix.MasklengthRange
	   conditionGetBulkInfo = "match Prefix " + cfg.Prefix.IpPrefix + "MasklengthRange " + cfg.Prefix.MasklengthRange
	} else if len(cfg.PrefixSet) != 0 {
		conditionInfo.usePrefixSet = true
		conditionInfo.prefixSet = cfg.PrefixSet
	    conditionGetBulkInfo = "match Prefix " + cfg.PrefixSet
	}
	policyCondition := PolicyConditionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyCondition == nil) {
	   logger.Println("Defining a new policy condition with name ", cfg.Name)
	   newPolicyCondition := PolicyCondition{name:cfg.Name,conditionType:ribdCommonDefs.PolicyConditionTypePrefixMatch,conditionInfo:conditionInfo ,localDBSliceIdx:(len(localPolicyConditionsDB))}
       newPolicyCondition.conditionGetBulkInfo = conditionGetBulkInfo 
	   if ok := PolicyConditionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyCondition); ok != true {
	   logger.Println(" return value not ok")
	   return val, err
	}
	updateLocalConditionsDB(patriciaDB.Prefix(cfg.Name))
    } else {
		logger.Println("Duplicate Condition name")
		err = errors.New("Duplicate policy condition definition")
		return val, err
	}	
	return val, err
}

func (m RouteServiceHandler) CreatePolicyDefinitionStmtMatchProtocolCondition(cfg *ribd.PolicyDefinitionStmtMatchProtocolCondition) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStmtMatchProtocolCondition")
	protoType := -1

	policyCondition := PolicyConditionsDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyCondition == nil) {
	   logger.Println("Defining a new policy condition with name ", cfg.Name)
	   retProto,found := RouteProtocolTypeMapDB[cfg.InstallProtocolEq]
	   if(found == false ) {
          logger.Println("Invalid protocol type %s ", cfg.InstallProtocolEq)
		  return val,err
	   }
	   protoType = retProto
	   logger.Printf("protoType for installProtocolEq %s is %d\n", cfg.InstallProtocolEq, protoType)
	   newPolicyCondition := PolicyCondition{name:cfg.Name,conditionType:ribdCommonDefs.PolicyConditionTypeProtocolMatch,conditionInfo:protoType ,localDBSliceIdx:(len(localPolicyConditionsDB))}
       newPolicyCondition.conditionGetBulkInfo = "match Protocol " + cfg.InstallProtocolEq
		if ok := PolicyConditionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyCondition); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
	    updateLocalConditionsDB(patriciaDB.Prefix(cfg.Name))
	} else {
		logger.Println("Duplicate Condition name")
		err = errors.New("Duplicate policy condition definition")
		return val, err
	}
	return val, err
}
/*
func (m RouteServiceHandler) GetBulkPolicyDefinitionStmtMatchProtocolConditions( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStmtMatchProtocolConditionsGetInfo, err error){
	logger.Println("getBulkPolicyDefinitionStmtMatchProtocolConditions")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionStmtMatchProtocolCondition = make ([]ribd.PolicyDefinitionStmtMatchProtocolCondition, rcount)
	var nextNode *ribd.PolicyDefinitionStmtMatchProtocolCondition
    var returnNodes []*ribd.PolicyDefinitionStmtMatchProtocolCondition
	var returnGetInfo ribd.PolicyDefinitionStmtMatchProtocolConditionsGetInfo
	i = 0
	policyConditions := &returnGetInfo
	more := true
    if(localPolicyConditionsDB == nil) {
		logger.Println("PolicyDefinitionStmtMatchProtocolConditionGetInfo not initialized")
		return policyConditions, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyConditionsDB))) {
			logger.Println("All the policy conditions fetched")
			more = false
			break
		}
		if(localPolicyConditionsDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy condition statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy conditions fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyStmtDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyConditionsDB.Get(localPolicyConditionsDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyCondition)
			if prefixNode.conditionType != ribdCommonDefs.PolicyConditionTypeProtocolMatch {
				continue
			}
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.InstallProtocolEq = ReverseRouteProtoTypeMapDB[prefixNode.conditionInfo.(int)]
			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyDefinitionStmtMatchProtocolCondition, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policyConditions", validCount)
	policyConditions.PolicyDefinitionStmtMatchProtocolConditionList = returnNodes
	policyConditions.StartIdx = fromIndex
	policyConditions.EndIdx = toIndex+1
	policyConditions.More = more
	policyConditions.Count = validCount
	return policyConditions, err
}
*/
func (m RouteServiceHandler) GetBulkPolicyDefinitionConditionState( fromIndex ribd.Int, rcount ribd.Int) (policyConditions *ribd.PolicyDefinitionConditionStateGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkPolicyDefinitionConditionState")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionConditionState = make ([]ribd.PolicyDefinitionConditionState, rcount)
	var nextNode *ribd.PolicyDefinitionConditionState
    var returnNodes []*ribd.PolicyDefinitionConditionState
	var returnGetInfo ribd.PolicyDefinitionConditionStateGetInfo
	i = 0
	policyConditions = &returnGetInfo
	more := true
    if(localPolicyConditionsDB == nil) {
		logger.Println("PolicyDefinitionStmtMatchProtocolConditionGetInfo not initialized")
		return policyConditions, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyConditionsDB))) {
			logger.Println("All the policy conditions fetched")
			more = false
			break
		}
		if(localPolicyConditionsDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy condition statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy conditions fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyConditionsDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyConditionsDB.Get(localPolicyConditionsDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyCondition)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.ConditionInfo = prefixNode.conditionGetBulkInfo
            if prefixNode.policyList != nil {
				nextNode.PolicyList = make([]string,0)
			}
			for idx := 0;idx < len(prefixNode.policyList);idx++ {
				nextNode.PolicyList = append(nextNode.PolicyList, prefixNode.policyList[idx])
			}
 			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyDefinitionConditionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policyConditions", validCount)
	policyConditions.PolicyDefinitionConditionStateList = returnNodes
	policyConditions.StartIdx = fromIndex
	policyConditions.EndIdx = toIndex+1
	policyConditions.More = more
	policyConditions.Count = validCount
	return policyConditions, err
}