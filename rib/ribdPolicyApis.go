// policyApis.go
package main

import (
	"ribd"
	"errors"
	"l3/rib/ribdCommonDefs"
	"utils/patriciaDB"
	"strconv"
	"strings"
	"net"
	"reflect"
)

type PolicyStmt struct {				//policy engine uses this
	name               string
	precedence         ribd.Int
	matchConditions    string
	conditions         []string
	actions            []string
	localDBSliceIdx        int8  
}

type Policy struct {
	name              string
	precedence        ribd.Int
	matchType         string
	policyStmtPrecedenceMap map[int]string
	hitCounter         int   
	routeList         []string
	routeInfoList     []ribd.Routes
	localDBSliceIdx        int8  
	importPolicy       bool
	exportPolicy       bool  
	globalPolicy       bool
}

var PolicyDB = patriciaDB.NewTrie()
var PolicyStmtDB = patriciaDB.NewTrie()
var PolicyStmtPolicyMapDB = make(map[string] []string) //policies using this statement
var PrefixPolicyListDB = patriciaDB.NewTrie()
var ProtocolPolicyListDB = make(map[string][]string)//policystmt names assoociated with every protocol type
var ImportPolicyPrecedenceMap = make(map[int] string)
var ExportPolicyPrecedenceMap = make(map[int] string)
var localPolicyStmtDB []localDB
var localPolicyDB []localDB

func addPolicyRouteMap(route ribd.Routes, policy Policy) {
	logger.Println("addPolicyRouteMap")
	policy.hitCounter++
	ipPrefix,err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Invalid ip prefix")
		return
	}
	maskIp, err := getIP(route.Mask)
	if err != nil {
		return
	}
	prefixLen,err := getPrefixLen(maskIp)
	if err != nil {
		return
	}
	logger.Println("prefixLen= ", prefixLen)
	var newRoute string
	found := false
	newRoute = route.Ipaddr + "/"+strconv.Itoa(prefixLen)
//	newRoute := string(ipPrefix[:])
	logger.Println("Adding ip prefix %s %v ", newRoute, ipPrefix)
	policyInfo:=PolicyDB.Get(patriciaDB.Prefix(policy.name))
	if policyInfo == nil {
		logger.Println("Unexpected:policyInfo nil for policy ", policy.name)
		return
	}
	tempPolicy:=policyInfo.(Policy)
	if tempPolicy.routeList == nil {
		logger.Println("routeList nil")
		tempPolicy.routeList = make([]string, 0)
	}
	logger.Println("routelist len= ", len(tempPolicy.routeList)," prefix list so far")
	for i:=0;i<len(tempPolicy.routeList);i++ {
		logger.Println(tempPolicy.routeList[i])
		if tempPolicy.routeList[i] == newRoute {
			logger.Println(newRoute, " already is a part of ", policy.name, "'s routelist")
			found = true
		}
	}
	if !found {
       tempPolicy.routeList = append(tempPolicy.routeList, newRoute)
	}
	found=false
	logger.Println("routeInfoList details")
	for i:=0;i<len(tempPolicy.routeInfoList);i++ {
		logger.Println("IP: ",tempPolicy.routeInfoList[i].Ipaddr, ":", tempPolicy.routeInfoList[i].Mask, " routeType: ", tempPolicy.routeInfoList[i].Prototype)
		if tempPolicy.routeInfoList[i].Ipaddr==route.Ipaddr && tempPolicy.routeInfoList[i].Mask == route.Mask && tempPolicy.routeInfoList[i].Prototype == route.Prototype {
			logger.Println("route already is a part of ", policy.name, "'s routeInfolist")
			found = true
		}
	}
	if tempPolicy.routeInfoList == nil {
		tempPolicy.routeInfoList = make([]ribd.Routes, 0)
	}
	if found == false {
       tempPolicy.routeInfoList = append(tempPolicy.routeInfoList, route)
	}
	PolicyDB.Set(patriciaDB.Prefix(policy.name), tempPolicy)
}
func deletePolicyRouteMap(route ribd.Routes, policy Policy) {
	logger.Println("deletePolicyRouteMap")
}
func updatePolicyRouteMap(route ribd.Routes, policy Policy, op int) {
	logger.Println("updatePolicyRouteMap")
	if op == add {
		addPolicyRouteMap(route, policy)
	} else if op == del {
		deletePolicyRouteMap(route, policy)
	}
	
}
func validMatchConditions(matchConditionStr string) (valid bool) {
    logger.Println("validMatchConditions for string ", matchConditionStr)
	if matchConditionStr == "any" || matchConditionStr == "all "{
		logger.Println("valid")
		valid = true
	}
	return valid
}
func updateProtocolPolicyTable(protoType string, name string, op int) {
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
	found :=false
	if op == del {
		for i =0; i< len(policyList);i++ {
			if policyList[i] == name {
				logger.Println("Found the policy in the protocol policy table, deleting it")
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
func updatePrefixPolicyTableWithPrefix(ipPrefix patriciaDB.Prefix, name string, op int){
	logger.Println("updatePrefixPolicyTableWithPrefix %v", ipPrefix)
	var i int
	var policyList []string
	policyListItem:= PrefixPolicyListDB.Get(ipPrefix)
	if policyListItem != nil && reflect.TypeOf(policyListItem).Kind() != reflect.Slice {
		logger.Println("Incorrect data type for this prefix ")
		return
	}
	if(policyListItem == nil) {
		if (op == del) {
			logger.Println("Cannot find the policy map for this prefix, so cannot delete")
			return
		}
		policyList = make([]string, 0)
	} else {
	   policyListSlice := reflect.ValueOf(policyListItem)
	   policyList = make([]string,0)
	   for i = 0;i<policyListSlice.Len();i++ {
	      policyList = append(policyList, policyListSlice.Index(i).Interface().(string))	
	   }
	}
    if op == add {
	   policyList = append(policyList, name)
	}
	found :=false
	if op == del {
		for i =0; i< len(policyList);i++ {
			if policyList[i] == name {
				logger.Println("Found the policy in the prefix policy table, deleting it")
				break
			}
		}
		if found {
		   policyList = append(policyList[:i], policyList[i+1:]...)
		}
	}
	PrefixPolicyListDB.Set(ipPrefix, policyList)
}
func updatePrefixPolicyTableWithMaskRange(ipAddrStr string, masklength string, name string, op int){
	logger.Println("updatePrefixPolicyTableWithMaskRange")
	    maskList := strings.Split(masklength,"..")
		if len(maskList) !=2 {
			logger.Println("Invalid masklength range")
			return 
		}
        lowRange,err := strconv.Atoi(maskList[0])
		if err != nil {
			logger.Println("maskList[0] not valid")
			return
		}
		highRange,err := strconv.Atoi(maskList[1])
		if err != nil {
			logger.Println("maskList[1] not valid")
			return
		}
		logger.Println("lowRange = ", lowRange, " highrange = ", highRange)
		for idx := lowRange;idx<highRange;idx ++ {
			ipMask:= net.CIDRMask(idx, 32)
			ipMaskStr := net.IP(ipMask).String()
			logger.Println("idx ", idx, "ipMaskStr = ", ipMaskStr)
			ipPrefix, err := getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
			if err != nil {
				logger.Println("Invalid prefix")
				return 
			}
			updatePrefixPolicyTableWithPrefix(ipPrefix, name, op)
		}
}
func updatePrefixPolicyTableWithPrefixSet(prefixSet string, name string, op int) {
	logger.Println("updatePrefixPolicyTableWithPrefixSet")
}
func updatePrefixPolicyTable(conditionInfo interface{}, name string, op int) {
    condition := conditionInfo.(MatchPrefixConditionInfo)
	logger.Printf("updatePrefixPolicyTable for prefixSet %s prefix %s policy name %s op %d\n", condition.prefixSet, condition.prefix, name, op)
    if condition.usePrefixSet {
		logger.Println("Need to look up Prefix set to get the prefixes")
		updatePrefixPolicyTableWithPrefixSet(condition.prefixSet, name, op)
	} else {
	   if condition.prefix.MasklengthRange == "exact" {
       ipPrefix, err := getNetworkPrefixFromCIDR(condition.prefix.IpPrefix)
	   if err != nil {
		logger.Println("ipPrefix invalid ")
		return 
	   }
	   updatePrefixPolicyTableWithPrefix(ipPrefix, name, op)
	 } else {
		logger.Println("Masklength= ", condition.prefix.MasklengthRange)
		ip, _, err := net.ParseCIDR(condition.prefix.IpPrefix)
	    if err != nil {
		   return 
	    }
	    ipAddrStr := ip.String()
		updatePrefixPolicyTableWithMaskRange(ipAddrStr, condition.prefix.MasklengthRange, name, op)
	 }
   }
}


func (m RouteServiceHandler) CreatePolicyPrefixSet(cfg *ribd.PolicyPrefixSet ) (val bool, err error) {
	logger.Println("CreatePolicyPrefixSet")
	return val, err
}
func updateStatements(policy  string, stmt string, op int) (err error){
   logger.Println("updateStatements stmt ", stmt, " with policy ", policy)
   var i int
    policyList := PolicyStmtPolicyMapDB[stmt]
	if(policyList == nil) {
		if (op == del) {
			logger.Println("Cannot find the policy map for this stmt, so cannot delete")
            err = errors.New("Cannot find the policy map for this stmt, so cannot delete")
			return err
		}
		policyList = make([]string, 0)
	}
    if op == add {
	   policyList = append(policyList, policy)
	}
	found :=false
	if op == del {
		for i =0; i< len(policyList);i++ {
			if policyList[i] == policy {
				logger.Println("Found the policy in the policy stmt table, deleting it")
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
func updateConditions(policyStmt PolicyStmt, conditionName string, op int) (err error){
	logger.Println("updateConditions for condition ", conditionName)
	conditionItem := PolicyConditionsDB.Get(patriciaDB.Prefix(conditionName))
	if(conditionItem != nil) {
		condition := conditionItem.(PolicyCondition)
		switch condition.conditionType {
			case ribdCommonDefs.PolicyConditionTypeProtocolMatch:
			   logger.Println("PolicyConditionTypeProtocolMatch")
			   updateProtocolPolicyTable(condition.conditionInfo.(string), policyStmt.name, op)
			   break
			case ribdCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
			   logger.Println("PolicyConditionTypeDstIpPrefixMatch")
			   updatePrefixPolicyTable(condition.conditionInfo, policyStmt.name, op)
			   break
		}
		if condition.policyStmtList == nil {
			condition.policyStmtList = make([]string,0)
		}
        condition.policyStmtList = append(condition.policyStmtList, policyStmt.name)
		logger.Println("Adding policy ", policyStmt.name, "to condition ", conditionName)
		PolicyConditionsDB.Set(patriciaDB.Prefix(conditionName), condition)
	} else {
		logger.Println("Condition name ", conditionName, " not defined")
		err = errors.New("Condition name not defined")
	}
	return err
}

func updateActions(policyStmt PolicyStmt, actionName string, op int) (err error) {
	logger.Println("updateActions for action ", actionName)
	actionItem := PolicyActionsDB.Get(patriciaDB.Prefix(actionName))
	if(actionItem != nil) {
		action := actionItem.(PolicyAction)
		if action.policyStmtList == nil {
			action.policyStmtList = make([]string,0)
		}
        action.policyStmtList = append(action.policyStmtList, policyStmt.name)
		PolicyActionsDB.Set(patriciaDB.Prefix(actionName), action)
	} else {
		logger.Println("action name ", actionName, " not defined")
		err = errors.New("action name not defined")
	}
	return err
}

func (m RouteServiceHandler) CreatePolicyStatement(cfg *ribd.PolicyStmtConfig) (val bool, err error) {
	logger.Println("CreatePolicyStatement")

	policyStmt := PolicyStmtDB.Get(patriciaDB.Prefix(cfg.Name))
	var i int
	if(policyStmt == nil) {
	   logger.Println("Defining a new policy statement with name ", cfg.Name)
	   var newPolicyStmt PolicyStmt
	   newPolicyStmt.name = cfg.Name
	   if !validMatchConditions(cfg.MatchConditions) {
	      logger.Println("Invalid match conditions - try any/all")
		  err = errors.New("Invalid match conditions - try any/all")	
		  return val, err
	   }
	   newPolicyStmt.matchConditions = cfg.MatchConditions
	   if len(cfg.Conditions) > 0 {
	      logger.Println("Policy Statement has %d ", len(cfg.Conditions)," number of conditions")	
		  newPolicyStmt.conditions = make([] string, 0)
		  for i=0;i<len(cfg.Conditions);i++ {
			newPolicyStmt.conditions = append(newPolicyStmt.conditions, cfg.Conditions[i])
			err = updateConditions(newPolicyStmt, cfg.Conditions[i], add)
			if err != nil {
				logger.Println("updateConditions returned err ", err)
				return val,err
			}
		}
	   }
	   if len(cfg.Actions) > 0 {
	      logger.Println("Policy Statement has %d ", len(cfg.Actions)," number of actions")	
		  newPolicyStmt.actions = make([] string, 0)
		  for i=0;i<len(cfg.Actions);i++ {
			newPolicyStmt.actions = append(newPolicyStmt.actions,cfg.Actions[i])
			err = updateActions(newPolicyStmt, cfg.Actions[i], add)
			if err != nil {
				logger.Println("updateActions returned err ", err)
				return val,err
			}
		}
	   }
        newPolicyStmt.localDBSliceIdx = int8(len(localPolicyStmtDB))
		if ok := PolicyStmtDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicyStmt); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyStmtDB == nil) {
			localPolicyStmtDB = make([]localDB, 0)
		} 
	    localPolicyStmtDB = append(localPolicyStmtDB, localDBRecord)
	    //PolicyEngineTraverseAndApply(newPolicyStmt)
	} else {
		logger.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
	}
	return val, err
}

func (m RouteServiceHandler) 	DeletePolicyStatement(cfg *ribd.PolicyStmtConfig) (val bool, err error) {
	logger.Println("DeletePolicyStatement for name ", cfg.Name)
	ok := PolicyStmtDB.Match(patriciaDB.Prefix(cfg.Name))
	if !ok {
		err = errors.New("No policy statement with this name found")
		return val, err
	}
	policyStmtInfoGet := PolicyStmtDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyStmtInfoGet != nil) {
       //invalidate localPolicyStmt 
	   policyStmtInfo := policyStmtInfoGet.(PolicyStmt)
	   if policyStmtInfo.localDBSliceIdx < int8(len(localPolicyStmtDB)) {
          logger.Println("local DB slice index for this policy stmt is ", policyStmtInfo.localDBSliceIdx)
		  localPolicyStmtDB[policyStmtInfo.localDBSliceIdx].isValid = false		
	   }
	  // PolicyEngineTraverseAndReverse(policyStmtInfo)
	   logger.Println("Deleting policy statement with name ", cfg.Name)
		if ok := PolicyStmtDB.Delete(patriciaDB.Prefix(cfg.Name)); ok != true {
			logger.Println(" return value not ok for delete PolicyDB")
			return val, err
		}
	   //update other tables
	   if len(policyStmtInfo.conditions) > 0 {
	      for i:=0;i<len(policyStmtInfo.conditions);i++ {
			updateConditions(policyStmtInfo, policyStmtInfo.conditions[i],del)
		}	
	   }
	   if len(policyStmtInfo.conditions) > 0 {
	      for i:=0;i<len(policyStmtInfo.conditions);i++ {
			updateActions(policyStmtInfo, policyStmtInfo.actions[i],del)
		}	
	   }
	} 
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyStmtState( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyStmtStateGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkPolicyStmtState")
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
		if(localPolicyStmtDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy statement")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policy statements fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyStmtDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyStmtDB.Get(localPolicyStmtDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(PolicyStmt)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.Conditions = prefixNode.conditions
			nextNode.Actions = prefixNode.actions
			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
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
	if cfg.Import && ImportPolicyPrecedenceMap != nil {
	   _,ok:=ImportPolicyPrecedenceMap[int(cfg.Precedence)]
	   if ok {
		logger.Println("There is already a import policy with this precedence.")
		err =  errors.New("There is already a import policy with this precedence.")
         return val,err
	   }
	} else if cfg.Export && ExportPolicyPrecedenceMap != nil {
	   _,ok:=ExportPolicyPrecedenceMap[int(cfg.Precedence)]
	   if ok {
		logger.Println("There is already a export policy with this precedence.")
		err =  errors.New("There is already a export policy with this precedence.")
         return val,err
	   }
	} else if cfg.Global {
		logger.Println("This is a global policy")
	}
	policy := PolicyDB.Get(patriciaDB.Prefix(cfg.Name))
	var i int
	if(policy == nil) {
	   logger.Println("Defining a new policy with name ", cfg.Name)
	   var newPolicy Policy
	   newPolicy.name = cfg.Name
	   newPolicy.precedence = cfg.Precedence
	   newPolicy.matchType = cfg.MatchType
       if cfg.Export == false && cfg.Import == false && cfg.Global == false {
			logger.Println("Need to set import, export or global to true")
			return val, err
	   }	  
	   newPolicy.exportPolicy = cfg.Export
	   newPolicy.importPolicy = cfg.Import
	   newPolicy.globalPolicy = cfg.Global
	   logger.Println("Policy has %d ", len(cfg.PolicyDefinitionStatements)," number of statements")
	   newPolicy.policyStmtPrecedenceMap = make(map[int]string)	
	   for i=0;i<len(cfg.PolicyDefinitionStatements);i++ {
		  logger.Println("Adding statement ", cfg.PolicyDefinitionStatements[i].Statement, " at precedence id ", cfg.PolicyDefinitionStatements[i].Precedence)
          newPolicy.policyStmtPrecedenceMap[int(cfg.PolicyDefinitionStatements[i].Precedence)] = cfg.PolicyDefinitionStatements[i].Statement 
		  err = updateStatements(newPolicy.name, cfg.PolicyDefinitionStatements[i].Statement, add)
		  if err != nil {
			logger.Println("updateStatements returned err ", err)
			return val,err
		  }
	   }
       for k:=range newPolicy.policyStmtPrecedenceMap {
		logger.Println("key k = ", k)
	   }
       newPolicy.localDBSliceIdx = int8(len(localPolicyDB))
	   if ok := PolicyDB.Insert(patriciaDB.Prefix(cfg.Name), newPolicy); ok != true {
			logger.Println(" return value not ok")
			return val, err
		}
        localDBRecord := localDB{prefix:patriciaDB.Prefix(cfg.Name), isValid:true}
		if(localPolicyDB == nil) {
			localPolicyDB = make([]localDB, 0)
		} 
	    localPolicyDB = append(localPolicyDB, localDBRecord)
		if cfg.Import {
		   logger.Println("Adding ", newPolicy.name, " as import policy")
		   if ImportPolicyPrecedenceMap == nil {
	          ImportPolicyPrecedenceMap = make(map[int]string)	
		   }
		   ImportPolicyPrecedenceMap[int(cfg.Precedence)]=cfg.Name
		} else if cfg.Export {
		   logger.Println("Adding ", newPolicy.name, " as export policy")
		   if ExportPolicyPrecedenceMap == nil {
	          ExportPolicyPrecedenceMap = make(map[int]string)	
		   }
		   ExportPolicyPrecedenceMap[int(cfg.Precedence)]=cfg.Name
		}
	     PolicyEngineTraverseAndApplyPolicy(newPolicy)
	} else {
		logger.Println("Duplicate Policy definition name")
		err = errors.New("Duplicate policy definition")
		return val, err
	}
	return val, err
}

func (m RouteServiceHandler) 	DeletePolicyDefinition(cfg *ribd.PolicyDefinitionConfig) (val bool, err error) {
	logger.Println("DeletePolicyDefinition for name ", cfg.Name)
	ok := PolicyDB.Match(patriciaDB.Prefix(cfg.Name))
	if !ok {
		err = errors.New("No policy with this name found")
		return val, err
	}
	policyInfoGet := PolicyDB.Get(patriciaDB.Prefix(cfg.Name))
	if(policyInfoGet != nil) {
       //invalidate localPolicy 
	   policyInfo := policyInfoGet.(Policy)
	   if policyInfo.localDBSliceIdx < int8(len(localPolicyDB)) {
          logger.Println("local DB slice index for this policy is ", policyInfo.localDBSliceIdx)
		  localPolicyDB[policyInfo.localDBSliceIdx].isValid = false		
	   }
	   PolicyEngineTraverseAndReversePolicy(policyInfo)
	   logger.Println("Deleting policy with name ", cfg.Name)
		if ok := PolicyDB.Delete(patriciaDB.Prefix(cfg.Name)); ok != true {
			logger.Println(" return value not ok for delete PolicyDB")
			return val, err
		}
		for _,v:=range policyInfo.policyStmtPrecedenceMap {
		  err = updateStatements(policyInfo.name, v, del)
		  if err != nil {
			logger.Println("updateStatements returned err ", err)
			return val,err
		  }
		}
		if policyInfo.exportPolicy{
			if ExportPolicyPrecedenceMap != nil {
				delete(ExportPolicyPrecedenceMap,int(policyInfo.precedence))
			}
		}
		if policyInfo.importPolicy{
			if ImportPolicyPrecedenceMap != nil {
				delete(ImportPolicyPrecedenceMap,int(policyInfo.precedence))
			}
		}
	} 
	return val, err
}

func (m RouteServiceHandler) GetBulkPolicyDefinitionState( fromIndex ribd.Int, rcount ribd.Int) (policyStmts *ribd.PolicyDefinitionStateGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkPolicyDefinitionState")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.PolicyDefinitionState = make ([]ribd.PolicyDefinitionState, rcount)
	var nextNode *ribd.PolicyDefinitionState
    var returnNodes []*ribd.PolicyDefinitionState
	var returnGetInfo ribd.PolicyDefinitionStateGetInfo
	i = 0
	policyStmts = &returnGetInfo
	more := true
    if(localPolicyDB == nil) {
		logger.Println("localPolicyDB not initialized")
		return policyStmts, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localPolicyDB))) {
			logger.Println("All the policies fetched")
			more = false
			break
		}
		if(localPolicyDB[i+fromIndex].isValid == false) {
			logger.Println("Invalid policy")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough policies fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (localPolicyDB[i+fromIndex].prefix))
		prefixNodeGet := PolicyDB.Get(localPolicyDB[i+fromIndex].prefix)
		if(prefixNodeGet != nil) {
			prefixNode := prefixNodeGet.(Policy)
			nextNode = &tempNode[validCount]
		    nextNode.Name = prefixNode.name
			nextNode.HitCounter = ribd.Int(prefixNode.hitCounter)
			nextNode.IpPrefixList = make([]string,0)
			for k:=0;k<len(prefixNode.routeList);k++ {
			   nextNode.IpPrefixList = append(nextNode.IpPrefixList,prefixNode.routeList[k])
			}
			toIndex = ribd.Int(prefixNode.localDBSliceIdx)
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
