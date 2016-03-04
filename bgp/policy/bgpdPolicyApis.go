// policyApis.go
package policy

import (
	"bgpd"
	"errors"
	"fmt"
	"utils/policy/policyCommonDefs"
	"reflect"
	"strconv"
	"strings"
	"utils/patriciaDB"
)

type PolicyStmt struct { //policy engine uses this
	name            string
	precedence      bgpd.Int
	matchConditions string
	conditions      []string
	actions         []string
	localDBSliceIdx int8
}

type Policy struct {
	name                    string
	precedence              bgpd.Int
	matchType               string
	policyStmtPrecedenceMap map[int]string
	hitCounter              int
	routeList               []string
	routeInfoList           []bgpd.BGPRoute
	localDBSliceIdx         int8
	importPolicy            bool
	exportPolicy            bool
	globalPolicy            bool
}

var PolicyDB = patriciaDB.NewTrie()
var PolicyStmtDB = patriciaDB.NewTrie()
var PolicyStmtPolicyMapDB = make(map[string][]string) //policies using this statement
var PrefixPolicyListDB = patriciaDB.NewTrie()
var ProtocolPolicyListDB = make(map[string][]string) //policystmt names assoociated with every protocol type
var ImportPolicyPrecedenceMap = make(map[int]string)
var ExportPolicyPrecedenceMap = make(map[int]string)
var localPolicyStmtDB []localDB
var localPolicyDB []localDB

type PrefixPolicyListInfo struct {
	ipPrefix   patriciaDB.Prefix
	policyName string
	lowRange   int
	highRange  int
}

func addPolicyRouteMap(route *bgpd.BGPRoute, policy Policy) {
	fmt.Println("addPolicyRouteMap")
	policy.hitCounter++
	//ipPrefix, err := getNetowrkPrefixFromStrings(route.Network, route.Mask)
	var newRoute string
	found := false
	newRoute = route.Network + "/" + strconv.Itoa(int(route.CIDRLen))
	ipPrefix, err := getNetworkPrefixFromCIDR(newRoute)
	if err != nil {
		fmt.Println("Invalid ip prefix")
		return
	}
	/*
		maskIp, err := getIP(route.Mask)
		if err != nil {
			return
		}
		prefixLen, err := getPrefixLen(maskIp)
		if err != nil {
			return
		}
	*/
	//	newRoute := string(ipPrefix[:])
	fmt.Println("Adding ip prefix %s %v ", newRoute, ipPrefix)
	policyInfo := PolicyDB.Get(patriciaDB.Prefix(policy.name))
	if policyInfo == nil {
		fmt.Println("Unexpected:policyInfo nil for policy ", policy.name)
		return
	}
	tempPolicy := policyInfo.(Policy)
	if tempPolicy.routeList == nil {
		fmt.Println("routeList nil")
		tempPolicy.routeList = make([]string, 0)
	}
	fmt.Println("routelist len= ", len(tempPolicy.routeList), " prefix list so far")
	for i := 0; i < len(tempPolicy.routeList); i++ {
		fmt.Println(tempPolicy.routeList[i])
		if tempPolicy.routeList[i] == newRoute {
			fmt.Println(newRoute, " already is a part of ", policy.name, "'s routelist")
			found = true
		}
	}
	if !found {
		tempPolicy.routeList = append(tempPolicy.routeList, newRoute)
	}
	found = false
	fmt.Println("routeInfoList details")
	for i := 0; i < len(tempPolicy.routeInfoList); i++ {
		fmt.Println("IP: ", tempPolicy.routeInfoList[i].Network, "/", tempPolicy.routeInfoList[i].CIDRLen, " nextHop: ", tempPolicy.routeInfoList[i].NextHop)
		if tempPolicy.routeInfoList[i].Network == route.Network && tempPolicy.routeInfoList[i].CIDRLen == route.CIDRLen && tempPolicy.routeInfoList[i].NextHop == route.NextHop {
			fmt.Println("route already is a part of ", policy.name, "'s routeInfolist")
			found = true
		}
	}
	if tempPolicy.routeInfoList == nil {
		tempPolicy.routeInfoList = make([]bgpd.BGPRoute, 0)
	}
	if found == false {
		tempPolicy.routeInfoList = append(tempPolicy.routeInfoList, *route)
	}
	PolicyDB.Set(patriciaDB.Prefix(policy.name), tempPolicy)
}
func deletePolicyRouteMap(route *bgpd.BGPRoute, policy Policy) {
	fmt.Println("deletePolicyRouteMap")
}
func updatePolicyRouteMap(route *bgpd.BGPRoute, policy Policy, op int) {
	fmt.Println("updatePolicyRouteMap")
	if op == add {
		addPolicyRouteMap(route, policy)
	} else if op == del {
		deletePolicyRouteMap(route, policy)
	}

}
func validMatchConditions(matchConditionStr string) (valid bool) {
	fmt.Println("validMatchConditions for string ", matchConditionStr)
	if matchConditionStr == "any" || matchConditionStr == "all " {
		fmt.Println("valid")
		valid = true
	}
	return valid
}
func updateProtocolPolicyTable(protoType string, name string, op int) {
	fmt.Printf("updateProtocolPolicyTable for protocol %d policy name %s op %d\n", protoType, name, op)
	var i int
	policyList := ProtocolPolicyListDB[protoType]
	if policyList == nil {
		if op == del {
			fmt.Println("Cannot find the policy map for this protocol, so cannot delete")
			return
		}
		policyList = make([]string, 0)
	}
	if op == add {
		policyList = append(policyList, name)
	}
	found := false
	if op == del {
		for i = 0; i < len(policyList); i++ {
			if policyList[i] == name {
				fmt.Println("Found the policy in the protocol policy table, deleting it")
				found = true
				break
			}
		}
		if found {
			policyList = append(policyList[:i], policyList[i+1:]...)
		}
	}
	ProtocolPolicyListDB[protoType] = policyList
}
func updatePrefixPolicyTableWithPrefix(ipAddr string, name string, op int, lowRange int, highRange int) {
	fmt.Println("updatePrefixPolicyTableWithPrefix ", ipAddr)
	var i int
	/*		ip, _, err := net.ParseCIDR(ipAddr)
	    if err != nil {
		   return
	    }*/
	ipPrefix, err := getNetworkPrefixFromCIDR(ipAddr)
	if err != nil {
		fmt.Println("ipPrefix invalid ")
		return
	}
	var policyList []PrefixPolicyListInfo
	var prefixPolicyListInfo PrefixPolicyListInfo
	policyListItem := PrefixPolicyListDB.Get(ipPrefix)
	if policyListItem != nil && reflect.TypeOf(policyListItem).Kind() != reflect.Slice {
		fmt.Println("Incorrect data type for this prefix ")
		return
	}
	if policyListItem == nil {
		if op == del {
			fmt.Println("Cannot find the policy map for this prefix, so cannot delete")
			return
		}
		policyList = make([]PrefixPolicyListInfo, 0)
	} else {
		policyListSlice := reflect.ValueOf(policyListItem)
		policyList = make([]PrefixPolicyListInfo, 0)
		for i = 0; i < policyListSlice.Len(); i++ {
			policyList = append(policyList, policyListSlice.Index(i).Interface().(PrefixPolicyListInfo))
		}
	}
	if op == add {
		prefixPolicyListInfo.ipPrefix = ipPrefix
		prefixPolicyListInfo.policyName = name
		prefixPolicyListInfo.lowRange = lowRange
		prefixPolicyListInfo.highRange = highRange
		policyList = append(policyList, prefixPolicyListInfo)
	}
	found := false
	if op == del {
		for i = 0; i < len(policyList); i++ {
			if policyList[i].policyName == name {
				fmt.Println("Found the policy in the prefix policy table, deleting it")
				break
			}
		}
		if found {
			policyList = append(policyList[:i], policyList[i+1:]...)
		}
	}
	PrefixPolicyListDB.Set(ipPrefix, policyList)
}
func updatePrefixPolicyTableWithMaskRange(ipAddr string, masklength string, name string, op int) {
	fmt.Println("updatePrefixPolicyTableWithMaskRange")
	maskList := strings.Split(masklength, "..")
	if len(maskList) != 2 {
		fmt.Println("Invalid masklength range")
		return
	}
	lowRange, err := strconv.Atoi(maskList[0])
	if err != nil {
		fmt.Println("maskList[0] not valid")
		return
	}
	highRange, err := strconv.Atoi(maskList[1])
	if err != nil {
		fmt.Println("maskList[1] not valid")
		return
	}
	fmt.Println("lowRange = ", lowRange, " highrange = ", highRange)
	updatePrefixPolicyTableWithPrefix(ipAddr, name, op, lowRange, highRange)
}
func updatePrefixPolicyTableWithPrefixSet(prefixSet string, name string, op int) {
	fmt.Println("updatePrefixPolicyTableWithPrefixSet")
}

func updatePrefixPolicyTable(conditionInfo interface{}, name string, op int) {
	condition := conditionInfo.(MatchPrefixConditionInfo)
	fmt.Printf("updatePrefixPolicyTable for prefixSet %s prefix %s policy name %s op %d\n", condition.prefixSet, condition.Prefix, name, op)
	if condition.usePrefixSet {
		fmt.Println("Need to look up Prefix set to get the prefixes")
		updatePrefixPolicyTableWithPrefixSet(condition.prefixSet, name, op)
	} else {
		if condition.Prefix.MasklengthRange == "exact" {
			updatePrefixPolicyTableWithPrefix(condition.Prefix.IpPrefix, name, op, -1, -1)
		} else {
			fmt.Println("Masklength= ", condition.Prefix.MasklengthRange)
			updatePrefixPolicyTableWithMaskRange(condition.Prefix.IpPrefix, condition.Prefix.MasklengthRange, name, op)
		}
	}
}

func CreateBGPPolicyPrefixSet(cfg *bgpd.BGPPolicyPrefixSet) (val bool, err error) {
	fmt.Println("CreatePolicyPrefixSet")
	return val, err
}
func updateStatements(policy string, stmt string, op int) (err error) {
	fmt.Println("updateStatements stmt ", stmt, " with policy ", policy)
	var i int
	policyList := PolicyStmtPolicyMapDB[stmt]
	if policyList == nil {
		if op == del {
			fmt.Println("Cannot find the policy map for this stmt, so cannot delete")
			err = errors.New("Cannot find the policy map for this stmt, so cannot delete")
			return err
		}
		policyList = make([]string, 0)
	}
	if op == add {
		policyList = append(policyList, policy)
	}
	found := false
	if op == del {
		for i = 0; i < len(policyList); i++ {
			if policyList[i] == policy {
				fmt.Println("Found the policy in the policy stmt table, deleting it")
				found = true
				break
			}
		}
		if found {
			policyList = append(policyList[:i], policyList[i+1:]...)
		}
	}
	PolicyStmtPolicyMapDB[stmt] = policyList
	return err
}
func updateConditions(policyStmt PolicyStmt, conditionName string, op int) (err error) {
	fmt.Println("updateConditions for condition ", conditionName)
	conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(conditionName))
	if conditionItem != nil {
		condition := conditionItem.(PolicyCondition)
		switch condition.ConditionType {
		case policyCommonDefs.PolicyConditionTypeProtocolMatch:
			fmt.Println("PolicyConditionTypeProtocolMatch")
			updateProtocolPolicyTable(condition.ConditionInfo.(string), policyStmt.name, op)
			break
		case policyCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
			fmt.Println("PolicyConditionTypeDstIpPrefixMatch")
			updatePrefixPolicyTable(condition.ConditionInfo, policyStmt.name, op)
			break
		}
		if condition.PolicyStmtList == nil {
			condition.PolicyStmtList = make([]string, 0)
		}
		condition.PolicyStmtList = append(condition.PolicyStmtList, policyStmt.name)
		fmt.Println("Adding policy ", policyStmt.name, "to condition ", conditionName)
		PolicyConditionsDB.Set(patriciaDB.Prefix(conditionName), condition)
	} else {
		fmt.Println("Condition name ", conditionName, " not defined")
		err = errors.New("Condition name not defined")
	}
	return err
}

func updateActions(policyStmt PolicyStmt, actionName string, op int) (err error) {
	fmt.Println("updateActions for action ", actionName)
	actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(actionName))
	if actionItem != nil {
		action := actionItem.(PolicyAction)
		if action.policyStmtList == nil {
			action.policyStmtList = make([]string, 0)
		}
		action.policyStmtList = append(action.policyStmtList, policyStmt.name)
		PolicyActionsDB.Set(patriciaDB.Prefix(actionName), action)
	} else {
		fmt.Println("action name ", actionName, " not defined")
		err = errors.New("action name not defined")
	}
	return err
}

func CreateBGPPolicyStmtConfig(cfg *bgpd.BGPPolicyStmtConfig) (val bool, err error) {
	fmt.Println("CreatePolicyStatement")

	policyStmt := PolicyStmtDB.Get(patriciaDB.Prefix(cfg.Name))
	var i int
	if policyStmt == nil {
		fmt.Println("Defining a new policy statement with name ", cfg.Name)
		var newPolicyStmt PolicyStmt
		newPolicyStmt.name = cfg.Name
		if !validMatchConditions(cfg.MatchConditions) {
			fmt.Println("Invalid match conditions - try any/all")
			err = errors.New("Invalid match conditions - try any/all")
			return val, err
		}
		newPolicyStmt.matchConditions = cfg.MatchConditions
		if len(cfg.Conditions) > 0 {
			fmt.Println("Policy Statement has %d ", len(cfg.Conditions), " number of conditions")
			newPolicyStmt.conditions = make([]string, 0)
			for i = 0; i < len(cfg.Conditions); i++ {
				newPolicyStmt.conditions = append(newPolicyStmt.conditions, cfg.Conditions[i])
				err = updateConditions(newPolicyStmt, cfg.Conditions[i], add)
				if err != nil {
					fmt.Println("updateConditions returned err ", err)
					return val, err
				}
			}
		}
		if len(cfg.Actions) > 0 {
			fmt.Println("Policy Statement has %d ", len(cfg.Actions), " number of actions")
			newPolicyStmt.actions = make([]string, 0)
			for i = 0; i < len(cfg.Actions); i++ {
				newPolicyStmt.actions = append(newPolicyStmt.actions, cfg.Actions[i])
				err = updateActions(newPolicyStmt, cfg.Actions[i], add)
				if err != nil {
					fmt.Println("updateActions returned err ", err)
					return val, err
				}
			}
		}
		newPolicyStmt.localDBSliceIdx = int8(len(localPolicyStmtDB))
		if ok := PolicyStmtDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyStmt); ok != true {
			fmt.Println(" return value not ok")
			return val, err
		}
		localDBRecord := localDB{prefix: patriciaDB.Prefix(cfg.Name), isValid: true}
		if localPolicyStmtDB == nil {
			localPolicyStmtDB = make([]localDB, 0)
		}
		localPolicyStmtDB = append(localPolicyStmtDB, localDBRecord)
		//PolicyEngineTraverseAndApply(newPolicyStmt)
	} else {
		fmt.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
	}
	return val, err
}

func DeleteBGPPolicyStmtConfig(name string) (val bool, err error) {
	fmt.Println("DeletePolicyStatement for name ", name)
	ok := PolicyStmtDB.Match(patriciaDB.Prefix(name))
	if !ok {
		err = errors.New("No policy statement with this name found")
		return val, err
	}
	policyStmtInfoGet := PolicyStmtDB.Get(patriciaDB.Prefix(name))
	if policyStmtInfoGet != nil {
		//invalidate localPolicyStmt
		policyStmtInfo := policyStmtInfoGet.(PolicyStmt)
		if policyStmtInfo.localDBSliceIdx < int8(len(localPolicyStmtDB)) {
			fmt.Println("local DB slice index for this policy stmt is ", policyStmtInfo.localDBSliceIdx)
			localPolicyStmtDB[policyStmtInfo.localDBSliceIdx].isValid = false
		}
		// PolicyEngineTraverseAndReverse(policyStmtInfo)
		fmt.Println("Deleting policy statement with name ", name)
		if ok := PolicyStmtDB.Delete(patriciaDB.Prefix(name)); ok != true {
			fmt.Println(" return value not ok for delete PolicyDB")
			return val, err
		}
		//update other tables
		if len(policyStmtInfo.conditions) > 0 {
			for i := 0; i < len(policyStmtInfo.conditions); i++ {
				updateConditions(policyStmtInfo, policyStmtInfo.conditions[i], del)
			}
		}
		if len(policyStmtInfo.conditions) > 0 {
			for i := 0; i < len(policyStmtInfo.conditions); i++ {
				updateActions(policyStmtInfo, policyStmtInfo.actions[i], del)
			}
		}
	}
	return val, err
}

func GetBulkBGPPolicyStmtState(fromIndex bgpd.Int, rcount bgpd.Int) (policyStmts *bgpd.BGPPolicyStmtStateGetInfo, err error) {
	fmt.Println("GetBulkPolicyStmtState")
	var i, validCount, toIndex bgpd.Int
	var tempNode []bgpd.BGPPolicyStmtState = make([]bgpd.BGPPolicyStmtState, rcount)
	var nextNode *bgpd.BGPPolicyStmtState
	var returnNodes []*bgpd.BGPPolicyStmtState
	var returnGetInfo bgpd.BGPPolicyStmtStateGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
	if localPolicyStmtDB == nil {
		fmt.Println("destNetSlice not initialized")
		return policyStmts, err
	}
	for ; ; i++ {
		fmt.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if i+fromIndex >= bgpd.Int(len(localPolicyStmtDB)) {
			fmt.Println("All the policy statements fetched")
			more = false
			break
		}
		if localPolicyStmtDB[i+fromIndex].isValid == false {
			fmt.Println("Invalid policy statement")
			continue
		}
		if validCount == rcount {
			fmt.Println("Enough policy statements fetched")
			break
		}
		fmt.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyStmtDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyStmtDB.Get(localPolicyStmtDB[i+fromIndex].prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(PolicyStmt)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.name
			nextNode.Conditions = prefixNode.conditions
			nextNode.Actions = prefixNode.actions
			toIndex = bgpd.Int(prefixNode.localDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*bgpd.BGPPolicyStmtState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	fmt.Printf("Returning %d list of policyStmts", validCount)
	policyStmts.PolicyStmtStateList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex + 1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}

func CreateBGPPolicyDefinitionConfig(cfg *bgpd.BGPPolicyDefinitionConfig) (val bool, err error) {
	fmt.Println("CreatePolicyDefinition")
	if cfg.Import && ImportPolicyPrecedenceMap != nil {
		_, ok := ImportPolicyPrecedenceMap[int(cfg.Precedence)]
		if ok {
			fmt.Println("There is already a import policy with this precedence.")
			err = errors.New("There is already a import policy with this precedence.")
			return val, err
		}
	} else if cfg.Export && ExportPolicyPrecedenceMap != nil {
		_, ok := ExportPolicyPrecedenceMap[int(cfg.Precedence)]
		if ok {
			fmt.Println("There is already a export policy with this precedence.")
			err = errors.New("There is already a export policy with this precedence.")
			return val, err
		}
	} else if cfg.Global {
		fmt.Println("This is a global policy")
	}
	policy := PolicyDB.Get(patriciaDB.Prefix(cfg.Name))
	var i int
	if policy == nil {
		fmt.Println("Defining a new policy with name ", cfg.Name)
		var newPolicy Policy
		newPolicy.name = cfg.Name
		newPolicy.precedence = cfg.Precedence
		newPolicy.matchType = cfg.MatchType
		if cfg.Export == false && cfg.Import == false && cfg.Global == false {
			fmt.Println("Need to set import, export or global to true")
			return val, err
		}
		newPolicy.exportPolicy = cfg.Export
		newPolicy.importPolicy = cfg.Import
		newPolicy.globalPolicy = cfg.Global
		fmt.Println("Policy has %d ", len(cfg.PolicyDefinitionStatements), " number of statements")
		newPolicy.policyStmtPrecedenceMap = make(map[int]string)
		for i = 0; i < len(cfg.PolicyDefinitionStatements); i++ {
			fmt.Println("Adding statement ", cfg.PolicyDefinitionStatements[i].Statement, " at precedence id ", cfg.PolicyDefinitionStatements[i].Precedence)
			newPolicy.policyStmtPrecedenceMap[int(cfg.PolicyDefinitionStatements[i].Precedence)] = cfg.PolicyDefinitionStatements[i].Statement
			err = updateStatements(newPolicy.name, cfg.PolicyDefinitionStatements[i].Statement, add)
			if err != nil {
				fmt.Println("updateStatements returned err ", err)
				return val, err
			}
		}
		for k := range newPolicy.policyStmtPrecedenceMap {
			fmt.Println("key k = ", k)
		}
		newPolicy.localDBSliceIdx = int8(len(localPolicyDB))
		if ok := PolicyDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicy); ok != true {
			fmt.Println(" return value not ok")
			return val, err
		}
		localDBRecord := localDB{prefix: patriciaDB.Prefix(cfg.Name), isValid: true}
		if localPolicyDB == nil {
			localPolicyDB = make([]localDB, 0)
		}
		localPolicyDB = append(localPolicyDB, localDBRecord)
		if cfg.Import {
			fmt.Println("Adding ", newPolicy.name, " as import policy")
			if ImportPolicyPrecedenceMap == nil {
				ImportPolicyPrecedenceMap = make(map[int]string)
			}
			ImportPolicyPrecedenceMap[int(cfg.Precedence)] = cfg.Name
		} else if cfg.Export {
			fmt.Println("Adding ", newPolicy.name, " as export policy")
			if ExportPolicyPrecedenceMap == nil {
				ExportPolicyPrecedenceMap = make(map[int]string)
			}
			ExportPolicyPrecedenceMap[int(cfg.Precedence)] = cfg.Name
		}
		PolicyEngineTraverseAndApplyPolicy(newPolicy)
	} else {
		fmt.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
	}
	return val, err
}

func DeleteBGPPolicyDefinitionConfig(name string) (val bool, err error) {
	fmt.Println("DeletePolicyDefinition for name ", name)
	ok := PolicyDB.Match(patriciaDB.Prefix(name))
	if !ok {
		err = errors.New("No policy with this name found")
		return val, err
	}
	policyInfoGet := PolicyDB.Get(patriciaDB.Prefix(name))
	if policyInfoGet != nil {
		//invalidate localPolicy
		policyInfo := policyInfoGet.(Policy)
		if policyInfo.localDBSliceIdx < int8(len(localPolicyDB)) {
			fmt.Println("local DB slice index for this policy is ", policyInfo.localDBSliceIdx)
			localPolicyDB[policyInfo.localDBSliceIdx].isValid = false
		}
		PolicyEngineTraverseAndReversePolicy(policyInfo)
		fmt.Println("Deleting policy with name ", name)
		if ok := PolicyDB.Delete(patriciaDB.Prefix(name)); ok != true {
			fmt.Println(" return value not ok for delete PolicyDB")
			return val, err
		}
		for _, v := range policyInfo.policyStmtPrecedenceMap {
			err = updateStatements(policyInfo.name, v, del)
			if err != nil {
				fmt.Println("updateStatements returned err ", err)
				return val, err
			}
		}
		if policyInfo.exportPolicy {
			if ExportPolicyPrecedenceMap != nil {
				delete(ExportPolicyPrecedenceMap, int(policyInfo.precedence))
			}
		}
		if policyInfo.importPolicy {
			if ImportPolicyPrecedenceMap != nil {
				delete(ImportPolicyPrecedenceMap, int(policyInfo.precedence))
			}
		}
	}
	return val, err
}

func GetBulkBGPPolicyDefinitionState(fromIndex bgpd.Int, rcount bgpd.Int) (policyStmts *bgpd.BGPPolicyDefinitionStateGetInfo, err error) { //(routes []*bgpd.BGPRoute, err error) {
	fmt.Println("GetBulkPolicyDefinitionState")
	var i, validCount, toIndex bgpd.Int
	var tempNode []bgpd.BGPPolicyDefinitionState = make([]bgpd.BGPPolicyDefinitionState, rcount)
	var nextNode *bgpd.BGPPolicyDefinitionState
	var returnNodes []*bgpd.BGPPolicyDefinitionState
	var returnGetInfo bgpd.BGPPolicyDefinitionStateGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
	if localPolicyDB == nil {
		fmt.Println("localPolicyDB not initialized")
		return policyStmts, err
	}
	for ; ; i++ {
		fmt.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if i+fromIndex >= bgpd.Int(len(localPolicyDB)) {
			fmt.Println("All the policies fetched")
			more = false
			break
		}
		if localPolicyDB[i+fromIndex].isValid == false {
			fmt.Println("Invalid policy")
			continue
		}
		if validCount == rcount {
			fmt.Println("Enough policies fetched")
			break
		}
		fmt.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyDB.Get(localPolicyDB[i+fromIndex].prefix)
		if prefixNodeGet != nil {
			prefixNode := prefixNodeGet.(Policy)
			nextNode = &tempNode[validCount]
			nextNode.Name = prefixNode.name
			nextNode.HitCounter = bgpd.Int(prefixNode.hitCounter)
			nextNode.IpPrefixList = make([]string, 0)
			for k := 0; k < len(prefixNode.routeList); k++ {
				nextNode.IpPrefixList = append(nextNode.IpPrefixList, prefixNode.routeList[k])
			}
			toIndex = bgpd.Int(prefixNode.localDBSliceIdx)
			if len(returnNodes) == 0 {
				returnNodes = make([]*bgpd.BGPPolicyDefinitionState, 0)
			}
			returnNodes = append(returnNodes, nextNode)
			validCount++
		}
	}
	fmt.Printf("Returning %d list of policies", validCount)
	policyStmts.PolicyDefinitionStateList = returnNodes
	policyStmts.StartIdx = fromIndex
	policyStmts.EndIdx = toIndex + 1
	policyStmts.More = more
	policyStmts.Count = validCount
	return policyStmts, err
}
