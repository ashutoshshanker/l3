// ribdRouteServiceApis.go
package main

import (
	"asicd/asicdConstDefs"
	"errors"
	"fmt"
	"reflect"
	"ribd"
	"ribdInt"
	"strconv"
	"utils/commonDefs"
	"strings"
	"net"
)

func (m RIBDServicesHandler) CreateIPv4Route(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Info(fmt.Sprintln("Received create route request for ip", cfg.DestinationNw, " mask ", cfg.NetworkMask, "cfg.OutgoingIntfType: ", cfg.OutgoingIntfType, "cfg.OutgoingInterface: ", cfg.OutgoingInterface))
	_, ok := RouteProtocolTypeMapDB[cfg.Protocol]
	if !ok {
		logger.Info(fmt.Sprintln("route type ", cfg.Protocol, " invalid"))
		err = errors.New("Invalid route protocol type")
		return false, err
	}
	var nextHopIfType int
	var nextHopIf int
	if cfg.OutgoingIntfType == "VLAN" {
		nextHopIfType = commonDefs.L2RefTypeVlan
	} else if cfg.OutgoingIntfType == "PHY" {
		nextHopIfType = commonDefs.L2RefTypePort
	} else if cfg.OutgoingIntfType == "NULL" {
		nextHopIfType = commonDefs.IfTypeNull
	} else if cfg.OutgoingIntfType == "Loopback" {
		nextHopIfType = commonDefs.IfTypeLoopback
	}
	nextHopIf, _ = strconv.Atoi(cfg.OutgoingInterface)
	ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(nextHopIf, nextHopIfType)
	logger.Info(fmt.Sprintln("IfId = ", ifId))
	_, ok = IntfIdNameMap[ifId]
	if !ok {
		logger.Err(fmt.Sprintln("Cannot create ip route on a unknown L3 interface"))
		return false, errors.New("Cannot create ip route on a unknown L3 interface")
	}
	_, err = getIP(cfg.NextHopIp)
	if err != nil {
		logger.Println("nextHopIpAddr invalid")
		return false,errors.New("Invalid next hop ip address")
	}
	isCidr := strings.Contains(cfg.DestinationNw, "/")
	if isCidr { //the given address is in CIDR format
	    ip, ipNet, err := net.ParseCIDR(cfg.DestinationNw)
	    if err != nil {    
		    logger.Err(fmt.Sprintln("Invalid Destination IP address"))
			return false, errors.New("Invalid Desitnation IP address")
	    }
	    _, err = getNetworkPrefixFromCIDR(cfg.DestinationNw)
	    if err != nil {
		    return false,errors.New("Invalid destination ip/network Mask")
	    }
		cfg.DestinationNw = ip.String() 
	    ipMask := make(net.IP, 4)
		copy(ipMask, ipNet.Mask)
		ipMaskStr := net.IP(ipMask).String()
		cfg.NetworkMask = ipMaskStr
	} else {
		//validate ip and mask string if not a CIDR address
	    destNetIpAddr, err := getIP(cfg.DestinationNw)
	    if err != nil {
		    logger.Println("destNetIpAddr invalid")
		    return false, errors.New("Invalid destination IP address")
	    }
	    networkMaskAddr,err := getIP(cfg.NetworkMask)
	    if err != nil {
		    logger.Println("networkMaskAddr invalid")
		    return false, errors.New("Invalid mask")
	    }
	    _, err = getPrefixLen(networkMaskAddr)
	    if err != nil {
		    return false, errors.New("Invalid networkMask")
	    }
	
	    _, err = getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	    if err != nil {
		    return false,errors.New("Invalid destination ip/network Mask")
	    }
	}
	m.RouteCreateConfCh <- cfg
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
	isCidr := strings.Contains(cfg.DestinationNw, "/")
	if isCidr { //the given address is in CIDR format
	    ip, ipNet, err := net.ParseCIDR(cfg.DestinationNw)
	    if err != nil {    
		    logger.Err(fmt.Sprintln("Invalid Destination IP address"))
			return false, errors.New("Invalid Desitnation IP address")
	    }
	    _, err = getNetworkPrefixFromCIDR(cfg.DestinationNw)
	    if err != nil {
		    return false,errors.New("Invalid destination ip/network Mask")
	    }
		cfg.DestinationNw = ip.String() 
	    ipMask := make(net.IP, 4)
		copy(ipMask, ipNet.Mask)
		ipMaskStr := net.IP(ipMask).String()
		cfg.NetworkMask = ipMaskStr
	}
	destNet, err := getNetowrkPrefixFromStrings(cfg.DestinationNw, cfg.NetworkMask)
	if err != nil {
		logger.Info(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return false, errors.New("Invalid destination ip address")
	}
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return false, err
	}
	m.RouteDeleteConfCh <- cfg
	return true, nil
}
func (m RIBDServicesHandler) OnewayDeleteIPv4Route(cfg *ribd.IPv4Route) (err error) {
	logger.Info(fmt.Sprintln("OnewayDeleteIPv4Route:RouteReceived Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "nextHopIP:", cfg.NextHopIp, "Protocol ", cfg.Protocol))
	m.DeleteIPv4Route(cfg)
	return err
}
func (m RIBDServicesHandler) UpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (val bool, err error) {
	logger.Println("UpdateIPv4Route: Received update route request")
	isCidr := strings.Contains(origconfig.DestinationNw, "/")
	if isCidr { //the given address is in CIDR format
	    ip, ipNet, err := net.ParseCIDR(origconfig.DestinationNw)
	    if err != nil {    
		    logger.Err(fmt.Sprintln("Invalid Destination IP address"))
			return false, errors.New("Invalid Desitnation IP address")
	    }
	    _, err = getNetworkPrefixFromCIDR(origconfig.DestinationNw)
	    if err != nil {
		    return false,errors.New("Invalid destination ip/network Mask")
	    }
		origconfig.DestinationNw = ip.String() 
	    ipMask := make(net.IP, 4)
		copy(ipMask, ipNet.Mask)
		ipMaskStr := net.IP(ipMask).String()
		origconfig.NetworkMask = ipMaskStr
	}
	destNet, err := getNetowrkPrefixFromStrings(origconfig.DestinationNw, origconfig.NetworkMask)
	if err != nil {
		logger.Info(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return false, errors.New("Invalid destination ip address")
	}
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return false, err
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Println("No route for destination network")
		return false, errors.New("No route for destination network")
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
	routeUpdateConfig := UpdateRouteInfo{origconfig, newconfig, attrset}
	m.RouteUpdateConfCh <- routeUpdateConfig
	return true, nil
}
func (m RIBDServicesHandler) OnewayUpdateIPv4Route(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (err error) {
	logger.Println("OneWayUpdateIPv4Route: Received update route request")
	m.UpdateIPv4Route(origconfig, newconfig, attrset)
	return err
}
func (m RIBDServicesHandler) GetIPv4RouteState(destNw string, nextHop string) (*ribd.IPv4RouteState, error) {
	logger.Info("Get state for IPv4Route")
	route := ribd.NewIPv4RouteState()
	return route, nil
}
