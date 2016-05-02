// ribDB.go
package server

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"models"
	"ribd"
)

func (ribdServiceHandler *RIBDServer) UpdateRoutesFromDB() (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutesFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	if dbHdl != nil {
		var dbObjCfg models.IPv4Route
		objList, err := dbObjCfg.GetAllObjFromDb(dbHdl)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewIPv4Route()
				dbObj := objList[idx].(models.IPv4Route)
				models.ConvertribdIPv4RouteObjToThrift(&dbObj, obj)
				rv, _ := ribdServiceHandler.ProcessRouteCreateConfig(obj)
				if rv == false {
					logger.Err("IPv4Route create failed during init")
				}
			}
		} else {
			logger.Err("DB Query failed during IPv4Route query: RIBd init")
		}
	}
	return err
}

func (ribdServiceHandler *RIBDServer) UpdatePolicyConditionsFromDB(dbHdl redis.Conn) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyConditionsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyCondition
		objList, err := dbObjCfg.GetAllObjFromDb(dbHdl)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyCondition()
				dbObj := objList[idx].(models.PolicyCondition)
				models.ConvertribdPolicyConditionObjToThrift(&dbObj, obj)
				rv, _ := ribdServiceHandler.ProcessPolicyConditionConfigCreate(obj)
				if rv == false {
					logger.Err("PolicyCondition create failed during init")
				}
			}
		} else {
			logger.Err("DB Query failed during PolicyCondition query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdatePolicyActionsFromDB(dbHdl redis.Conn) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyActionsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyAction
		objList, err := dbObjCfg.GetAllObjFromDb(dbHdl)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyAction()
				dbObj := objList[idx].(models.PolicyAction)
				models.ConvertribdPolicyActionObjToThrift(&dbObj, obj)
				rv, _ := ribdServiceHandler.ProcessPolicyActionConfigCreate(obj)
				if rv == false {
					logger.Err("PolicyAction create failed during init")
				}
			}
		} else {
			logger.Err("DB Query failed during PolicyAction query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdatePolicyStmtsFromDB(dbHdl redis.Conn) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyStmtsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyStmt
		objList, err := dbObjCfg.GetAllObjFromDb(dbHdl)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyStmt()
				dbObj := objList[idx].(models.PolicyStmt)
				models.ConvertribdPolicyStmtObjToThrift(&dbObj, obj)
				err = ribdServiceHandler.ProcessPolicyStmtConfigCreate(obj)
				if err != nil {
					logger.Err("PolicStmt create failed during init")
				}
			}
		} else {
			logger.Err("DB Query failed during PolicyStmt query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdatePolicyFromDB(dbHdl redis.Conn) (err error) {
	logger.Info(fmt.Sprintln("UpdatePolicyFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyDefinition
		objList, err := dbObjCfg.GetAllObjFromDb(dbHdl)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyDefinition()
				dbObj := objList[idx].(models.PolicyDefinition)
				models.ConvertribdPolicyDefinitionObjToThrift(&dbObj, obj)
				err = ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(obj)
				if err != nil {
					logger.Err("PolicyDefinition create failed during init")
				}
			}
		} else {
			logger.Err("DB Query failed during PolicyDefinition query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdatePolicyObjectsFromDB() { //(paramsDir string) (err error) {
	logger.Info(fmt.Sprintln("UpdateFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	ribdServiceHandler.UpdatePolicyConditionsFromDB(dbHdl) //paramsDir, dbHdl)
	ribdServiceHandler.UpdatePolicyActionsFromDB(dbHdl)    //paramsDir, dbHdl)
	ribdServiceHandler.UpdatePolicyStmtsFromDB(dbHdl)
	ribdServiceHandler.UpdatePolicyFromDB(dbHdl)
	return
}
