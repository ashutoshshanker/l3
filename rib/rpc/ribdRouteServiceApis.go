// ribdRouteServiceApis.go
package rpc

import (
	"errors"
	"fmt"
	"reflect"
	"ribd"
	"ribdInt"
	"l3/rib/server"
)

func (m RIBDServicesHandler) CreateIPv4Route(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Info(fmt.Sprintln("Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.OutgoingIntfType: ", cfg.OutgoingIntfType, "cfg.OutgoingInterface: ", cfg.OutgoingInterface))
    err = m.server.RouteConfigValidationCheck(cfg,"add")
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
	}
	m.server.RouteCreateConfCh <- cfg
	return true, nil
}
func (m RIBDServicesHandler) OnewayCreateIPv4Route(cfg *ribd.IPv4Route) (err error) {
	logger.Info(fmt.Sprintln("OnewayCreateIPv4Route - Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.OutgoingIntfType: ", cfg.OutgoingIntfType, "cfg.OutgoingInterface: ", cfg.OutgoingInterface))
	m.CreateIPv4Route(cfg)
	return err
}
func (m RIBDServicesHandler) OnewayCreateBulkIPv4Route(cfg []*ribdInt.IPv4Route) (err error) {
	//logger.Info(fmt.Sprintln("OnewayCreateIPv4Route - Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.OutgoingIntfType: ", cfg.OutgoingIntfType, "cfg.OutgoingInterface: ", cfg.OutgoingInterface))
     logger.Info(fmt.Sprintln("OnewayCreateBulkIPv4Route for ", len(cfg), " routes"))
    for i := 0;i<len(cfg);i++ {
		newCfg := ribd.IPv4Route {cfg[i].DestinationNw, cfg[i].NetworkMask, cfg[i].NextHopIp, cfg[i].Cost, cfg[i].OutgoingIntfType, cfg[i].OutgoingInterface, cfg[i].Protocol}
		m.CreateIPv4Route(&newCfg)
	}
	return err
}
func (m RIBDServicesHandler) DeleteIPv4Route(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Info(fmt.Sprintln("DeleteIPv4:RouteReceived Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "nextHopIP:", cfg.NextHopIp, "Protocol ", cfg.Protocol))
    err = m.server.RouteConfigValidationCheck(cfg,"del")
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
	}
	m.server.RouteDeleteConfCh <- cfg
	return true, nil
}
func (m RIBDServicesHandler) OnewayDeleteIPv4Route(cfg *ribd.IPv4Route) (err error) {
	logger.Info(fmt.Sprintln("OnewayDeleteIPv4Route:RouteReceived Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "nextHopIP:", cfg.NextHopIp, "Protocol ", cfg.Protocol))
	m.DeleteIPv4Route(cfg)
	return err
}
func (m RIBDServicesHandler) UpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (val bool, err error) {
	logger.Println("UpdateIPv4Route: Received update route request")
    err = m.server.RouteConfigValidationCheck(newconfig,"update")
	if err != nil {
		logger.Err(fmt.Sprintln("validation check failed with error ", err))
	}
	objTyp := reflect.TypeOf(*origconfig)
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		if objName == "OutgoingIntfType" {
			if newconfig.OutgoingIntfType == "NULL" {
				logger.Err("Cannot update the type to NULL interface: delete and create the route")
				return false, errors.New("Cannot update the type to NULL interface: delete and create the route")
			}
			if origconfig.OutgoingIntfType == "NULL" {
				logger.Err("Cannot update NULL interface type with another type: delete and create the route")
				return false, errors.New("Cannot update NULL interface type with another type: delete and create the route")
			}
			break
		}
	}
	routeUpdateConfig := server.UpdateRouteInfo{origconfig, newconfig, attrset}
	m.server.RouteUpdateConfCh <- routeUpdateConfig
	return true, nil
}
func (m RIBDServicesHandler) OnewayUpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (err error) {
	logger.Println("OneWayUpdateIPv4Route: Received update route request")
	m.UpdateIPv4Route(origconfig, newconfig, attrset)
	return err
}
func (m RIBDServicesHandler) GetIPv4RouteState(destNw string) (*ribd.IPv4RouteState, error) {
	logger.Info("Get state for IPv4Route")
	route := ribd.NewIPv4RouteState()
	return route, nil
}

func (m RIBDServicesHandler) GetBulkIPv4RouteState(fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.IPv4RouteStateGetInfo, err error) { 
    ret,err := m.server.GetBulkIPv4RouteState(fromIndex, rcount)
	return ret,err
}

func (m RIBDServicesHandler) GetIPv4EventState(index int32) (*ribd.IPv4EventState, error) {
	logger.Info("Get state for IPv4EventState")
	route := ribd.NewIPv4EventState()
	return route, nil
}

func (m RIBDServicesHandler) GetBulkIPv4EventState(fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.IPv4EventStateGetInfo, err error) { 
    ret,err := m.server.GetBulkIPv4EventState(fromIndex, rcount)
	return ret,err
}

func (m RIBDServicesHandler) GetBulkRoutesForProtocol(srcProtocol string, fromIndex ribdInt.Int, rcount ribdInt.Int) (routes *ribdInt.RoutesGetInfo, err error) {
	ret,err := m.server.GetBulkRoutesForProtocol(srcProtocol,fromIndex,rcount)
	return ret,err
}

func (m RIBDServicesHandler) GetBulkRouteDistanceState(fromIndex ribd.Int, rcount ribd.Int) (routeDistanceStates *ribd.RouteDistanceStateGetInfo, err error) {
	ret,err := m.server.GetBulkRouteDistanceState(fromIndex,rcount)
	return ret,err
}
func (m RIBDServicesHandler) GetRouteDistanceState(protocol string) (*ribd.RouteDistanceState, error) {
	logger.Info("Get state for RouteDistanceState")
	route := ribd.NewRouteDistanceState()
	return route, nil
}
func (m RIBDServicesHandler) GetNextHopIfTypeStr(nextHopIfType ribdInt.Int) (nextHopIfTypeStr string, err error) {
	nhStr,err := m.server.GetNextHopIfTypeStr(nextHopIfType)
	return nhStr,err
}
func (m RIBDServicesHandler) GetRoute(destNetIp string, networkMask string) (route *ribdInt.Routes, err error) {
	ret,err := m.server.GetRoute(destNetIp,networkMask)
	return ret,err
}
func (m RIBDServicesHandler) GetRouteReachabilityInfo(destNet string) (nextHopIntf *ribdInt.NextHopInfo, err error) {
	nh, err := m.server.GetRouteReachabilityInfo(destNet)
	return nh,err
}
func (m RIBDServicesHandler) TrackReachabilityStatus(ipAddr string, protocol string, op string) (err error) {
	m.server.TrackReachabilityCh <- server.TrackReachabilityInfo{ipAddr,protocol,op}
	return nil
}