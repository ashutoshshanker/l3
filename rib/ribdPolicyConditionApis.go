// ribdPolicyConditionApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
)

var PolicyConditionsDB = patriciaDB.NewTrie()
type PolicyCondition struct {
	name          string
	conditionType int
	conditionInfo interface {}
	localDBSliceIdx int
}
var localPolicyConditionsDB []localDB

func (m RouteServiceHandler) CreatePolicyDefinitionStmtMatchPrefixSetCondition(cfg *ribd.PolicyDefinitionStmtMatchPrefixSetCondition) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStmtMatchPrefixSetCondition")
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
		if ok := PolicyConditionsDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyCondition); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyConditionsDB == nil) {
			localPolicyConditionsDB = make([]localDB, 0)
		} 
	    localPolicyConditionsDB = append(localPolicyConditionsDB, localDBRecord)
	} else {
		logger.Println("Duplicate Condition name")
		err = errors.New("Duplicate policy condition definition")
		return val, err
	}
	return val, err
}

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
