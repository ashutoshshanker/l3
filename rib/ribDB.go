// ribDB.go
package main

import (
	"ribd"
	"strconv"
	"asicd/portd/portdCommonDefs"
//    "utils/dbutils"
	_ "github.com/mattn/go-sqlite3"
    "database/sql"
)

func UpdateRoutesFromDB(paramsDir string) (err error) {
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
			outIntfType = portdCommonDefs.VLAN
		} else {
			outIntfType = portdCommonDefs.PHY
		}
		proto, _ := strconv.Atoi(ipRoute.Protocol)
		_,err = createV4Route(ipRoute.DestinationNw, ipRoute.NetworkMask, ribd.Int(ipRoute.Cost), ipRoute.NextHopIp, outIntfType,ribd.Int(outIntf), ribd.Int(proto),  FIBAndRIB,ribd.Int(len(destNetSlice)))
		if(err != nil) {
			logger.Printf("Route create failed with err %s\n", err)
			return err
		}
	}
	return err
}
