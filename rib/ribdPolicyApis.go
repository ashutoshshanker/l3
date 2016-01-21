// policyApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
)
var PolicyDB = patriciaDB.NewTrie()
var PolicyConfigDB = patriciaDB.NewTrie()

type PolicyConditions struct {
	conditionType int
	conditionInfo interface {}
}
type PolicyActions struct {
	actionType int
	actionInfo interface {}
}
type PolicyStmt struct {				//policy engine uses this
	name               string
	conditions         []PolicyConditions
	actions            []PolicyActions
}
type PolicyStmtInfo struct {			//config
	name                   string
	//conditions
	prefixSetMatchInfo     ribd.PolicyDefinitionStatementMatchPrefixSet
	routeProtocolType      int		//ribdCommonDefs.PtypesInstallProtocolTypePtypes
    //action
	routeDisposition       string
	//setTag
	redistribute           bool
	redistributeTargetProtocol int
	localDBSliceIdx        int8       
}
var RouteProtocolTypeMapDB = make(map[string]int)
var ReverseRouteProtoTypeMapDB = make(map[int]string)
var ProtocolPolicyListDB = make(map[int][]string)//policystmt names assoociated with every protocol type
var localPolicyStmtDB []localDB

func updateProtocolPolicyTable(protoType int, name string, op int) {
	logger.Printf("updateProtocolPolicyTable for protocol %d policy name %s op %d\n", protoType, name, op)
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
	ProtocolPolicyListDB[protoType] = policyList
}

func updatePolicyDB(cfg PolicyStmtInfo) {
	logger.Println("updatePolicyDB")
	policyStmtInfo := PolicyDB.Get(patriciaDB.Prefix(cfg.name))
	var tempCondition PolicyConditions
	var tempAction PolicyActions
	if(policyStmtInfo == nil) {
	   var	newPolicyStmt PolicyStmt
	   logger.Println("Defining a new policy statement with name in the policy engine DB", cfg.name)
        newPolicyStmt.name = cfg.name
		
       //check what conditions need to be installed
	   if len(cfg.prefixSetMatchInfo.PrefixSet) != 0 {
         logger.Println("Add Prefix match condition")
		 if newPolicyStmt.conditions == nil {
			newPolicyStmt.conditions = make([] PolicyConditions, 0)
		}
		tempCondition.conditionType = ribdCommonDefs.PolicyConditionTypePrefixMatch
		tempCondition.conditionInfo = cfg.prefixSetMatchInfo
		newPolicyStmt.conditions = append(newPolicyStmt.conditions, tempCondition)
	   }	
        if(cfg.routeProtocolType != -1) {
         logger.Println("Add Protocol match condition")
		 if newPolicyStmt.conditions == nil {
			newPolicyStmt.conditions = make([] PolicyConditions, 0)
		 }
		 tempCondition.conditionType = ribdCommonDefs.PolicyConditionTypeProtocolMatch
		 tempCondition.conditionInfo = cfg.routeProtocolType	
		}
		
		//actions
		if len(cfg.routeDisposition) > 0 {
			logger.Println("Add routeDisposition action")
			if newPolicyStmt.actions == nil {
				newPolicyStmt.actions = make ([] PolicyActions, 0)
			}
			tempAction.actionType = ribdCommonDefs.PolicyActionTypeRouteDisposition
			tempAction.actionInfo = cfg.routeDisposition
		}
		
		if(cfg.redistribute == true) {
			logger.Println("Add redistribute action")
			if newPolicyStmt.actions == nil {
				newPolicyStmt.actions = make ([] PolicyActions, 0)
			}
			tempAction.actionType = ribdCommonDefs.PolicyActionTypeRouteRedistribute
			tempAction.actionInfo = cfg.redistributeTargetProtocol
		}
		if ok := PolicyConfigDB.Insert(patriciaDB.Prefix(cfg.name), newPolicyStmt); ok != true {
			logger.Println(" return value not ok")
			return
		}
	}
}

func BuildRouteProtocolTypeMapDB() {
	RouteProtocolTypeMapDB["Connected"] = ribdCommonDefs.CONNECTED
	RouteProtocolTypeMapDB["BGP"]       = ribdCommonDefs.BGP
	RouteProtocolTypeMapDB["Static"]       = ribdCommonDefs.STATIC
	
	//reverse
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.CONNECTED] = "Connected"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.BGP] = "BGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.STATIC] = "Static"
}
func (m RouteServiceHandler) CreatePolicyDefinitionSetsPrefixSet(cfg *ribd.PolicyDefinitionSetsPrefixSet ) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionSetsPrefixSet")
	return val, err
}

func (m RouteServiceHandler) CreatePolicyDefinitionStatementMatchPrefixSet(cfg *ribd.PolicyDefinitionStatementMatchPrefixSet) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStatementMatchPrefixSet")
	return val, err
}

func (m RouteServiceHandler) CreatePolicyDefinitionStatement(cfg *ribd.PolicyDefinitionStatement) (val bool, err error) {
	logger.Println("CreatePolicyDefinitionStatement")
	policyStmtInfo := PolicyConfigDB.Get(patriciaDB.Prefix(cfg.Name))
	protoType := -1
	targetProtoType := -1
	var tempMatchPrefixSetInfo ribd.PolicyDefinitionStatementMatchPrefixSet
	if(policyStmtInfo == nil) {
	   logger.Println("Defining a new policy statement with name ", cfg.Name)
	   if cfg.MatchPrefixSetInfo != nil {
	      tempMatchPrefixSetInfo = *(cfg.MatchPrefixSetInfo)
	   }	
	   retProto,found := RouteProtocolTypeMapDB[cfg.InstallProtocolEq]
	   if(found == true ) {
	      protoType = retProto
	   }
	   logger.Printf("protoType for installProtocolEq %s is %d\n", cfg.InstallProtocolEq, protoType)
	   retProto,found = RouteProtocolTypeMapDB[cfg.RedistributeTargetProtocol]
	   if(found == true ) {
	      targetProtoType = retProto
	   }
	   logger.Printf("protoType for installProtocolEq %s is %d\n", cfg.InstallProtocolEq, protoType)
	   newPolicyStmtInfo := PolicyStmtInfo{name:cfg.Name, prefixSetMatchInfo:tempMatchPrefixSetInfo, routeProtocolType:protoType, routeDisposition:cfg.RouteDisposition, redistribute:cfg.Redistribute,redistributeTargetProtocol:targetProtoType,localDBSliceIdx:int8(len(localPolicyStmtDB))}
		if ok := PolicyConfigDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyStmtInfo); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyStmtDB == nil) {
			localPolicyStmtDB = make([]localDB, 0)
		} 
	    localPolicyStmtDB = append(localPolicyStmtDB, localDBRecord)
	    updatePolicyDB(newPolicyStmtInfo)
	} else {
		logger.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
	}
	//update other tables
    if protoType != -1 {
		updateProtocolPolicyTable(protoType, cfg.Name, add)
	}
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyStmts( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStatementGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("getBulkPolicyStmts")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionStatement = make ([]ribd.PolicyDefinitionStatement, rcount)
    var tempMatchPrefixSetInfo []ribd.PolicyDefinitionStatementMatchPrefixSet = make ([]ribd.PolicyDefinitionStatementMatchPrefixSet, rcount)
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
		prefixNodeGet := PolicyConfigDB.Get(localPolicyStmtDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyStmtInfo)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
            if(prefixNode.routeProtocolType != -1) {
			   nextNode.InstallProtocolEq = ReverseRouteProtoTypeMapDB[prefixNode.routeProtocolType]
			}
			tempMatchPrefixSetInfo[validCount] = prefixNode.prefixSetMatchInfo
			nextNode.MatchPrefixSetInfo = &tempMatchPrefixSetInfo[validCount]
		    nextNode.RouteDisposition = prefixNode.routeDisposition
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
