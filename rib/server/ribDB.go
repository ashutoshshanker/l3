// ribDB.go
package server

import (
	"fmt"
	"models"
	"ribd"
	"utils/dbutils"
)

func (ribdServiceHandler *RIBDServer) UpdateRoutesFromDB() (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutesFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	if dbHdl != nil {
		var dbObjCfg models.IPv4Route
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
		    logger.Info(fmt.Sprintln("Number of routes from DB: ", len((objList))))
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

func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyConditionsFromDB(ddbHdl *dbutils.DBUtil) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyConditionsFromDB"))
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyConditionsFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Info(fmt.Sprintln("UpdateGlobalPolicyConditionsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyCondition
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyCondition()
				dbObj := objList[idx].(models.PolicyCondition)
				models.ConvertribdPolicyConditionObjToThrift(&dbObj, obj)
	             ribdServiceHandler.PolicyConditionCreateConfCh <- obj
				/*rv, _ := ribdServiceHandler.ProcessPolicyConditionConfigCreate(obj,GlobalPolicyEngineDB)
				if rv == false {
					logger.Err("PolicyCondition create failed during init")
				}*/
			}
		} else {
			logger.Err("DB Query failed during PolicyCondition query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyStmtsFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyStmtsFromDB"))
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyStmtsFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Info(fmt.Sprintln("UpdateGlobalPolicyStmtsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyAction
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
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
func (ribdServiceHandler *RIBDServer) UpdateRoutePolicyFromDB(ddbHdl *dbutils.DBUtil) (err error) {
	logger.Info(fmt.Sprintln("UpdateRoutePolicyFromDB"))
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Info(fmt.Sprintln("UpdateGlobalPolicyFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyDefinition
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyDefinition()
				dbObj := objList[idx].(models.PolicyDefinition)
				models.ConvertribdPolicyDefinitionObjToThrift(&dbObj, obj)
				ribdServiceHandler.PolicyDefinitionCreateConfCh <- obj
				/*err = ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(obj,GlobalPolicyEngineDB)
				if err != nil {
					logger.Err("PolicyDefinition create failed during init")
				}*/
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
	ribdServiceHandler.UpdateGlobalPolicyConditionsFromDB(dbHdl) //paramsDir, dbHdl)
	ribdServiceHandler.UpdateGlobalPolicyStmtsFromDB(dbHdl)
	ribdServiceHandler.UpdateGlobalPolicyFromDB(dbHdl)
    //local route policies
	ribdServiceHandler.UpdateRoutePolicyConditionsFromDB(dbHdl) //paramsDir, dbHdl)
	ribdServiceHandler.UpdateRoutePolicyStmtsFromDB(dbHdl)
	ribdServiceHandler.UpdateRoutePolicyFromDB(dbHdl)
	return
}
