// policyApis.go
package main

import (
	"ribd"
)

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
	return val, err
}

func (m RouteServiceHandler) CreatePolicyDefinition(cfg *ribd.PolicyDefinition) (val bool, err error) {
	logger.Println("CreatePolicyDefinition")
	return val, err
}
