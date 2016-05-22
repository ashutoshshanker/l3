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

// ribdRouteServiceApis.go
package rpc

import (
	"fmt"
	"l3/rib/server"
	"ribd"
	"ribdInt"
)

/* Create route API
 */

func (m RIBDServicesHandler) CreateIPv4Route(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Info(fmt.Sprintln("Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.NextHopIntRef: ", cfg.NextHop[0].NextHopIntRef))
    /* Validate Route config parameters for "add" operation
    */
	err = m.server.RouteConfigValidationCheck(cfg, "add")
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
		return false, err
	}
	m.server.RouteConfCh <- server.RIBdServerConfig{
	                                   OrigConfigObject:cfg,
	                                   Op : "add",
	                              }
	return true, nil
}

/*
    OnewayCreate API for route
*/
func (m RIBDServicesHandler) OnewayCreateIPv4Route(cfg *ribd.IPv4Route) (err error) {
	logger.Info(fmt.Sprintln("OnewayCreateIPv4Route - Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.NextHopIntRef: ", cfg.NextHop[0].NextHopIntRef))
	m.CreateIPv4Route(cfg)
	return err
}

/*
    Create Routes in Bulk using Oneway create API
*/
func (m RIBDServicesHandler) OnewayCreateBulkIPv4Route(cfg []*ribdInt.IPv4Route) (err error) {
	//logger.Info(fmt.Sprintln("OnewayCreateIPv4Route - Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.OutgoingIntfType: ", cfg.OutgoingIntfType, "cfg.OutgoingInterface: ", cfg.OutgoingInterface))
	logger.Info(fmt.Sprintln("OnewayCreateBulkIPv4Route for ", len(cfg), " routes"))
	for i := 0; i < len(cfg); i++ {
		newCfg := ribd.IPv4Route{
			DestinationNw: cfg[i].DestinationNw,
			NetworkMask:   cfg[i].NetworkMask,
			Cost:          cfg[i].Cost,
			Protocol:      cfg[i].Protocol,
		}
		newCfg.NextHop = make([]*ribd.NextHopInfo, 0)
		nextHop := ribd.NextHopInfo{
			NextHopIp:     cfg[i].NextHopIp,
			NextHopIntRef: cfg[i].NextHopIntRef,
			Weight:        cfg[i].Weight,
		}
		newCfg.NextHop = append(newCfg.NextHop, &nextHop)
		m.CreateIPv4Route(&newCfg)
	}
	return err
}
/*
    Delete Route
*/
func (m RIBDServicesHandler) DeleteIPv4Route(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeleteIPv4:RouteReceived Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "Protocol ", cfg.Protocol, "number of nextHops: ", len(cfg.NextHop)))
	/*
	    Validate route config parameters for "del" operation
	*/
	err = m.server.RouteConfigValidationCheck(cfg, "del")
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
		return false, err
	}
	m.server.RouteConfCh <- server.RIBdServerConfig{
	                                   OrigConfigObject:cfg,
	                                   Op : "del",
	                              }
	return true, nil
}
/*
    Delete route using Oneway Api
*/
func (m RIBDServicesHandler) OnewayDeleteIPv4Route(cfg *ribd.IPv4Route) (err error) {
	logger.Info(fmt.Sprintln("OnewayDeleteIPv4Route:RouteReceived Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "nextHopIP:", cfg.NextHop[0].NextHopIp, "Protocol ", cfg.Protocol))
	m.DeleteIPv4Route(cfg)
	return err
}
/*
    Update route
*/
func (m RIBDServicesHandler) UpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool, op string) (val bool, err error) { //[]*ribd.PatchOpInfo) (val bool, err error) {
	logger.Println("UpdateIPv4Route: Received update route request")
	/*
	    validate route config parameters for update operation
	*/
/*	if op == nil {
	    err = m.server.RouteConfigValidationCheckForUpdate(origconfig, newconfig, attrset)
	    if err != nil {
		    logger.Err(fmt.Sprintln("validation check failed with error ", err))
		    return false, err
	    }
	    routeUpdateConfig := server.UpdateRouteInfo{origconfig, newconfig, attrset}
	    m.server.RouteUpdateConfCh <- routeUpdateConfig
        return true,nil
	}
	err = m.server.RouteConfigValidationCheckForPatchUpdate(origconfig, newconfig,op)
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
		return false, err
	}
	routePatchUpdateConfig := server.PatchUpdateRouteInfo{origconfig, newconfig, op}
	m.server.RoutePatchUpdateConfCh <- routePatchUpdateConfig
	*/
	err = m.server.RouteConfigValidationCheckForUpdate(origconfig, newconfig, attrset,op)
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
		return false, err
	}
	m.server.RouteConfCh <- server.RIBdServerConfig{
	                                   OrigConfigObject: origconfig,
									NewConfigObject : newconfig,
									AttrSet         : attrset,
	                                   Op              : "update",
									PatchOp         : op,
	                              }
	return true, nil
}

/*
    one way update route function
*/
func (m RIBDServicesHandler) OnewayUpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (err error) {
	logger.Println("OneWayUpdateIPv4Route: Received update route request")
	m.UpdateIPv4Route(origconfig, newconfig, attrset,"replace")
	return err
}
/*
    Applications call this function to fetch all the routes that need to be redistributed into them.
*/
func (m RIBDServicesHandler) GetBulkRoutesForProtocol(srcProtocol string, fromIndex ribdInt.Int, rcount ribdInt.Int) (routes *ribdInt.RoutesGetInfo, err error) {
	ret, err := m.server.GetBulkRoutesForProtocol(srcProtocol, fromIndex, rcount)
	return ret, err
}
/*
    Api to track a route's reachability status
*/
func (m RIBDServicesHandler) TrackReachabilityStatus(ipAddr string, protocol string, op string) (err error) {
	m.server.TrackReachabilityCh <- server.TrackReachabilityInfo{ipAddr, protocol, op}
	return nil
}

func (m RIBDServicesHandler) GetIPv4RouteState(destNw string) (*ribd.IPv4RouteState, error) {
	logger.Info("Get state for IPv4Route")
	route := ribd.NewIPv4RouteState()
	return route, nil
}

func (m RIBDServicesHandler) GetBulkIPv4RouteState(fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.IPv4RouteStateGetInfo, err error) {
	ret, err := m.server.GetBulkIPv4RouteState(fromIndex, rcount)
	return ret, err
}

func (m RIBDServicesHandler) GetIPv4EventState(index int32) (*ribd.IPv4EventState, error) {
	logger.Info("Get state for IPv4EventState")
	route := ribd.NewIPv4EventState()
	return route, nil
}

func (m RIBDServicesHandler) GetBulkIPv4EventState(fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.IPv4EventStateGetInfo, err error) {
	ret, err := m.server.GetBulkIPv4EventState(fromIndex, rcount)
	return ret, err
}

func (m RIBDServicesHandler) GetBulkRouteDistanceState(fromIndex ribd.Int, rcount ribd.Int) (routeDistanceStates *ribd.RouteDistanceStateGetInfo, err error) {
	ret, err := m.server.GetBulkRouteDistanceState(fromIndex, rcount)
	return ret, err
}
func (m RIBDServicesHandler) GetRouteDistanceState(protocol string) (*ribd.RouteDistanceState, error) {
	logger.Info("Get state for RouteDistanceState")
	route := ribd.NewRouteDistanceState()
	return route, nil
}
func (m RIBDServicesHandler) GetNextHopIfTypeStr(nextHopIfType ribdInt.Int) (nextHopIfTypeStr string, err error) {
	nhStr, err := m.server.GetNextHopIfTypeStr(nextHopIfType)
	return nhStr, err
}
func (m RIBDServicesHandler) GetRoute(destNetIp string, networkMask string) (route *ribdInt.Routes, err error) {
	ret, err := m.server.GetRoute(destNetIp, networkMask)
	return ret, err
}
func (m RIBDServicesHandler) GetRouteReachabilityInfo(destNet string) (nextHopIntf *ribdInt.NextHopInfo, err error) {
	nh, err := m.server.GetRouteReachabilityInfo(destNet)
	return nh, err
}
