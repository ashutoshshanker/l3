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

// ribDBServer.go
package server

import (
	"errors"
	"fmt"
	"models"
	"ribd"
	"strconv"
)

type RouteDBInfo struct {
	entry     RouteInfoRecord
	routeList RouteInfoRecordList
}

func (m RIBDServer) WriteIPv4RouteStateEntryToDB(dbInfo RouteDBInfo) error {
	logger.Info(fmt.Sprintln("WriteIPv4RouteStateEntryToDB"))
	entry := dbInfo.entry
	routeList := dbInfo.routeList
	m.DelIPv4RouteStateEntryFromDB(dbInfo)
	var dbObj models.IPv4RouteState
	obj := ribd.NewIPv4RouteState()
	obj.DestinationNw = entry.networkAddr
	/*	obj.NextHopIp = entry.nextHopIp.String()
		nextHopIfTypeStr, _ := m.GetNextHopIfTypeStr(ribdInt.Int(entry.nextHopIfType))
		obj.OutgoingIntfType = nextHopIfTypeStr
		obj.OutgoingInterface = strconv.Itoa(int(entry.nextHopIfIndex))*/
	obj.Protocol = ReverseRouteProtoTypeMapDB[int(entry.protocol)]
	obj.NextHopList = make([]*ribd.NextHopInfo, 0)
	routeInfoList := routeList.routeInfoProtocolMap[routeList.selectedRouteProtocol]
	logger.Info(fmt.Sprintln("len of routeInfoList - ", len(routeInfoList), "selected route protocol = ", routeList.selectedRouteProtocol, " route Protocol: ", entry.protocol, " route nwAddr: ", entry.networkAddr))
	nextHopInfo := make([]ribd.NextHopInfo, len(routeInfoList))
	i := 0
	for sel := 0; sel < len(routeInfoList); sel++ {
		logger.Info(fmt.Sprintln("nextHop ", sel, " weight = ", routeInfoList[sel].weight, " ip ", routeInfoList[sel].nextHopIp, " intref ", routeInfoList[sel].nextHopIfIndex))
		nextHopInfo[i].NextHopIp = routeInfoList[sel].nextHopIp.String()
		nextHopInfo[i].NextHopIntRef = strconv.Itoa(int(routeInfoList[sel].nextHopIfIndex))
		nextHopInfo[i].Weight = int32(routeInfoList[sel].weight)
		obj.NextHopList = append(obj.NextHopList, &nextHopInfo[i])
		i++
	}
	obj.RouteCreatedTime = entry.routeCreatedTime
	obj.RouteUpdatedTime = entry.routeUpdatedTime
	obj.IsNetworkReachable = entry.resolvedNextHopIpIntf.IsReachable
	obj.PolicyList = make([]string, 0)
	routePolicyListInfo := ""
	if routeList.policyList != nil {
		for k := 0; k < len(routeList.policyList); k++ {
			routePolicyListInfo = "policy " + routeList.policyList[k] + "["
			policyRouteIndex := PolicyRouteIndex{destNetIP: entry.networkAddr, policy: routeList.policyList[k]}
			policyStmtMap, ok := PolicyEngineDB.PolicyEntityMap[policyRouteIndex]
			if !ok || policyStmtMap.PolicyStmtMap == nil {
				continue
			}
			routePolicyListInfo = routePolicyListInfo + " stmtlist[["
			for stmt, conditionsAndActionsList := range policyStmtMap.PolicyStmtMap {
				routePolicyListInfo = routePolicyListInfo + stmt + ":[conditions:"
				for c := 0; c < len(conditionsAndActionsList.ConditionList); c++ {
					routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.ConditionList[c].Name + ","
				}
				routePolicyListInfo = routePolicyListInfo + "],[actions:"
				for a := 0; a < len(conditionsAndActionsList.ActionList); a++ {
					routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.ActionList[a].Name + ","
				}
				routePolicyListInfo = routePolicyListInfo + "]]"
			}
			routePolicyListInfo = routePolicyListInfo + "]"
			obj.PolicyList = append(obj.PolicyList, routePolicyListInfo)
		}
	}
	models.ConvertThriftToribdIPv4RouteStateObj(obj, &dbObj)
	err := dbObj.StoreObjectInDb(m.DbHdl)
	if err != nil {
		logger.Err(fmt.Sprintln("Failed to store IPv4RouteState entry in DB, err - ", err))
		return errors.New(fmt.Sprintln("Failed to add IPv4RouteState db : ", entry))
	}
	logger.Info(fmt.Sprintln("returned successfully after write to DB for IPv4RouteState"))
	return nil
}

func (m RIBDServer) DelIPv4RouteStateEntryFromDB(dbInfo RouteDBInfo) error {
	logger.Info(fmt.Sprintln("DelIPv4RouteStateEntryFromDB"))
	entry := dbInfo.entry
	var dbObj models.IPv4RouteState
	obj := ribd.NewIPv4RouteState()
	obj.DestinationNw = entry.networkAddr
	models.ConvertThriftToribdIPv4RouteStateObj(obj, &dbObj)
	err := dbObj.DeleteObjectFromDb(m.DbHdl)
	if err != nil {
		return errors.New(fmt.Sprintln("Failed to delete IPv4RouteState from state db : ", entry))
	}
	return nil
}

func (ribdServiceHandler *RIBDServer) StartDBServer() {
	logger.Info("Starting the arpdserver loop")
	for {
		select {
		case info := <-ribdServiceHandler.DBRouteAddCh:
			logger.Info(" received message on DBRouteAddCh")
			ribdServiceHandler.WriteIPv4RouteStateEntryToDB(info)
		case info := <-ribdServiceHandler.DBRouteDelCh:
			logger.Info(" received message on DBRouteDelCh")
			ribdServiceHandler.DelIPv4RouteStateEntryFromDB(info)
		}
	}
}
