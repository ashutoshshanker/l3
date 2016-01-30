// ribDB.go
package main

import (
	"ribd"
	"strconv"
	"utils/commonDefs"
	"l3/rib/ribdCommonDefs"
//    "utils/dbutils"
	_ "github.com/mattn/go-sqlite3"
    "database/sql"
)

func UpdateRoutesFromDB(paramsDir string, dbHdl *sql.DB) (err error) {
    logger.Println("UpdateRoutesFromDB")
    dbCmd := "select * from IPV4Route"
	rows, err := dbHdl.Query(dbCmd)
	if(err != nil) {
		logger.Printf("DB Query failed for %s with err %s\n", dbCmd, err)
		return err
	}
	var ipRoute IPRoute
	for rows.Next() {
		if err = rows.Scan(&ipRoute.DestinationNw, &ipRoute.NetworkMask,&ipRoute.Cost, &ipRoute.NextHopIp, &ipRoute.OutgoingIntfType, &ipRoute.OutgoingInterface, &ipRoute.Protocol); err != nil {
			logger.Printf("DB Scan failed when iterating over IPV4Route rows with error %s\n", err)
			return err
		}
		outIntf, _ := strconv.Atoi(ipRoute.OutgoingInterface)
		var outIntfType ribd.Int
		if ipRoute.OutgoingIntfType == "VLAN" {
			outIntfType = commonDefs.L2RefTypeVlan
		} else {
			outIntfType = commonDefs.L2RefTypePort
		}
		proto, _ := strconv.Atoi(ipRoute.Protocol)
		_,err = createV4Route(ipRoute.DestinationNw, ipRoute.NetworkMask, ribd.Int(ipRoute.Cost), ipRoute.NextHopIp, outIntfType,ribd.Int(outIntf), ribd.Int(proto),  FIBAndRIB,ribdCommonDefs.RoutePolicyStateChangetoValid,ribd.Int(len(destNetSlice)))
		if(err != nil) {
			logger.Printf("Route create failed with err %s\n", err)
			return err
		}
	}
	return err
}

func UpdatePolicyConditionsFromDB(paramsDir string, dbHdl *sql.DB) (err error) {
      logger.Println("UpdatePolicyConditionsFromDB")
    dbCmd := "select * from PolicyDefinitionStmtMatchProtocolCondition"
	rows, err := dbHdl.Query(dbCmd)
	if(err != nil) {
		logger.Printf("DB Query failed for %s with err %s\n", dbCmd, err)
		return err
	}
	var condition ribd.PolicyDefinitionStmtMatchProtocolCondition
	for rows.Next() {
		if err = rows.Scan(&condition.Name, &condition.InstallProtocolEq); err != nil {
			logger.Printf("DB Scan failed when iterating over PolicyDefinitionStmtMatchProtocolCondition rows with error %s\n", err)
			return err
		}
		_,err = routeServiceHandler.CreatePolicyDefinitionStmtMatchProtocolCondition(&condition)
		if(err != nil) {
			logger.Printf("Condition create failed with err %s\n", err)
			return err
		}
	}
	return err
}
func UpdateFromDB(paramsDir string) (err error) {
      logger.Println("UpdateFromDB")
	  DbName := paramsDir + "/UsrConfDb.db"
      logger.Println("DB Location: ", DbName)
      dbHdl, err := sql.Open("sqlite3", DbName)
      if err != nil {
        logger.Println("Failed to create the handle with err ", err)
        return err
      }

    if err = dbHdl.Ping(); err != nil {
        logger.Println("Failed to keep DB connection alive")
        return err
    }
	UpdateRoutesFromDB(paramsDir, dbHdl)
	UpdatePolicyConditionsFromDB(paramsDir, dbHdl)
	return err
}

