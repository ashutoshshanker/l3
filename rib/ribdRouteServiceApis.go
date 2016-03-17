// ribdRouteServiceApis.go
package main

import (
	"fmt"
	"ribd"
	"errors"
	"reflect"
)

func (m RIBDServicesHandler) CreateIPv4Route(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Info(fmt.Sprintf("Received create route request for ip %s mask %s\n", cfg.DestinationNw, cfg.NetworkMask))
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return 0, err
	}
	_, ok := RouteProtocolTypeMapDB[cfg.Protocol]
	if !ok {
		logger.Info(fmt.Sprintln("route type ", cfg.Protocol, " invalid"))
		err = errors.New("Invalid route protocol type")
		return false, err
	}
    m.RouteCreateConfCh <- cfg
	return true,err	
}
func (m RIBDServicesHandler) DeleteIPv4Route(cfg *ribd.IPv4Route) (val bool, err error){
	logger.Info(fmt.Sprintln(":DeleteIPv4RouteReceived Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "nextHopIP:", cfg.NextHopIp, "Protocol ", cfg.Protocol))
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return 0,err
	}
	m.RouteDeleteConfCh <- cfg
	return true, err
}
func (m RIBDServicesHandler) UpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (val bool, err error) {
	logger.Println("UpdateIPv4Route: Received update route request")
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return err
	}
	destNet, err := getNetowrkPrefixFromStrings(origconfig.DestinationNw, origconfig.NetworkMask)
	if err != nil {
		logger.Info(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return val, err
	}
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return val, err
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Println("No route for destination network")
		return val, err
	}
	objTyp := reflect.TypeOf(*origconfig)
	for i:=0;i<objTyp.NumField(); i++ {
	    objName := objTyp.Field(i).Name
	    if objName == "OutgoingIntfType" {
            if newconfig.OutgoingIntfType == "NULL" {
		        logger.Err("Cannot update the type to NULL interface: delete and create the route")
			    return val,errors.New("Cannot update the type to NULL interface: delete and create the route")
		    }
            if origconfig.OutgoingIntfType == "NULL" {
			    logger.Err("Cannot update NULL interface type with another type: delete and create the route")
			    return val,errors.New("Cannot update NULL interface type with another type: delete and create the route")
		    }
	        break
	    }
	}
	routeUpdateConfig := UpdateRouteInfo{origconfig,newconfig,attrset}
	m.RouteUpdateConfCh <- routeUpdateConfig
	return val, err
}
