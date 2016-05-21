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

// ribdUtils.go
package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/op/go-nanomsg"
	"l3/rib/ribdCommonDefs"
	"models"
	"net"
	"reflect"
	"ribd"
	"ribdInt"
	"sort"
	"strconv"
	"strings"
	"utils/netUtils"
	"utils/patriciaDB"
	"utils/policy"
)

type RouteDistanceConfig struct {
	defaultDistance    int
	configuredDistance int
}
type AdminDistanceSlice []ribd.RouteDistanceState
type RedistributeRouteInfo struct {
	route ribdInt.Routes
}
type RedistributionPolicyInfo struct {
	policy     string
	policyStmt string
}
type PublisherMapInfo struct {
	pub_ipc    string
	pub_socket *nanomsg.PubSocket
}

var RedistributeRouteMap map[string][]RedistributeRouteInfo
var RedistributionPolicyMap map[string]RedistributionPolicyInfo
var TrackReachabilityMap map[string][]string //map[ipAddr][]protocols
var RouteProtocolTypeMapDB map[string]int
var ReverseRouteProtoTypeMapDB map[int]string
var ProtocolAdminDistanceMapDB map[string]RouteDistanceConfig
var ProtocolAdminDistanceSlice AdminDistanceSlice
var PublisherInfoMap map[string]PublisherMapInfo
var RIBD_PUB *nanomsg.PubSocket
var RIBD_POLICY_PUB *nanomsg.PubSocket

func InitPublisher(pub_str string) (pub *nanomsg.PubSocket) {
	logger.Info(fmt.Sprintln("Setting up %s", pub_str, "publisher"))
	pub, err := nanomsg.NewPubSocket()
	if err != nil {
		logger.Println("Failed to open pub socket")
		return nil
	}
	ep, err := pub.Bind(pub_str)
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to bind pub socket - ", ep))
		return nil
	}
	err = pub.SetSendBuffer(1024 * 1024)
	if err != nil {
		logger.Println("Failed to set send buffer size")
		return nil
	}
	return pub
}

func BuildPublisherMap() {
	RIBD_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_ADDR)
	RIBD_POLICY_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_POLICY_ADDR)
	for k, _ := range RouteProtocolTypeMapDB {
		logger.Info(fmt.Sprintln("Building publisher map for protocol ", k))
		if k == "CONNECTED" || k == "STATIC" {
			logger.Info(fmt.Sprintln("Publisher info for protocol ", k, " not required"))
			continue
		}
		if k == "IBGP" || k == "EBGP" {
			continue
		}
		pub_ipc := "ipc:///tmp/ribd_" + strings.ToLower(k) + "d.ipc"
		logger.Info(fmt.Sprintln("pub_ipc:", pub_ipc))
		pub := InitPublisher(pub_ipc)
		PublisherInfoMap[k] = PublisherMapInfo{pub_ipc, pub}
	}
	PublisherInfoMap["EBGP"] = PublisherInfoMap["BGP"]
	PublisherInfoMap["IBGP"] = PublisherInfoMap["BGP"]
	PublisherInfoMap["BFD"] = PublisherMapInfo{ribdCommonDefs.PUB_SOCKET_BFDD_ADDR, InitPublisher(ribdCommonDefs.PUB_SOCKET_BFDD_ADDR)}
}
func BuildRouteProtocolTypeMapDB() {
	RouteProtocolTypeMapDB["CONNECTED"] = ribdCommonDefs.CONNECTED
	RouteProtocolTypeMapDB["EBGP"] = ribdCommonDefs.EBGP
	RouteProtocolTypeMapDB["IBGP"] = ribdCommonDefs.IBGP
	RouteProtocolTypeMapDB["BGP"] = ribdCommonDefs.BGP
	RouteProtocolTypeMapDB["OSPF"] = ribdCommonDefs.OSPF
	RouteProtocolTypeMapDB["STATIC"] = ribdCommonDefs.STATIC

	//reverse
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.CONNECTED] = "CONNECTED"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.IBGP] = "IBGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.EBGP] = "EBGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.BGP] = "BGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.STATIC] = "STATIC"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.OSPF] = "OSPF"
}
func BuildProtocolAdminDistanceMapDB() {
	ProtocolAdminDistanceMapDB["CONNECTED"] = RouteDistanceConfig{defaultDistance: 0, configuredDistance: -1}
	ProtocolAdminDistanceMapDB["STATIC"] = RouteDistanceConfig{defaultDistance: 1, configuredDistance: -1}
	ProtocolAdminDistanceMapDB["EBGP"] = RouteDistanceConfig{defaultDistance: 20, configuredDistance: -1}
	ProtocolAdminDistanceMapDB["IBGP"] = RouteDistanceConfig{defaultDistance: 200, configuredDistance: -1}
	ProtocolAdminDistanceMapDB["OSPF"] = RouteDistanceConfig{defaultDistance: 110, configuredDistance: -1}
}
func (slice AdminDistanceSlice) Len() int {
	return len(slice)
}
func (slice AdminDistanceSlice) Less(i, j int) bool {
	return slice[i].Distance < slice[j].Distance
}
func (slice AdminDistanceSlice) Swap(i, j int) {
	slice[i].Protocol, slice[j].Protocol = slice[j].Protocol, slice[i].Protocol
	slice[i].Distance, slice[j].Distance = slice[j].Distance, slice[i].Distance
}
func BuildProtocolAdminDistanceSlice() {
	distance := 0
	protocol := ""
	ProtocolAdminDistanceSlice = nil
	ProtocolAdminDistanceSlice = make([]ribd.RouteDistanceState, 0)
	for k, v := range ProtocolAdminDistanceMapDB {
		protocol = k
		distance = v.defaultDistance
		if v.configuredDistance != -1 {
			distance = v.configuredDistance
		}
		routeDistance := ribd.RouteDistanceState{Protocol: protocol, Distance: int32(distance)}
		ProtocolAdminDistanceSlice = append(ProtocolAdminDistanceSlice, routeDistance)
	}
	sort.Sort(ProtocolAdminDistanceSlice)
}
func (m RIBDServer) ConvertIntfStrToIfIndexStr(intfString string) (ifIndex string, err error) {
	if val, err := strconv.Atoi(intfString); err == nil {
		//Verify ifIndex is valid
		logger.Info(fmt.Sprintln("IfIndex = ", val))
		_, ok := IntfIdNameMap[int32(val)]
		if !ok {
			logger.Err(fmt.Sprintln("Cannot create ip route on a unknown L3 interface"))
			return ifIndex, errors.New("Cannot create ip route on a unknown L3 interface")
		}
		ifIndex = intfString
	} else {
		//Verify ifName is valid
		if _, ok := IfNameToIfIndex[intfString]; !ok {
			return ifIndex, errors.New("Invalid ifName value")
		}
		ifIndex = strconv.Itoa(int(IfNameToIfIndex[intfString]))
	}
	return ifIndex, nil
}
/*
    This function performs config parameters validation for Route update operation.
	Key validations performed by this fucntion include:
	   - Validate destinationNw. If provided in CIDR notation, convert to ip addr and mask values
	   - Check if the route is present in the DB
*/
func (m RIBDServer) RouteConfigValidationCheckForUpdate(oldcfg *ribd.IPv4Route, cfg *ribd.IPv4Route, attrset []bool, op string) (err error) {
	logger.Info(fmt.Sprintln("RouteConfigValidationCheckForUpdate"))
	isCidr := strings.Contains(cfg.DestinationNw, "/")
	if isCidr { 
	    /*
		    the given address is in CIDR format
		*/
		ip, ipNet, err := net.ParseCIDR(cfg.DestinationNw)
		if err != nil {
			logger.Err(fmt.Sprintln("Invalid Destination IP address"))
			return errors.New("Invalid Desitnation IP address")
		}
		_, err = getNetworkPrefixFromCIDR(cfg.DestinationNw)
		if err != nil {
			return errors.New("Invalid destination ip/network Mask")
		}
		cfg.DestinationNw = ip.String()
		ipMask := make(net.IP, 4)
		copy(ipMask, ipNet.Mask)
		ipMaskStr := net.IP(ipMask).String()
		cfg.NetworkMask = ipMaskStr
	}
	destNet, err := validateNetworkPrefix(cfg.DestinationNw, cfg.NetworkMask)
	if err != nil {
		logger.Info(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return errors.New("Invalid destination ip address")
	}
	/*
	    Check if the route being updated is present in RIB DB
	*/
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return err
	}
	if op == "add" {
		/*
		   This is a update add operation. 
		   "add" option is set for an update call when the user wants to add a new value
		   instead of modifying existing ones. 
		*/
		logger.Debug(fmt.Sprintln("Add operation in update"))
		if attrset != nil {
			logger.Debug("attr set not nil, set individual attributes")
			objTyp := reflect.TypeOf(*cfg)
			for i := 0; i < objTyp.NumField(); i++ {
				objName := objTyp.Field(i).Name
				if attrset[i] {
					/*
					    Currently, we can only add next hop info via route update
					*/
					if objName != "NextHop" {
						logger.Err(fmt.Sprintln("Cannot add any other object ", objName, " other than next hop"))
						return errors.New("Cannot add any other object other than next hop")
					}
					if len(cfg.NextHop) == 0 {
						/*
						   If route update is trying to add next hop, non zero nextHop info is expected
						*/
						logger.Err("Must specify next hop")
						return errors.New("Next hop ip not specified")
					}
					for i :=0 ;i<len(cfg.NextHop);i++ {
					    /*
					        Check if the next hop ip valid
					    */
					    _, err = getIP(cfg.NextHop[i].NextHopIp)
					    if err != nil {
						    logger.Err(fmt.Sprintln("nextHopIpAddr invalid"))
						    return errors.New("Invalid next hop ip address")
					    }
						/*
						    Check if the next hop ref is valid L3 interface
						*/
					    logger.Debug(fmt.Sprintln("IntRef before : ", cfg.NextHop[i].NextHopIntRef))
					    cfg.NextHop[i].NextHopIntRef, err = m.ConvertIntfStrToIfIndexStr(cfg.NextHop[i].NextHopIntRef)
					    if err != nil {
						    logger.Err(fmt.Sprintln("Invalid NextHop IntRef ", cfg.NextHop[i].NextHopIntRef))
						    return errors.New("Invalid NextHop Intref")
					    }
					    logger.Debug(fmt.Sprintln("IntRef after : ", cfg.NextHop[i].NextHopIntRef))
					}
				}
			}
		}
		return err 
	} //end of update add operation
	
	if op == "remove" {
		/*
		   This is a update remove operation. 
		   "remove" option is set for an update call when the user wants to remove a new value
		   instead of modifying existing ones. 
		*/
		logger.Debug(fmt.Sprintln("remove operation in update"))
		if attrset != nil {
			logger.Debug("attr set not nil, set individual attributes")
			objTyp := reflect.TypeOf(*cfg)
			for i := 0; i < objTyp.NumField(); i++ {
				objName := objTyp.Field(i).Name
				if attrset[i] {
					/*
					    Currently, we can only add next hop info via route update
					*/
					if objName != "NextHop" {
						logger.Err(fmt.Sprintln("Cannot remove any other object ", objName, " other than next hop"))
						return errors.New("Cannot remove any other object other than next hop")
					}
					if len(cfg.NextHop) == 0 {
						/*
						   If route update is trying to remove next hop, non zero nextHop info is expected
						*/
						logger.Err("Must specify next hop")
						return errors.New("Next hop ip not specified")
					}
					for i :=0 ;i<len(cfg.NextHop);i++ {
					    /*
					        Check if the next hop ip valid
					    */
					    _, err = getIP(cfg.NextHop[i].NextHopIp)
					    if err != nil {
						    logger.Err(fmt.Sprintln("nextHopIpAddr invalid"))
						    return errors.New("Invalid next hop ip address")
					    }
					}
				}
			}
		}
		return err 
	} //end of update remove operation
	
	/*
	    Default operation for update function is to update route Info. The following 
		logic deals with updating route attributes
	*/
	if attrset != nil {
		logger.Debug("attr set not nil, set individual attributes")
		objTyp := reflect.TypeOf(*cfg)
		for i := 0; i < objTyp.NumField(); i++ {
			objName := objTyp.Field(i).Name
			if attrset[i] {
				logger.Debug(fmt.Sprintf("ProcessRouteUpdateConfig (server): changed ", objName))
				if objName == "Protocol" {
					/*
					    Updating route protocol type is not allowed
					*/
					logger.Err("Cannot update Protocol value of a route")
					return errors.New("Cannot set Protocol field")
				}
				if objName == "NextHop" {
					/*
					   Next hop info is being updated
					*/
					if len(cfg.NextHop) == 0 {
						/*
						   Expects non-zero nexthop info
						*/
						logger.Err("Must specify next hop")
						return errors.New("Next hop ip not specified")
					}
					/*
					    Check if next hop IP is valid
					*/
					for i:=0;i<len(cfg.NextHop);i++ {
					    _, err = getIP(cfg.NextHop[i].NextHopIp)
					    if err != nil {
						    logger.Err(fmt.Sprintln("nextHopIpAddr invalid"))
						    return errors.New("Invalid next hop ip address")
					    }
					    /*
					        Check if next hop intf is valid L3 interface
					    */
					    if cfg.NextHop[i].NextHopIntRef != "" {
					        logger.Debug(fmt.Sprintln("IntRef before : ", cfg.NextHop[i].NextHopIntRef))
					        cfg.NextHop[i].NextHopIntRef, err = m.ConvertIntfStrToIfIndexStr(cfg.NextHop[i].NextHopIntRef)
					        if err != nil {
						        logger.Err(fmt.Sprintln("Invalid NextHop IntRef ", cfg.NextHop[i].NextHopIntRef))
						        return errors.New("Invalid Nexthop Intref")
					        }
					        logger.Debug(fmt.Sprintln("IntRef after : ", cfg.NextHop[0].NextHopIntRef))
						} else {
							if len(oldcfg.NextHop) == 0 || len(oldcfg.NextHop) < i {
								logger.Err("Number of nextHops for old cfg < new cfg")
								return errors.New("number of nexthops not correct for update replace operation")
							}
					        logger.Debug(fmt.Sprintln("IntRef not provided, take the old value",oldcfg.NextHop[i].NextHopIntRef))
					        cfg.NextHop[i].NextHopIntRef, err = m.ConvertIntfStrToIfIndexStr(oldcfg.NextHop[i].NextHopIntRef)
					        if err != nil {
						        logger.Err(fmt.Sprintln("Invalid NextHop IntRef ", oldcfg.NextHop[i].NextHopIntRef))
						        return errors.New("Invalid Nexthop Intref")
					        }
						}
					}
				}
			}
		}
	}
	return nil
}

/*
    This function performs config parameters validation for op = "add" and "del" values.
	Key validations performed by this fucntion include:
	   - if the Protocol specified is valid (STATIC/CONNECTED/EBGP/OSPF)
	   - Validate destinationNw. If provided in CIDR notation, convert to ip addr and mask values
	   - In case of op == "del", check if the route is present in the DB
	   - for each of the nextHop info, check:
	       - if the next hop ip is valid 
		   - if the nexthopIntf is valid L3 intf and if so, convert to string value
*/
func (m RIBDServer) RouteConfigValidationCheck(cfg *ribd.IPv4Route, op string) (err error) {
	logger.Debug(fmt.Sprintln("RouteConfigValidationCheck"))
	isCidr := strings.Contains(cfg.DestinationNw, "/")
	if isCidr { 
	    /*
		    the given address is in CIDR format
		*/
		ip, ipNet, err := net.ParseCIDR(cfg.DestinationNw)
		if err != nil {
			logger.Err(fmt.Sprintln("Invalid Destination IP address"))
			return errors.New("Invalid Desitnation IP address")
		}
		_, err = getNetworkPrefixFromCIDR(cfg.DestinationNw)
		if err != nil {
			return errors.New("Invalid destination ip/network Mask")
		}
		/*
		    Convert the CIDR format address to IP and mask strings
		*/
		cfg.DestinationNw = ip.String()
		ipMask := make(net.IP, 4)
		copy(ipMask, ipNet.Mask)
		ipMaskStr := net.IP(ipMask).String()
		cfg.NetworkMask = ipMaskStr
		/*
			In case where user provides CIDR address, the DB cannot verify if the route is present, so check here
		*/
		if m.DbHdl != nil {
			var dbObjCfg models.IPv4Route
			dbObjCfg.DestinationNw = cfg.DestinationNw
			dbObjCfg.NetworkMask = cfg.NetworkMask
			key := "IPv4Route#" + cfg.DestinationNw + "#" + cfg.NetworkMask
			_, err := m.DbHdl.GetObjectFromDb(dbObjCfg, key)
			if err == nil {
				logger.Err("Duplicate entry")
				return errors.New("Duplicate entry")
			}
		}
	}
	destNet, err := validateNetworkPrefix(cfg.DestinationNw, cfg.NetworkMask)
	if err != nil {
		logger.Info(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return err
	}
	/*
	    Check if route present.
	*/
    routeInfoRecordItem := RouteInfoMap.Get(destNet) 
	if routeInfoRecordItem == nil && op == "del"{
	/*
	    If delete operation, err if no route found
	*/
        err = errors.New("No route found")
        return err
    }
	/*
	    op is to add new route
	*/
	if op == "add" {
		/*
		    check if route protocol type is valid
		*/
		_, ok := RouteProtocolTypeMapDB[cfg.Protocol]
		if !ok {
			logger.Err(fmt.Sprintln("route type ", cfg.Protocol, " invalid"))
			err = errors.New("Invalid route protocol type")
			return err
		}
		logger.Debug(fmt.Sprintln("Number of nexthops = ", len(cfg.NextHop)))
		for i := 0; i < len(cfg.NextHop); i++ {
			/*
			    Check if the NextHop IP valid
			*/
			_, err = getIP(cfg.NextHop[i].NextHopIp)
			if err != nil {
				logger.Err(fmt.Sprintln("nextHopIpAddr invalid"))
				return errors.New("Invalid next hop ip address")
			}
			logger.Debug(fmt.Sprintln("IntRef before : ", cfg.NextHop[i].NextHopIntRef))
			/*
			   Validate if nextHopIntRef is a valid L3 interface
			*/
			if cfg.NextHop[i].NextHopIntRef == "" {
				logger.Info(fmt.Sprintln("NextHopIntRef not set"))
				nhIntf,err := RouteServiceHandler.GetRouteReachabilityInfo(cfg.NextHop[i].NextHopIp)
				if err != nil {
					logger.Err(fmt.Sprintln("next hop ip ", cfg.NextHop[i].NextHopIp, " not reachable"))
					return errors.New(fmt.Sprintln("next hop ip ", cfg.NextHop[i].NextHopIp, " not reachable"))
				}
				cfg.NextHop[i].NextHopIntRef = strconv.Itoa(int(nhIntf.NextHopIfIndex))
			} else {
			    cfg.NextHop[i].NextHopIntRef, err = m.ConvertIntfStrToIfIndexStr(cfg.NextHop[i].NextHopIntRef)
			    if err != nil {
				    logger.Err(fmt.Sprintln("Invalid NextHop IntRef ", cfg.NextHop[i].NextHopIntRef))
				    return err
			    }
			}
			logger.Debug(fmt.Sprintln("IntRef after : ", cfg.NextHop[i].NextHopIntRef))
		}
	}
	return nil
}
func arpResolveCalled(key NextHopInfoKey) bool {
	if RouteServiceHandler.NextHopInfoMap == nil {
		return false
	}
	info, ok := RouteServiceHandler.NextHopInfoMap[key]
	if !ok || info.refCount == 0 {
		logger.Info(fmt.Sprintln("Arp resolve not called for ", key.nextHopIp))
		return false
	}
	return true
}
func updateNextHopMap(key NextHopInfoKey, op int) (count int) {
	opStr := ""
	if op == add {
		opStr = "incrementing"
	} else if op == del {
		opStr = "decrementing"
	}
	logger.Info(fmt.Sprintln(opStr, " nextHop Map for ", key.nextHopIp))
	if RouteServiceHandler.NextHopInfoMap == nil {
		return -1
	}
	info, ok := RouteServiceHandler.NextHopInfoMap[key]
	if !ok {
		RouteServiceHandler.NextHopInfoMap[key] = NextHopInfo{1}
		count = 1
	} else {
		if op == add {
			info.refCount++
		} else if op == del {
			info.refCount--
		}
		RouteServiceHandler.NextHopInfoMap[key] = info
		count = info.refCount
	}
	logger.Info(fmt.Sprintln("Updated refcount = ", count))
	return count
}
func findElement(list []string, element string) int {
	index := -1
	for i := 0; i < len(list); i++ {
		if list[i] == element {
			logger.Info(fmt.Sprintln("Found element ", element, " at index ", i))
			return i
		}
	}
	logger.Info(fmt.Sprintln("Element ", element, " not added to the list"))
	return index
}
func buildPolicyEntityFromRoute(route ribdInt.Routes, params interface{}) (entity policy.PolicyEngineFilterEntityParams, err error) {
	routeInfo := params.(RouteParams)
	logger.Info(fmt.Sprintln("buildPolicyEntityFromRoute: createType: ", routeInfo.createType, " delete type: ", routeInfo.deleteType))
	destNetIp, err := netUtils.GetCIDR(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Info(fmt.Sprintln("error getting CIDR address for ", route.Ipaddr, ":", route.Mask))
		return entity, err
	}
	entity.DestNetIp = destNetIp
	logger.Info(fmt.Sprintln("buildPolicyEntityFromRoute: destNetIp:", entity.DestNetIp))
	entity.NextHopIp = route.NextHopIp
	entity.RouteProtocol = ReverseRouteProtoTypeMapDB[int(route.Prototype)]
	if routeInfo.createType != Invalid {
		entity.CreatePath = true
	}
	if routeInfo.deleteType != Invalid {
		entity.DeletePath = true
	}
	return entity, err
}
func BuildRouteParamsFromRouteInoRecord(routeInfoRecord RouteInfoRecord) RouteParams {
	var params RouteParams
	params.routeType = ribd.Int(routeInfoRecord.protocol)
	params.destNetIp = routeInfoRecord.destNetIp.String()
	params.sliceIdx = ribd.Int(routeInfoRecord.sliceIdx)
	params.networkMask = routeInfoRecord.networkMask.String()
	params.metric = routeInfoRecord.metric
	params.nextHopIp = routeInfoRecord.nextHopIp.String()
	params.nextHopIfIndex = routeInfoRecord.nextHopIfIndex
	return params
}
func BuildRouteParamsFromribdIPv4Route(cfg *ribd.IPv4Route, createType int, deleteType int, sliceIdx int) RouteParams {
	nextHopIp := cfg.NextHop[0].NextHopIp
	if cfg.NullRoute == true { //commonDefs.IfTypeNull {
		logger.Info("null route create request")
		nextHopIp = "255.255.255.255"
	}
	nextHopIntRef, _ := strconv.Atoi(cfg.NextHop[0].NextHopIntRef)
	params := RouteParams{destNetIp: cfg.DestinationNw,
		networkMask:    cfg.NetworkMask,
		nextHopIp:      nextHopIp,
		nextHopIfIndex: ribd.Int(nextHopIntRef),
		weight:         ribd.Int(cfg.NextHop[0].Weight),
		metric:         ribd.Int(cfg.Cost),
		routeType:      ribd.Int(RouteProtocolTypeMapDB[cfg.Protocol]),
		sliceIdx:       ribd.Int(sliceIdx),
		createType:     ribd.Int(createType),
		deleteType:     ribd.Int(deleteType),
	}
	return params
}
func BuildPolicyRouteFromribdIPv4Route(cfg *ribd.IPv4Route) (policyRoute ribdInt.Routes) {
	nextHopIp := cfg.NextHop[0].NextHopIp
	if cfg.NullRoute == true { //commonDefs.IfTypeNull {
		logger.Info("null route create request")
		nextHopIp = "255.255.255.255"
	}
	nextHopIntRef, _ := strconv.Atoi(cfg.NextHop[0].NextHopIntRef)
	policyRoute = ribdInt.Routes{Ipaddr: cfg.DestinationNw,
		Mask:      cfg.NetworkMask,
		NextHopIp: nextHopIp,
		IfIndex:   ribdInt.Int(nextHopIntRef), //cfg.NextHopInfp[0].NextHopIntRef,
		Weight:    ribdInt.Int(cfg.NextHop[0].Weight),
		Metric:    ribdInt.Int(cfg.Cost),
		Prototype: ribdInt.Int(RouteProtocolTypeMapDB[cfg.Protocol]),
	}
	return policyRoute
}
func findRouteWithNextHop(routeInfoList []RouteInfoRecord, nextHopIP string) (found bool, routeInfoRecord RouteInfoRecord, index int) {
	logger.Println("findRouteWithNextHop")
	index = -1
	for i := 0; i < len(routeInfoList); i++ {
		if routeInfoList[i].nextHopIp.String() == nextHopIP {
			logger.Println("Next hop IP present")
			found = true
			routeInfoRecord = routeInfoList[i]
			index = i
			break
		}
	}
	return found, routeInfoRecord, index
}
func newNextHopIP(ip string, routeInfoList []RouteInfoRecord) (isNewNextHopIP bool) {
	logger.Println("newNextHopIP")
	isNewNextHopIP = true
	for i := 0; i < len(routeInfoList); i++ {
		if routeInfoList[i].nextHopIp.String() == ip {
			logger.Println("Next hop IP already present")
			isNewNextHopIP = false
		}
	}
	return isNewNextHopIP
}
func isSameRoute(selectedRoute ribdInt.Routes, route ribdInt.Routes) (same bool) {
	logger.Println("isSameRoute")
	if selectedRoute.Ipaddr == route.Ipaddr && selectedRoute.Mask == route.Mask && selectedRoute.Prototype == route.Prototype {
		same = true
	}
	return same
}
func getPolicyRouteMapIndex(entity policy.PolicyEngineFilterEntityParams, policy string) (policyRouteIndex policy.PolicyEntityMapIndex) {
	logger.Println("getPolicyRouteMapIndex")
	policyRouteIndex = PolicyRouteIndex{destNetIP: entity.DestNetIp, policy: policy}
	logger.Info(fmt.Sprintln("Returning policyRouteIndex as : ", policyRouteIndex))
	return policyRouteIndex
}
/*
   Update routelist for policy
*/
func addPolicyRouteMap(route ribdInt.Routes, policyName string) {
	logger.Println("addPolicyRouteMap")
	ipPrefix, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Invalid ip prefix")
		return
	}
	maskIp, err := getIP(route.Mask)
	if err != nil {
		return
	}
	prefixLen, err := getPrefixLen(maskIp)
	if err != nil {
		return
	}
	logger.Info(fmt.Sprintln("prefixLen= ", prefixLen))
	var newRoute string
	found := false
	newRoute = route.Ipaddr + "/" + strconv.Itoa(prefixLen)
	//	newRoute := string(ipPrefix[:])
	logger.Info(fmt.Sprintln("Adding ip prefix %s %v ", newRoute, ipPrefix))
	policyInfo := PolicyEngineDB.PolicyDB.Get(patriciaDB.Prefix(policyName))
	if policyInfo == nil {
		logger.Info(fmt.Sprintln("Unexpected:policyInfo nil for policy ", policyName))
		return
	}
	tempPolicyInfo := policyInfo.(policy.Policy)
	tempPolicy := tempPolicyInfo.Extensions.(PolicyExtensions)
	tempPolicy.hitCounter++
	if tempPolicy.routeList == nil {
		logger.Println("routeList nil")
		tempPolicy.routeList = make([]string, 0)
	}
	logger.Info(fmt.Sprintln("routelist len= ", len(tempPolicy.routeList), " prefix list so far"))
	for i := 0; i < len(tempPolicy.routeList); i++ {
		logger.Info(fmt.Sprintln(tempPolicy.routeList[i]))
		if tempPolicy.routeList[i] == newRoute {
			logger.Info(fmt.Sprintln(newRoute, " already is a part of ", policyName, "'s routelist"))
			found = true
		}
	}
	if !found {
		tempPolicy.routeList = append(tempPolicy.routeList, newRoute)
	}
	found = false
	logger.Println("routeInfoList details")
	for i := 0; i < len(tempPolicy.routeInfoList); i++ {
		logger.Info(fmt.Sprintln("IP: ", tempPolicy.routeInfoList[i].Ipaddr, ":", tempPolicy.routeInfoList[i].Mask, " routeType: ", tempPolicy.routeInfoList[i].Prototype))
		if tempPolicy.routeInfoList[i].Ipaddr == route.Ipaddr && tempPolicy.routeInfoList[i].Mask == route.Mask && tempPolicy.routeInfoList[i].Prototype == route.Prototype {
			logger.Info(fmt.Sprintln("route already is a part of ", policyName, "'s routeInfolist"))
			found = true
		}
	}
	if tempPolicy.routeInfoList == nil {
		tempPolicy.routeInfoList = make([]ribdInt.Routes, 0)
	}
	if found == false {
		tempPolicy.routeInfoList = append(tempPolicy.routeInfoList, route)
	}
	tempPolicyInfo.Extensions = tempPolicy
	PolicyEngineDB.PolicyDB.Set(patriciaDB.Prefix(policyName), tempPolicyInfo)
}
func deletePolicyRouteMap(route ribdInt.Routes, policyName string) {
	logger.Println("deletePolicyRouteMap")
}
func updatePolicyRouteMap(route ribdInt.Routes, policy string, op int) {
	logger.Println("updatePolicyRouteMap")
	if op == add {
		addPolicyRouteMap(route, policy)
	} else if op == del {
		deletePolicyRouteMap(route, policy)
	}

}

func deleteRoutePolicyStateAll(route ribdInt.Routes) {
	logger.Println("deleteRoutePolicyStateAll")
	destNet, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		return
	}

	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Info(fmt.Sprintln(" entry not found for prefix %v", destNet))
		return
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	routeInfoRecordList.policyHitCounter = ribd.Int(route.PolicyHitCounter)
	routeInfoRecordList.policyList = nil //append(routeInfoRecordList.policyList[:0])
	RouteInfoMap.Set(destNet, routeInfoRecordList)
	return
}
func addRoutePolicyState(route ribdInt.Routes, policy string, policyStmt string) {
	logger.Println("addRoutePolicyState")
	destNet, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		return
	}

	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Info(fmt.Sprintln("Unexpected - entry not found for prefix %v", destNet))
		return
	}
	logger.Info(fmt.Sprintln("Adding policy ", policy, " to route ", destNet))
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found := false
	idx := 0
	for idx = 0; idx < len(routeInfoRecordList.policyList); idx++ {
		if routeInfoRecordList.policyList[idx] == policy {
			found = true
			break
		}
	}
	if found {
		logger.Info(fmt.Sprintln("Policy ", policy, "already a part of policyList of route ", destNet))
		return
	}
	routeInfoRecordList.policyHitCounter = ribd.Int(route.PolicyHitCounter)
	if routeInfoRecordList.policyList == nil {
		routeInfoRecordList.policyList = make([]string, 0)
	}
	/*	policyStmtList := routeInfoRecordList.policyList[policy]
		if policyStmtList == nil {
		   policyStmtList = make([]string,0)
		}
		policyStmtList = append(policyStmtList,policyStmt)
	    routeInfoRecordList.policyList[policy] = policyStmtList*/
	routeInfoRecordList.policyList = append(routeInfoRecordList.policyList, policy)
	RouteInfoMap.Set(destNet, routeInfoRecordList)
	//RouteServiceHandler.DBRouteAddCh <- RouteDBInfo{routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0],routeInfoRecordList}
	RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0], routeInfoRecordList})
	return
}
func deleteRoutePolicyState(ipPrefix patriciaDB.Prefix, policyName string) {
	logger.Println("deleteRoutePolicyState")
	found := false
	idx := 0
	routeInfoRecordListItem := RouteInfoMap.Get(ipPrefix)
	if routeInfoRecordListItem == nil {
		logger.Info(fmt.Sprintln("routeInfoRecordListItem nil for prefix ", ipPrefix))
		return
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	/*    if routeInfoRecordList.policyList[policyName] != nil {
		delete(routeInfoRecordList.policyList, policyName)
	}*/
	for idx = 0; idx < len(routeInfoRecordList.policyList); idx++ {
		if routeInfoRecordList.policyList[idx] == policyName {
			found = true
			break
		}
	}
	if !found {
		logger.Info(fmt.Sprintln("Policy ", policyName, "not found in policyList of route ", ipPrefix))
		return
	}
	if len(routeInfoRecordList.policyList) <= idx+1 {
		logger.Println("last element")
		routeInfoRecordList.policyList = routeInfoRecordList.policyList[:idx]
	} else {
		routeInfoRecordList.policyList = append(routeInfoRecordList.policyList[:idx], routeInfoRecordList.policyList[idx+1:]...)
	}
	RouteInfoMap.Set(ipPrefix, routeInfoRecordList)
}

func updateRoutePolicyState(route ribdInt.Routes, op int, policy string, policyStmt string) {
	logger.Println("updateRoutePolicyState")
	if op == delAll {
		deleteRoutePolicyStateAll(route)
	} else if op == add {
		addRoutePolicyState(route, policy, policyStmt)
	}
}
func UpdateRedistributeTargetMap(evt int, protocol string, route ribdInt.Routes) {
	logger.Println("UpdateRedistributeTargetMap")
	if evt == ribdCommonDefs.NOTIFY_ROUTE_CREATED {
		redistributeMapInfo := RedistributeRouteMap[protocol]
		if redistributeMapInfo == nil {
			redistributeMapInfo = make([]RedistributeRouteInfo, 0)
		}
		redistributeRouteInfo := RedistributeRouteInfo{route: route}
		redistributeMapInfo = append(redistributeMapInfo, redistributeRouteInfo)
		RedistributeRouteMap[protocol] = redistributeMapInfo
	} else if evt == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
		redistributeMapInfo := RedistributeRouteMap[protocol]
		if redistributeMapInfo != nil {
			found := false
			i := 0
			for i = 0; i < len(redistributeMapInfo); i++ {
				if isSameRoute((redistributeMapInfo[i].route), route) {
					logger.Info(fmt.Sprintln("Found the route that is to be taken off the redistribution list for ", protocol))
					found = true
					break
				}
			}
			if found {
				if len(redistributeMapInfo) <= i+1 {
					redistributeMapInfo = redistributeMapInfo[:i]
				} else {
					redistributeMapInfo = append(redistributeMapInfo[:i], redistributeMapInfo[i+1:]...)
				}
			}
			RedistributeRouteMap[protocol] = redistributeMapInfo
		}
	}
}
func RedistributionNotificationSend(PUB *nanomsg.PubSocket, route ribdInt.Routes, evt int, targetProtocol string) {
	logger.Println("RedistributionNotificationSend")
	msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo: route}
	msgbufbytes, err := json.Marshal(msgBuf)
	msg := ribdCommonDefs.RibdNotifyMsg{MsgType: uint16(evt), MsgBuf: msgbufbytes}
	buf, err := json.Marshal(msg)
	if err != nil {
		logger.Println("Error in marshalling Json")
		return
	}
	var evtStr string
	if evt == ribdCommonDefs.NOTIFY_ROUTE_CREATED {
		evtStr = " NOTIFY_ROUTE_CREATED "
	} else if evt == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
		evtStr = " NOTIFY_ROUTE_DELETED "
	}
	eventInfo := "Redistribute "
	if route.NetworkStatement == true {
		eventInfo = " Advertise Network Statement "
	}
	eventInfo = eventInfo + evtStr + " for route " + route.Ipaddr + " " + route.Mask + " type " + ReverseRouteProtoTypeMapDB[int(route.Prototype)] + " to " + targetProtocol
	logger.Info(fmt.Sprintln("Adding ", evtStr, " for route ", route.Ipaddr, " ", route.Mask, " to notification channel"))
	RouteServiceHandler.NotificationChannel <- NotificationMsg{PUB, buf, eventInfo}
}
func RouteReachabilityStatusNotificationSend(targetProtocol string, info RouteReachabilityStatusInfo) {
	logger.Info(fmt.Sprintln("RouteReachabilityStatusNotificationSend for protocol ", targetProtocol))
	publisherInfo, ok := PublisherInfoMap[targetProtocol]
	if !ok {
		logger.Info(fmt.Sprintln("Publisher not found for protocol ", targetProtocol))
		return
	}
	evt := ribdCommonDefs.NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE
	PUB := publisherInfo.pub_socket
	msgInfo := ribdCommonDefs.RouteReachabilityStatusMsgInfo{}
	msgInfo.Network = info.destNet
	if info.status == "Up" || info.status == "Updated" {
		msgInfo.IsReachable = true
	}
	msgInfo.NextHopIntf = info.nextHopIntf
	msgBuf := msgInfo
	msgbufbytes, err := json.Marshal(msgBuf)
	msg := ribdCommonDefs.RibdNotifyMsg{MsgType: uint16(evt), MsgBuf: msgbufbytes}
	buf, err := json.Marshal(msg)
	if err != nil {
		logger.Println("Error in marshalling Json")
		return
	}
	eventInfo := "Update Route Reachability status " + info.status + " for network " + info.destNet + " for protocol " + targetProtocol
	if info.status == "Up" {
		eventInfo = eventInfo + " NextHop IP: " + info.nextHopIntf.NextHopIp + " Index: " + strconv.Itoa(int(info.nextHopIntf.NextHopIfIndex))
	}
	logger.Info(fmt.Sprintln("Adding  NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE with status ", info.status, " for network ", info.destNet, " to notification channel"))
	RouteServiceHandler.NotificationChannel <- NotificationMsg{PUB, buf, eventInfo}
}
func RouteReachabilityStatusUpdate(targetProtocol string, info RouteReachabilityStatusInfo) {
	logger.Info(fmt.Sprintln("RouteReachabilityStatusUpdate targetProtocol ", targetProtocol))
	if targetProtocol != "NONE" {
		RouteReachabilityStatusNotificationSend(targetProtocol, info)
	}
	var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(info.destNet)
	if err != nil {
		logger.Err(fmt.Sprintln("Error getting IP from cidr: ", info.destNet))
		return
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	destIpPrefix, err := getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
	if err != nil {
		logger.Err(fmt.Sprintln("Error getting ip prefix for ip:", ipAddrStr, " mask:", ipMaskStr))
		return
	}
	//check the TrackReachabilityMap to see if any other protocols are interested in receiving updates for this network
	for k, list := range TrackReachabilityMap {
		prefix, err := getNetowrkPrefixFromStrings(k, ipMaskStr)
		if err != nil {
			logger.Err(fmt.Sprintln("Error getting ip prefix for ip:", k, " mask:", ipMaskStr))
			return
		}
		if bytes.Equal(destIpPrefix, prefix) {
			for idx := 0; idx < len(list); idx++ {
				logger.Info(fmt.Sprintln(" protocol ", list[idx], " interested in receving reachability updates for ipAddr ", info.destNet))
				info.destNet = k
				RouteReachabilityStatusNotificationSend(list[idx], info)
			}
		}
	}
	return
}
func getIPInt(ip net.IP) (ipInt int, err error) {
	if ip == nil {
		logger.Info(fmt.Sprintf("ip address %v invalid\n", ip))
		return ipInt, errors.New("Invalid destination network IP Address")
	}
	ip = ip.To4()
	parsedPrefixIP := int(ip[3]) | int(ip[2])<<8 | int(ip[1])<<16 | int(ip[0])<<24
	ipInt = parsedPrefixIP
	return ipInt, nil
}

func getIP(ipAddr string) (ip net.IP, err error) {
	ip = net.ParseIP(ipAddr)
	if ip == nil {
		return ip, errors.New("Invalid destination network IP Address")
	}
	ip = ip.To4()
	return ip, nil
}

func getPrefixLen(networkMask net.IP) (prefixLen int, err error) {
	ipInt, err := getIPInt(networkMask)
	if err != nil {
		return -1, err
	}
	for prefixLen = 0; ipInt != 0; ipInt >>= 1 {
		prefixLen += ipInt & 1
	}
	return prefixLen, nil
}
func validateNetworkPrefix(ipAddr string, mask string) (destNet patriciaDB.Prefix, err error) {
	logger.Debug(fmt.Sprintln("validateNetworkPrefix for ip ", ipAddr, " mask: ", mask))
	destNetIp, err := getIP(ipAddr)
	if err != nil {
		logger.Err(fmt.Sprintln("destNetIpAddr ", ipAddr, " invalid"))
		return destNet, err
	}
	networkMask, err := getIP(mask)
	if err != nil {
		logger.Err("networkMaskAddr invalid")
		return destNet, err
	}
	prefixLen, err := getPrefixLen(networkMask)
	if err != nil {
		logger.Err(fmt.Sprintln("err when getting prefixLen, err= ", err))
		return destNet, errors.New(fmt.Sprintln("Invalid networkmask ", networkMask))
	}
	vdestMask := net.IPv4Mask(networkMask[0], networkMask[1], networkMask[2], networkMask[3])
	netIp := destNetIp.Mask(vdestMask)
	logger.Debug(fmt.Sprintln("netIP: ", netIp, " destNetIp ", destNetIp))
	if ! ( bytes.Equal(destNetIp, netIp)) {
		logger.Err(fmt.Sprintln("Cannot have ip : ", destNetIp, " more specific than mask "))
		return destNet, errors.New(fmt.Sprintln("IP address ", destNetIp ," more specific than mask ", networkMask))
	}
	numbytes := prefixLen / 8
	if (prefixLen % 8) != 0 {
		numbytes++
	}
	destNet = make([]byte, numbytes)
	for i := 0; i < numbytes; i++ {
		destNet[i] = netIp[i]
	}
	return destNet, err
}
func getNetworkPrefix(destNetIp net.IP, networkMask net.IP) (destNet patriciaDB.Prefix, err error) {
	logger.Debug(fmt.Sprintln("getNetworkPrefix for ip: ", destNetIp, "  networkMask: ", networkMask))
	prefixLen, err := getPrefixLen(networkMask)
	if err != nil {
		logger.Err(fmt.Sprintln("err when getting prefixLen, err= ", err))
		return destNet, errors.New(fmt.Sprintln("Invalid networkmask ", networkMask))
	}
	vdestMask := net.IPv4Mask(networkMask[0], networkMask[1], networkMask[2], networkMask[3])
	netIp := destNetIp.Mask(vdestMask)
	numbytes := prefixLen / 8
	if (prefixLen % 8) != 0 {
		numbytes++
	}
	destNet = make([]byte, numbytes)
	for i := 0; i < numbytes; i++ {
		destNet[i] = netIp[i]
	}
	return destNet, err
}
func getNetowrkPrefixFromStrings(ipAddr string, mask string) (prefix patriciaDB.Prefix, err error) {
	logger.Debug(fmt.Sprintln("getNetowrkPrefixFromStrings for ip ", ipAddr, " mask: ", mask))
	destNetIpAddr, err := getIP(ipAddr)
	if err != nil {
		logger.Info(fmt.Sprintln("destNetIpAddr ", ipAddr, " invalid"))
		return prefix, err
	}
	networkMaskAddr, err := getIP(mask)
	if err != nil {
		logger.Println("networkMaskAddr invalid")
		return prefix, err
	}
	prefix, err = getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		logger.Info(fmt.Sprintln("err=", err))
		return prefix, err
	}
	return prefix, err
}
func getNetworkPrefixFromCIDR(ipAddr string) (ipPrefix patriciaDB.Prefix, err error) {
	var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return ipPrefix, err
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	ipPrefix, err = getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
	return ipPrefix, err
}
