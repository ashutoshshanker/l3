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

package server

import (
	"asicdServices"
	"fmt"
	//	"encoding/json"
	"l3/rib/ribdCommonDefs"
	"ribd"
	"ribdInt"
	"utils/patriciaDB"
	"utils/policy/policyCommonDefs"
	//		"patricia"
	"asicd/asicdCommonDefs"
	"bytes"
	"errors"
	"utils/commonDefs"
	//	"github.com/op/go-nanomsg"
	"encoding/json"
	"net"
	"reflect"
	"strconv"
	"time"
)
/*
    Data type of each route stored in the DB
*/
type RouteInfoRecord struct {
	destNetIp               net.IP 
	networkMask             net.IP 
	nextHopIp               net.IP
	resolvedNextHopIpIntf   ribdInt.NextHopInfo  //immediate next hop info
	networkAddr             string //cidr
	nextHopIfIndex          ribd.Int
	metric                  ribd.Int
	weight                  ribd.Int
	sliceIdx                int
	protocol                int8
	isPolicyBasedStateValid bool
	routeCreatedTime        string
	routeUpdatedTime        string
}

/*
    Map of routeInfoRecords for each protocol type along with few other attributes
*/
type RouteInfoRecordList struct {
	selectedRouteProtocol   string
	routeInfoProtocolMap    map[string][]RouteInfoRecord
	policyHitCounter        ribd.Int
	policyList              []string
	isPolicyBasedStateValid bool
}

/*
    data-structure used for add/delete operations 
*/
type RouteOpInfoRecord struct {
	routeInfoRecord RouteInfoRecord
	opType          int
}

/*
    event notification data-structure
*/
type RouteEventInfo struct {
	timeStamp string
	eventInfo string
}
/*
    to track reachability of a route
*/
type RouteReachabilityStatusInfo struct {
	destNet     string
	status      string
	protocol    string
	nextHopIntf ribdInt.NextHopInfo
}

var RouteInfoMap *patriciaDB.Trie          //Routes are stored in patricia trie 
var DummyRouteInfoRecord RouteInfoRecord 
var destNetSlice []localDB
var localRouteEventsDB []RouteEventInfo

/*
   Update Connected route info
*/
func updateConnectedRoutes(destNetIPAddr string, networkMaskAddr string, nextHopIP string, nextHopIfIndex ribd.Int, op int, sliceIdx ribd.Int) {
	var temproute ribdInt.Routes
	route := &temproute
	logger.Debug(fmt.Sprintln("number of connectd routes = ", len(ConnectedRoutes), "current op is to ", op, " ipAddr:mask = ", destNetIPAddr, ":", networkMaskAddr))
	if len(ConnectedRoutes) == 0 {
		if op == del {
			logger.Debug("Cannot delete a non-existent connected route")
			return
		}
		ConnectedRoutes = make([]*ribdInt.Routes, 1)
		route.Ipaddr = destNetIPAddr
		route.Mask = networkMaskAddr
		route.NextHopIp = nextHopIP
		route.IfIndex = ribdInt.Int(nextHopIfIndex)
		route.IsValid = true
		route.SliceIdx = ribdInt.Int(sliceIdx)
		ConnectedRoutes[0] = route
		return
	}
	for i := 0; i < len(ConnectedRoutes); i++ {
		//		if(!strings.EqualFold(ConnectedRoutes[i].Ipaddr,destNetIPAddr) && !strings.EqualFold(ConnectedRoutes[i].Mask,networkMaskAddr)){
		if ConnectedRoutes[i].Ipaddr == destNetIPAddr && ConnectedRoutes[i].Mask == networkMaskAddr {
			if op == del {
				if len(ConnectedRoutes) <= i+1 {
					ConnectedRoutes = ConnectedRoutes[:i]
				} else {
					ConnectedRoutes = append(ConnectedRoutes[:i], ConnectedRoutes[i+1:]...)
				}
			} else if op == invalidate { //op is invalidate when a link on which the connectedroutes is configured goes down
				ConnectedRoutes[i].IsValid = false
			}
			return
		}
	}
	if op == del {
		return
	}
	route.Ipaddr = destNetIPAddr
	route.Mask = networkMaskAddr
	route.NextHopIp = nextHopIP
	route.IfIndex = ribdInt.Int(nextHopIfIndex)
	route.IsValid = true
	route.SliceIdx = ribdInt.Int(sliceIdx)
	ConnectedRoutes = append(ConnectedRoutes, route)
}
/*
    Api to find if a route of a protocol type is present
*/
func IsRoutePresent(routeInfoRecordList RouteInfoRecordList,
	protocol string) (found bool) {
	logger.Debug(fmt.Sprintln("Trying to look for route type ", protocol))
	routeInfoList, ok := routeInfoRecordList.routeInfoProtocolMap[protocol]
	if ok && len(routeInfoList) > 0 {
		logger.Debug(fmt.Sprintln(len(routeInfoList), " number of routeInfoRecords stored for this protocol"))
		found = true
	}
	return found
}

func getConnectedRoutes() {
	logger.Debug("Getting connected routes from portd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		logger.Debug(fmt.Sprintf("Getting ", count, " objects from currMarker ",currMarker))
		IPIntfBulk, err := asicdclnt.ClientHdl.GetBulkIPv4IntfState(currMarker, count)
		if err != nil {
			logger.Debug(fmt.Sprintln("GetBulkIPv4IntfState with err ", err))
			return
		}
		if IPIntfBulk.Count == 0 {
			logger.Println("0 objects returned from GetBulkIPv4IntfState")
			return
		}
		logger.Debug(fmt.Sprintf("len(IPIntfBulk.IPv4IntfStateList)  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfStateList), IPIntfBulk.Count))
		for i := 0; i < int(IPIntfBulk.Count); i++ {
			var ipMask net.IP
			ip, ipNet, err := net.ParseCIDR(IPIntfBulk.IPv4IntfStateList[i].IpAddr)
			if err != nil {
				return
			}
			ipMask = make(net.IP, 4)
			copy(ipMask, ipNet.Mask)
			ipAddrStr := ip.String()
			ipMaskStr := net.IP(ipMask).String()
			logger.Debug(fmt.Sprintln("Calling createv4Route with ipaddr ", ipAddrStr, " mask ", ipMaskStr, "ifIndex : ", IPIntfBulk.IPv4IntfStateList[i].IfIndex))
			cfg := ribd.IPv4Route{
				DestinationNw: ipAddrStr,
				Protocol:      "CONNECTED",
				Cost:          0,
				NetworkMask:   ipMaskStr,
			}
			nextHop := ribd.NextHopInfo{
				NextHopIp:     "0.0.0.0",
				NextHopIntRef: strconv.Itoa(int(IPIntfBulk.IPv4IntfStateList[i].IfIndex)),//strconv.Itoa(int(asicdCommonDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex))),
			}
			cfg.NextHop = make([]*ribd.NextHopInfo, 0)
			cfg.NextHop = append(cfg.NextHop, &nextHop)
			_, err = RouteServiceHandler.ProcessRouteCreateConfig(&cfg) //ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), "CONNECTED") // FIBAndRIB, ribd.Int(len(destNetSlice)))
			if err != nil {
				logger.Debug(fmt.Sprintf("Failed to create connected route for ip Addr %s/%s intfType %d intfId %d\n", ipAddrStr, ipMaskStr, ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfStateList[i].IfIndex))))
			}
		}
		if IPIntfBulk.More == false {
			logger.Debug("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(IPIntfBulk.EndIdx)
	}
}

func (m RIBDServer) GetRouteDistanceState(protocol string) (*ribd.RouteDistanceState, error) {
	logger.Debug("Get state for RouteDistanceState")
	state := ribd.NewRouteDistanceState()
	if ProtocolAdminDistanceMapDB == nil {
		logger.Debug("ProtocolAdminDistanceMapDB not initialized")
		return state, errors.New("ProtocolAdminDistanceMapDB not initialized")
	}
	val,ok := ProtocolAdminDistanceMapDB[protocol]
	if !ok {
		logger.Err(fmt.Sprintln("Admin Distance for protocol ", protocol, " not set"))
		return state, errors.New(fmt.Sprintln("Admin Distance for protocol ", protocol, " not set"))
	}
	state.Protocol = protocol
	state.Distance = int32(val.configuredDistance)
	return state, nil
}

//thrift API definitions
func (m RIBDServer) GetBulkRouteDistanceState(fromIndex ribd.Int, rcount ribd.Int) (routeDistanceStates *ribd.RouteDistanceStateGetInfo, err error) {
	logger.Debug("GetBulkRouteDistanceState")
	var i, validCount, toIndex ribd.Int
	var tempNode []ribd.RouteDistanceState = make([]ribd.RouteDistanceState, rcount)
	var nextNode *ribd.RouteDistanceState
	var returnNodes []*ribd.RouteDistanceState
	var returnGetInfo ribd.RouteDistanceStateGetInfo
	i = 0
	routeDistanceStates = &returnGetInfo
	more := true
	BuildProtocolAdminDistanceSlice()
	if ProtocolAdminDistanceSlice == nil {
		logger.Debug("ProtocolAdminDistanceSlice not initialized")
		return routeDistanceStates, err
	}
	for ; ; i++ {
		logger.Debug(fmt.Sprintf("Fetching record for index %d\n", i+fromIndex))
		if i+fromIndex >= ribd.Int(len(ProtocolAdminDistanceSlice)) {
			logger.Debug("All the events fetched")
			more = false
			break
		}
		if validCount == rcount {
			logger.Debug("Enough events fetched")
			break
		}
		logger.Debug(fmt.Sprintf("Fetching event record for index ", i+fromIndex))
		nextNode = &tempNode[validCount]
		nextNode.Protocol = ProtocolAdminDistanceSlice[i+fromIndex].Protocol
		nextNode.Distance = ProtocolAdminDistanceSlice[i+fromIndex].Distance
		toIndex = ribd.Int(i + fromIndex)
		if len(returnNodes) == 0 {
			returnNodes = make([]*ribd.RouteDistanceState, 0)
		}
		returnNodes = append(returnNodes, nextNode)
		validCount++
	}
	logger.Debug(fmt.Sprintf("Returning ", validCount, " list of dtsnace vector nodes"))
	routeDistanceStates.RouteDistanceStateList = returnNodes
	routeDistanceStates.StartIdx = fromIndex
	routeDistanceStates.EndIdx = toIndex + 1
	routeDistanceStates.More = more
	routeDistanceStates.Count = validCount
	return routeDistanceStates, err
}

func (m RIBDServer) GetBulkIPv4EventState(fromIndex ribd.Int, rcount ribd.Int) (events *ribd.IPv4EventStateGetInfo, err error) {
	logger.Debug("GetBulkIPv4EventState")
	var i, validCount, toIndex ribd.Int
	var tempNode []ribd.IPv4EventState = make([]ribd.IPv4EventState, rcount)
	var nextNode *ribd.IPv4EventState
	var returnNodes []*ribd.IPv4EventState
	var returnGetInfo ribd.IPv4EventStateGetInfo
	i = 0
	events = &returnGetInfo
	more := true
	if localRouteEventsDB == nil {
		logger.Debug("localRouteEventsDB not initialized")
		return events, err
	}
	for ; ; i++ {
		logger.Debug(fmt.Sprintf("Fetching record for index ", i+fromIndex))
		if i+fromIndex >= ribd.Int(len(localRouteEventsDB)) {
			logger.Debug("All the events fetched")
			more = false
			break
		}
		if validCount == rcount {
			logger.Debug("Enough events fetched")
			break
		}
		logger.Debug(fmt.Sprintf("Fetching event record for index ", i+fromIndex))
		nextNode = &tempNode[validCount]
		nextNode.TimeStamp = localRouteEventsDB[i+fromIndex].timeStamp
		nextNode.EventInfo = localRouteEventsDB[i+fromIndex].eventInfo
		toIndex = ribd.Int(i + fromIndex)
		if len(returnNodes) == 0 {
			returnNodes = make([]*ribd.IPv4EventState, 0)
		}
		returnNodes = append(returnNodes, nextNode)
		validCount++
	}
	logger.Debug(fmt.Sprintf("Returning ", validCount, " list of events"))
	events.IPv4EventStateList = returnNodes
	events.StartIdx = fromIndex
	events.EndIdx = toIndex + 1
	events.More = more
	events.Count = validCount
	return events, err
}
/*
   Application daemons like BGPD/OSPFD can call this API to get list of routes that 
   they should have, typically called at startup time
*/
func (m RIBDServer) GetBulkRoutesForProtocol(srcProtocol string, fromIndex ribdInt.Int, rcount ribdInt.Int) (routes *ribdInt.RoutesGetInfo, err error) {
	logger.Debug("GetBulkRoutesForProtocol")
	var i, validCount, toIndex ribdInt.Int
	var nextRoute *ribdInt.Routes
	var returnRoutes []*ribdInt.Routes
	var returnRouteGetInfo ribdInt.RoutesGetInfo
	i = 0
	routes = &returnRouteGetInfo
	moreRoutes := true
	redistributeRouteMap := RedistributeRouteMap[srcProtocol]
	if redistributeRouteMap == nil {
		logger.Debug(fmt.Sprintln("no routes to be advertised for this protocol ", srcProtocol))
		return routes, err
	}
	for ; ; i++ {
		logger.Debug(fmt.Sprintf("Fetching trie record for index ", i+fromIndex))
		if i+fromIndex >= ribdInt.Int(len(redistributeRouteMap)) {
			logger.Debug("All the routes fetched")
			moreRoutes = false
			break
		}
		if validCount == rcount {
			logger.Debug("Enough routes fetched")
			break
		}
		logger.Debug(fmt.Sprintf("Fetching route for index ", i+fromIndex))
		nextRoute = &redistributeRouteMap[i+fromIndex].route
		if len(returnRoutes) == 0 {
			returnRoutes = make([]*ribdInt.Routes, 0)
		}
		returnRoutes = append(returnRoutes, nextRoute)
		validCount++
	}
	logger.Debug(fmt.Sprintf("Returning ", validCount, " list of routes"))
	routes.RouteList = returnRoutes
	routes.StartIdx = fromIndex
	routes.EndIdx = toIndex + 1
	routes.More = moreRoutes
	routes.Count = validCount
	return routes, err
}
func (m RIBDServer) GetBulkIPv4RouteState(fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.IPv4RouteStateGetInfo, err error) { //(routes []*ribdInt.Routes, err error) {
	logger.Debug("GetBulkIPv4RouteState")
	var i, validCount ribd.Int
	var toIndex ribd.Int
	var temproute []ribd.IPv4RouteState = make([]ribd.IPv4RouteState, rcount)
	var nextRoute *ribd.IPv4RouteState
	var returnRoutes []*ribd.IPv4RouteState
	var returnRouteGetInfo ribd.IPv4RouteStateGetInfo
	var prefixNodeRouteList RouteInfoRecordList
	var prefixNodeRoute RouteInfoRecord
	i = 0
	sel := 0
	found := false
	routes = &returnRouteGetInfo
	moreRoutes := true
	if destNetSlice == nil {
		logger.Debug("destNetSlice not initialized: No Routes installed in RIB")
		return routes, err
	}
	for ; ; i++ {
		logger.Debug(fmt.Sprintf("Fetching trie record for index %d\n", i+fromIndex))
		found = false
		if i+fromIndex >= ribd.Int(len(destNetSlice)) {
			logger.Debug("All the routes fetched")
			moreRoutes = false
			break
		}
		/*		if destNetSlice[i+fromIndex].isValid == false {
				logger.Debug("Invalid route")
				continue
			}*/
		if validCount == rcount {
			logger.Debug("Enough routes fetched")
			break
		}
		logger.Debug(fmt.Sprintf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (destNetSlice[i+fromIndex].prefix)))
		prefixNode := RouteInfoMap.Get(destNetSlice[i+fromIndex].prefix)
		if prefixNode != nil {
			prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
			if prefixNodeRouteList.isPolicyBasedStateValid == false {
				logger.Debug("Route invalidated based on policy")
				continue
			}
			logger.Debug(fmt.Sprintln("selectedRouteProtocol = ", prefixNodeRouteList.selectedRouteProtocol))
			if prefixNodeRouteList.routeInfoProtocolMap == nil || prefixNodeRouteList.selectedRouteProtocol == "INVALID" || prefixNodeRouteList.routeInfoProtocolMap[prefixNodeRouteList.selectedRouteProtocol] == nil {
				logger.Debug("selected route not valid")
				continue
			}
			routeInfoList := prefixNodeRouteList.routeInfoProtocolMap[prefixNodeRouteList.selectedRouteProtocol]
			for sel = 0; sel < len(routeInfoList); sel++ {
				if routeInfoList[sel].nextHopIp.String() == destNetSlice[i+fromIndex].nextHopIp {
					logger.Debug("Found the entry corresponding to the nextHop ip")
					found = true
					break
				}
			}
			if !found {
				logger.Debug("The corresponding route with nextHopIP was not found in the record DB")
				continue
			}
			prefixNodeRoute = routeInfoList[sel] //prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
			nextRoute = &temproute[validCount]
			nextRoute.DestinationNw = prefixNodeRoute.networkAddr
			/*			nextRoute.NextHopIp = prefixNodeRoute.nextHopIp.String()
						nextHopIfTypeStr, _ := m.GetNextHopIfTypeStr(ribdInt.Int(prefixNodeRoute.nextHopIfType))
						nextRoute.OutgoingIntfType = nextHopIfTypeStr
						nextRoute.OutgoingInterface = strconv.Itoa(int(prefixNodeRoute.nextHopIfIndex))
						nextRoute.Protocol = ReverseRouteProtoTypeMapDB[int(prefixNodeRoute.protocol)]*/
			nextRoute.RouteCreatedTime = prefixNodeRoute.routeCreatedTime
			nextRoute.RouteUpdatedTime = prefixNodeRoute.routeUpdatedTime
			nextRoute.IsNetworkReachable = prefixNodeRoute.resolvedNextHopIpIntf.IsReachable
			nextRoute.PolicyList = make([]string, 0)
			routePolicyListInfo := ""
			if prefixNodeRouteList.policyList != nil {
				for k := 0; k < len(prefixNodeRouteList.policyList); k++ {
					routePolicyListInfo = "policy " + prefixNodeRouteList.policyList[k] + "["
					policyRouteIndex := PolicyRouteIndex{destNetIP: prefixNodeRoute.networkAddr, policy: prefixNodeRouteList.policyList[k]}
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
					nextRoute.PolicyList = append(nextRoute.PolicyList, routePolicyListInfo)
				}
			}
			toIndex = ribd.Int(i + fromIndex)
			if len(returnRoutes) == 0 {
				returnRoutes = make([]*ribd.IPv4RouteState, 0)
			}
			returnRoutes = append(returnRoutes, nextRoute)
			validCount++
		}
	}
	logger.Debug(fmt.Sprintf("Returning %d list of routes\n", validCount))
	routes.IPv4RouteStateList = returnRoutes
	routes.StartIdx = fromIndex
	routes.EndIdx = toIndex + 1
	routes.More = moreRoutes
	routes.Count = validCount
	return routes, err
}

/*
    API called by external applications interested in tracking reachability status of a network
	This function adds and removes ipAddr from the TrachReachabilityMap based on the op value
*/	
func (m RIBDServer) TrackReachabilityStatus(ipAddr string, protocol string, op string) error {
	logger.Debug(fmt.Sprintln("TrackReachabilityStatus for ipAddr: ", ipAddr, " by protocol ", protocol, " op = ", op))
	if op != "add" && op != "del" {
		logger.Err(fmt.Sprintln("Invalid operation ", op))
		return errors.New("Invalid operation")
	}
	/*
	   Check if this ipAddr is being tracked.
	*/
	protocolList, ok := TrackReachabilityMap[ipAddr]
	if !ok {
		if op == "del" {
			logger.Err(fmt.Sprintln("ipAddr ", ipAddr, " not being tracked currently"))
			return errors.New("ipAddr not being tracked currently")
		}
		/*
		    no application is tracking this, create the list for "add" op
		*/
		if op == "add" {
		    protocolList = make([]string, 0)
		}
	}
	index := -1
	index = findElement(protocolList, protocol)
	if index != -1 {
		if op == "del" {
			protocolList = append(protocolList[:index], protocolList[index:]...)
		} else if op == "add" {
			logger.Debug(fmt.Sprintln(protocol, " already tracking ip ", ipAddr))
			return nil
		}
	} else { //index = -1, protocol not tracking the ipAddr
		if op == "del" {
			logger.Err(fmt.Sprintln(protocol, " not tracking ipAddr ", ipAddr))
			return errors.New(" ipAddr not being tracked by the protocol")
		} else if op == "add" {
			protocolList = append(protocolList, protocol)
		}
	}
	/*
	    Update the TrackReachabilityMap for this ip
	*/
	TrackReachabilityMap[ipAddr] = protocolList
	return nil
}
/*
    Returns the longest prefix match route to reach the destination network destNet
*/
func (m RIBDServer) GetRouteReachabilityInfo(destNet string) (nextHopIntf *ribdInt.NextHopInfo, err error) {
	logger.Debug(fmt.Sprintln("GetRouteReachabilityInfo of ", destNet))
	t1 := time.Now()
	var retnextHopIntf ribdInt.NextHopInfo
	nextHopIntf = &retnextHopIntf
	var found bool
	destNetIp, err := getIP(destNet)
	if err != nil {
		logger.Debug(fmt.Sprintln("getIP returned Invalid dest ip address for ", destNet))
		return nextHopIntf, errors.New("Invalid dest ip address")
	}
	rmapInfoListItem := RouteInfoMap.GetLongestPrefixNode(patriciaDB.Prefix(destNetIp))
	if rmapInfoListItem != nil {
		rmapInfoList := rmapInfoListItem.(RouteInfoRecordList)
		if rmapInfoList.selectedRouteProtocol != "INVALID" {
			found = true
			routeInfoList, ok := rmapInfoList.routeInfoProtocolMap[rmapInfoList.selectedRouteProtocol]
			if !ok {
				logger.Debug("Selected route not found")
				return nextHopIntf, err
			}
			v := routeInfoList[0]
			nextHopIntf.NextHopIp = v.nextHopIp.String()
			nextHopIntf.NextHopIfIndex = ribdInt.Int(v.nextHopIfIndex)
			nextHopIntf.Metric = ribdInt.Int(v.metric)
			nextHopIntf.Ipaddr = v.destNetIp.String()
			nextHopIntf.Mask = v.networkMask.String()
			nextHopIntf.IsReachable = true
		}
	}

	if found == false {
		logger.Debug(fmt.Sprintln("dest IP", destNetIp, " not reachable "))
		err = errors.New("dest ip address not reachable")
		return nextHopIntf,err
	}
	duration := time.Since(t1)
	logger.Debug(fmt.Sprintln("time to get longestPrefixLen = ", duration.Nanoseconds(), " ipAddr of the route: ", nextHopIntf.Ipaddr, " next hop ip of the route = ", nextHopIntf.NextHopIp, " ifIndex: ", nextHopIntf.NextHopIfIndex))
	return nextHopIntf, err
}
func (m RIBDServer) GetRoute(destNetIp string, networkMask string) (route *ribdInt.Routes, err error) {
	var returnRoute ribdInt.Routes
	route = &returnRoute
	destNetIpAddr, err := getIP(destNetIp)
	if err != nil {
		return route, err
	}
	networkMaskAddr, err := getIP(networkMask)
	if err != nil {
		return route, err
	}
	destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		return route, err
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Debug("No such route")
		err = errors.New("Route does not exist")
		return route, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
	if routeInfoRecordList.selectedRouteProtocol == "INVALID" {
		logger.Debug("No selected route for this network")
		err = errors.New("No selected route for this network")
		return route, err
	}
	routeInfoList := routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]
	routeInfoRecord := routeInfoList[0]
	//	routeInfoRecord := routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
	route.Ipaddr = destNetIp
	route.Mask = networkMask
	route.NextHopIp = routeInfoRecord.nextHopIp.String()
	route.IfIndex = ribdInt.Int(routeInfoRecord.nextHopIfIndex)
	route.Metric = ribdInt.Int(routeInfoRecord.metric)
	route.Prototype = ribdInt.Int(routeInfoRecord.protocol)
	return route, err
}
/*
    Function updates the route reachability status of a network. When a route is created/deleted/state changes,
	we traverse the entire route map and call this function for each of the destination network with :
	    prefix = route prefix of route being visited
		handle = routeInfoList data stored at this node
		item - reachabilityInfo data formed with route that is modified and the state
*/
func UpdateRouteReachabilityStatus(prefix patriciaDB.Prefix, //prefix of the node being traversed
	handle patriciaDB.Item, //data interface (routeInforRecordList) for this node
	item patriciaDB.Item) /*RouteReachabilityStatusInfo data */ (err error) {

	if handle == nil {
		logger.Err(fmt.Sprintln("nil handle"))
		return err
	}
	routeReachabilityStatusInfo := item.(RouteReachabilityStatusInfo)
	var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(routeReachabilityStatusInfo.destNet)
	if err != nil {
		logger.Err(fmt.Sprintln("Error getting IP from cidr: ", routeReachabilityStatusInfo.destNet))
		return err
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	destIpPrefix, err := getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
	if err != nil {
		logger.Err(fmt.Sprintln("Error getting ip prefix for ip:", ipAddrStr, " mask:", ipMaskStr))
		return err
	}
	logger.Debug(fmt.Sprintln("UpdateRouteReachabilityStatus network: ", routeReachabilityStatusInfo.destNet, " status:", routeReachabilityStatusInfo.status, "ip: ", ip.String(), " destIPPrefix: ", destIpPrefix, " ipMaskStr:", ipMaskStr))
	rmapInfoRecordList := handle.(RouteInfoRecordList)
	//for each of the routes for this destination, check if the nexthop ip matches destPrefix - which is the route being modified
	for k, v := range rmapInfoRecordList.routeInfoProtocolMap {
		logger.Debug(fmt.Sprintln("UpdateRouteReachabilityStatus - protocol: ", k))
		for i := 0; i < len(v); i++ {
			vPrefix, err := getNetowrkPrefixFromStrings(v[i].nextHopIp.String(), ipMaskStr)
			if err != nil {
				logger.Err(fmt.Sprintln("Error getting ip prefix for v[i].nextHopIp:", v[i].nextHopIp.String(), " mask:", ipMaskStr))
				return err
			}
			nextHopIntf := ribdInt.NextHopInfo{
				NextHopIp:      v[i].nextHopIp.String(),
				NextHopIfIndex: ribdInt.Int(v[i].nextHopIfIndex),
			}
			//is the next hop same as the modified route
			if bytes.Equal(vPrefix, destIpPrefix) {
				if routeReachabilityStatusInfo.status == "Down" && v[i].resolvedNextHopIpIntf.IsReachable == true {
					v[i].resolvedNextHopIpIntf.IsReachable = false
					rmapInfoRecordList.routeInfoProtocolMap[k] = v
					RouteInfoMap.Set(prefix, rmapInfoRecordList)
					//RouteServiceHandler.DBRouteCh <- RouteDBInfo {v[i], rmapInfoRecordList}//
					RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{v[i], rmapInfoRecordList})
					logger.Debug(fmt.Sprintln("Bringing down route : ip: ", v[i].networkAddr))
					RouteReachabilityStatusUpdate(k, RouteReachabilityStatusInfo{v[i].networkAddr, "Down", k, nextHopIntf})
					/*
					   The reachability status for this network has been updated, now check if there are routes dependent on
					   this prefix and call reachability status 
					*/
					if RouteServiceHandler.NextHopInfoMap[NextHopInfoKey{string(prefix)}].refCount > 0 {
						logger.Debug(fmt.Sprintln("There are dependent routes for this ip ", v[i].networkAddr))
						RouteInfoMap.VisitAndUpdate(UpdateRouteReachabilityStatus, RouteReachabilityStatusInfo{v[i].networkAddr, "Down", k, nextHopIntf})
					}
				} else if routeReachabilityStatusInfo.status == "Up" && v[i].resolvedNextHopIpIntf.IsReachable == false {
					logger.Debug(fmt.Sprintln("Bringing up route : ip: ", v[i].networkAddr))
					v[i].resolvedNextHopIpIntf.IsReachable = true
					rmapInfoRecordList.routeInfoProtocolMap[k] = v
					RouteInfoMap.Set(prefix, rmapInfoRecordList)
					//RouteServiceHandler.DBRouteCh <- RouteDBInfo {v[i], rmapInfoRecordList} 
					RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{v[i], rmapInfoRecordList})
					RouteReachabilityStatusUpdate(k, RouteReachabilityStatusInfo{v[i].networkAddr, "Up", k, nextHopIntf})
					/*
					   The reachability status for this network has been updated, now check if there are routes dependent on
					   this prefix and call reachability status 
					*/
					if RouteServiceHandler.NextHopInfoMap[NextHopInfoKey{string(prefix)}].refCount > 0 {
						logger.Debug(fmt.Sprintln("There are dependent routes for this ip ", v[i].networkAddr))
						RouteInfoMap.VisitAndUpdate(UpdateRouteReachabilityStatus, RouteReachabilityStatusInfo{v[i].networkAddr, "Up", k, nextHopIntf})
					}
				}
			}
		}
	}
	return err
}
/*
    Resolve and determine the immediate next hop info for a given ipAddr
*/
func ResolveNextHop(ipAddr string) (nextHopIntf ribdInt.NextHopInfo, resolvedNextHopIntf ribdInt.NextHopInfo, err error) {
	logger.Debug(fmt.Sprintln("ResolveNextHop for ", ipAddr))
	var prev_intf ribdInt.NextHopInfo
	nextHopIntf.NextHopIp = ipAddr
	prev_intf.NextHopIp = ipAddr
	first := true
	if ipAddr == "0.0.0.0" {
		nextHopIntf.IsReachable = true
		return nextHopIntf, nextHopIntf, err
	}
	ip := ipAddr
	for {
		intf, err := RouteServiceHandler.GetRouteReachabilityInfo(ip)
		if err != nil {
			logger.Err(fmt.Sprintln("next hop ", ip, " not reachable"))
			return nextHopIntf, nextHopIntf, err
		}
		if first {
			nextHopIntf = *intf
			first = false
			logger.Debug(fmt.Sprintln("First nexthop network is : ", nextHopIntf.Ipaddr))
		}
		logger.Debug(fmt.Sprintln("intf.nextHopIp ", intf.NextHopIp, " intf.Ipaddr:", intf.Ipaddr))
		if intf.NextHopIp == "0.0.0.0" {
			logger.Debug(fmt.Sprintln("Marking ip ", ip, " as reachable"))
			intf.NextHopIp = intf.Ipaddr
			intf.IsReachable = true
			prev_intf.IsReachable = true
			return nextHopIntf, prev_intf, err //*intf,err
		}
		ip = intf.NextHopIp
		prev_intf = *intf
	}
	return nextHopIntf, nextHopIntf, err
}
/*
    Function which determines the best route when a route is deleted or updated
*/
func SelectBestRoute(routeInfoRecordList RouteInfoRecordList) (addRouteList []RouteOpInfoRecord, deleteRouteList []RouteOpInfoRecord, newSelectedProtocol string) {
	logger.Debug(fmt.Sprintln("SelectBestRoute, the current selected route protocol is ", routeInfoRecordList.selectedRouteProtocol))
	tempSelectedProtocol := "INVALID"
	newSelectedProtocol = "INVALID"
	deleteRouteList = make([]RouteOpInfoRecord, 0)
	addRouteList = make([]RouteOpInfoRecord, 0)
	var routeOpInfoRecord RouteOpInfoRecord
	logger.Debug(fmt.Sprintln("len(protocolAdminDistanceSlice):", len(ProtocolAdminDistanceSlice)))
	/*
	   Build protocol admin distance slice based on the current admin distance values
	*/
	BuildProtocolAdminDistanceSlice()
	/*
	   go over the protocol admin distance slice, select the protocols from best to worst
	   and check if there are any routes configured with that protocol type
	   If yes, then verify if that route is eligible to be selected.
	   If yes, then check if it is the same protocol as the incoming protocol
	   If not, then delete all the routes configured with the old selected protocol in FIB
	   and configure the routes of the new selected type
	*/
	for i := 0; i < len(ProtocolAdminDistanceSlice); i++ {
		tempSelectedProtocol = ProtocolAdminDistanceSlice[i].Protocol
		logger.Debug(fmt.Sprintln("Best preferred protocol ", tempSelectedProtocol))
		routeInfoList := routeInfoRecordList.routeInfoProtocolMap[tempSelectedProtocol]
		if routeInfoList == nil || len(routeInfoList) == 0 {
			logger.Debug(fmt.Sprintln("No routes are configured with this protocol ", tempSelectedProtocol, " for this route"))
			tempSelectedProtocol = "INVALID"
			continue
		}
		tempSelectedProtocol = "INVALID"
		for j := 0; j < len(routeInfoList); j++ {
			routeInfoRecord := routeInfoList[j]
			policyRoute := ribdInt.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), IfIndex: ribdInt.Int(routeInfoRecord.nextHopIfIndex), Metric: ribdInt.Int(routeInfoRecord.metric), Prototype: ribdInt.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid: routeInfoRecordList.isPolicyBasedStateValid}
			entity, _ := buildPolicyEntityFromRoute(policyRoute, RouteParams{})
			actionList := PolicyEngineDB.PolicyEngineCheckActionsForEntity(entity, policyCommonDefs.PolicyConditionTypeProtocolMatch)
			if !PolicyEngineDB.ActionNameListHasAction(actionList, policyCommonDefs.PolicyActionTypeRouteDisposition, "Reject") {
				logger.Debug("atleast one of the routes of this protocol will not be rejected by the policy engine")
				tempSelectedProtocol = ProtocolAdminDistanceSlice[i].Protocol
				break
			}
		}
		if tempSelectedProtocol != "INVALID" {
			logger.Debug(fmt.Sprintln("Found a valid protocol ", tempSelectedProtocol))
			break
		}
	}
	if tempSelectedProtocol == routeInfoRecordList.selectedRouteProtocol {
		logger.Debug("The current protocol remains the new selected protocol")
		return addRouteList, deleteRouteList, newSelectedProtocol
	}
	if routeInfoRecordList.selectedRouteProtocol != "INVALID" {
		logger.Debug(fmt.Sprintln("Valid protocol currently selected as ", routeInfoRecordList.selectedRouteProtocol))
		for j := 0; j < len(routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]); j++ {
			routeOpInfoRecord.opType = FIBOnly
			routeOpInfoRecord.routeInfoRecord = routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][j]
			deleteRouteList = append(deleteRouteList, routeOpInfoRecord)
		}
	}
	if tempSelectedProtocol != "INVALID" {
		logger.Debug(fmt.Sprintln("New Valid protocol selected as ", tempSelectedProtocol))
		for j := 0; j < len(routeInfoRecordList.routeInfoProtocolMap[tempSelectedProtocol]); j++ {
			routeOpInfoRecord.opType = FIBOnly
			routeOpInfoRecord.routeInfoRecord = routeInfoRecordList.routeInfoProtocolMap[tempSelectedProtocol][j]
			logger.Debug(fmt.Sprintln("Adding route with nexthop ip ", routeOpInfoRecord.routeInfoRecord.nextHopIp.String(), "/", routeOpInfoRecord.routeInfoRecord.nextHopIfIndex))
			addRouteList = append(addRouteList, routeOpInfoRecord)
		}
		newSelectedProtocol = tempSelectedProtocol
	}
	return addRouteList, deleteRouteList, newSelectedProtocol
}

//this function is called when a route is being added after it has cleared import policies
func selectBestRouteOnAdd(routeInfoRecordList RouteInfoRecordList, routeInfoRecord RouteInfoRecord) (addRouteList []RouteOpInfoRecord, deleteRouteList []RouteOpInfoRecord, newSelectedProtocol string) {
	logger.Debug(fmt.Sprintln("selectBestRouteOnAdd current selected protocol = ", routeInfoRecordList.selectedRouteProtocol))
	deleteRouteList = make([]RouteOpInfoRecord, 0)
	addRouteList = make([]RouteOpInfoRecord, 0)
	newSelectedProtocol = routeInfoRecordList.selectedRouteProtocol
	newRouteProtocol := ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]
	add := false
	del := false
	var addrouteOpInfoRecord RouteOpInfoRecord
	var delrouteOpInfoRecord RouteOpInfoRecord
	
	if routeInfoRecordList.selectedRouteProtocol == "INVALID" {
		/*
		    Currently, no route has been selected
		*/
		if routeInfoRecord.protocol != PROTOCOL_NONE {
			logger.Debug("Selecting the new route because the current selected route is invalid")
			add = true
			addrouteOpInfoRecord.opType = FIBAndRIB
			newSelectedProtocol = newRouteProtocol
		}
	} else if ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance > ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance {
		/*
		    If the configured admin distance is more than the incoming route, add the route in RIB
		*/
		add = true
		addrouteOpInfoRecord.opType = RIBOnly
	} else if ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance < ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance {
		logger.Debug(fmt.Sprintln(" Selecting the new route because the admin distance of the new routetype ", newRouteProtocol, ":", ProtocolAdminDistanceMapDB[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]].configuredDistance, "is better than the selected route protocol ", routeInfoRecordList.selectedRouteProtocol, "'s admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol]))
		del = true
		add = true
		addrouteOpInfoRecord.opType = FIBAndRIB
		delrouteOpInfoRecord.opType = FIBOnly
		newSelectedProtocol = newRouteProtocol
	} else if ProtocolAdminDistanceMapDB[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]].configuredDistance == ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance {
		logger.Debug("Same admin distance ")
		if newRouteProtocol == routeInfoRecordList.selectedRouteProtocol {
			logger.Debug("Same protocol as the selected route")
			if routeInfoRecord.metric == routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0].metric {
				logger.Debug("Adding a same cost route as the current selected routes")
				if !newNextHopIP(routeInfoRecord.nextHopIp.String(), routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]) {
					logger.Debug("Not a new next hop ip, so do nothing")
				} else {
					logger.Debug("This is a new route with a new next hop IP")
					addrouteOpInfoRecord.opType = FIBAndRIB
					add = true
				}
			} else if routeInfoRecord.metric < routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0].metric {
				logger.Debug(fmt.Sprintln("New metric ", routeInfoRecord.metric, " is lower than the current metric ", routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0].metric))
				del = true
				delrouteOpInfoRecord.opType = FIBAndRIB
				add = true
				addrouteOpInfoRecord.opType = FIBAndRIB
			}
		} else {
			logger.Debug(fmt.Sprintln("Protocol ", newRouteProtocol, " has the same admin distance ", ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance, " as the protocol", routeInfoRecordList.selectedRouteProtocol, "'s configured admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance))
			if ProtocolAdminDistanceMapDB[newRouteProtocol].defaultDistance < ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].defaultDistance {
				logger.Debug(fmt.Sprintln("Protocol ", newRouteProtocol, " has lower default admin distance ", ProtocolAdminDistanceMapDB[newRouteProtocol].defaultDistance, " than the protocol", routeInfoRecordList.selectedRouteProtocol, "'s default admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].defaultDistance))
				del = true
				delrouteOpInfoRecord.opType = FIBOnly
				add = true
				addrouteOpInfoRecord.opType = FIBAndRIB
				newSelectedProtocol = newRouteProtocol
			} else {
				logger.Debug(fmt.Sprintln("Protocol ", newRouteProtocol, " has higher default admin distance ", ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance, " than the protocol", routeInfoRecordList.selectedRouteProtocol, "'s default admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance))
				add = true
				addrouteOpInfoRecord.opType = RIBOnly
			}
		}
	}
	logger.Debug(fmt.Sprintln("At the end of the route selection logic, add = ", add, " del = ", del))
	if add == true {
		addrouteOpInfoRecord.routeInfoRecord = routeInfoRecord
		addRouteList = append(addRouteList, addrouteOpInfoRecord)
	}
	if del == true {
		for i := 0; i < len(routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]); i++ {
			delrouteOpInfoRecord.routeInfoRecord = routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][i]
			deleteRouteList = append(deleteRouteList, delrouteOpInfoRecord)
		}
	}
	return addRouteList, deleteRouteList, newSelectedProtocol
}
/*
    This function adds the route in RIB RouteMap after it has cleared the import policies
	If the route being added is the selected route protocol and this function is called with export policy path, then 
	it adds the route in FIB and also calls the policyenginefilter with export path
	It then updates reachabilitystatus info of this route and dependent routes
	
	Flow to reach in and out of this function:
	event/user                       import                        accept                                                        export
	------------->ProcessRouteCreate--------->policyEngineFilter----------->createv4route------>selectV4Route------>addNewRoute--------->policyEngineFilter----------->export_actions(redistribute)
*/
func addNewRoute(destNetPrefix patriciaDB.Prefix,
	routeInfoRecord RouteInfoRecord,
	routeInfoRecordList RouteInfoRecordList,
	policyPath int) {
	policyPathStr := ""
	if policyPath == policyCommonDefs.PolicyPath_Export {
		policyPathStr = "Export"
	} else {
		policyPathStr = "Import"
	}
	logger.Debug(fmt.Sprintln(" addNewRoute for nwAddr: ", routeInfoRecord.networkAddr, "protocol ", routeInfoRecord.protocol, " policy path: ", policyPathStr, " next hop ip: ", routeInfoRecord.nextHopIp.String(), "/", routeInfoRecord.nextHopIfIndex))
	if destNetSlice != nil && (len(destNetSlice) > int(routeInfoRecord.sliceIdx)) { //&& bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNet)) {
		if bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNetPrefix) == false {
			logger.Debug(fmt.Sprintln("Unexpected destination network prefix ", destNetSlice[routeInfoRecord.sliceIdx].prefix, " found at the slice Idx ", routeInfoRecord.sliceIdx, " expected prefix ", destNetPrefix))
			return
		}
		//There is already an entry in the destNetSlice at the route index and was invalidated earlier because  of a link down of the nexthop intf of the route or if the route was deleted
		//In this case since the old route was invalid, there is nothing to delete
		logger.Debug(fmt.Sprintln("sliceIdx ", routeInfoRecord.sliceIdx))
		destNetSlice[routeInfoRecord.sliceIdx].isValid = true
	} else {
		logger.Debug(fmt.Sprintln("This is a new route for selectedProtocolType being added, create destNetSlice entry at index ", len(destNetSlice)))
		routeInfoRecord.sliceIdx = len(destNetSlice)
		localDBRecord := localDB{prefix: destNetPrefix, isValid: true, nextHopIp: routeInfoRecord.nextHopIp.String()}
		if destNetSlice == nil {
			destNetSlice = make([]localDB, 0)
		}
		destNetSlice = append(destNetSlice, localDBRecord)
	}
	if routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] == nil {
		routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = make([]RouteInfoRecord, 0)
	}
	if newNextHopIP(routeInfoRecord.nextHopIp.String(), routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]]) {
		routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = append(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]], routeInfoRecord)
	} else {
		//already existing route needs to be updated
		found, currRecord, idx := findRouteWithNextHop(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]], routeInfoRecord.nextHopIp.String())
		if !found {
			logger.Err(fmt.Sprintln("Unexpected error - did not find route with ip: ", routeInfoRecord.destNetIp.String(), " next hop: ", routeInfoRecord.nextHopIp.String()))
			return
		}
		//update the patriciaDB trie with the updated route info record list
		t1 := time.Now()
		currRecord.routeUpdatedTime = t1.String()
		currRecord.resolvedNextHopIpIntf.IsReachable = true
		routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]][idx] = currRecord
	}
    /*
	    Update route info in RouteMap
	*/
	RouteInfoMap.Set(patriciaDB.Prefix(destNetPrefix), routeInfoRecordList)

	if ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)] != routeInfoRecordList.selectedRouteProtocol {
		logger.Debug("This is not a selected route, so nothing more to do here")
		return
	}
	logger.Debug("This is a selected route, so install and parse through export policy engine")

	//RouteServiceHandler.DBRouteCh <- RouteDBInfo { routeInfoRecord, routeInfoRecordList}
	RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{routeInfoRecord, routeInfoRecordList})

	policyRoute := ribdInt.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), IfIndex: ribdInt.Int(routeInfoRecord.nextHopIfIndex), Metric: ribdInt.Int(routeInfoRecord.metric), Prototype: ribdInt.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid: routeInfoRecordList.isPolicyBasedStateValid}
	var params RouteParams
	params = BuildRouteParamsFromRouteInoRecord(routeInfoRecord)
	if policyPath == policyCommonDefs.PolicyPath_Export {
		routeInfoRecord.resolvedNextHopIpIntf.NextHopIp = routeInfoRecord.nextHopIp.String()
		routeInfoRecord.resolvedNextHopIpIntf.NextHopIfIndex = ribdInt.Int(routeInfoRecord.nextHopIfIndex)
		/*
		    Find resolved next hop
		*/
		nhIntf, resolvedNextHopIntf, res_err := ResolveNextHop(routeInfoRecord.nextHopIp.String())
		logger.Debug(fmt.Sprintln("nhIntf:ipAddr:mask = ", nhIntf.Ipaddr, ":", nhIntf.Mask, " nexthop ip :", routeInfoRecord.nextHopIp.String()))
		routeInfoRecord.resolvedNextHopIpIntf = resolvedNextHopIntf
		//call asicd to add
		if asicdclnt.IsConnected {
			logger.Debug(fmt.Sprintln("New route selected, call asicd to install a new route - ip", routeInfoRecord.destNetIp.String(), " mask ", routeInfoRecord.networkMask.String(), " nextHopIP ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp))
			RouteServiceHandler.AsicdRouteCh <- RIBdServerConfig{OrigConfigObject:routeInfoRecord,Op:"add"}
		}
		/*
		    Call Arp to resolve the next hop if this is not a connected route
		*/
		if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
			/*
			    Call arp resolve only if it has not yet been called for this next hop 
			*/
			if !arpResolveCalled(NextHopInfoKey{routeInfoRecord.resolvedNextHopIpIntf.NextHopIp}) {
				//call arpd to resolve the ip
				logger.Debug(fmt.Sprintln("Adding ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, " to ArpdRouteCh"))
				RouteServiceHandler.ArpdRouteCh <- RIBdServerConfig{OrigConfigObject:routeInfoRecord,Op:"add"}
			}
			/*
			    Update next hop map for this next hop ip
			*/
			updateNextHopMap(NextHopInfoKey{routeInfoRecord.resolvedNextHopIpIntf.NextHopIp}, add)
		}
		//update in the event log
		eventInfo := "Installed " + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)] + " route " + policyRoute.Ipaddr + ":" + policyRoute.Mask + " nextHopIp :" + routeInfoRecord.nextHopIp.String() + " in Hardware and RIB "
		t1 := time.Now()
		routeEventInfo := RouteEventInfo{timeStamp: t1.String(), eventInfo: eventInfo}
		localRouteEventsDB = append(localRouteEventsDB, routeEventInfo)

		//get the network address associated with the nexthop and update its refcount
		if res_err == nil {
			nhPrefix, err := getNetowrkPrefixFromStrings(nhIntf.Ipaddr, nhIntf.Mask)
			if err == nil {
				updateNextHopMap(NextHopInfoKey{string(nhPrefix)}, add)
			}
		}
		if routeInfoRecord.resolvedNextHopIpIntf.IsReachable {
			logger.Debug(fmt.Sprintln("Mark this network reachable"))
			nextHopIntf := ribdInt.NextHopInfo{
				NextHopIp:      routeInfoRecord.nextHopIp.String(),
				NextHopIfIndex: ribdInt.Int(routeInfoRecord.nextHopIfIndex),
			}
			//check if there are routes depending on this network as next hop
			if RouteServiceHandler.NextHopInfoMap[NextHopInfoKey{string(destNetPrefix)}].refCount > 0 {
				routeReachabilityStatusInfo := RouteReachabilityStatusInfo{routeInfoRecord.networkAddr, "Up", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], nextHopIntf}
				RouteReachabilityStatusUpdate(routeReachabilityStatusInfo.protocol, routeReachabilityStatusInfo)
				RouteInfoMap.VisitAndUpdate(UpdateRouteReachabilityStatus, routeReachabilityStatusInfo)
			}
		}
	}
	params.deleteType = Invalid
	PolicyEngineFilter(policyRoute, policyPath, params)
}
func addNewRouteList(destNetPrefix patriciaDB.Prefix,
	addRouteList []RouteOpInfoRecord,
	routeInfoRecordList RouteInfoRecordList,
	policyPath int) {
	logger.Debug("addNewRoutes")
	for i := 0; i < len(addRouteList); i++ {
		logger.Debug(fmt.Sprintln("Calling addNewRoute for next hop ip: ", addRouteList[i].routeInfoRecord.nextHopIp.String(), "/", addRouteList[i].routeInfoRecord.nextHopIfIndex))
		addNewRoute(destNetPrefix, addRouteList[i].routeInfoRecord, routeInfoRecordList, policyPath)
	}
}

//note: selectedrouteProtocol should not have been set to INVALID by either of the selects when this function is called
func deleteRoute(destNetPrefix patriciaDB.Prefix, //route prefix of the route being deleted
	routeInfoRecord RouteInfoRecord,              //route info record of the route being deleted
	routeInfoRecordList RouteInfoRecordList,      
	policyPath int, //Import/Export
	delType int,    //FIBOnly/RIBAndFIB
	) {
		
	logger.Debug(" deleteRoute")
	deleteNode := true
	nodeDeleted := false
	if destNetSlice == nil || int(routeInfoRecord.sliceIdx) >= len(destNetSlice) {
		logger.Debug(fmt.Sprintln("Destination slice not found at the expected slice index ", routeInfoRecord.sliceIdx))
		return
	}
	destNetSlice[routeInfoRecord.sliceIdx].isValid = false //invalidate this entry in the local db
	//the following operations delete this node from the RIB DB
	if delType == FIBAndRIB {
		logger.Debug("Del type = FIBAndRIB, so delete the entry in RIB DB")
		routeInfoList := routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]]
		found, _, index := findRouteWithNextHop(routeInfoList, routeInfoRecord.nextHopIp.String())
		if !found || index == -1 {
			logger.Debug("Invalid nextHopIP")
			return
		}
		logger.Debug(fmt.Sprintln("Found the route at index ", index))
		if len(routeInfoList) <= index+1 {
			routeInfoList = routeInfoList[:index]
		} else {
			routeInfoList = append(routeInfoList[:index], routeInfoList[index+1:]...)
		}
		routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = routeInfoList
		if len(routeInfoList) == 0 {
			/*
			    If all the routes from this protocol have been deleted
			*/
			logger.Debug(fmt.Sprintln("All routes for this destination from protocol ", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], " deleted"))
			routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = nil
			deleteNode = true
			/*
			    Go over the routeInfoProtocolMap for this network to see if routes from other protocol are configured
			*/
			for k, v := range routeInfoRecordList.routeInfoProtocolMap {
				if v != nil && len(v) != 0 {
					logger.Debug(fmt.Sprintln("There are still other protocol ", k, " routes for this destination"))
					deleteNode = false
				}
			}
			if deleteNode == true {
				logger.Debug(fmt.Sprintln("Route deleted for this destination, traverse dependent routes to update routeReachability status"))
				//check if there are routes dependent on this network
				if RouteServiceHandler.NextHopInfoMap[NextHopInfoKey{string(destNetPrefix)}].refCount > 0 {
					nextHopIntf := ribdInt.NextHopInfo{}
					routeReachabilityStatusInfo := RouteReachabilityStatusInfo{routeInfoRecord.networkAddr, "Down", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], nextHopIntf}
					RouteReachabilityStatusUpdate(routeReachabilityStatusInfo.protocol, routeReachabilityStatusInfo)
					RouteInfoMap.VisitAndUpdate(UpdateRouteReachabilityStatus, routeReachabilityStatusInfo)
				}
				//get the network address associated with the nexthop and update its refcount
				nhIntf, err := RouteServiceHandler.GetRouteReachabilityInfo(routeInfoRecord.nextHopIp.String())
				if err == nil {
					nhPrefix, err := getNetowrkPrefixFromStrings(nhIntf.Ipaddr, nhIntf.Mask)
					if err == nil {
						updateNextHopMap(NextHopInfoKey{string(nhPrefix)}, del)
					}
				}
				/*
				    delete the route in state db and routeInfoMap
				*/
				RouteServiceHandler.DelIPv4RouteStateEntryFromDB(RouteDBInfo{routeInfoRecord, routeInfoRecordList})
				//RouteServiceHandler.DBRouteCh <- RouteDBInfo {routeInfoRecord,routeInfoRecordList}
				RouteInfoMap.Delete(destNetPrefix)
				nodeDeleted = true
			}
		}
		if !nodeDeleted {
			logger.Debug(fmt.Sprintln("node not deleted, write the updated list of len ", len(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]])))
			//RouteServiceHandler.DBRouteCh <- RouteDBInfo{routeInfoRecord, routeInfoRecordList}//.WriteIPv4RouteStateEntryToDB(routeInfoRecord, routeInfoRecordList)
			RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{routeInfoRecord, routeInfoRecordList})
			RouteInfoMap.Set(destNetPrefix, routeInfoRecordList)
		}
	} else if delType == FIBOnly { 
	    /*
		    in cases where the interface goes down
		*/
		logger.Debug("Del type = FIBOnly, so don't delete the entry in RIB DB")
		/*
		    Mark the network not reachable
		*/
		routeInfoList := routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]]
		for i := 0; i < len(routeInfoList); i++ {
			routeInfoList[i].resolvedNextHopIpIntf.IsReachable = false
		}
		routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = routeInfoList
		logger.Debug(fmt.Sprintln("Route deleted for this destination, traverse dependent routes to update routeReachability status"))
		//check if there are routes dependent on this network
		if RouteServiceHandler.NextHopInfoMap[NextHopInfoKey{string(destNetPrefix)}].refCount > 0 {
			nextHopIntf := ribdInt.NextHopInfo{}
			routeReachabilityStatusInfo := RouteReachabilityStatusInfo{routeInfoRecord.networkAddr, "Down", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], nextHopIntf}
			RouteReachabilityStatusUpdate(routeReachabilityStatusInfo.protocol, routeReachabilityStatusInfo)
			RouteInfoMap.VisitAndUpdate(UpdateRouteReachabilityStatus, routeReachabilityStatusInfo)
		}
		//get the network address associated with the nexthop and update its refcount
		nhIntf, err := RouteServiceHandler.GetRouteReachabilityInfo(routeInfoRecord.nextHopIp.String())
		if err == nil {
			nhPrefix, err := getNetowrkPrefixFromStrings(nhIntf.Ipaddr, nhIntf.Mask)
			if err == nil {
				updateNextHopMap(NextHopInfoKey{string(nhPrefix)}, del)
			}
		}
		//RouteServiceHandler.DBRouteCh <- RouteDBInfo{routeInfoRecord, routeInfoRecordList}
		RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{routeInfoRecord, routeInfoRecordList})
		RouteInfoMap.Set(destNetPrefix, routeInfoRecordList)
	}
	if routeInfoRecordList.selectedRouteProtocol != ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)] {
		logger.Debug("This is not the selected protocol, nothing more to do here")
		return
	}
	policyRoute := ribdInt.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), IfIndex: ribdInt.Int(routeInfoRecord.nextHopIfIndex), Metric: ribdInt.Int(routeInfoRecord.metric), Prototype: ribdInt.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid: routeInfoRecordList.isPolicyBasedStateValid}
	if policyPath != policyCommonDefs.PolicyPath_Export {
		logger.Debug("Expected export path for delete op")
		return
	}
	logger.Debug("This is the selected protocol")
	//delete in asicd
	if asicdclnt.IsConnected {
		logger.Debug(fmt.Sprintln("Calling asicd to delete this route- ip", routeInfoRecord.destNetIp.String(), " mask ", routeInfoRecord.networkMask.String(), " nextHopIP ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp))
			RouteServiceHandler.AsicdRouteCh <- RIBdServerConfig{OrigConfigObject:routeInfoRecord,Op:"del"}
	}
	if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
		if !arpResolveCalled(NextHopInfoKey{routeInfoRecord.resolvedNextHopIpIntf.NextHopIp}) {
			logger.Debug(fmt.Sprintln("ARP resolve was never called for ", routeInfoRecord.nextHopIp.String()))
		} else {
			refCount := updateNextHopMap(NextHopInfoKey{routeInfoRecord.resolvedNextHopIpIntf.NextHopIp}, del)
			if refCount == 0 {
				logger.Debug(fmt.Sprintln("Adding ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, " to ArpdRouteCh"))
				RouteServiceHandler.ArpdRouteCh <- RIBdServerConfig { OrigConfigObject:routeInfoRecord,Op:"del"}
			}
		}
	}
	//update in the event log
	delStr := "Route Uninstalled in Hardware "
	if delType == FIBAndRIB {
		delStr = delStr + " and deleted from RIB "
	}
	eventInfo := delStr + ":" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)] + " " + policyRoute.Ipaddr + ":" + policyRoute.Mask + " nextHopIp :" + routeInfoRecord.nextHopIp.String()
	t1 := time.Now()
	routeEventInfo := RouteEventInfo{timeStamp: t1.String(), eventInfo: eventInfo}
	localRouteEventsDB = append(localRouteEventsDB, routeEventInfo)

	var params RouteParams
	params = BuildRouteParamsFromRouteInoRecord(routeInfoRecord)
	params.createType = Invalid
	policyRoute.PolicyList = routeInfoRecordList.policyList
	PolicyEngineFilter(policyRoute, policyPath, params)
}
func deleteRoutes(destNetPrefix patriciaDB.Prefix,
	deleteRouteList []RouteOpInfoRecord,
	routeInfoRecordList RouteInfoRecordList,
	policyPath int) {
	logger.Debug("deleteRoutes")
	for i := 0; i < len(deleteRouteList); i++ {
		deleteRoute(destNetPrefix, deleteRouteList[i].routeInfoRecord, routeInfoRecordList, policyPath, deleteRouteList[i].opType)
	}
}
/*
    This function is called whenever a route is added or deleted. In either of the cases, 
	this function selects the best route to be programmed in FIB.
*/
func SelectV4Route(destNetPrefix patriciaDB.Prefix,
	routeInfoRecordList RouteInfoRecordList, //the current list of routes for this prefix
	routeInfoRecord RouteInfoRecord, //the route to be added or deleted or invalidated or validated
	op ribd.Int,                     //add or delete of the route
	opType int,                       //whether this operation is at FIB only or RIBAndFIB
	) (err error) {
	logger.Debug(fmt.Sprintln("Selecting the best Route for destNetPrefix ", destNetPrefix))
	if op == add {
		logger.Debug("Op is to add the new route")
		_, deleteRouteList, newSelectedProtocol := selectBestRouteOnAdd(routeInfoRecordList, routeInfoRecord)
		/*
		   If any of the routes need to be deleted as part of adding the new route, call delete of those routes
		*/
		if len(deleteRouteList) > 0 {
			deleteRoutes(destNetPrefix, deleteRouteList, routeInfoRecordList, policyCommonDefs.PolicyPath_Export)
		}
		routeInfoRecordList.selectedRouteProtocol = newSelectedProtocol
		addNewRoute(destNetPrefix, routeInfoRecord, routeInfoRecordList, policyCommonDefs.PolicyPath_Export)
	} else if op == del {
		logger.Debug("Op is to delete new route")
		deleteRoute(destNetPrefix, routeInfoRecord, routeInfoRecordList, policyCommonDefs.PolicyPath_Export, opType)
		addRouteList, _, newSelectedProtocol := SelectBestRoute(routeInfoRecordList)
		routeInfoRecordList.selectedRouteProtocol = newSelectedProtocol
		if len(addRouteList) > 0 {
			addNewRouteList(destNetPrefix, addRouteList, routeInfoRecordList, policyCommonDefs.PolicyPath_Import)
		}
	}
	return err
}
func updateBestRoute(destNetPrefix patriciaDB.Prefix, routeInfoRecordList RouteInfoRecordList) {
	logger.Debug(fmt.Sprintln("updateBestRoute for ip network ", destNetPrefix))
	addRouteList, deleteRouteList, newSelectedProtocol := SelectBestRoute(routeInfoRecordList)
	if len(deleteRouteList) > 0 {
		logger.Debug(fmt.Sprintln(len(deleteRouteList), " to be deleted"))
		deleteRoutes(destNetPrefix, deleteRouteList, routeInfoRecordList, policyCommonDefs.PolicyPath_Export)
	}
	routeInfoRecordList.selectedRouteProtocol = newSelectedProtocol
	if len(addRouteList) > 0 {
		logger.Debug(fmt.Sprintln("New ", len(addRouteList), " to be added"))
		addNewRouteList(destNetPrefix, addRouteList, routeInfoRecordList, policyCommonDefs.PolicyPath_Import)
	}
}

/**
   This function is called when :
 - a user/routing protocol installs a new route. In that case, addType will be RIBAndFIB
 - when a operationally down link comes up. In this case, the addType will be FIBOnly because on a link down, the route is still preserved in the RIB database and only deleted from FIB (Asic)
**/
func createV4Route(destNetIp string,
	networkMask string,
	metric ribd.Int,
	weight ribd.Int,
	nextHopIp string,
	nextHopIfIndex ribd.Int,
	routeType ribd.Int,
	addType ribd.Int,
	policyStateChange int,
	sliceIdx ribd.Int) (rc ribd.Int, err error) {
	logger.Debug(fmt.Sprintln("createV4Route for ip ", destNetIp, " mask ", networkMask, " next hop ip ", nextHopIp, " nextHopIfIndex : ", nextHopIfIndex, " addType ", addType, " weight ", weight))

	callSelectRoute := false
	destNetIpAddr, err := getIP(destNetIp)
	if err != nil {
		logger.Debug("destNetIpAddr invalid")
		return 0, err
	}
	networkMaskAddr, err := getIP(networkMask)
	if err != nil {
		logger.Debug("networkMaskAddr invalid")
		return 0, err
	}
	nextHopIpAddr, err := getIP(nextHopIp)
	if err != nil {
		logger.Debug("nextHopIpAddr invalid")
		return 0, err
	}
	prefixLen, err := getPrefixLen(networkMaskAddr)
	if err != nil {
		return -1, err
	}
	destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		return -1, err
	}
	routePrototype := int8(routeType)
	nwAddr := (destNetIpAddr.Mask(net.IPMask(networkMaskAddr))).String() + "/" + strconv.Itoa(prefixLen)
	
	routeInfoRecord := RouteInfoRecord{destNetIp: destNetIpAddr, networkMask: networkMaskAddr, protocol: routePrototype, nextHopIp: nextHopIpAddr, networkAddr: nwAddr, nextHopIfIndex: nextHopIfIndex, metric: metric, sliceIdx: int(sliceIdx), weight: weight}

	policyRoute := ribdInt.Routes{Ipaddr: destNetIp, Mask: networkMask, NextHopIp: nextHopIp, IfIndex: ribdInt.Int(nextHopIfIndex), Metric: ribdInt.Int(metric), Prototype: ribdInt.Int(routeType), Weight: ribdInt.Int(weight)}
	routeInfoRecord.resolvedNextHopIpIntf.NextHopIp = routeInfoRecord.nextHopIp.String()
	routeInfoRecord.resolvedNextHopIpIntf.NextHopIfIndex = ribdInt.Int(routeInfoRecord.nextHopIfIndex)

	nhIntf, resolvedNextHopIntf, res_err := ResolveNextHop(routeInfoRecord.nextHopIp.String())
	routeInfoRecord.resolvedNextHopIpIntf = resolvedNextHopIntf
	logger.Info(fmt.Sprintln("nhIntf ipaddr/mask: ", nhIntf.Ipaddr, ":", nhIntf.Mask, " resolvedNex ", resolvedNextHopIntf.NextHopIp, " nexthop ", nextHopIp))

	routeInfoRecord.routeCreatedTime = time.Now().String()

	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		/*
		    no routes for this destination are currently configured
		*/
		if addType == FIBOnly {
			logger.Debug("route record list not found in RIB")
			err = errors.New("Unexpected: route record list not found in RIB")
			return 0, err
		}
		var newRouteInfoRecordList RouteInfoRecordList
		newRouteInfoRecordList.routeInfoProtocolMap = make(map[string][]RouteInfoRecord)
		newRouteInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeType)]] = make([]RouteInfoRecord, 0)
		newRouteInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeType)]] = append(newRouteInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeType)]], routeInfoRecord)
		newRouteInfoRecordList.selectedRouteProtocol = ReverseRouteProtoTypeMapDB[int(routeType)]

		if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
			newRouteInfoRecordList.isPolicyBasedStateValid = false
		} else if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
			newRouteInfoRecordList.isPolicyBasedStateValid = true
		}
		if ok := RouteInfoMap.Insert(destNet, newRouteInfoRecordList); ok != true {
			logger.Debug(" return value not ok")
		}
		localDBRecord := localDB{prefix: destNet, isValid: true, nextHopIp: nextHopIp}
		if destNetSlice == nil {
			destNetSlice = make([]localDB, 0)
		}
		destNetSlice = append(destNetSlice, localDBRecord)
		//call asicd
		if asicdclnt.IsConnected {
			logger.Debug(fmt.Sprintln("New route selected, call asicd to install a new route - ip", routeInfoRecord.destNetIp.String(), " mask ", routeInfoRecord.networkMask.String(), " nextHopIP ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp))
			RouteServiceHandler.AsicdRouteCh <- RIBdServerConfig{OrigConfigObject:routeInfoRecord,Op:"add"}
		}
		if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
			if !arpResolveCalled(NextHopInfoKey{routeInfoRecord.resolvedNextHopIpIntf.NextHopIp}) {
				//call arpd to resolve the ip
				logger.Debug(fmt.Sprintln("Adding ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, " to ArpdRouteCh"))
				RouteServiceHandler.ArpdRouteCh <- RIBdServerConfig{OrigConfigObject:routeInfoRecord,Op:"add"}
			}
			//update the ref count for the resolved next hop ip
			updateNextHopMap(NextHopInfoKey{routeInfoRecord.resolvedNextHopIpIntf.NextHopIp}, add)
		}
		//RouteServiceHandler.DBRouteCh <- RouteDBInfo{routeInfoRecord, newRouteInfoRecordList}//.WriteIPv4RouteStateEntryToDB(routeInfoRecord, routeInfoRecordList)
		RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{routeInfoRecord, newRouteInfoRecordList})
		//update in the event log
		eventInfo := "Installed " + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)] + " route " + policyRoute.Ipaddr + ":" + policyRoute.Mask + " nextHopIp :" + routeInfoRecord.nextHopIp.String() + " in Hardware and RIB "
		t1 := time.Now()
		routeEventInfo := RouteEventInfo{timeStamp: t1.String(), eventInfo: eventInfo}
		localRouteEventsDB = append(localRouteEventsDB, routeEventInfo)

		//update the ref count for the next hop ip
		//nhIntf,err := routeServiceHandler.GetRouteReachabilityInfo(routeInfoRecord.nextHopIp.String())
		if res_err == nil {
			nhPrefix, err := getNetowrkPrefixFromStrings(nhIntf.Ipaddr, nhIntf.Mask)
			if err == nil {
				logger.Debug(fmt.Sprintln("network address of the nh route: ", nhPrefix))
				updateNextHopMap(NextHopInfoKey{string(nhPrefix)}, add)
			}
		}
		if routeInfoRecord.resolvedNextHopIpIntf.IsReachable {
			logger.Debug(fmt.Sprintln("Mark this network reachable"))
			nextHopIntf := ribdInt.NextHopInfo{
				NextHopIp:      routeInfoRecord.nextHopIp.String(),
				NextHopIfIndex: ribdInt.Int(routeInfoRecord.nextHopIfIndex),
			}
			if RouteServiceHandler.NextHopInfoMap[NextHopInfoKey{string(destNet)}].refCount > 0 {
				routeReachabilityStatusInfo := RouteReachabilityStatusInfo{routeInfoRecord.networkAddr, "Up", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], nextHopIntf}
				RouteReachabilityStatusUpdate(routeReachabilityStatusInfo.protocol, routeReachabilityStatusInfo)
				//If there are dependent routes for this ip, then bring them up
				RouteInfoMap.VisitAndUpdate(UpdateRouteReachabilityStatus, routeReachabilityStatusInfo)
			}
		}

		var params RouteParams
		params = BuildRouteParamsFromRouteInoRecord(routeInfoRecord)
		params.createType = addType
		params.deleteType = Invalid
		policyRoute.IsPolicyBasedStateValid = newRouteInfoRecordList.isPolicyBasedStateValid
		PolicyEngineFilter(policyRoute, policyCommonDefs.PolicyPath_Export, params)
	} else {
		logger.Debug("routeInfoRecordListItem not nil")
		routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
		found := IsRoutePresent(routeInfoRecordList, ReverseRouteProtoTypeMapDB[int(routeType)])
		if found && (addType == FIBAndRIB) {
			routeInfoList := routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]]
			logger.Debug(fmt.Sprintln("Trying to create a duplicate route of protocol type ", ReverseRouteProtoTypeMapDB[int(routePrototype)]))
			if routeInfoList[0].metric > metric {
				logger.Debug("New route has a better metric")
				//delete all existing routes
				//call asicd to delete if it is the selected protocol
				//add this new route and configure in asicd
				logger.Debug("Adding a better cost route for the selected route")
				callSelectRoute = true
			} else if routeInfoList[0].metric == metric {
				if !newNextHopIP(nextHopIp, routeInfoList) {
					logger.Debug("same cost and next hop ip, so reject this route")
					err = errors.New("Duplicate route creation")
					return 0, err
				}
				//adding equal cost route
				//routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]] = append(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]], routeInfoRecord)
				// if routeInfoRecordList.selectedRouteProtocol == ReverseRouteProtoTypeMapDB[int(routePrototype)] {
				logger.Debug("Adding a equal cost route for the selected route")
				callSelectRoute = true
				//}
			} else { //if metric > routeInfoRecordList.routeInfoList[idx].metric
				logger.Debug("Duplicate route creation with higher cost, rejecting the route")
				err = errors.New("Duplicate route creation with higher cost, rejecting the route")
				return 0, err
			}
		} else if !found {
			if addType != FIBOnly {
				callSelectRoute = true
				//routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]] = make([]RouteInfoRecord,0)
				//routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]] = append(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]], routeInfoRecord)
				//routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList, routeInfoRecord)
			}
		} else {
			callSelectRoute = true
		}
		if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
			routeInfoRecordList.isPolicyBasedStateValid = false
		} else if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
			routeInfoRecordList.isPolicyBasedStateValid = true
		}
		if callSelectRoute {
			err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, add, int(addType)) //, len(routeInfoRecordList.routeInfoList)-1)
		}
	}
	if addType != FIBOnly && routePrototype == ribdCommonDefs.CONNECTED { //PROTOCOL_CONNECTED {
		updateConnectedRoutes(destNetIp, networkMask, nextHopIp, nextHopIfIndex, add, sliceIdx)
	}
	return 0, err

}

func (m RIBDServer) ProcessRouteCreateConfig(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Debug(fmt.Sprintln("ProcessRouteCreate: Received create route request for ip ", cfg.DestinationNw, " mask ", cfg.NetworkMask, " number of next hops: ", len(cfg.NextHop)))
    newCfg := ribd.IPv4Route{ 
	    DestinationNw: cfg.DestinationNw,
		NetworkMask  : cfg.NetworkMask,
		Protocol     : cfg.Protocol,
		Cost         : cfg.Cost,
		NullRoute    : cfg.NullRoute,
	}
	for i := 0; i< len(cfg.NextHop) ; i++ {
		logger.Debug(fmt.Sprintln("nexthop info: ip: ", cfg.NextHop[i].NextHopIp, " intref: ", cfg.NextHop[i].NextHopIntRef))
        nh := ribd.NextHopInfo {
			NextHopIp     : cfg.NextHop[i].NextHopIp,
			NextHopIntRef : cfg.NextHop[i].NextHopIntRef,
			Weight        : cfg.NextHop[i].Weight, 
		}		
		newCfg.NextHop = make([]*ribd.NextHopInfo,0)
		newCfg.NextHop = append(newCfg.NextHop, &nh)
	    policyRoute := BuildPolicyRouteFromribdIPv4Route(&newCfg)
	    params := BuildRouteParamsFromribdIPv4Route(&newCfg, FIBAndRIB, Invalid, len(destNetSlice))
	    logger.Debug(fmt.Sprintln("createType = ", params.createType, "deleteType = ", params.deleteType))
	    PolicyEngineFilter(policyRoute, policyCommonDefs.PolicyPath_Import, params)
	}

	return true, err
}

/**
   This function is called when:
   -  a user/protocol deletes a route - delType = FIBAndRIB
   - when a link goes down and we have connected routes on that link - delType = FIBOnly
**/
func deleteV4Route(destNetIp string,
	networkMask string,
	routeType string,
	nextHopIP string,
	delType ribd.Int,
	policyStateChange int) (rc ribd.Int, err error) {
	logger.Debug(fmt.Sprintln("deleteV4Route  with del type ", delType))

	destNetIpAddr, err := getIP(destNetIp)
	if err != nil {
		return 0, err
	}
	networkMaskAddr, err := getIP(networkMask)
	if err != nil {
		return 0, err
	}
	destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		return -1, err
	}
	logger.Debug(fmt.Sprintln("destNet = ", destNet))
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		logger.Err(fmt.Sprintln("Destnet ", destNet , " not found"))
		return 0, nil
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		return 0, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found := IsRoutePresent(routeInfoRecordList, routeType)
	if !found {
		logger.Err(fmt.Sprintln("Route with protocol ", routeType, " not found"))
		return 0, err
	}
	if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
		routeInfoRecordList.isPolicyBasedStateValid = false
	} else if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
		routeInfoRecordList.isPolicyBasedStateValid = true
	}
	found, routeInfoRecord, _ := findRouteWithNextHop(routeInfoRecordList.routeInfoProtocolMap[routeType], nextHopIP)
	if !found {
		logger.Err(fmt.Sprintln("Route with nextHop IP ", nextHopIP, " not found"))
		return 0, err
	}
	/*
	    Call selectv4Route to select the best route
	*/
	SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, del, int(delType))

	if routeType == "CONNECTED" { //PROTOCOL_CONNECTED {
		if delType == FIBOnly { //link gone down, just invalidate the connected route
			updateConnectedRoutes(destNetIp, networkMask, "", 0, invalidate, 0)
		} else {
			updateConnectedRoutes(destNetIp, networkMask, "", 0, del, 0)
		}
	}
	return 0, err
}

/*func (m RIBDServer) DeleteV4Route(destNetIp string,
networkMask string,
routeTypeString string,
nextHopIP string) (rc ribd.Int, err error) {*/
func (m RIBDServer) ProcessRouteDeleteConfig(cfg *ribd.IPv4Route) (val bool, err error) {
	logger.Debug(fmt.Sprintln("ProcessRouteDeleteConfig:Received Route Delete request for ", cfg.DestinationNw, ":", cfg.NetworkMask, "number of nextHops:", len(cfg.NextHop), "Protocol ", cfg.Protocol))
	if !RouteServiceHandler.AcceptConfig {
		logger.Debug("Not ready to accept config")
		//return 0,err
	}
	for i := 0; i< len(cfg.NextHop) ; i++ {
		logger.Debug(fmt.Sprintln("nexthop info: ip: ", cfg.NextHop[i].NextHopIp, " intref: ", cfg.NextHop[i].NextHopIntRef))
	    _, err = deleteV4Route(cfg.DestinationNw, cfg.NetworkMask, cfg.Protocol, cfg.NextHop[i].NextHopIp, FIBAndRIB, ribdCommonDefs.RoutePolicyStateChangetoInValid)
	}
	return true, err
}

func (m RIBDServer) ProcessRoutePatchUpdateConfig(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, op []*ribd.PatchOpInfo) (val bool, err error) {
	logger.Debug(fmt.Sprintln("ProcessRoutePatchUpdateConfig:Received update route request with number of patch ops: ", len(op)))
	if !RouteServiceHandler.AcceptConfig {
		logger.Debug("Not ready to accept config")
		//return err
	}
	destNet, err := getNetowrkPrefixFromStrings(origconfig.DestinationNw, origconfig.NetworkMask)
	if err != nil {
		logger.Debug(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return val, err
	}
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return val, err
	}
    for idx := 0;idx < len(op);idx++ {
		switch op[idx].Path {
			case "NextHop":
			    logger.Debug("Patch update for next hop")
				/*newconfig should only have the next hops that have to be added or deleted*/
				newconfig.NextHop = make([]*ribd.NextHopInfo,0)
				logger.Debug(fmt.Sprintln("value = ", op[idx].Value))
				valueObjArr := []ribd.NextHopInfo{}
	             err = json.Unmarshal([]byte (op[idx].Value),&valueObjArr)
	             if err != nil {
		             logger.Debug(fmt.Sprintln("error unmarshaling value:",err))
		             return val,errors.New(fmt.Sprintln("error unmarshaling value:",err))
	             }
				logger.Debug(fmt.Sprintln("Number of nextHops:", len(valueObjArr)))
				for _,val := range valueObjArr {
					logger.Debug(fmt.Sprintln("nextHop info: ip - ", val.NextHopIp, " intf: ", val.NextHopIntRef, " wt:", val.Weight))
					//wt,_ := strconv.Atoi((op[idx].Value[j]["Weight"]))
					nh := ribd.NextHopInfo {
					    NextHopIp : val.NextHopIp,
					    NextHopIntRef : val.NextHopIntRef,
					    Weight      : val.Weight,
					}
					newconfig.NextHop = append(newconfig.NextHop,&nh)
				}
				switch op[idx].Op {
					case "add":
					    m.ProcessRouteCreateConfig(newconfig)
					case "remove":
					    m.ProcessRouteDeleteConfig(newconfig)
					default:
					    logger.Err(fmt.Sprintln("Operation ", op[idx].Op, " not supported"))
				}
			default:
			    logger.Err(fmt.Sprintln("Patch update for attribute:", op[idx].Path, " not supported"))
		}
	}
	return val, err
}

func (m RIBDServer) ProcessRouteUpdateConfig(origconfig *ribd.IPv4Route, newconfig *ribd.IPv4Route, attrset []bool) (val bool, err error) {
	logger.Debug(fmt.Sprintln("ProcessRouteUpdateConfig:Received update route request "))
	if !RouteServiceHandler.AcceptConfig {
		logger.Debug("Not ready to accept config")
		//return err
	}
	destNet, err := getNetowrkPrefixFromStrings(origconfig.DestinationNw, origconfig.NetworkMask)
	if err != nil {
		logger.Debug(fmt.Sprintln(" getNetowrkPrefixFromStrings returned err ", err))
		return val, err
	}
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New(fmt.Sprintln("No route found for ip ", destNet))
		return val, err
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Debug(fmt.Sprintln("No route for destination network", destNet))
		return val, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	callUpdate := true
	if attrset != nil {
		found, routeInfoRecord, index := findRouteWithNextHop(routeInfoRecordList.routeInfoProtocolMap[origconfig.Protocol], origconfig.NextHop[0].NextHopIp)
		if !found || index == -1 {
			logger.Debug("Invalid nextHopIP")
			return val, err
		}
		objTyp := reflect.TypeOf(*origconfig)
		for i := 0; i < objTyp.NumField(); i++ {
			objName := objTyp.Field(i).Name
			if attrset[i] {
				logger.Debug(fmt.Sprintf("ProcessRouteUpdateConfig (server): changed ", objName))
				if objName == "NextHop" {
					if len(newconfig.NextHop) == 0 {
						logger.Err("Must specify next hop")
						return val, err
					} else {
						nextHopIpAddr, err := getIP(newconfig.NextHop[0].NextHopIp)
						if err != nil {
							logger.Debug("nextHopIpAddr invalid")
							return val, errors.New("Invalid next hop")
						}
						logger.Debug(fmt.Sprintln("Update the next hop info old ip: ", origconfig.NextHop[0].NextHopIp, " new value: ", newconfig.NextHop[0].NextHopIp, " weight : ", newconfig.NextHop[0].Weight))
						routeInfoRecord.nextHopIp = nextHopIpAddr
						routeInfoRecord.weight = ribd.Int(newconfig.NextHop[0].Weight)
						if newconfig.NextHop[0].NextHopIntRef != "" {
							nextHopIntRef, _ := strconv.Atoi(newconfig.NextHop[0].NextHopIntRef)
							routeInfoRecord.nextHopIfIndex = ribd.Int(nextHopIntRef)
						}
					}
				}
				if objName == "Cost" {
					routeInfoRecord.metric = ribd.Int(newconfig.Cost)
				}
				/*				if objName == "OutgoingInterface" {
								nextHopIfIndex, _ := strconv.Atoi(newconfig.OutgoingInterface)
								routeInfoRecord.nextHopIfIndex = ribd.Int(nextHopIfIndex)
								callUpdate = false
							}*/
			}
		}
		routeInfoRecordList.routeInfoProtocolMap[origconfig.Protocol][index] = routeInfoRecord
		RouteInfoMap.Set(destNet, routeInfoRecordList)
		//RouteServiceHandler.DBRouteAddCh <- RouteDBInfo{routeInfoRecord, routeInfoRecordList}//.WriteIPv4RouteStateEntryToDB(routeInfoRecord, routeInfoRecordList)
		RouteServiceHandler.WriteIPv4RouteStateEntryToDB(RouteDBInfo{routeInfoRecord, routeInfoRecordList})
		if callUpdate == false {
			return val, err
		}
	}
	updateBestRoute(destNet, routeInfoRecordList)
	return val, err
}


func printRoutesInfo(prefix patriciaDB.Prefix, item patriciaDB.Item) (err error) {
	rmapInfoRecordList := item.(RouteInfoRecordList)
	for _, v := range rmapInfoRecordList.routeInfoProtocolMap {
		if v == nil || len(v) == 0 {
			continue
		}
		for i := 0; i < len(v); i++ {
			//   logger.Printf("%v-> %d %d %d %d\n", prefix, v.destNetIp, v.networkMask, v.protocol)
			count++
		}
	}
	return nil
}

func (m RIBDServer) PrintV4Routes() (err error) {
	count = 0
	logger.Debug("Received print route")
	RouteInfoMap.Visit(printRoutesInfo)
	logger.Debug(fmt.Sprintf("total count = %d\n", count))
	return nil
}
func (m RIBDServer) GetNextHopIfTypeStr(nextHopIfType ribdInt.Int) (nextHopIfTypeStr string, err error) {
	nextHopIfTypeStr = ""
	switch nextHopIfType {
	case commonDefs.IfTypePort:
		nextHopIfTypeStr = "PHY"
		break
	case commonDefs.IfTypeVlan:
		nextHopIfTypeStr = "VLAN"
		break
	case commonDefs.IfTypeNull:
		nextHopIfTypeStr = "NULL"
		break
	case commonDefs.IfTypeLoopback:
		nextHopIfTypeStr = "Loopback"
		break
	}
	return nextHopIfTypeStr, err
}
