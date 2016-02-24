// policyApis.go
package main

import (
	"ribd"
	"errors"
	"utils/policy"
	  "utils/policy/policyCommonDefs"
	"utils/patriciaDB"
	"strconv"
	"strings"
	//"net"
	"reflect"
)

type Policy struct {
    *policy.Policy 
	hitCounter         int   
	routeList         []string
	routeInfoList     []ribd.Routes
}

var PolicyDB *patriciaDB.Trie
var PolicyStmtDB *patriciaDB.Trie

var localPolicyStmtDB []policy.LocalDB
var localPolicyDB []policy.LocalDB

func (m RouteServiceHandler) CreatePolicyPrefixSet(cfg *ribd.PolicyPrefixSet ) (val bool, err error) {
	logger.Println("CreatePolicyPrefixSet")
	return val, err
}

func (m RouteServiceHandler) CreatePolicyStatement(cfg *ribd.PolicyStmtConfig) (val bool, err error) {
	logger.Println("CreatePolicyStatement")
	newPolicyStmt:=policy.PolicyStmtConfig{Name:cfg.Name, Precedence:cfg.Precedence,MatchConditions:cfg.MatchConditions}
	newPolicyStmt.Conditions = make([]string, 0)
	for i:=0;i<len(cfg.Conditions);i++ {
		newPolicyStmt.Conditions = append(newPolicyStmt.Conditions,cfg.Conditions[i])
	}
	for i:=0;i<len(cfg.Actions);i++ {
		newPolicyStmt.Actions = append(newPolicyStmt.Actions,cfg.Actions[i])
	}
	err = policy.CreatePolicyStatement(newPolicyStmt)
	return val,err
}

func (m RouteServiceHandler) 	DeletePolicyStatement(cfg *ribd.PolicyStmtConfig) (val bool, err error) {
	logger.Println("DeletePolicyStatement for name ", cfg.Name)
	stmt:=policy.PolicyStmtConfig{Name:cfg.Name}
	err = policy.DeletePolicyStatement(stmt)
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyStmtState( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyStmtStateGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkPolicyStmtState")
	PolicyStmtDB,err = policy.GetPolicyStmtDB()
	if err != nil {
		logger.Println("Failed to get policyStmtDB")
		return val,err
	}
	localPolicyStmtDB,err = policy.GetLocalPolicyStmtDB()
	if err != nil {
		logger.Println("Failed to get localpolicyStmtDB")
		return val,err
	}
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyStmtState = make ([]ribd.PolicyStmtState, rcount)
	var nextNode *ribd.PolicyStmtState
    var returnNodes []*ribd.PolicyStmtState
	var returnGetInfo ribd.PolicyStmtStateGetInfo
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
		if(localPolicyStmtDB[i+fromIndex].IsValid == false) {
			logger.Println("Invalid policy statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy statements fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyStmtDB[i+fromIndex].prefix))
		prefixNodeGet := policy.PolicyStmtDB.Get(localPolicyStmtDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(policy.PolicyStmt)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.Name
			nextNode.Conditions = prefixNode.Conditions
			nextNode.Actions = prefixNode.Actions
			toIndex = ribd.Int(prefixNode.LocalDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyStmtState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policyStmts", validCount)
	policyStmts.PolicyStmtStateList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex+1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}

func (m RouteServiceHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinitionConfig) (val bool, err error) {
	logger.Println("CreatePolicyDefinition")
	newPolicy:=policy.PolicyDefinitionConfig{Name:cfg.Name, Precedence:cfg.Precedence,MatchType:cfg.MatchType,Export:cfg.Export,Import:cfg.Import,Global:cfg.Global}
	newPolicy.PolicyDefinitionStatements = make([]PolicyDefinitionStmtPrecedence,0)
	var policyDefinitionStatement policy.PolicyDefinitionStmtPrecedence
	for i:=0;i<len(newPolicy.PolicyDefinitionStatements);i++ {
		policyDefinitionStatement.Precedence = cfg.PolicyDefinitionStatements[i].Precedence
		policyDefinitionStatement.Statement = cfg.PolicyDefinitionStatements[i].Statement
		newPolicy.PolicyDefinitionStatements = append(newPolicy.PolicyDefinitionStatements,policyDefinitionStatement)
	}
	err = policy.CreatePolicyDefinition(cfg)
	return val, err
}

func (m RouteServiceHandler) 	DeletePolicyDefinition(cfg *ribd.PolicyDefinitionConfig) (val bool, err error) {
	logger.Println("DeletePolicyDefinition for name ", cfg.Name)
	policy:=policy.PolicyDefinitionConfig{Name:cfg.Name}
	err = policy.DeletePolicyDefinition(policy)
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyDefinitionState( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStateGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkPolicyDefinitionState")
	PolicyDB,err = policy.GetPolicyDB()
	if err != nil {
		logger.Println("Failed to get policyDB")
		return val,err
	}
	localPolicyDB,err = policy.GetLocalPolicyDB()
	if err != nil {
		logger.Println("Failed to get localpolicyDB")
		return val,err
	}
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionState = make ([]ribd.PolicyDefinitionState, rcount)
	var nextNode *ribd.PolicyDefinitionState
    var returnNodes []*ribd.PolicyDefinitionState
	var returnGetInfo ribd.PolicyDefinitionStateGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
    if(LocalPolicyDB == nil) {
		logger.Println("LocalPolicyDB not initialized")
		return policyStmts, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyDB))) {
			logger.Println("All the policies fetched")
			more = false
			break
		}
		if(localPolicyDB[i+fromIndex].IsValid == false) {
			logger.Println("Invalid policy")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policies fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyDB.Get(localPolicyDB[i+fromIndex].Prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(policy.Policy)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.Name
			nextNode.HitCounter = ribd.Int(prefixNode.hitCounter)
			nextNode.IpPrefixList = make([]string,0)
			for k:=0;k<len(prefixNode.routeList);k++ {
			   nextNode.IpPrefixList = append(nextNode.IpPrefixList,prefixNode.routeList[k])
			}
			toIndex = ribd.Int(prefixNode.LocalDBSliceIdx)
			if(len(returnNodes) == 0){
				returnNodes = make([]*ribd.PolicyDefinitionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	logger.Printf("Returning %d list of policies", validCount)
	policyStmts.PolicyDefinitionStateList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex+1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}