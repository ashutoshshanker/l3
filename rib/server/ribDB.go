//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

// ribDB.go
package server

import (
	"fmt"
	"models"
	"ribd"
	"utils/dbutils"
)

func (ribdServiceHandler *RIBDServer) UpdateRoutesFromDB() (err error) {
	logger.Debug(fmt.Sprintln("UpdateRoutesFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	if dbHdl != nil {
		var dbObjCfg models.IPv4Route
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
			logger.Debug(fmt.Sprintln("Number of routes from DB: ", len((objList))))
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewIPv4Route()
				dbObj := objList[idx].(models.IPv4Route)
				models.ConvertribdIPv4RouteObjToThrift(&dbObj, obj)
				err = ribdServiceHandler.RouteConfigValidationCheck(obj, "add")
				if err != nil {
					logger.Err("Route validation failed when reading from db")
					continue
				}
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

func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyConditionsFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Debug(fmt.Sprintln("UpdateGlobalPolicyConditionsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyCondition
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyCondition()
				dbObj := objList[idx].(models.PolicyCondition)
				models.ConvertribdPolicyConditionObjToThrift(&dbObj, obj)
	             ribdServiceHandler.PolicyConditionConfCh <-RIBdServerConfig {
	                                   OrigConfigObject:obj,
	                                   Op : "add",
	                              }
			}
		} else {
			logger.Err("DB Query failed during PolicyCondition query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyStmtsFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Debug(fmt.Sprintln("UpdateGlobalPolicyStmtsFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyStmt
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyStmt()
				dbObj := objList[idx].(models.PolicyStmt)
				models.ConvertribdPolicyStmtObjToThrift(&dbObj, obj)
	            ribdServiceHandler.PolicyStmtConfCh <- RIBdServerConfig {
	                                   OrigConfigObject:obj,
	                                   Op : "add",
	                              }
			}
		} else {
			logger.Err("DB Query failed during PolicyStmt query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdateGlobalPolicyFromDB(dbHdl *dbutils.DBUtil) (err error) {
	logger.Debug(fmt.Sprintln("UpdateGlobalPolicyFromDB"))
	if dbHdl != nil {
		var dbObjCfg models.PolicyDefinition
		objList, err := dbHdl.GetAllObjFromDb(dbObjCfg)
		if err == nil {
			for idx := 0; idx < len(objList); idx++ {
				obj := ribd.NewPolicyDefinition()
				dbObj := objList[idx].(models.PolicyDefinition)
				models.ConvertribdPolicyDefinitionObjToThrift(&dbObj, obj)
	             ribdServiceHandler.PolicyDefinitionConfCh <- RIBdServerConfig{
	                                   OrigConfigObject:obj,
	                                   Op : "add",
	                              }
			}
		} else {
			logger.Err("DB Query failed during PolicyDefinition query: RIBd init")
		}
	}
	return err
}
func (ribdServiceHandler *RIBDServer) UpdatePolicyObjectsFromDB() { //(paramsDir string) (err error) {
	logger.Debug(fmt.Sprintln("UpdateFromDB"))
	dbHdl := ribdServiceHandler.DbHdl
	ribdServiceHandler.UpdateGlobalPolicyConditionsFromDB(dbHdl) //paramsDir, dbHdl)
	ribdServiceHandler.UpdateGlobalPolicyStmtsFromDB(dbHdl)
	ribdServiceHandler.UpdateGlobalPolicyFromDB(dbHdl)
	return
}
