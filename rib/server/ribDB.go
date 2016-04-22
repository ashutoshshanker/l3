// ribDB.go
package server

import (
	"fmt"
	"ribd"
	"ribdInt"
	//"utils/commonDefs"
	//    "utils/dbutils"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func (ribdServiceHandler *RIBDServer) UpdateRoutesFromDB() (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutesFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	defer dbHdl.Close()
	dbCmd := "select * from IPv4Route"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var ipRoute IPRoute
	for rows.Next() {
		if err = rows.Scan(&ipRoute.DestinationNw, &ipRoute.NetworkMask, &ipRoute.NextHopIp, &ipRoute.Cost, &ipRoute.OutgoingIntfType, &ipRoute.OutgoingInterface, &ipRoute.Protocol); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over IPV4Route rows with error %s\n", err))
			return err
		}
		cfg := ribd.IPv4Route{
			DestinationNw:     ipRoute.DestinationNw,
			Protocol:          ipRoute.Protocol,
			OutgoingInterface: ipRoute.OutgoingInterface,
			OutgoingIntfType:  ipRoute.OutgoingIntfType,
			Cost:              int32(ipRoute.Cost),
			NetworkMask:       ipRoute.NetworkMask,
			NextHopIp:         ipRoute.NextHopIp}
		_, err = ribdServiceHandler.ProcessRouteCreateConfig(&cfg) //ipRoute.DestinationNw, ipRoute.NetworkMask, ribd.Int(ipRoute.Cost), ipRoute.NextHopIp, outIntfType, ribd.Int(outIntf), ipRoute.Protocol)
		//_,err = createV4Route(ipRoute.DestinationNw, ipRoute.NetworkMask, ribd.Int(ipRoute.Cost), ipRoute.NextHopIp, outIntfType,ribd.Int(outIntf), ribd.Int(proto),  FIBAndRIB,ribdCommonDefs.RoutePolicyStateChangetoValid,ribd.Int(len(destNetSlice)))
		if err != nil {
			logger.Info(fmt.Sprintf("Route create failed with err %s\n", err))
			return err
		}
	}
	return err
}

func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyConditionsFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyConditionsFromDB"))
	dbCmd := "select * from PolicyCondition"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var condition ribd.PolicyCondition
	var IpPrefix, MaskLengthRange string
	for rows.Next() {
		if err = rows.Scan(&condition.Name, &condition.ConditionType, &condition.Protocol, &IpPrefix, &MaskLengthRange); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionStmtMatchProtocolCondition rows with error %s\n", err))
			return err
		}
		//var cfgIpPrefix ribd.PolicyPrefix
		//var dstIpMatchPrefixconditionCfg ribd.PolicyDstIpMatchPrefixSetCondition
		//cfgIpPrefix.IpPrefix = IpPrefix
		//cfgIpPrefix.MasklengthRange = MaskLengthRange
		//dstIpMatchPrefixconditionCfg.Prefix = &cfgIpPrefix
		//condition.MatchDstIpPrefixConditionInfo = &dstIpMatchPrefixconditionCfg
		condition.IpPrefix = IpPrefix
		condition.MaskLengthRange = MaskLengthRange
		ribdServiceHandler.ProcessPolicyConditionConfigCreate(&condition,GlobalPolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("Condition create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyConditionsFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyConditionsFromDB"))
	dbCmd := "select * from RoutePolicyCondition"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var condition ribd.PolicyCondition
	var IpPrefix, MaskLengthRange string
	for rows.Next() {
		if err = rows.Scan(&condition.Name, &condition.ConditionType, &condition.Protocol, &IpPrefix, &MaskLengthRange); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionStmtMatchProtocolCondition rows with error %s\n", err))
			return err
		}
		//var cfgIpPrefix ribd.PolicyPrefix
		//var dstIpMatchPrefixconditionCfg ribd.PolicyDstIpMatchPrefixSetCondition
		//cfgIpPrefix.IpPrefix = IpPrefix
		//cfgIpPrefix.MasklengthRange = MaskLengthRange
		//dstIpMatchPrefixconditionCfg.Prefix = &cfgIpPrefix
		//condition.MatchDstIpPrefixConditionInfo = &dstIpMatchPrefixconditionCfg
		condition.IpPrefix = IpPrefix
		condition.MaskLengthRange = MaskLengthRange
		ribdServiceHandler.ProcessPolicyConditionConfigCreate(&condition,PolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("Condition create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyActionsFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyActionsFromDB"))
	dbCmd := "select * from RoutePolicyAction"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var action ribdInt.PolicyAction
	for rows.Next() {
		if err = rows.Scan(&action.Name, &action.ActionType, &action.SetAdminDistanceValue, &action.Accept, &action.Reject, &action.RedistributeAction, &action.RedistributeTargetProtocol, &action.NetworkStatementTargetProtocol); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionStmtMatchProtocolCondition rows with error %s\n", err))
			return err
		}
		_,err = ribdServiceHandler.ProcessPolicyActionConfigCreate(&action,PolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("Action create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyStmtsFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyStmtsFromDB"))
	dbCmd := "select * from PolicyStmt"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var stmt ribd.PolicyStmt
	for rows.Next() {
		if err = rows.Scan(&stmt.Name, &stmt.MatchConditions,&stmt.Action); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionStmtMatchProtocolCondition rows with error %s\n", err))
			return err
		}
		logger.Info(fmt.Sprintln("Scanning stmt ", stmt.Name, "MatchConditions:", stmt.MatchConditions))
		dbCmdCond := "select * from PolicyStmtConditions"
		conditionrows, err := dbHdl.Query(dbCmdCond)
		if err != nil {
			logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmdCond, err))
			return err
		}
		stmt.Conditions = make([]string, 0)
		var Conditions, stmtName string
		for conditionrows.Next() {
			if err = conditionrows.Scan(&stmtName, &Conditions); err != nil {
				logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyStmtConfigConditions rows with error %s\n", err))
				return err
			}
			if stmtName != stmt.Name {
				logger.Info(fmt.Sprintln("Not a condition for this statement"))
				continue
			}
			logger.Info(fmt.Sprintln("Fetching condition ", Conditions))
			stmt.Conditions = append(stmt.Conditions, Conditions)
		}

		err = ribdServiceHandler.ProcessPolicyStmtConfigCreate(&stmt,GlobalPolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("Action create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyStmtsFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyStmtsFromDB"))
	dbCmd := "select * from RoutePolicyStmt"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var stmt ribd.PolicyStmt
	for rows.Next() {
		if err = rows.Scan(&stmt.Name, &stmt.MatchConditions,&stmt.Action); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionStmtMatchProtocolCondition rows with error %s\n", err))
			return err
		}
		logger.Info(fmt.Sprintln("Scanning stmt ", stmt.Name, "MatchConditions:", stmt.MatchConditions))
		dbCmdCond := "select * from PolicyStmtConditions"
		conditionrows, err := dbHdl.Query(dbCmdCond)
		if err != nil {
			logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmdCond, err))
			return err
		}
		stmt.Conditions = make([]string, 0)
		var Conditions, stmtName string
		for conditionrows.Next() {
			if err = conditionrows.Scan(&stmtName, &Conditions); err != nil {
				logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyStmtConfigConditions rows with error %s\n", err))
				return err
			}
			if stmtName != stmt.Name {
				logger.Info(fmt.Sprintln("Not a condition for this statement"))
				continue
			}
			logger.Info(fmt.Sprintln("Fetching condition ", Conditions))
			stmt.Conditions = append(stmt.Conditions, Conditions)
		}

		err = ribdServiceHandler.ProcessPolicyStmtConfigCreate(&stmt,PolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("Action create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyFromDB"))
	dbCmd := "select * from PolicyDefinition"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var policy ribd.PolicyDefinition
	for rows.Next() {
		if err = rows.Scan(&policy.Name, &policy.Priority, &policy.MatchType); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionConfig rows with error %s\n", err))
			return err
		}
		logger.Info(fmt.Sprintln("executed cmd ", dbCmd, "policy name = ", policy.Name, " precedence: ", policy.Priority))
		dbCmdPrecedence := "select * from PolicyDefinitionStatementList"
		conditionrows, err := dbHdl.Query(dbCmdPrecedence)
		if err != nil {
			logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmdPrecedence, err))
			return err
		}
		policy.StatementList = make([]*ribd.PolicyDefinitionStmtPriority, 0)
		var stmt, policyName string
		var precedence int32
		for conditionrows.Next() {
			if err = conditionrows.Scan(&policyName, &precedence, &stmt); err != nil {
				logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionConfigStatementList rows with error %s\n", err))
				return err
			}
			if policyName != policy.Name {
				logger.Info(fmt.Sprintln("Not a stmt for this policy, policyName: ", policyName))
				continue
			}
			logger.Info(fmt.Sprintln("Fetching stmt ", stmt, "Priority:", precedence))
			policyStmtPrecedence := ribd.PolicyDefinitionStmtPriority{Priority: int32(precedence), Statement: stmt}
			policy.StatementList = append(policy.StatementList, &policyStmtPrecedence)
		}

		err = ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(&policy,GlobalPolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("policy create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyFromDB(dbHdl *sql.DB) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyFromDB"))
	dbCmd := "select * from PolicyDefinition"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmd, err))
		return err
	}
	var policy ribd.PolicyDefinition
	for rows.Next() {
		if err = rows.Scan(&policy.Name, &policy.Priority, &policy.MatchType); err != nil {
			logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionConfig rows with error %s\n", err))
			return err
		}
		logger.Info(fmt.Sprintln("executed cmd ", dbCmd, "policy name = ", policy.Name, " precedence: ", policy.Priority))
		dbCmdPrecedence := "select * from PolicyDefinitionStatementList"
		conditionrows, err := dbHdl.Query(dbCmdPrecedence)
		if err != nil {
			logger.Info(fmt.Sprintf("DB Query failed for %s with err %s\n", dbCmdPrecedence, err))
			return err
		}
		policy.StatementList = make([]*ribd.PolicyDefinitionStmtPriority, 0)
		var stmt, policyName string
		var precedence int32
		for conditionrows.Next() {
			if err = conditionrows.Scan(&policyName, &precedence, &stmt); err != nil {
				logger.Info(fmt.Sprintf("DB Scan failed when iterating over PolicyDefinitionConfigStatementList rows with error %s\n", err))
				return err
			}
			if policyName != policy.Name {
				logger.Info(fmt.Sprintln("Not a stmt for this policy, policyName: ", policyName))
				continue
			}
			logger.Info(fmt.Sprintln("Fetching stmt ", stmt, "Priority:", precedence))
			policyStmtPrecedence := ribd.PolicyDefinitionStmtPriority{Priority: int32(precedence), Statement: stmt}
			policy.StatementList = append(policy.StatementList, &policyStmtPrecedence)
		}

		err = ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(&policy,PolicyEngineDB)
		if err != nil {
			logger.Info(fmt.Sprintf("policy create failed with err %s\n", err))
			return err
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdatePolicyObjectsFromDB() { //(paramsDir string) (err error) {
	logger.Info(fmt.Sprintln("UpdateFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	ribdServiceHandler.UpdateGlobalPolicyConditionsFromDB(dbHdl) //paramsDir, dbHdl)
	ribdServiceHandler.UpdateGlobalPolicyStmtsFromDB(dbHdl)
	ribdServiceHandler.UpdateGlobalPolicyFromDB(dbHdl)
    //local route policies
	ribdServiceHandler.UpdateRoutePolicyConditionsFromDB(dbHdl) //paramsDir, dbHdl)
	ribdServiceHandler.UpdateRoutePolicyActionsFromDB(dbHdl)    //paramsDir, dbHdl)
	ribdServiceHandler.UpdateRoutePolicyStmtsFromDB(dbHdl)
	ribdServiceHandler.UpdateRoutePolicyFromDB(dbHdl)
	return
}
