// ribdUtils.go
package main

import (
	"ribd"
	"encoding/json"
	"github.com/op/go-nanomsg"
	"net"
	"errors"
	"strconv"
	"utils/patriciaDB"
	"github.com/vishvananda/netlink"
	"asicd/asicdConstDefs"
	"l3/rib/ribdCommonDefs"
	"time"
	"sort"
)
type RouteDistanceConfig struct{
	defaultDistance int
	configuredDistance int
}
type AdminDistanceSlice []ribd.RouteDistanceState
var RouteProtocolTypeMapDB = make(map[string]int)
var ReverseRouteProtoTypeMapDB = make(map[int]string)
var ProtocolAdminDistanceMapDB = make(map[string]RouteDistanceConfig)
var ProtocolAdminDistanceSlice AdminDistanceSlice 

func BuildRouteProtocolTypeMapDB() {
	RouteProtocolTypeMapDB["CONNECTED"] = ribdCommonDefs.CONNECTED
	RouteProtocolTypeMapDB["EBGP"]       = ribdCommonDefs.EBGP
	RouteProtocolTypeMapDB["IBGP"]       = ribdCommonDefs.IBGP
	RouteProtocolTypeMapDB["BGP"]       = ribdCommonDefs.BGP
	RouteProtocolTypeMapDB["STATIC"]       = ribdCommonDefs.STATIC
	
	//reverse
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.CONNECTED] = "CONNECTED"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.IBGP] = "IBGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.EBGP] = "EBGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.BGP] = "BGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.STATIC] = "STATIC"
}
func BuildProtocolAdminDistanceMapDB() {
	ProtocolAdminDistanceMapDB["CONNECTED"] = RouteDistanceConfig{defaultDistance:0, configuredDistance:-1}
	ProtocolAdminDistanceMapDB["STATIC"]       = RouteDistanceConfig{defaultDistance:1, configuredDistance:-1}	
	ProtocolAdminDistanceMapDB["EBGP"]       = RouteDistanceConfig{defaultDistance:20, configuredDistance:-1}
	ProtocolAdminDistanceMapDB["IBGP"]       = RouteDistanceConfig{defaultDistance:200, configuredDistance:-1}
	ProtocolAdminDistanceMapDB["OSPF"]       = RouteDistanceConfig{defaultDistance:110, configuredDistance:-1}
}
func (slice AdminDistanceSlice ) Len() int {
	return len(slice )
}
func (slice AdminDistanceSlice ) Less(i,j int) bool {
	return slice[i].Distance < slice[j].Distance
}
func (slice AdminDistanceSlice ) Swap(i,j int) {
     slice[i].Protocol,slice[j].Protocol = slice[j].Protocol, slice[i].Protocol
     slice[i].Distance,slice[j].Distance = slice[j].Distance, slice[i].Distance
}
func BuildProtocolAdminDistanceSlice() {
	distance :=0
	protocol:=""
	ProtocolAdminDistanceSlice = nil
	ProtocolAdminDistanceSlice = make([]ribd.RouteDistanceState,0)
	for k,v:=range ProtocolAdminDistanceMapDB {
		protocol=k
		distance = v.defaultDistance
		if v.configuredDistance != -1 {
			distance = v.configuredDistance
		}
		routeDistance:=ribd.RouteDistanceState{Protocol:protocol,Distance:ribd.Int(distance)}
		ProtocolAdminDistanceSlice = append(ProtocolAdminDistanceSlice,routeDistance)
	}
	sort.Sort(ProtocolAdminDistanceSlice)
}
/*func isBetterRoute(selectedRoute RouteInfoRecord, routeInfoRecord RouteInfoRecord) (isBetter bool){ 
   logger.Println("isBetterRoute ")
   if (selectedRoute.protocol == PROTOCOL_NONE && routeInfoRecord.protocol != PROTOCOL_NONE) {
      logger.Println("new route is better route because the the current route protocol is ", PROTOCOL_NONE)
      isBetter = true
   } else if ProtocolAdminDistanceMapDB[int(routeInfoRecord.protocol)].configuredDistance < ProtocolAdminDistanceMapDB[int(selectedRoute.protocol)].configuredDistance {
      logger.Println("New route is better because configured admin distance", ProtocolAdminDistanceMapDB[int(routeInfoRecord.protocol)].configuredDistance ," of new route ", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], " is lower than the current protocol ",ReverseRouteProtoTypeMapDB[int(selectedRoute.protocol)],"'s configured admin distane: ", ProtocolAdminDistanceMapDB[int(selectedRoute.protocol)].configuredDistance)
      isBetter = true	
   } else if ProtocolAdminDistanceMapDB[int(routeInfoRecord.protocol)].defaultDistance < ProtocolAdminDistanceMapDB[int(selectedRoute.protocol)].defaultDistance {
      logger.Println("New route is better because default admin distance", ProtocolAdminDistanceMapDB[int(routeInfoRecord.protocol)].defaultDistance ," of new route ", ReverseRouteProtoTypeMapDB[int(routeInfoRecord.protocol)], " is lower than the current protocol ",ReverseRouteProtoTypeMapDB[int(selectedRoute.protocol)],"'s default admin distane: ", ProtocolAdminDistanceMapDB[int(selectedRoute.protocol)].defaultDistance)
      isBetter = true	
   } else if routeInfoRecord.metric < selectedRoute.metric {
      logger.Println("New route is better becayse its cost: ", routeInfoRecord.metric, " is lower than the selected route's cost ", selectedRoute.metric)
      isBetter = true	
   }
   return isBetter
}*/
func findRouteWithNextHop(routeInfoList []RouteInfoRecord, nextHopIP string) (found bool, routeInfoRecord RouteInfoRecord, index int) {
	logger.Println("findRouteWithNextHop")
	index = -1
	for i:=0;i<len(routeInfoList);i++ {
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
	for i:=0;i<len(routeInfoList);i++ {
		if routeInfoList[i].nextHopIp.String() == ip {
			logger.Println("Next hop IP already present")
			isNewNextHopIP = false
		}
	}
	return isNewNextHopIP
}
func isSameRoute(selectedRoute ribd.Routes, route ribd.Routes) (same bool) {
	logger.Println("isSameRoute")
	if selectedRoute.Ipaddr == route.Ipaddr && selectedRoute.Mask == route.Mask && selectedRoute.Prototype == route.Prototype {
		same = true
	}
	return same
} 
func getNetowrkPrefixFromStrings(ipAddr string, mask string) (prefix patriciaDB.Prefix, err error) {
	destNetIpAddr, err := getIP(ipAddr)
	if err != nil {
		logger.Println("destNetIpAddr invalid")
		return prefix, err
	}
	networkMaskAddr, err := getIP(mask)
	if err != nil {
		logger.Println("networkMaskAddr invalid")
		return prefix, err
	}
	prefix, err = getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		logger.Println("err=", err)
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
	ipPrefix ,err= getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
    return ipPrefix, err
}
func addPolicyRouteMap(route ribd.Routes, policyName string) {
	logger.Println("addPolicyRouteMap")
	ipPrefix,err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		logger.Println("Invalid ip prefix")
		return
	}
	maskIp, err := getIP(route.Mask)
	if err != nil {
		return
	}
	prefixLen,err := getPrefixLen(maskIp)
	if err != nil {
		return
	}
	logger.Println("prefixLen= ", prefixLen)
	var newRoute string
	found := false
	newRoute = route.Ipaddr + "/"+strconv.Itoa(prefixLen)
//	newRoute := string(ipPrefix[:])
	logger.Println("Adding ip prefix %s %v ", newRoute, ipPrefix)
	policyInfo:=PolicyEngineDB.PolicyDB.Get(patriciaDB.Prefix(policyName))
	if policyInfo == nil {
		logger.Println("Unexpected:policyInfo nil for policy ", policyName)
		return
	}
	tempPolicy:=policyInfo.(Policy)
	tempPolicy.hitCounter++
	if tempPolicy.routeList == nil {
		logger.Println("routeList nil")
		tempPolicy.routeList = make([]string, 0)
	}
	logger.Println("routelist len= ", len(tempPolicy.routeList)," prefix list so far")
	for i:=0;i<len(tempPolicy.routeList);i++ {
		logger.Println(tempPolicy.routeList[i])
		if tempPolicy.routeList[i] == newRoute {
			logger.Println(newRoute, " already is a part of ", policyName, "'s routelist")
			found = true
		}
	}
	if !found {
       tempPolicy.routeList = append(tempPolicy.routeList, newRoute)
	}
	found=false
	logger.Println("routeInfoList details")
	for i:=0;i<len(tempPolicy.routeInfoList);i++ {
		logger.Println("IP: ",tempPolicy.routeInfoList[i].Ipaddr, ":", tempPolicy.routeInfoList[i].Mask, " routeType: ", tempPolicy.routeInfoList[i].Prototype)
		if tempPolicy.routeInfoList[i].Ipaddr==route.Ipaddr && tempPolicy.routeInfoList[i].Mask == route.Mask && tempPolicy.routeInfoList[i].Prototype == route.Prototype {
			logger.Println("route already is a part of ", policyName, "'s routeInfolist")
			found = true
		}
	}
	if tempPolicy.routeInfoList == nil {
		tempPolicy.routeInfoList = make([]ribd.Routes, 0)
	}
	if found == false {
       tempPolicy.routeInfoList = append(tempPolicy.routeInfoList, route)
	}
	PolicyEngineDB.PolicyDB.Set(patriciaDB.Prefix(policyName), tempPolicy)
}
func deletePolicyRouteMap(route ribd.Routes, policyName string) {
	logger.Println("deletePolicyRouteMap")
}
func updatePolicyRouteMap(route ribd.Routes, policy string, op int) {
	logger.Println("updatePolicyRouteMap")
	if op == add {
		addPolicyRouteMap(route, policy)
	} else if op == del {
		deletePolicyRouteMap(route, policy)
	}
	
}

func deleteRoutePolicyStateAll(route ribd.Routes) {
	logger.Println("deleteRoutePolicyStateAll")
	destNet, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		return 
	}

	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
       logger.Println(" entry not found for prefix %v", destNet)
	   return
	}
    routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	routeInfoRecordList.policyHitCounter = route.PolicyHitCounter
	routeInfoRecordList.policyList = nil//append(routeInfoRecordList.policyList[:0])
	RouteInfoMap.Set(destNet,routeInfoRecordList)
	return
}
func addRoutePolicyState(route ribd.Routes, policy string, policyStmt string) {
	logger.Println("addRoutePolicyState")
	destNet, err := getNetowrkPrefixFromStrings(route.Ipaddr, route.Mask)
	if err != nil {
		return 
	}

	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
       logger.Println("Unexpected - entry not found for prefix %v", destNet)
	   return
	}
	logger.Println("Adding policy ", policy, " to route %v", destNet)
    routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	routeInfoRecordList.policyHitCounter = route.PolicyHitCounter
	if routeInfoRecordList.policyList == nil {
		routeInfoRecordList.policyList = make([]string,0)
	}
/*	policyStmtList := routeInfoRecordList.policyList[policy]
	if policyStmtList == nil {
	   policyStmtList = make([]string,0)
	}
	policyStmtList = append(policyStmtList,policyStmt)
    routeInfoRecordList.policyList[policy] = policyStmtList*/
	routeInfoRecordList.policyList = append(routeInfoRecordList.policyList, policy)
	RouteInfoMap.Set(destNet,routeInfoRecordList)
	return
}
func	 addPolicyRouteMapEntry(route *ribd.Routes, policy string, policyStmt string, conditionList []string, actionList []string) {
	logger.Println("addPolicyRouteMapEntry")
	var policyStmtMap PolicyStmtMap
	var conditionsAndActionsList ConditionsAndActionsList
	if PolicyRouteMap == nil {
		PolicyRouteMap = make(map[PolicyRouteIndex]PolicyStmtMap)
	}
	policyRouteIndex := PolicyRouteIndex{routeIP:route.Ipaddr, routeMask:route.Mask,policy:policy}
	policyStmtMap, ok:= PolicyRouteMap[policyRouteIndex]
	if !ok {
		policyStmtMap.policyStmtMap = make(map[string]ConditionsAndActionsList)
	}
	_, ok = policyStmtMap.policyStmtMap[policyStmt]
	if ok {
		logger.Println("policy statement map for statement ", policyStmt, " already in place for policy ", policy)
		return
	} 
	conditionsAndActionsList.conditionList = make([]string,0)
	conditionsAndActionsList.actionList = make([]string,0)
	for i:=0;conditionList != nil && i<len(conditionList);i++ {
		conditionsAndActionsList.conditionList = append(conditionsAndActionsList.conditionList,conditionList[i])
	}
	for i:=0;actionList != nil && i<len(actionList);i++ {
		conditionsAndActionsList.actionList = append(conditionsAndActionsList.actionList,actionList[i])
	}
	policyStmtMap.policyStmtMap[policyStmt]=conditionsAndActionsList
	PolicyRouteMap[policyRouteIndex]=policyStmtMap
}
func deleteRoutePolicyState( ipPrefix patriciaDB.Prefix, policyName string) {
	logger.Println("deleteRoutePolicyState")
	found := false
	idx :=0
	routeInfoRecordListItem := RouteInfoMap.Get(ipPrefix)
	if routeInfoRecordListItem == nil {
		logger.Println("routeInfoRecordListItem nil for prefix ",ipPrefix)
		return
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
/*    if routeInfoRecordList.policyList[policyName] != nil {
		delete(routeInfoRecordList.policyList, policyName)
	}*/
	for idx = 0;idx<len(routeInfoRecordList.policyList);idx++ {
		if routeInfoRecordList.policyList[idx] == policyName {
			found = true
			break
		}
	}
	if !found {
		logger.Println("Policy ", policyName, "not found in policyList of route ", ipPrefix)
		return
	}
	routeInfoRecordList.policyList = append(routeInfoRecordList.policyList[:idx], routeInfoRecordList.policyList[idx+1:]...)
	RouteInfoMap.Set(ipPrefix, routeInfoRecordList)
}
func	 deletePolicyRouteMapEntry(route ribd.Routes, policy string) {
	logger.Println("deletePolicyRouteMapEntry for policy ", policy, "route ", route.Ipaddr, ":", route.Mask)
	if PolicyRouteMap == nil {
		logger.Println("PolicyRouteMap empty")
		return
	}
	policyRouteIndex := PolicyRouteIndex{routeIP:route.Ipaddr,routeMask:route.Mask, policy:policy}
	//PolicyRouteMap[policyRouteIndex].policyStmtMap=nil
	delete(PolicyRouteMap,policyRouteIndex)
}

func updateRoutePolicyState(route ribd.Routes, op int, policy string, policyStmt string) {
	logger.Println("updateRoutePolicyState")
	if op == delAll {
		deleteRoutePolicyStateAll(route)
		deletePolicyRouteMapEntry(route, policy)
	} else if op == add {
		addRoutePolicyState(route, policy, policyStmt)
    }
}
func RouteNotificationSend(PUB *nanomsg.PubSocket, route ribd.Routes, evt int) {
	logger.Println("RouteNotificationSend") 
	msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo : route}
	msgbufbytes, err := json.Marshal( msgBuf)
    msg := ribdCommonDefs.RibdNotifyMsg {MsgType:uint16(evt), MsgBuf: msgbufbytes}
	buf, err := json.Marshal( msg)
	if err != nil {
	   logger.Println("Error in marshalling Json")
	   return
	}
	var evtStr string
	if evt == ribdCommonDefs.NOTIFY_ROUTE_CREATED {
		evtStr = "NOTIFY_ROUTE_CREATED"
	} else if evt == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
		evtStr = "NOTIFY_ROUTE_DELETED"
	}
	eventInfo := "Redistribute "+evtStr+" for route "+route.Ipaddr+" "+route.Mask+" type" + ReverseRouteProtoTypeMapDB[int(route.Prototype)]
   	logger.Println("Sending ", evtStr, " for route ", route.Ipaddr, " ", route.Mask, " ", buf)
	t1 := time.Now()
    routeEventInfo := RouteEventInfo{timeStamp:t1.String(),eventInfo:eventInfo}
	localRouteEventsDB = append(localRouteEventsDB,routeEventInfo)
   	PUB.Send(buf, nanomsg.DontWait)
}

func delLinuxRoute(route RouteInfoRecord) {
	logger.Println("delLinuxRoute")
	if route.protocol == ribdCommonDefs.CONNECTED {
		logger.Println("This is a connected route, do nothing")
		return
	}
	mask:=net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3])
	maskedIP:=route.destNetIp.Mask(mask)
	logger.Println("mask = ", mask, " destip:= ", route.destNetIp, " maskedIP ",maskedIP)
		dst := &net.IPNet{
		IP:   maskedIP,//route.destNetIp,
		Mask: mask,//net.CIDRMask(prefixLen, 32),//net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3]),
	    }
		ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(route.nextHopIfIndex), int(route.nextHopIfType))
        logger.Println("IfId = ", ifId)
		intfEntry,ok:=IntfIdNameMap[ifId]
		if !ok {
			logger.Println("IfName not updated for ifId ", ifId)
			return
		}
		ifName := intfEntry.name
		logger.Println("ifName = ", ifName, " for ifId ", ifId)
	    link, err := netlink.LinkByName(ifName)
	    if err != nil {
			logger.Println("LinkByIndex call failed with error ", err, "for linkName ", ifName)
			return
  	    }

	    lxroute := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst}
        err = netlink.RouteDel(&lxroute)
		if err != nil {
			logger.Println("Route delete call failed with error ", err)
		}
    return
}

func addLinuxRoute(route RouteInfoRecord) {
	logger.Println("addLinuxRoute")
	if route.protocol == ribdCommonDefs.CONNECTED {
		logger.Println("This is a connected route, do nothing")
		return
	}
	mask:=net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3])
	maskedIP:=route.destNetIp.Mask(mask)
	logger.Println("mask = ", mask, " destip:= ", route.destNetIp, " maskedIP ",maskedIP)
		dst := &net.IPNet{
		IP:   maskedIP,//route.destNetIp,
		Mask: mask,//net.CIDRMask(prefixLen, 32),//net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3]),
	    }
		ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(route.nextHopIfIndex), int(route.nextHopIfType))
        logger.Println("IfId = ", ifId)
		intfEntry,ok:=IntfIdNameMap[ifId]
		if !ok {
			logger.Println("IfName not updated for ifId ", ifId)
			return
		}
		ifName := intfEntry.name
		logger.Println("ifName = ", ifName, " for ifId ", ifId)
	    link, err := netlink.LinkByName(ifName)
	    if err != nil {
			logger.Println("LinkByIndex call failed with error ", err, "for linkName ", ifName)
			return
  	    }

        logger.Println("adding linux route for dst.ip= ", dst.IP.String(), " mask: ", dst.Mask.String(), "Gw: ", route.nextHopIp)
	    lxroute := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Gw:route.nextHopIp}
        err = netlink.RouteAdd(&lxroute)
		if err != nil {
			logger.Println("Route add call failed with error ", err)
		}
    return
}
func getIPInt(ip net.IP) (ipInt int, err error) {
	if ip == nil {
		logger.Printf("ip address %v invalid\n", ip)
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

func getNetworkPrefix(destNetIp net.IP, networkMask net.IP) (destNet patriciaDB.Prefix, err error) {
	prefixLen, err := getPrefixLen(networkMask)
	if err != nil {
		logger.Println("err when getting prefixLen, err= ", err)
		return destNet, err
	}
	/*   ip, err := getIP(destNetIp)
	    if err != nil {
	        logger.Println("Invalid destination network IP Address")
			return destNet, err
	    }
	    vdestMaskIp,err := getIP(networkMask)
	    if err != nil {
	        logger.Println("Invalid network mask")
			return destNet, err
	    }*/
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