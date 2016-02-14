package main

import (
	"arpd"
	"asicdServices"
	//	"encoding/json"
	"l3/rib/ribdCommonDefs"
	"ribd"
	"utils/patriciaDB"
	//		"patricia"
	"asicd/asicdConstDefs"
	"bytes"
	"errors"
	//	"github.com/op/go-nanomsg"
	"net"
	"time"
)
type RouteInfoRecord struct {
	destNetIp      net.IP //string
	networkMask    net.IP //string
	nextHopIp      net.IP
	nextHopIfType  int8
	nextHopIfIndex ribd.Int
	metric         ribd.Int
	sliceIdx       int
	protocol       int8
	isPolicyBasedStateValid bool
}
type ConditionsAndActionsList struct {
	conditionList []string
	actionList    []string
}
type PolicyStmtMap struct {
	policyStmtMap map[string]ConditionsAndActionsList
}
//implement priority queue of the routes
type RouteInfoRecordList struct {
	selectedRouteProtocol   string
	selectedRouteIdx        int8
	routeInfoProtocolMap    map[string][]RouteInfoRecord 
	policyHitCounter        ribd.Int
	policyList              []string
	isPolicyBasedStateValid bool
	routeCreatedTime        string
	routeUpdatedTime        string
}

type RouteParams struct {
	destNetIp      string
	networkMask    string
	nextHopIp      string
	nextHopIfType  ribd.Int
	nextHopIfIndex ribd.Int
	metric         ribd.Int
	sliceIdx       ribd.Int
	routeType      ribd.Int
	createType     ribd.Int
	deleteType     ribd.Int
}
type RouteEventInfo struct {
	timeStamp     string
	eventInfo     string
}
type PolicyRouteIndex struct {
	routeIP string// patriciaDB.Prefix
	routeMask string
	policy string
}
var RouteInfoMap = patriciaDB.NewTrie()
var DummyRouteInfoRecord RouteInfoRecord //{destNet:0, prefixLen:0, protocol:0, nextHop:0, nextHopIfIndex:0, metric:0, selected:false}
var destNetSlice []localDB
var localRouteEventsDB []RouteEventInfo
var PolicyRouteMap map[PolicyRouteIndex]PolicyStmtMap

/*func getSelectedRoute(routeInfoRecordList RouteInfoRecordList) (routeInfoRecord RouteInfoRecord, err error) {
	logger.Println("getSelectedRoute routeInfoRecordList.selectedRouteProtocol = ", routeInfoRecordList.selectedRouteProtocol)
    routeInfoRecord.protocol = PROTOCOL_NONE
	if routeInfoRecordList.selectedRouteProtocol == PROTOCOL_NONE {
		err = errors.New("No route selected")
	} else {
		routeInfoRecord = routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]
	}
	return routeInfoRecord, err
}*/

func updateConnectedRoutes(destNetIPAddr string, networkMaskAddr string, nextHopIP string, nextHopIfIndex ribd.Int, nextHopIfType ribd.Int, op int, sliceIdx ribd.Int) {
	var temproute ribd.Routes
	route := &temproute
	logger.Println("number of connectd routes = ", len(ConnectedRoutes), "current op is to ", op, " ipAddr:mask = ", destNetIPAddr,":",networkMaskAddr)
	if len(ConnectedRoutes) == 0 {
		if op == del {
			logger.Println("Cannot delete a non-existent connected route")
			return
		}
		ConnectedRoutes = make([]*ribd.Routes, 1)
		route.Ipaddr = destNetIPAddr
		route.Mask = networkMaskAddr
		route.NextHopIp = nextHopIP
		route.NextHopIfType = nextHopIfType
		route.IfIndex = nextHopIfIndex
		route.IsValid = true
		route.SliceIdx = sliceIdx
		ConnectedRoutes[0] = route
		return
	}
	for i := 0; i < len(ConnectedRoutes); i++ {
		//		if(!strings.EqualFold(ConnectedRoutes[i].Ipaddr,destNetIPAddr) && !strings.EqualFold(ConnectedRoutes[i].Mask,networkMaskAddr)){
		if ConnectedRoutes[i].Ipaddr == destNetIPAddr && ConnectedRoutes[i].Mask == networkMaskAddr {
			if op == del {
				ConnectedRoutes = append(ConnectedRoutes[:i], ConnectedRoutes[i+1:]...)
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
	route.IfIndex = nextHopIfIndex
	route.NextHopIfType = nextHopIfType
	route.IsValid = true
	route.SliceIdx = sliceIdx
	ConnectedRoutes = append(ConnectedRoutes, route)
}
func IsRoutePresent(routeInfoRecordList RouteInfoRecordList,
	protocol string) (found bool) {
	logger.Println("Trying to look for route type ", protocol)
	routeInfoList,ok:= routeInfoRecordList.routeInfoProtocolMap[protocol]
	if ok && len(routeInfoList) > 0 {
		logger.Println(len(routeInfoList)," number of routeInfoRecords stored for this protocol")
		found = true
	}
/*	for i := 0; i < len(routeInfoRecordList.routeInfoList); i++ {
		logger.Printf("len = %d i=%d routePrototype=%d\n", len(routeInfoRecordList.routeInfoList), i, routeInfoRecordList.routeInfoList[i].protocol)
		if routeInfoRecordList.routeInfoList[i].protocol == routePrototype {
			found = true
             idxList = append(idxList,i)			
		}
	}
	logger.Printf("returning found = %d at indices :", found)
	for j:=0;j<len(idxList);j++ {
		logger.Printf("%d ", idxList[j])
	}
	logger.Printf("\n")*/
	return found
}

func getConnectedRoutes() {
	logger.Println("Getting ip intfs from portd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		logger.Printf("Getting %d objects from currMarker %d\n", count, currMarker)
		IPIntfBulk, err := asicdclnt.ClientHdl.GetBulkIPv4Intf(currMarker, count)
		if err != nil {
			logger.Println("GetBulkIPv4Intf with err ", err)
			return
		}
		if IPIntfBulk.Count == 0 {
			logger.Println("0 objects returned from GetBulkIPv4Intf")
			return
		}
		logger.Printf("len(IPIntfBulk.IPv4IntfList)  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfList), IPIntfBulk.Count)
		for i := 0; i < int(IPIntfBulk.Count); i++ {
			var ipMask net.IP
			ip, ipNet, err := net.ParseCIDR(IPIntfBulk.IPv4IntfList[i].IpAddr)
			if err != nil {
				return
			}
			ipMask = make(net.IP, 4)
			copy(ipMask, ipNet.Mask)
			ipAddrStr := ip.String()
			ipMaskStr := net.IP(ipMask).String()
			logger.Printf("Calling createv4Route with ipaddr %s mask %s\n", ipAddrStr, ipMaskStr)
			_, err = routeServiceHandler.CreateV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)),"CONNECTED") // FIBAndRIB, ribd.Int(len(destNetSlice)))
			if err != nil {
				logger.Printf("Failed to create connected route for ip Addr %s/%s intfType %d intfId %d\n", ipAddrStr, ipMaskStr, ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)))
			}
		}
		if IPIntfBulk.More == false {
			logger.Println("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(IPIntfBulk.EndIdx)
	}
}

//thrift API definitions
func (m RouteServiceHandler)	GetBulkRouteDistanceState(fromIndex ribd.Int, rcount ribd.Int) (routeDistanceStates *ribd.RouteDistanceStateGetInfo , err error) {
	logger.Println("GetBulkRouteDistanceState")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.RouteDistanceState = make ([]ribd.RouteDistanceState, rcount)
	var nextNode *ribd.RouteDistanceState
    var returnNodes []*ribd.RouteDistanceState
	var returnGetInfo ribd.RouteDistanceStateGetInfo
	i = 0
	routeDistanceStates = &returnGetInfo
	more := true
	BuildProtocolAdminDistanceSlice()
    if(ProtocolAdminDistanceSlice== nil) {
		logger.Println("ProtocolAdminDistanceSlice not initialized")
		return routeDistanceStates, err
	}
	for ;;i++ {
		logger.Printf("Fetching record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(ProtocolAdminDistanceSlice))) {
			logger.Println("All the events fetched")
			more = false
			break
		}
		if(validCount==rcount) {
			logger.Println("Enough events fetched")
			break
		}
		logger.Printf("Fetching event record for index %d \n", i+fromIndex)
		nextNode = &tempNode[validCount]
		nextNode.Protocol = ProtocolAdminDistanceSlice[i+fromIndex].Protocol
		nextNode.Distance = ProtocolAdminDistanceSlice[i+fromIndex].Distance
	    toIndex = ribd.Int(i+fromIndex)
		if(len(returnNodes) == 0){
			returnNodes = make([]*ribd.RouteDistanceState, 0)
		}
		returnNodes = append(returnNodes, nextNode)
		validCount++
	}
	logger.Printf("Returning %d list of dtsnace vector nodes", validCount)
	routeDistanceStates.RouteDistanceStateList = returnNodes
	routeDistanceStates.StartIdx = fromIndex
	routeDistanceStates.EndIdx = toIndex+1
	routeDistanceStates.More = more
	routeDistanceStates.Count = validCount
	return routeDistanceStates, err
}

func (m RouteServiceHandler) 	 GetBulkIPV4EventState( fromIndex ribd.Int, rcount ribd.Int	) (events *ribd.IPV4EventStateGetInfo, err error) {
	logger.Println("GetBulkIPV4EventState")
    var i, validCount, toIndex ribd.Int
	var tempNode []ribd.IPV4EventState = make ([]ribd.IPV4EventState, rcount)
	var nextNode *ribd.IPV4EventState
    var returnNodes []*ribd.IPV4EventState
	var returnGetInfo ribd.IPV4EventStateGetInfo
	i = 0
	events = &returnGetInfo
	more := true
    if(localRouteEventsDB == nil) {
		logger.Println("localRouteEventsDB not initialized")
		return events, err
	}
	for ;;i++ {
		logger.Printf("Fetching record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(localRouteEventsDB))) {
			logger.Println("All the events fetched")
			more = false
			break
		}
		if(validCount==rcount) {
			logger.Println("Enough events fetched")
			break
		}
		logger.Printf("Fetching event record for index %d \n", i+fromIndex)
		nextNode = &tempNode[validCount]
		nextNode.TimeStamp = localRouteEventsDB[i+fromIndex].timeStamp
		nextNode.EventInfo = localRouteEventsDB[i+fromIndex].eventInfo
	    toIndex = ribd.Int(i+fromIndex)
		if(len(returnNodes) == 0){
			returnNodes = make([]*ribd.IPV4EventState, 0)
		}
		returnNodes = append(returnNodes, nextNode)
		validCount++
	}
	logger.Printf("Returning %d list of events", validCount)
	events.IPV4EventStateList = returnNodes
	events.StartIdx = fromIndex
	events.EndIdx = toIndex+1
	events.More = more
	events.Count = validCount
	return events, err
}

func (m RouteServiceHandler) GetBulkRoutes(fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.RoutesGetInfo, err error) { //(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkRoutes")
	var i, validCount, toIndex ribd.Int
	var temproute []ribd.Routes = make([]ribd.Routes, rcount)
	var nextRoute *ribd.Routes
	var returnRoutes []*ribd.Routes
	var returnRouteGetInfo ribd.RoutesGetInfo
	var prefixNodeRouteList RouteInfoRecordList
	var prefixNodeRoute RouteInfoRecord
	i = 0
	sel:=0
	found := false
	routes = &returnRouteGetInfo
	moreRoutes := true
	if destNetSlice == nil {
		logger.Println("destNetSlice not initialized")
		return routes, err
	}
	for ; ; i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		found = false
		if i+fromIndex >= ribd.Int(len(destNetSlice)) {
			logger.Println("All the routes fetched")
			moreRoutes = false
			break
		}
		if destNetSlice[i+fromIndex].isValid == false {
			logger.Println("Invalid route")
			continue
		}
		if validCount == rcount {
			logger.Println("Enough routes fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (destNetSlice[i+fromIndex].prefix))
		prefixNode := RouteInfoMap.Get(destNetSlice[i+fromIndex].prefix)
		if prefixNode != nil{
			prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
			if prefixNodeRouteList.isPolicyBasedStateValid == false {
				logger.Println("Route invalidated based on policy")
				continue
			}
			logger.Println("selectedRouteProtocol = ", prefixNodeRouteList.selectedRouteProtocol)
			if prefixNodeRouteList.routeInfoProtocolMap == nil || prefixNodeRouteList.selectedRouteProtocol == "INVALID" || prefixNodeRouteList.routeInfoProtocolMap[prefixNodeRouteList.selectedRouteProtocol] ==nil {
				logger.Println("selected route not valid")
				continue
			}
			routeInfoList := prefixNodeRouteList.routeInfoProtocolMap[prefixNodeRouteList.selectedRouteProtocol]
			for sel =0;sel<len(routeInfoList);sel++ {
				if routeInfoList[sel].nextHopIp.String() == destNetSlice[i+fromIndex].nextHopIp {
				    logger.Println("Found the entry corresponding to the nextHop ip")
                     found = true
					break
				}
			}
			if !found {
				logger.Println("The corresponding route with nextHopIP was not found in the record DB")
				continue
			}
			prefixNodeRoute = routeInfoList[sel] //prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
			nextRoute = &temproute[validCount]
			nextRoute.Ipaddr = prefixNodeRoute.destNetIp.String()
			nextRoute.Mask = prefixNodeRoute.networkMask.String()
			nextRoute.NextHopIp = prefixNodeRoute.nextHopIp.String()
			nextRoute.NextHopIfType = ribd.Int(prefixNodeRoute.nextHopIfType)
			nextRoute.IfIndex = prefixNodeRoute.nextHopIfIndex
			nextRoute.Metric = prefixNodeRoute.metric
			nextRoute.RoutePrototypeString = ReverseRouteProtoTypeMapDB[int(prefixNodeRoute.protocol)]
			nextRoute.IsValid = destNetSlice[i+fromIndex].isValid
			nextRoute.RouteCreated = prefixNodeRouteList.routeCreatedTime
			nextRoute.RouteUpdated = prefixNodeRouteList.routeUpdatedTime
			nextRoute.PolicyList = make([]string,0)
			routePolicyListInfo := ""
			if prefixNodeRouteList.policyList != nil {
				for k:=0;k<len(prefixNodeRouteList.policyList);k++ {
					routePolicyListInfo = "policy "+prefixNodeRouteList.policyList[k]+"["
	                 policyRouteIndex := PolicyRouteIndex{routeIP:prefixNodeRoute.destNetIp.String(),routeMask:prefixNodeRoute.networkMask.String(), policy:prefixNodeRouteList.policyList[k]}
					policyStmtMap, ok := PolicyRouteMap[policyRouteIndex]
					if !ok || policyStmtMap.policyStmtMap == nil{
						continue
					}
					routePolicyListInfo = routePolicyListInfo + " stmtlist[["
					for stmt,conditionsAndActionsList := range policyStmtMap.policyStmtMap {
						routePolicyListInfo = routePolicyListInfo + stmt+":[conditions:"
						for c:=0;c<len(conditionsAndActionsList.conditionList);c++ {
							routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.conditionList[c]+","
						}  
						routePolicyListInfo = routePolicyListInfo+"],[actions:"
						for a:=0;a<len(conditionsAndActionsList.actionList);a++ {
							routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.actionList[a]+","
						}  
						routePolicyListInfo = routePolicyListInfo+"]]"
					}
					routePolicyListInfo = routePolicyListInfo+"]"
				    nextRoute.PolicyList = append(nextRoute.PolicyList,routePolicyListInfo)
				}
			}
/*			if prefixNodeRouteList.policyList != nil {
			  for k,v := range prefixNodeRouteList.policyList {
				routePolicyListInfo = k+":"
			    for vv:=range v {
			       routePolicyListInfo = routePolicyListInfo + vv+"," 	
			    }
			  }	
			}
			nextRoute.PolicyList = routePolicyListInfo
			nextRoute.PolicyList = make(map[string][]string)
			if prefixNodeRouteList.policyList != nil {
			  for k,v := range prefixNodeRouteList.policyList {
			    nextRoute.PolicyList[k] = make([]string,0)
			    for idx:=0;idx<len(v);idx++ {
					nextRoute.PolicyList[k] = append(nextRoute.PolicyList[k],v[idx])
			    }
			  }	
			}*/
			toIndex = ribd.Int(prefixNodeRoute.sliceIdx)
			if len(returnRoutes) == 0 {
				returnRoutes = make([]*ribd.Routes, 0)
			}
			returnRoutes = append(returnRoutes, nextRoute)
			validCount++
		}
	}
	logger.Printf("Returning %d list of routes\n", validCount)
	routes.RouteList = returnRoutes
	routes.StartIdx = fromIndex
	routes.EndIdx = toIndex + 1
	routes.More = moreRoutes
	routes.Count = validCount
	return routes, err
}

func (m RouteServiceHandler) GetConnectedRoutesInfo() (routes []*ribd.Routes, err error) {
	var returnRoutes []*ribd.Routes
	var nextRoute *ribd.Routes
	logger.Println("Received GetConnectedRoutesInfo")
	returnRoutes = make([]*ribd.Routes, 0)
	//	routes = ConnectedRoutes
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].IsValid == true {
			nextRoute = ConnectedRoutes[i]
			returnRoutes = append(returnRoutes, nextRoute)
		} else {
			logger.Println("Invalid connected route present")
		}
	}
	routes = returnRoutes
	return routes, err
}
func (m RouteServiceHandler) GetRouteReachabilityInfo(destNet string) (nextHopIntf *ribd.NextHopInfo, err error) {
	t1 := time.Now()
	var retnextHopIntf ribd.NextHopInfo
	nextHopIntf = &retnextHopIntf
	var found bool
	destNetIp, err := getIP(destNet)
	if err != nil {
		return nextHopIntf, errors.New("Invalid dest ip address")
	}
	rmapInfoListItem := RouteInfoMap.GetLongestPrefixNode(patriciaDB.Prefix(destNetIp))
	if rmapInfoListItem != nil {
		rmapInfoList := rmapInfoListItem.(RouteInfoRecordList)
		if rmapInfoList.selectedRouteProtocol != "INVALID" {
			found = true
			routeInfoList, ok := rmapInfoList.routeInfoProtocolMap[rmapInfoList.selectedRouteProtocol]
            if !ok {
				logger.Println("Selected route not found")
	             return nextHopIntf, err
			}
			v := routeInfoList[0]
			nextHopIntf.NextHopIfType = ribd.Int(v.nextHopIfType)
			nextHopIntf.NextHopIfIndex = v.nextHopIfIndex
			nextHopIntf.NextHopIp = v.nextHopIp.String()
			nextHopIntf.Metric = v.metric
			nextHopIntf.Ipaddr = v.destNetIp.String()
			nextHopIntf.Mask = v.networkMask.String()
		}
	}

	if found == false {
		logger.Printf("dest IP %s not reachable\n", destNetIp)
		err = errors.New("dest ip address not reachable")
	}
	duration := time.Since(t1)
	logger.Printf("time to get longestPrefixLen = %d\n", duration.Nanoseconds())
	logger.Printf("next hop ip of the route = %s\n", nextHopIntf.NextHopIfIndex)
	return nextHopIntf, err
}
func (m RouteServiceHandler) GetRoute(destNetIp string, networkMask string) (route *ribd.Routes, err error) {
	var returnRoute ribd.Routes
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
		logger.Println("No such route")
		err = errors.New("Route does not exist")
		return route, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
	if routeInfoRecordList.selectedRouteProtocol == "INVALID" {
		logger.Println("No selected route for this network")
		err = errors.New("No selected route for this network")
		return route, err
	}
	routeInfoList := routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]
	routeInfoRecord := routeInfoList[0]
//	routeInfoRecord := routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
	route.Ipaddr = destNetIp
	route.Mask = networkMask
	route.NextHopIp = routeInfoRecord.nextHopIp.String()
	route.NextHopIfType = ribd.Int(routeInfoRecord.nextHopIfType)
	route.IfIndex = routeInfoRecord.nextHopIfIndex
	route.Metric = routeInfoRecord.metric
	route.Prototype = ribd.Int(routeInfoRecord.protocol)
	return route, err
}
func SelectBestRoute(routeInfoRecordList RouteInfoRecordList) (addRouteList []RouteInfoRecord, deleteRouteList []RouteInfoRecord, newSelectedProtocol string) {
	logger.Println("SelectBestRoute, the current selected route protocol is ", routeInfoRecordList.selectedRouteProtocol)
	tempSelectedProtocol := "INVALID"
	newSelectedProtocol = "INVALID"
	deleteRouteList = make([]RouteInfoRecord, 0)
	addRouteList = make([]RouteInfoRecord, 0)
	logger.Println("len(protocolAdminDistanceSlice):", len(ProtocolAdminDistanceSlice))
	BuildProtocolAdminDistanceSlice()
	for i:=0;i<len(ProtocolAdminDistanceSlice);i++ {
		tempSelectedProtocol = ProtocolAdminDistanceSlice[i].Protocol
		logger.Println("Best preferred protocol ", tempSelectedProtocol)
		routeInfoList := routeInfoRecordList.routeInfoProtocolMap[tempSelectedProtocol]
		if routeInfoList == nil || len(routeInfoList) == 0 {
			logger.Println("No routes are configured with this protocol ", tempSelectedProtocol, " for this route")
	        tempSelectedProtocol = "INVALID"
			continue
		}
	    tempSelectedProtocol = "INVALID"
        for j :=0;j<len(routeInfoList);j++ {
		   routeInfoRecord := routeInfoList[j]
           policyRoute := ribd.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecord.nextHopIfType), IfIndex: routeInfoRecord.nextHopIfIndex, Metric: routeInfoRecord.metric, Prototype: ribd.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
		   actionList := PolicyEngineCheck(policyRoute, ribdCommonDefs.PolicyConditionTypeProtocolMatch)
		   if !actionListHasAction(actionList, ribdCommonDefs.PolicyActionTypeRouteDisposition,"Reject") {
		       logger.Println("atleast one of the routes of this protocol will not be rejected by the policy engine")
		       tempSelectedProtocol = ProtocolAdminDistanceSlice[i].Protocol
			   break
		   }
		}
		if tempSelectedProtocol != "INVALID" {
			logger.Println("Found a valid protocol ", tempSelectedProtocol)
			break
		}		
	}
	if tempSelectedProtocol == routeInfoRecordList.selectedRouteProtocol {
		logger.Println("The current protocol remains the new selected protocol")
		return addRouteList, deleteRouteList,newSelectedProtocol
	}
	if routeInfoRecordList.selectedRouteProtocol != "INVALID" {
		logger.Println("Valid protocol currently selected as ", routeInfoRecordList.selectedRouteProtocol)
         for j :=0;j<len(routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]);j++ {
			deleteRouteList = append(deleteRouteList,routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][j])
		}		
	}
	if tempSelectedProtocol != "INVALID" {
		logger.Println("New Valid protocol selected as ", tempSelectedProtocol)
         for j :=0;j<len(routeInfoRecordList.routeInfoProtocolMap[tempSelectedProtocol]);j++ {
			addRouteList = append(addRouteList,routeInfoRecordList.routeInfoProtocolMap[tempSelectedProtocol][j])
		}		
		newSelectedProtocol = tempSelectedProtocol
	}
	return addRouteList, deleteRouteList,newSelectedProtocol
}
//this function is called when a route is being added after it has cleared import policies
func selectBestRouteOnAdd(routeInfoRecordList RouteInfoRecordList,  routeInfoRecord RouteInfoRecord)  (addRouteList []RouteInfoRecord, deleteRouteList []RouteInfoRecord, newSelectedProtocol string) {
	logger.Println("selectBestRouteOnAdd current selected protocol = ", routeInfoRecordList.selectedRouteProtocol)
	deleteRouteList = make([]RouteInfoRecord, 0)
	addRouteList = make([]RouteInfoRecord, 0)
    newSelectedProtocol = routeInfoRecordList.selectedRouteProtocol
	newRouteProtocol := ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]
	add := false
	del := false
	if routeInfoRecordList.selectedRouteProtocol == "INVALID" {
		if routeInfoRecord.protocol != PROTOCOL_NONE {
		   logger.Println("Selecting the new route because the current selected route is invalid")
		   add = true
		   newSelectedProtocol = newRouteProtocol
		}
	} else if ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance > ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance {
	    logger.Println(" Rejecting the new route because the admin distance of the new routetype ", newRouteProtocol, ":", ProtocolAdminDistanceMapDB[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]].configuredDistance, "is configured to be higher than the selected route protocol ", routeInfoRecordList.selectedRouteProtocol, "'s admin distance ",ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol]) 	
	} else if ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance < ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance {
	    logger.Println(" Selecting the new route because the admin distance of the new routetype ", newRouteProtocol, ":", ProtocolAdminDistanceMapDB[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]].configuredDistance, "is better than the selected route protocol ", routeInfoRecordList.selectedRouteProtocol, "'s admin distance ",ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol]) 	
        del = true
		add = true
		newSelectedProtocol = newRouteProtocol
	} else if ProtocolAdminDistanceMapDB[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]].configuredDistance == ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance{
	    logger.Println("Same admin distance ")
		if newRouteProtocol == routeInfoRecordList.selectedRouteProtocol {
			logger.Println("Same protocol as the selected route")
			if routeInfoRecord.metric == routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0].metric {
			   logger.Println("Adding a same cost route as the current selected routes")
			   if !newNextHopIP(routeInfoRecord.nextHopIp.String(), routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]) {
			      logger.Println("Not a new next hop ip, so do nothing")
			   } else {
			      logger.Println("This is a new route with a new next hop IP")
		          add = true
			   }
			} else if routeInfoRecord.metric < routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0].metric {
			  logger.Println("New metric ", routeInfoRecord.metric, " is lower than the current metric ", routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][0].metric)	
               del = true
			   add = true
			}
		} else {
			logger.Println("Protocol ", newRouteProtocol, " has the same admin distance ", ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance, " as the protocol", routeInfoRecordList.selectedRouteProtocol, "'s configured admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance )
             if ProtocolAdminDistanceMapDB[newRouteProtocol].defaultDistance < ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].defaultDistance {
			   logger.Println("Protocol ", newRouteProtocol, " has lower default admin distance ", ProtocolAdminDistanceMapDB[newRouteProtocol].defaultDistance, " than the protocol", routeInfoRecordList.selectedRouteProtocol, "'s default admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].defaultDistance )
                del = true
				add = true
		       newSelectedProtocol = newRouteProtocol
			} else {
				logger.Println("Protocol ", newRouteProtocol, " has higher default admin distance ", ProtocolAdminDistanceMapDB[newRouteProtocol].configuredDistance, " than the protocol", routeInfoRecordList.selectedRouteProtocol, "'s default admin distance ", ProtocolAdminDistanceMapDB[routeInfoRecordList.selectedRouteProtocol].configuredDistance )
                 add = true
			}
		}	
	}
	logger.Println("At the end of the route selection logic, add = ", add, " del = ", del)
	if add == true {
	   addRouteList = append(addRouteList,routeInfoRecord)
    } 
	if del == true {
		for i:=0;i<len(routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol]);i++ {
			deleteRouteList = append(deleteRouteList,routeInfoRecordList.routeInfoProtocolMap[routeInfoRecordList.selectedRouteProtocol][i])
		}
	}
	return addRouteList, deleteRouteList,newSelectedProtocol
}
func addNewRoute(destNetPrefix patriciaDB.Prefix, 
                  routeInfoRecord RouteInfoRecord, 
				  routeInfoRecordList RouteInfoRecordList,
				  policyPath int){
   logger.Println("addNewRoute")
   policyRoute := ribd.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecord.nextHopIfType), IfIndex: routeInfoRecord.nextHopIfIndex, Metric: routeInfoRecord.metric, Prototype: ribd.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
   var params RouteParams
	   if destNetSlice != nil && (len(destNetSlice) > int(routeInfoRecord.sliceIdx)) { //&& bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNet)) {
	      if bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNetPrefix) == false {
			logger.Println("Unexpected destination network prefix ", destNetSlice[routeInfoRecord.sliceIdx].prefix, " found at the slice Idx ", routeInfoRecord.sliceIdx, " expected prefix ", destNetPrefix)
			return 
		  }
		  //There is already an entry in the destNetSlice at the route index and was invalidated earlier because  of a link down of the nexthop intf of the route or if the route was deleted
		  //In this case since the old route was invalid, there is nothing to delete
		  logger.Println("sliceIdx ", routeInfoRecord.sliceIdx)
		  destNetSlice[routeInfoRecord.sliceIdx].isValid = true
	   } 
	   if routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] == nil {
	      routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = make([]RouteInfoRecord,0)	
	   }
	   if newNextHopIP(routeInfoRecord.nextHopIp.String(),routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]]) {
	      routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = append(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]], routeInfoRecord)
	   }
	
	    //update the patriciaDB trie with the updated route info record list
	    t1 := time.Now()
	    routeInfoRecordList.routeUpdatedTime = t1.String()
	    RouteInfoMap.Set(patriciaDB.Prefix(destNetPrefix), routeInfoRecordList)

        if ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)] != routeInfoRecordList.selectedRouteProtocol {
			logger.Println("This is not a selected route, so nothing more to do here")
			return
		}
		logger.Println("This is a selected route, so install and parse through export policy engine")
        if policyPath == ribdCommonDefs.PolicyPath_Import && destNetSlice != nil && (len(destNetSlice) <= int(routeInfoRecord.sliceIdx)){
          logger.Println("This is a new route for selectedProtocolType being added, create destNetSlice entry")
	      routeInfoRecord.sliceIdx = len(destNetSlice)
		  localDBRecord := localDB{prefix: destNetPrefix, isValid: true,nextHopIp:routeInfoRecord.nextHopIp.String()}
		  if destNetSlice == nil {
			destNetSlice = make([]localDB, 0)
		  }
		  destNetSlice = append(destNetSlice, localDBRecord)
		}
		policyRoute.Prototype = ribd.Int(routeInfoRecord.protocol)
		params.routeType = policyRoute.Prototype
		params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
		policyRoute.Ipaddr = routeInfoRecord.destNetIp.String()
		policyRoute.Mask = routeInfoRecord.networkMask.String()
		if policyPath == ribdCommonDefs.PolicyPath_Export {
		  logger.Println("New route selected, call asicd to install a new route - ip", routeInfoRecord.destNetIp.String(), " mask ", routeInfoRecord.networkMask.String(), " nextHopIP ",routeInfoRecord.nextHopIp.String())
		  //call asicd to add
		  if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String(), routeInfoRecord.nextHopIp.String())
		  }
		  if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
			//call arpd to resolve the ip
			logger.Println("### Sending ARP Resolve for ", routeInfoRecord.nextHopIp.String(), routeInfoRecord.nextHopIfType)
			arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.nextHopIp.String(), arpd.Int(routeInfoRecord.nextHopIfType), arpd.Int(routeInfoRecord.nextHopIfIndex))
		  }
		  addLinuxRoute(routeInfoRecord)
		  //update in the event log
	      eventInfo := "Created route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	      t1 := time.Now()
          routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	      localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
		}
		params.deleteType = Invalid
	    PolicyEngineFilter(policyRoute, policyPath,params )
}
func addNewRouteList(destNetPrefix patriciaDB.Prefix, 
                  addRouteList []RouteInfoRecord, 
				  routeInfoRecordList RouteInfoRecordList,
				  policyPath int){
   logger.Println("addNewRoutes")
   for i:=0;i<len(addRouteList);i++ {
	   addNewRoute(destNetPrefix,addRouteList[i],routeInfoRecordList,policyPath)
   }
}
//note: selectedrouteProtocol should not have been set to INVALID by either of the selects when this function is called
func deleteRoute(destNetPrefix patriciaDB.Prefix, 
                 routeInfoRecord RouteInfoRecord, 
				routeInfoRecordList RouteInfoRecordList,
				policyPath int) {
	logger.Println(" deleteRoute")
	if destNetSlice == nil || int(routeInfoRecord.sliceIdx) >= len(destNetSlice) {
		logger.Println("Destination slice not found at the expected slice index ", routeInfoRecord.sliceIdx)
		return 
	}
	destNetSlice[routeInfoRecord.sliceIdx].isValid = false //invalidate this entry in the local db
	routeInfoList := routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]]
    found,_,index := findRouteWithNextHop(routeInfoList,routeInfoRecord.nextHopIp.String())
    if !found || index == -1 {
		logger.Println("Invalid nextHopIP")
		return
	}
	logger.Println("Found the route at index ", index)
	deleteNode := true
	routeInfoList = append(routeInfoList[:index], routeInfoList[index+1:]...)
	routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = routeInfoList
    if len(routeInfoList) == 0{
        logger.Println("All routes for this destination from protocol ", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], " deleted")
		routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)]] = nil
        deleteNode = true
        for k,v:=range routeInfoRecordList.routeInfoProtocolMap {
			if v!= nil && len(v) != 0 {
				logger.Println("There are still other protocol ", k," routes for this destination")
                 deleteNode = false
			}
		}
		if deleteNode == true {
		   logger.Println("No routes to this destination , delete node")
	       RouteInfoMap.Delete(destNetPrefix)
		} else {
			RouteInfoMap.Set(destNetPrefix,routeInfoRecordList)
		}
	}
	if routeInfoRecordList.selectedRouteProtocol != ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)] {
		logger.Println("This is not the selected protocol, nothing more to do here")
		return
	}
    policyRoute := ribd.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecord.nextHopIfType), IfIndex: routeInfoRecord.nextHopIfIndex, Metric: routeInfoRecord.metric, Prototype: ribd.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
    var params RouteParams
	if policyPath != ribdCommonDefs.PolicyPath_Export {
		logger.Println("Expected export path for delete op")
		return
	}
	logger.Println("This is the selected protocol")
		//delete in asicd
		if asicdclnt.IsConnected {
			logger.Println("Calling asicd to delete this route")
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
		}
		//update in the event log
	    eventInfo := "Deleted route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	    t1 := time.Now()
        routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	    localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
	    params.createType = Invalid
	    params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
		policyRoute.PolicyList = routeInfoRecordList.policyList
		PolicyEngineFilter(policyRoute, policyPath, params)
}
func deleteRoutes (destNetPrefix patriciaDB.Prefix, 
                     deleteRouteList []RouteInfoRecord, 
				    routeInfoRecordList RouteInfoRecordList,
				    policyPath int) {
   logger.Println("deleteRoutes")
   for i := 0;i<len(deleteRouteList);i++ {
      deleteRoute(destNetPrefix, deleteRouteList[i],routeInfoRecordList,policyPath)	
   }				 	
}
func SelectV4Route(destNetPrefix patriciaDB.Prefix,
	routeInfoRecordList RouteInfoRecordList,  //the current list of routes for this prefix
	routeInfoRecord RouteInfoRecord,          //the route to be added or deleted or invalidated or validated
	op ribd.Int) (err error) {
//	index int) (err error) {
   logger.Println("Selecting the best Route for destNetPrefix ", destNetPrefix)
   if op == add {
      logger.Println("Op is to add the new route")
	  _, deleteRouteList,newSelectedProtocol := selectBestRouteOnAdd(routeInfoRecordList,routeInfoRecord)
	  if len(deleteRouteList) >0 {
	     deleteRoutes(destNetPrefix, deleteRouteList, routeInfoRecordList, ribdCommonDefs.PolicyPath_Export)	
	  }	
	  routeInfoRecordList.selectedRouteProtocol = newSelectedProtocol
	  addNewRoute(destNetPrefix, routeInfoRecord,routeInfoRecordList, ribdCommonDefs.PolicyPath_Export)
   } else if op == del {
	    logger.Println("Op is to delete new route")
	    deleteRoute(destNetPrefix, routeInfoRecord, routeInfoRecordList, ribdCommonDefs.PolicyPath_Export)	
		addRouteList,_,newSelectedProtocol := SelectBestRoute(routeInfoRecordList)
	    routeInfoRecordList.selectedRouteProtocol = newSelectedProtocol
	    if len(addRouteList) > 0 {
		  logger.Println("Number of routes to be added = ", len(addRouteList))
		  addNewRouteList(destNetPrefix, addRouteList,routeInfoRecordList, ribdCommonDefs.PolicyPath_Import)
	  }
   }
   return err
}
func updateBestRoute(destNetPrefix patriciaDB.Prefix, routeInfoRecordList RouteInfoRecordList) {
	logger.Println("updateBestRoute for ip network ", destNetPrefix)
	addRouteList,deleteRouteList,newSelectedProtocol := SelectBestRoute(routeInfoRecordList)
	if len(deleteRouteList) > 0 {
		logger.Println(len(addRouteList), " to be deleted")
		deleteRoutes(destNetPrefix,addRouteList,routeInfoRecordList,ribdCommonDefs.PolicyPath_Export)
	}
	routeInfoRecordList.selectedRouteProtocol = newSelectedProtocol
	if len(addRouteList) > 0 {
		logger.Println("New ", len(addRouteList), " to be added")
		addNewRouteList(destNetPrefix,addRouteList,routeInfoRecordList,ribdCommonDefs.PolicyPath_Import)
	}
}
/*func SelectV4Route(destNetPrefix patriciaDB.Prefix,
	routeInfoRecordList RouteInfoRecordList,  //the current list of routes for this prefix
	routeInfoRecord RouteInfoRecord,          //the route to be added or deleted or invalidated or validated
	op ribd.Int,
	index int) (err error) {
	var routeInfoRecordNew RouteInfoRecord
	var routeInfoRecordOld RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	routeInfoRecordNew.protocol = PROTOCOL_NONE
	routeInfoRecordOld.protocol = PROTOCOL_NONE
	routeInfoRecordTemp.protocol = PROTOCOL_NONE
	var i int8
	var deleteRoute bool
	var policyPath int
	policyRoute := ribd.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecord.nextHopIfType), IfIndex: routeInfoRecord.nextHopIfIndex, Metric: routeInfoRecord.metric, Prototype: ribd.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
	var params RouteParams
	logger.Printf("Selecting the best Route for destNetPrefix %v, index = %d\n", destNetPrefix, index)
	if op == add {
		selectedRoute, err := getSelectedRoute(routeInfoRecordList)
		logger.Printf("Selected route protocol = %s, routeinforecord.protocol=%s\n", ReverseRouteProtoTypeMapDB[int(selectedRoute.protocol)], ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)])
        logger.Println("routeInfoRecordList.selectedRouteIdx = ", routeInfoRecordList.selectedRouteIdx, " selectedRoute.sliceIdx = ",selectedRoute.sliceIdx)
		if err == nil && isBetterRoute(selectedRoute, routeInfoRecord)  {
			routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx] = selectedRoute
			routeInfoRecordOld = selectedRoute
			destNetSlice[routeInfoRecordOld.sliceIdx].isValid = false
			//destNetSlice is a slice of localDB maintained for a getBulk operations. An entry is created in this db when we create a new route
			if destNetSlice != nil && (len(destNetSlice) > int(routeInfoRecord.sliceIdx)) { //&& bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNet)) {
				if bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNetPrefix) == false {
					logger.Println("Unexpected destination network prefix ", destNetSlice[routeInfoRecord.sliceIdx].prefix, " found at the slice Idx ", routeInfoRecord.sliceIdx, " expected prefix ", destNetPrefix)
					return err
				}
				//There is already an entry in the destNetSlice at the route index and was invalidated earlier because  of a link down of the nexthop intf of the route or if the route was deleted
				//In this case since the old route was invalid, there is nothing to delete
				routeInfoRecordOld.protocol = PROTOCOL_NONE
				logger.Println("sliceIdx ", routeInfoRecord.sliceIdx)
				destNetSlice[routeInfoRecord.sliceIdx].isValid = true
			} else { //this is a new route being added
				routeInfoRecord.sliceIdx = len(destNetSlice)
				localDBRecord := localDB{prefix: destNetPrefix, isValid: true}
				if destNetSlice == nil {
					destNetSlice = make([]localDB, 0)
				}
				destNetSlice = append(destNetSlice, localDBRecord)
			}
		   routeInfoRecordList.routeInfoList[index] = routeInfoRecord
		   routeInfoRecordNew = routeInfoRecord
		   routeInfoRecordList.selectedRouteIdx = int8(index)
		   policyPath = ribdCommonDefs.PolicyPath_Export
		   logger.Printf("new selected route idx = %d\n", routeInfoRecordList.selectedRouteIdx)
		} else if isBetterRoute(selectedRoute, routeInfoRecord) {//case when route is being updated
		   logger.Println("current selected route is invalid, new selected route protocol is ", routeInfoRecord.protocol)	
		   routeInfoRecordList.routeInfoList[index] = routeInfoRecord
		   routeInfoRecordNew = routeInfoRecord
		   routeInfoRecordList.selectedRouteIdx = int8(index)
		   policyPath = ribdCommonDefs.PolicyPath_Export
		   logger.Printf("new selected route idx = %d\n", routeInfoRecordList.selectedRouteIdx)
		}
	} else if op == del {
		logger.Println(" in del index selectedrouteIndex", index, routeInfoRecordList.selectedRouteIdx)
		if destNetSlice == nil || int(routeInfoRecord.sliceIdx) >= len(destNetSlice) {
			logger.Println("Destination slice not found at the expected slice index ", routeInfoRecord.sliceIdx)
			return err
		}
		destNetSlice[routeInfoRecord.sliceIdx].isValid = false //invalidate this entry in the local db
		if len(routeInfoRecordList.routeInfoList) == 0 {
			logger.Println(" in del,numRoutes now 0, so delete the node")
			RouteInfoMap.Delete(destNetPrefix)
			//call asicd to del
			if asicdclnt.IsConnected {
				asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
			}
		    //update in the event log
	        eventInfo := "Deleted route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	        t1 := time.Now()
            routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	        localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
			params.createType = Invalid
			params.destNetIp = routeInfoRecord.destNetIp.String()
			params.networkMask = routeInfoRecord.networkMask.String()
			policyRoute.PolicyList = routeInfoRecordList.policyList
			PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export, params)
			return nil
		}
		if int8(index) == routeInfoRecordList.selectedRouteIdx {
			logger.Println("Deleting the selected route")
			deleteRoute = true
		    routeInfoRecordList.selectedRouteIdx = PROTOCOL_NONE
			routeInfoRecordTempSelected := routeInfoRecord
			selectedRouteIdx := -1
			routeInfoRecordTempSelected.protocol = PROTOCOL_NONE
			for i = 0; i < int8(len(routeInfoRecordList.routeInfoList)); i++ {
				routeInfoRecordTemp = routeInfoRecordList.routeInfoList[i]
				/*if i == int8(index) { //if(ok != true || i==routeInfoRecord.protocol) {
					continue
				}*/
/*				logger.Printf("temp protocol=%d, routeInfoRecordTempSelected.protocol=%d\n", routeInfoRecordTemp.protocol, routeInfoRecordTempSelected.protocol)
				if (isBetterRoute(routeInfoRecordTempSelected, routeInfoRecordTemp)) {//} && destNetSlice[routeInfoRecord.sliceIdx].isValid) {
					logger.Println(" evaluating route at index ",i, "routeInfoRecordTemp.protocol = ", ReverseRouteProtoTypeMapDB[int(routeInfoRecordTemp.protocol)], " tempselected protocol = ", ReverseRouteProtoTypeMapDB[int(routeInfoRecordTempSelected.protocol)])
                     routeInfoRecordTempSelected = routeInfoRecordTemp
					selectedRouteIdx = int(i)
				}
			}
				logger.Println("selected route at index ",selectedRouteIdx, " tempselected protocol = ", ReverseRouteProtoTypeMapDB[int(routeInfoRecordTempSelected.protocol)])
				if routeInfoRecordTempSelected.protocol != PROTOCOL_NONE{
					routeInfoRecordNew = routeInfoRecordTempSelected
					//routeInfoRecordList.selectedRouteIdx = int8(selectedRouteIdx)
					logger.Println("routeRecordInfo.sliceIdx = ", routeInfoRecord.sliceIdx)
					logger.Println("routeRecordInfoNew.sliceIdx = ", routeInfoRecordNew.sliceIdx)
					routeInfoRecordNew.sliceIdx = routeInfoRecordTempSelected.sliceIdx
					policyRoute.SliceIdx  = ribd.Int(routeInfoRecordNew.sliceIdx)
					params.sliceIdx = ribd.Int(routeInfoRecordNew.sliceIdx)
					policyPath = ribdCommonDefs.PolicyPath_Import //since this is dynamic, we need to go over the poilicyEngine from the import path
					destNetSlice[routeInfoRecord.sliceIdx].isValid = true
				}
		} else {
			logger.Println("Deleted route was not the selected route")
			if routeInfoRecordList.selectedRouteIdx < int8(index) {
				logger.Println("Selected route index less than the deleted route index, no adjustments needed")
			} else {
				logger.Println("Selected route index greater than deleted route index, adjust the selected route index")
				routeInfoRecordList.selectedRouteIdx--
			}
		}
	}
	//update the patriciaDB trie with the updated route info record list
	t1 := time.Now()
	routeInfoRecordList.routeUpdatedTime = t1.String()
	RouteInfoMap.Set(patriciaDB.Prefix(destNetPrefix), routeInfoRecordList)

	if deleteRoute == true || routeInfoRecordOld.protocol != PROTOCOL_NONE {
		params.routeType = policyRoute.Prototype
		params.createType = Invalid
		params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
		policyRoute.PolicyList = routeInfoRecordList.policyList
		if deleteRoute == true {
			logger.Println("Deleting the selected route, so call asicd to delete")
		}
		if routeInfoRecordOld.protocol != PROTOCOL_NONE {
			logger.Println("routeInfoRecordOld.protocol != PROTOCOL_NONE - adding a better route, so call asicd to delete")
		    policyRoute.Prototype = ribd.Int(routeInfoRecordOld.protocol)
		    params.routeType = policyRoute.Prototype
		    params.destNetIp = routeInfoRecordOld.destNetIp.String()
		    params.networkMask = routeInfoRecordOld.networkMask.String()
		    policyRoute.Ipaddr = routeInfoRecordOld.destNetIp.String()
		    policyRoute.Mask = routeInfoRecordOld.networkMask.String()
		}
		//call asicd to del
		if asicdclnt.IsConnected {
		    logger.Println("call asicd to delete route - ip", routeInfoRecord.destNetIp.String(), " mask ", routeInfoRecord.networkMask.String())
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
		}
		delLinuxRoute(routeInfoRecord)
		//update in the event log
	    eventInfo := "Deleted route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	    t1 := time.Now()
        routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	    localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
		routeInfoRecordList.selectedRouteIdx = PROTOCOL_NONE
	    PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export,params )
	}
	if routeInfoRecordNew.protocol != PROTOCOL_NONE {
		policyRoute.Prototype = ribd.Int(routeInfoRecordNew.protocol)
		params.routeType = policyRoute.Prototype
		params.destNetIp = routeInfoRecordNew.destNetIp.String()
		params.networkMask = routeInfoRecordNew.networkMask.String()
		policyRoute.Ipaddr = routeInfoRecordNew.destNetIp.String()
		policyRoute.Mask = routeInfoRecordNew.networkMask.String()
		if policyPath == ribdCommonDefs.PolicyPath_Export {
		  logger.Println("New route selected, call asicd to install a new route - ip", routeInfoRecordNew.destNetIp.String(), " mask ", routeInfoRecordNew.networkMask.String(), " nextHopIP ",routeInfoRecordNew.nextHopIp.String())
		  //call asicd to add
		  if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecordNew.destNetIp.String(), routeInfoRecordNew.networkMask.String(), routeInfoRecordNew.nextHopIp.String())
		  }
		  if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
			//call arpd to resolve the ip
			logger.Println("### Sending ARP Resolve for ", routeInfoRecordNew.nextHopIp.String(), routeInfoRecordNew.nextHopIfType)
			arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecordNew.nextHopIp.String(), arpd.Int(routeInfoRecordNew.nextHopIfType), arpd.Int(routeInfoRecordNew.nextHopIfIndex))
			//arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.destNetIp.String(), arpd.Int(routeInfoRecord.nextHopIfIndex))
		  }
		  addLinuxRoute(routeInfoRecordNew)
		  //update in the event log
	      eventInfo := "Created route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	      t1 := time.Now()
          routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	      localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
		}
		params.deleteType = Invalid
	    PolicyEngineFilter(policyRoute, policyPath,params )
	}
	return nil
}*/
/*func updateBestRoute(destNet patriciaDB.Prefix, routeInfoRecordList RouteInfoRecordList) {
	logger.Println("updateBestRoute for ip network ", destNet)
	var routeInfoRecordNew RouteInfoRecord
    var routeInfoRecord RouteInfoRecord
	var routeInfoRecordTempSelected RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	var deleteRoute, addRoute bool
	selectedRouteIdx := routeInfoRecordList.selectedRouteIdx
	var policyPath,i int
	var policyRoute ribd.Routes
	var params RouteParams
	if routeInfoRecordList.routeInfoList == nil {
		logger.Println("routeInfoList empty")
		return
	}
	if routeInfoRecordList.selectedRouteIdx == -1 {
		logger.Println("routeInfoList selectedrouteIdx invalid")
	    routeInfoRecord.protocol = PROTOCOL_NONE
	} else {
	    routeInfoRecord = routeInfoRecordList.routeInfoList[selectedRouteIdx]
	}
		
	routeInfoRecordTempSelected.protocol = PROTOCOL_NONE
	routeInfoRecordNew.protocol = PROTOCOL_NONE
	for i = 0; i < len(routeInfoRecordList.routeInfoList); i++ {
	   routeInfoRecordTemp = routeInfoRecordList.routeInfoList[i]
	   logger.Printf("temp protocol=%d, routeInfoRecordTempSelected.protocol=%d\n", routeInfoRecordTemp.protocol, routeInfoRecordTempSelected.protocol)
	 /*  if (isBetterRoute(routeInfoRecordTempSelected, routeInfoRecordTemp)) {
			logger.Println(" evaluating route at index ",i, "routeInfoRecordTemp.protocol = ", ReverseRouteProtoTypeMapDB[int(routeInfoRecordTemp.protocol)], " tempselected protocol = ", ReverseRouteProtoTypeMapDB[int(routeInfoRecordTempSelected.protocol)])
             routeInfoRecordTempSelected = routeInfoRecordTemp
			 selectedRouteIdx = int8(i)
	   }*/
/*	}
	logger.Println("selected route at index ",selectedRouteIdx, " tempselected protocol = ", ReverseRouteProtoTypeMapDB[int(routeInfoRecordTempSelected.protocol)])
	if selectedRouteIdx == routeInfoRecordList.selectedRouteIdx {
		logger.Println("update route selected the same route as the best route")
		return
	}
	logger.Println("New route selected")
	if routeInfoRecord.protocol != PROTOCOL_NONE {
		logger.Println("There was a valid route selected earlier, delete that")
		deleteRoute = true
		routeInfoRecordList.selectedRouteIdx = PROTOCOL_NONE
	}
	if routeInfoRecordTempSelected.protocol != PROTOCOL_NONE{
		logger.Println("There is a valid new route selected, add that")
		routeInfoRecordNew = routeInfoRecordTempSelected
		//routeInfoRecordList.selectedRouteIdx = int8(selectedRouteIdx)
		logger.Println("routeRecordInfo.sliceIdx = ", routeInfoRecord.sliceIdx)
		logger.Println("routeRecordInfoNew.sliceIdx = ", routeInfoRecordNew.sliceIdx)
		routeInfoRecordNew.sliceIdx = routeInfoRecordTempSelected.sliceIdx
		policyRoute.SliceIdx  = ribd.Int(routeInfoRecordNew.sliceIdx)
		params.sliceIdx = ribd.Int(routeInfoRecordNew.sliceIdx)
		policyPath = ribdCommonDefs.PolicyPath_Import //since this is dynamic, we need to go over the poilicyEngine from the import path
		addRoute = true
	}
	//update the patriciaDB trie with the updated route info record list
	t1 := time.Now()
	routeInfoRecordList.routeUpdatedTime = t1.String()
	RouteInfoMap.Set(patriciaDB.Prefix(destNet), routeInfoRecordList)
	
	if deleteRoute == true  {
		params.routeType = policyRoute.Prototype
		params.createType = Invalid
		params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
		policyRoute.PolicyList = routeInfoRecordList.policyList
		if deleteRoute == true {
			logger.Println("Deleting the selected route, so call asicd to delete")
		}
	    policyRoute = ribd.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecord.nextHopIfType), IfIndex: routeInfoRecord.nextHopIfIndex, Metric: routeInfoRecord.metric, Prototype: ribd.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
		params.routeType = policyRoute.Prototype
		params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
		//call asicd to del
		if asicdclnt.IsConnected {
		    logger.Println("call asicd to delete route - ip", routeInfoRecord.destNetIp.String(), " mask ", routeInfoRecord.networkMask.String())
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
		}
		delLinuxRoute(routeInfoRecord)
		//update in the event log
	    eventInfo := "Deleted route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	    t1 := time.Now()
        routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	    localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
	    PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export,params )
	}
	if addRoute == true && routeInfoRecordNew.protocol != PROTOCOL_NONE {
	   policyRoute := ribd.Routes{Ipaddr: routeInfoRecordNew.destNetIp.String(), Mask: routeInfoRecordNew.networkMask.String(), NextHopIp: routeInfoRecordNew.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecordNew.nextHopIfType), IfIndex: routeInfoRecordNew.nextHopIfIndex, Metric: routeInfoRecordNew.metric, Prototype: ribd.Int(routeInfoRecordNew.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
		params.routeType = policyRoute.Prototype
		params.destNetIp = routeInfoRecordNew.destNetIp.String()
		params.networkMask = routeInfoRecordNew.networkMask.String()
		if policyPath == ribdCommonDefs.PolicyPath_Export {
		  logger.Println("New route selected, call asicd to install a new route - ip", routeInfoRecordNew.destNetIp.String(), " mask ", routeInfoRecordNew.networkMask.String(), " nextHopIP ",routeInfoRecordNew.nextHopIp.String())
		  //call asicd to add
		  if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecordNew.destNetIp.String(), routeInfoRecordNew.networkMask.String(), routeInfoRecordNew.nextHopIp.String())
		  }
		  if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
			//call arpd to resolve the ip
			logger.Println("### Sending ARP Resolve for ", routeInfoRecordNew.nextHopIp.String(), routeInfoRecordNew.nextHopIfType)
			arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecordNew.nextHopIp.String(), arpd.Int(routeInfoRecordNew.nextHopIfType), arpd.Int(routeInfoRecordNew.nextHopIfIndex))
			//arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.destNetIp.String(), arpd.Int(routeInfoRecord.nextHopIfIndex))
		  }
		  addLinuxRoute(routeInfoRecordNew)
		  //update in the event log
	      eventInfo := "Created route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	      t1 := time.Now()
          routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	      localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
		}
		params.deleteType = Invalid
	    PolicyEngineFilter(policyRoute, policyPath,params )
	}
}
*/
/**
   This function is called when :
 - a user/routing protocol installs a new route. In that case, addType will be RIBAndFIB
 - when a operationally down link comes up. In this case, the addType will be FIBOnly because on a link down, the route is still preserved in the RIB database and only deleted from FIB (Asic)
**/
func createV4Route(destNetIp string,
	networkMask string,
	metric ribd.Int,
	nextHopIp string,
	nextHopIfType ribd.Int,
	nextHopIfIndex ribd.Int,
	routeType ribd.Int,
	addType ribd.Int,
	policyStateChange int,
	sliceIdx ribd.Int) (rc ribd.Int, err error) {
	logger.Printf("createV4Route for ip %s mask %s next hop ip %s addType %d\n", destNetIp, networkMask, nextHopIp, addType)

    callSelectRoute := false
	destNetIpAddr, err := getIP(destNetIp)
	if err != nil {
		logger.Println("destNetIpAddr invalid")
		return 0, err
	}
	networkMaskAddr, err := getIP(networkMask)
	if err != nil {
		logger.Println("networkMaskAddr invalid")
		return 0, err
	}
	nextHopIpAddr, err := getIP(nextHopIp)
	if err != nil {
		logger.Println("nextHopIpAddr invalid")
		return 0, err
	}
	/*	prefixLen, err := getPrefixLen(networkMaskAddr)
		if(err != nil) {
			return -1, err
		}*/
	destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		return -1, err
	}
	routePrototype := int8(routeType)
	/*	routePrototype, err := setProtocol(routeType)
		if err != nil {
			return 0, err
		}*/
	logger.Printf("routePrototype %d for routeType %d prefix %v", routePrototype, routeType, destNet)
	policyRoute := ribd.Routes{Ipaddr: destNetIp, Mask: networkMask, NextHopIp: nextHopIp, NextHopIfType: nextHopIfType, IfIndex: nextHopIfIndex, Metric: metric, Prototype: routeType}
	routeInfoRecord := RouteInfoRecord{destNetIp: destNetIpAddr, networkMask: networkMaskAddr, protocol: routePrototype, nextHopIp: nextHopIpAddr, nextHopIfType: int8(nextHopIfType), nextHopIfIndex: nextHopIfIndex, metric: metric, sliceIdx: int(sliceIdx)}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		if addType == FIBOnly {
			logger.Println("route record list not found in RIB")
			err = errors.New("Unexpected: route record list not found in RIB")
			return 0, err
		}
		var newRouteInfoRecordList RouteInfoRecordList
		newRouteInfoRecordList.routeInfoProtocolMap = make(map[string][]RouteInfoRecord)
        newRouteInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeType)]] = make([]RouteInfoRecord,0)
		//newRouteInfoRecordList.routeInfoList = make([]RouteInfoRecord, 0)
		newRouteInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeType)]] = append(newRouteInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routeType)]], routeInfoRecord)
		newRouteInfoRecordList.selectedRouteProtocol = ReverseRouteProtoTypeMapDB[int(routeType)]
		if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
			newRouteInfoRecordList.isPolicyBasedStateValid = false
		} else if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
			newRouteInfoRecordList.isPolicyBasedStateValid = true
		}
		t1 := time.Now()
		newRouteInfoRecordList.routeCreatedTime = t1.String()
		if ok := RouteInfoMap.Insert(destNet, newRouteInfoRecordList); ok != true {
			logger.Println(" return value not ok")
		}
		localDBRecord := localDB{prefix: destNet, isValid: true, nextHopIp:nextHopIp}
		if destNetSlice == nil {
			destNetSlice = make([]localDB, 0)
		}
		destNetSlice = append(destNetSlice, localDBRecord)
		//call asicd
		if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String(), routeInfoRecord.nextHopIp.String())
		}
		 
		if arpdclnt.IsConnected && routeType != ribdCommonDefs.CONNECTED {
			logger.Println("### 22 Sending ARP Resolve for ", routeInfoRecord.nextHopIp.String(), routeInfoRecord.nextHopIfType)
			arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.nextHopIp.String(), arpd.Int(routeInfoRecord.nextHopIfType), arpd.Int(routeInfoRecord.nextHopIfIndex))
		}
		addLinuxRoute(routeInfoRecord)
		//update in the event log
	    eventInfo := "Created route "+policyRoute.Ipaddr+" "+policyRoute.Mask+" type" + ReverseRouteProtoTypeMapDB[int(policyRoute.Prototype)]
	    t1 = time.Now()
        routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	    localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
		
	    var params RouteParams
		params.destNetIp = destNetIp
		params.networkMask = networkMask
		params.routeType = routeType
		params.createType = addType
		params.deleteType = Invalid
		policyRoute.IsPolicyBasedStateValid = newRouteInfoRecordList.isPolicyBasedStateValid
		PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export, params)
	} else {
		logger.Println("routeInfoRecordListItem not nil")
		routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
		found := IsRoutePresent(routeInfoRecordList, ReverseRouteProtoTypeMapDB[int(routeType)])
		if found && (addType == FIBAndRIB) {
			routeInfoList := routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]]
			logger.Println("Trying to create a duplicate route of protocol type ", ReverseRouteProtoTypeMapDB[int(routePrototype)])
            if routeInfoList[0].metric > metric {
				logger.Println("New route has a better metric")
				//delete all existing routes 
				//call asicd to delete if it is the selected protocol
				//add this new route and configure in asicd
                 if routeInfoRecordList.selectedRouteProtocol == ReverseRouteProtoTypeMapDB[int(routePrototype)] {
					logger.Println("Adding a equal cost route for the selected route")
					callSelectRoute = true
				}
			} else if routeInfoList[0].metric == metric {
			    if !newNextHopIP(nextHopIp,routeInfoList) {
				  logger.Println("same cost and next hop ip, so reject this route")
			      err = errors.New("Duplicate route creation")
				  return 0,err
			    }
				//adding equal cost route
				routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]] = append(routeInfoRecordList.routeInfoProtocolMap[ReverseRouteProtoTypeMapDB[int(routePrototype)]], routeInfoRecord)
                 if routeInfoRecordList.selectedRouteProtocol == ReverseRouteProtoTypeMapDB[int(routePrototype)] {
					logger.Println("Adding a equal cost route for the selected route")
					callSelectRoute = true
				}
			} else {//if metric > routeInfoRecordList.routeInfoList[idx].metric 
			    err = errors.New("Duplicate route creation with higher cost, rejecting the route")
				return 0,err
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
		   err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, add)//, len(routeInfoRecordList.routeInfoList)-1)
        }
	}
	if addType != FIBOnly && routePrototype == ribdCommonDefs.CONNECTED { //PROTOCOL_CONNECTED {
		updateConnectedRoutes(destNetIp, networkMask, nextHopIp, nextHopIfIndex, nextHopIfType, add, sliceIdx)
	}
	return 0, err

}
func (m RouteServiceHandler) CreateV4Route(destNetIp string,
	networkMask string,
	metric ribd.Int,
	nextHopIp string,
	nextHopIfType ribd.Int,
	nextHopIfIndex ribd.Int,
	routeTypeString string) (rc ribd.Int, err error) {
	logger.Printf("Received create route request for ip %s mask %s\n", destNetIp, networkMask)
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return 0, err
	}
    routeType,ok := RouteProtocolTypeMapDB[routeTypeString]
	if !ok {
		logger.Println("route type ", routeTypeString, " invalid")
		err=errors.New("Invalid route protocol type")
		return rc,err
	}
	policyRoute := ribd.Routes{Ipaddr: destNetIp, Mask: networkMask, NextHopIp: nextHopIp, NextHopIfType: nextHopIfType, IfIndex: nextHopIfIndex, Metric: metric, Prototype: ribd.Int(routeType)}
	params := RouteParams{destNetIp: destNetIp, networkMask: networkMask, nextHopIp: nextHopIp, nextHopIfType: nextHopIfType, nextHopIfIndex: nextHopIfIndex, metric: metric, routeType: ribd.Int(routeType), sliceIdx: ribd.Int(len(destNetSlice)), createType: FIBAndRIB, deleteType: Invalid}
	logger.Println("createType = ", params.createType, "deleteType = ", params.deleteType)
	PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Import, params)

	/*	_, err = createV4Route(destNetIp, networkMask, metric, nextHopIp, nextHopIfType, nextHopIfIndex, routeType, FIBAndRIB, ribd.Int(len(destNetSlice)))

		if err != nil {
			logger.Println("creating v4 route failed with err ", err)
			return 0, err
		}

		//pass through policy engine
		policyRoute := ribd.Routes{Ipaddr: destNetIp, Mask: networkMask, NextHopIp: nextHopIp, NextHopIfType: nextHopIfType, IfIndex: nextHopIfIndex, Metric: metric, Prototype: routeType}
		PolicyEngineFilter(policyRoute, ribdCommonDefs.NOTIFY_ROUTE_CREATED)*/

	/*
		//If this is not a connected route, then nothing more to do
		if routeType == ribdCommonDefs.CONNECTED {
			logger.Println("This is a connected route, so send a route add event")
		} else if routeType == ribdCommonDefs.STATIC {
			logger.Println("This is a static route, so send a route add event")
		} else {
			logger.Println(" This is neither a connected nor a static route, so nothing more to do")
			return 0, err
		}

		//Send a event
		logger.Println("This is a temporary notification till policies take effect")
		route := ribd.Routes{Ipaddr: destNetIp, Mask: networkMask, NextHopIp: nextHopIp, NextHopIfType: nextHopIfType, IfIndex: nextHopIfIndex, Metric: metric}
		RouteNotificationSend(RIBD_PUB, route)
	*/
	return 0, err
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
	logger.Println("deleteV4Route  with del type ", delType)

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
	logger.Printf("destNet = %v\n", destNet)
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		return 0, nil
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		return 0, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found := IsRoutePresent(routeInfoRecordList, routeType)
	if !found {
		logger.Println("Route with protocol ", routeType, " not found")
		return 0, err
	}
	if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
		routeInfoRecordList.isPolicyBasedStateValid = false
	} else if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
		routeInfoRecordList.isPolicyBasedStateValid = true
	}
	found,routeInfoRecord,_ := findRouteWithNextHop(routeInfoRecordList.routeInfoProtocolMap[routeType], nextHopIP)
	if !found {
		logger.Println("Route with nextHop IP ", nextHopIP, " not found")
		return 0, err
	}
	SelectV4Route(destNet,routeInfoRecordList,routeInfoRecord,del)
/*	routeInfoRecord := routeInfoRecordList.routeInfoList[idxList[0]]
	var prefixNodeRouteList RouteInfoRecordList
	var prefixNodeRoute RouteInfoRecord
	logger.Printf("Fetching trie record for prefix %v\n", destNet)
	prefixNode := RouteInfoMap.Get(destNet)
	if prefixNode != nil {
		prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
		logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
		prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
		logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
	}
	if delType != FIBOnly { //if this is not FIBOnly, then we have to delete this route from the RIB data base as well.
		routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList[:idxList[0]], routeInfoRecordList.routeInfoList[idxList[0]+1:]...)
	}
	logger.Printf("Fetching trie record for prefix after append%v\n", destNet)
	prefixNode = RouteInfoMap.Get(destNet)
	if prefixNode != nil {
		prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
		logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
		prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
		logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
	}
	err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, del, int(idxList[0])) //this function will invalidate the route in destNetSlice and also delete the entry in FIB (Asic)
	logger.Printf("Fetching trie record for prefix after selectv4route%v\n", destNet)
	prefixNode = RouteInfoMap.Get(destNet)
	if prefixNode != nil {
		prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
		logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
		prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
		logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
	}*/

	if routeType == "CONNECTED" { //PROTOCOL_CONNECTED {
		if delType == FIBOnly { //link gone down, just invalidate the connected route
			updateConnectedRoutes(destNetIp, networkMask, "", 0, 0, invalidate, 0)
		} else {
			updateConnectedRoutes(destNetIp, networkMask, "", 0, 0, del, 0)
		}
	}
	return 0, err
}

func (m RouteServiceHandler) DeleteV4Route(destNetIp string,
	networkMask string,
	routeTypeString string,
	nextHopIP string) (rc ribd.Int, err error) {
	logger.Println("Received Route Delete request for ", destNetIp,":",networkMask, "nextHopIP:",nextHopIP,"Protocol ",routeTypeString )
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return 0,err
	}
	_, err = deleteV4Route(destNetIp, networkMask, routeTypeString, nextHopIP,FIBAndRIB, ribdCommonDefs.RoutePolicyStateChangetoInValid)
	return 0, err
}
func (m RouteServiceHandler) UpdateIPV4Route(origconfig *ribd.Routes, newconfig *ribd.Routes, attrset []bool) (val bool, err error) {

/*func (m RouteServiceHandler) UpdateV4Route(destNetIp string,
	networkMask string,
	routeType ribd.Int,
	nextHopIp string,
	//	nextHopIfType ribd.Int,
	nextHopIfIndex ribd.Int,
	metric ribd.Int) (err error) {*/
	logger.Println("Received update route request")
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return err
	}
	destNet, err := getNetowrkPrefixFromStrings(origconfig.Ipaddr, origconfig.Mask)
	if err != nil {
		logger.Println(" getNetowrkPrefixFromStrings returned err ", err)
		return val, err
	}
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return val,err
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Println("No route for destination network")
		return val,err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	if attrset != nil {
		logger.Println("attr set not nil, set individual attributes")
	}
	updateBestRoute(destNet, routeInfoRecordList)
	return val,err
}

func printRoutesInfo(prefix patriciaDB.Prefix, item patriciaDB.Item) (err error) {
	rmapInfoRecordList := item.(RouteInfoRecordList)
	for _, v := range rmapInfoRecordList.routeInfoProtocolMap {
		if v == nil || len(v) == 0 {
			continue
		}
		for i:=0;i<len(v);i++ {
		//   logger.Printf("%v-> %d %d %d %d\n", prefix, v.destNetIp, v.networkMask, v.protocol)
		count++
		}
	}
	return nil
}

func (m RouteServiceHandler) PrintV4Routes() (err error) {
	count = 0
	logger.Println("Received print route")
	RouteInfoMap.Visit(printRoutesInfo)
	logger.Printf("total count = %d\n", count)
	return nil
}
