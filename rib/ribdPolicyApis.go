// policyApis.go
package main

import (
	"ribd"
	"utils/patriciaDB"
)
var PolicyDB = patriciaDB.NewTrie()
type PolicyStmtInfo struct {
	name                   string
	//conditions
	prefixSetMatchInfo     ribd.PolicyDefinitionStatementMatchPrefixSet
	routeProtocolType      int		//ribdCommonDefs.PtypesInstallProtocolTypePtypes
    //action
	routeDisposition       string
	//setTag
	//redistribute
	localDBSliceIdx        int8       
}
var localPolicyStmtDB []localDB
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
	policyStmtInfo := PolicyDB.Get(patriciaDB.Prefix(cfg.Name))
	var tempMatchPrefixSetInfo ribd.PolicyDefinitionStatementMatchPrefixSet
	if(policyStmtInfo == nil) {
	   logger.Println("Defining a new policy statement with name ", cfg.Name)
	   if cfg.MatchPrefixSetInfo != nil {
	      tempMatchPrefixSetInfo = *(cfg.MatchPrefixSetInfo)
	   }	
	   newPolicyStmtInfo :=PolicyStmtInfo{name:cfg.Name, prefixSetMatchInfo:tempMatchPrefixSetInfo, routeProtocolType:int(cfg.InstallProtocolEq), routeDisposition:cfg.RouteDisposition, localDBSliceIdx:int8(len(localPolicyStmtDB))}
		if ok := PolicyDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyStmtInfo); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyStmtDB == nil) {
			localPolicyStmtDB = make([]localDB, 0)
		} 
	    localPolicyStmtDB = append(localPolicyStmtDB, localDBRecord)
	} else {
		logger.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
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
		prefixNodeGet := PolicyDB.Get(localPolicyStmtDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyStmtInfo)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.InstallProtocolEq = ribd.Int(prefixNode.routeProtocolType)
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
