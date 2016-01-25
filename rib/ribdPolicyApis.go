// policyApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
)
var PolicyDB = patriciaDB.NewTrie()

type PolicyStmt struct {				//policy engine uses this
	name               string
	conditions         []string
	actions            []string
	localDBSliceIdx        int8       
}
var ProtocolPolicyListDB = make(map[int][]string)//policystmt names assoociated with every protocol type
var localPolicyStmtDB []localDB

func updateProtocolPolicyTable(protoType int, name string, op int) {
	logger.Printf("updateProtocolPolicyTable for protocol %d policy name %s op %d\n", protoType, name, op)
    var i int
    policyList := ProtocolPolicyListDB[protoType]
	if(policyList == nil) {
		if (op == del) {
			logger.Println("Cannot find the policy map for this protocol, so cannot delete")
			return
		}
		policyList = make([]string, 0)
	}
    if op == add {
	   policyList = append(policyList, name)
	}
	if op == del {
		for i =0; i< len(policyList);i++ {
			if policyList[i] == name {
				logger.Println("Found the policy in the protocol policy table, deleting it")
				break
			}
		}
		policyList = append(policyList[:i], policyList[i+1:]...)
	}
	ProtocolPolicyListDB[protoType] = policyList
}


func (m RouteServiceHandler) CreatePolicyDefinitionSetsPrefixSet(cfg *ribd.PolicyDefinitionSetsPrefixSet ) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionSetsPrefixSet")
	return val, err
}

func updateConditions(policyStmt PolicyStmt, conditionName string, op int) {
	logger.Println("updateConditions for condition ", conditionName)
	conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(conditionName))
	if(conditionItem != nil) {
		condition := conditionItem.(PolicyCondition)
		switch condition.conditionType {
			case ribdCommonDefs.PolicyConditionTypeProtocolMatch:
			   logger.Println("PolicyConditionTypeProtocolMatch")
			   updateProtocolPolicyTable(condition.conditionInfo.(int), policyStmt.name, op)
			   break
			case ribdCommonDefs.PolicyConditionTypePrefixMatch:
			   logger.Println("PolicyConditionTypePrefixMatch")
			   break
		}
	}
}

func (m RouteServiceHandler) CreatePolicyDefinitionStatement(cfg *ribd.PolicyDefinitionStatement) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStatement")

	policyStmt := PolicyDB.Get(patriciaDB.Prefix(cfg.Name))
	var i int
	if(policyStmt == nil) {
	   logger.Println("Defining a new policy statement with name ", cfg.Name)
	   var newPolicyStmt PolicyStmt
	   newPolicyStmt.name = cfg.Name
	   if len(cfg.Conditions) > 0 {
	      logger.Println("Policy Statement has %d ", len(cfg.Conditions)," number of conditions")	
		  newPolicyStmt.conditions = make([] string, 0)
		  for i=0;i<len(cfg.Conditions);i++ {
			newPolicyStmt.conditions = append(newPolicyStmt.conditions, cfg.Conditions[i])
			updateConditions(newPolicyStmt, cfg.Conditions[i], add)
		}
	   }
	   if len(cfg.Actions) > 0 {
	      logger.Println("Policy Statement has %d ", len(cfg.Actions)," number of actions")	
		  newPolicyStmt.actions = make([] string, 0)
		  for i=0;i<len(cfg.Actions);i++ {
			newPolicyStmt.actions = append(newPolicyStmt.actions,cfg.Actions[i])
		}
	   }
		if ok := PolicyDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyStmt); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyStmtDB == nil) {
			localPolicyStmtDB = make([]localDB, 0)
		} 
	    localPolicyStmtDB = append(localPolicyStmtDB, localDBRecord)
	    PolicyEngineTraverseAndApply(newPolicyStmt)
	} else {
		logger.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
	}
	return val, err
}

func (m RouteServiceHandler) 	DeletePolicyDefinitionStatement(cfg *ribd.PolicyDefinitionStatement) (val bool, err error) {
	logger.Println("DeletePolicyDefinitionStatement for name ", cfg.Name)
	ok := PolicyDB.Match(patriciaDB.Prefix(cfg.Name))
	if !ok {
		err = errors.New("No policy statement with this name found")
		return val, err
	}
	policyStmtInfoGet := PolicyDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyStmtInfoGet != nil) {
       //invalidate localPolicyStmt 
	   policyStmtInfo := policyStmtInfoGet.(PolicyStmt)
	   if policyStmtInfo.localDBSliceIdx < int8(len(localPolicyStmtDB)) {
          logger.Println("local DB slice index for this policy stmt is ", policyStmtInfo.localDBSliceIdx)
		  localPolicyStmtDB[policyStmtInfo.localDBSliceIdx].isValid = false		
	   }
	   logger.Println("Deleting policy statement with name ", cfg.Name)
		if ok := PolicyDB.Delete(patriciaDB.Prefix(cfg.Name)); ok != true {
			logger.Println(" return value not ok for delete PolicyDB")
			return val, err
		}
	   //update other tables
	   if len(policyStmtInfo.conditions) > 0 {
	      for i:=0;i<len(policyStmtInfo.conditions);i++ {
			updateConditions(policyStmtInfo, policyStmtInfo.conditions[i],del)
		}	
	   }
	} 
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyStmts( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStatementGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("getBulkPolicyStmts")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionStatement = make ([]ribd.PolicyDefinitionStatement, rcount)
	var nextNode *ribd.PolicyDefinitionStatement
    var returnNodes []*ribd.PolicyDefinitionStatement
	var returnGetInfo ribd.PolicyDefinitionStatementGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
    if(localPolicyStmtDB == nil) {
		logger.Println("destNetSlice not initialized")
		return policyStmts, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyStmtDB))) {
			logger.Println("All the policy statements fetched")
			more = false
			break
		}
		if(localPolicyStmtDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy statements fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyStmtDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyDB.Get(localPolicyStmtDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyStmt)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.Conditions = prefixNode.conditions
			nextNode.Actions = prefixNode.actions
			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyDefinitionStatement, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policyStmts", validCount)
	policyStmts.PolicyDefinitionStatementList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex+1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}

func (m RouteServiceHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Println("CreatePolicyDefinition")
	return val, err
}
