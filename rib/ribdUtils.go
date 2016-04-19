// ribdUtils.go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"l3/rib/ribdCommonDefs"
	"net"
	"ribd"
	"ribdInt"
	"sort"
	"strconv"
	"strings"
	"bytes"
	"utils/netUtils"
	"utils/patriciaDB"
	"utils/policy"
	"github.com/op/go-nanomsg"
)

type RouteDistanceConfig struct {
	defaultDistance    int
	configuredDistance int
}
type AdminDistanceSlice []ribd.RouteDistanceState
type RedistributeRouteInfo struct {
	route ribdInt.Routes
}

type PublisherMapInfo struct {
	pub_ipc    string
	pub_socket *nanomsg.PubSocket
}
var RedistributeRouteMap map[string][]RedistributeRouteInfo
var TrackReachabilityMap map[string][]string //map[ipAddr][]protocols
var RouteProtocolTypeMapDB map[string]int
var ReverseRouteProtoTypeMapDB map[int]string
var ProtocolAdminDistanceMapDB map[string]RouteDistanceConfig
var ProtocolAdminDistanceSlice AdminDistanceSlice
var PublisherInfoMap map[string]PublisherMapInfo

func BuildPublisherMap() {
	RIBD_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_ADDR)
	for k,_ := range RouteProtocolTypeMapDB {
		logger.Info(fmt.Sprintln("Building publisher map for protocol ", k))
		if k == "CONNECTED" || k == "STATIC" {
			logger.Info(fmt.Sprintln("Publisher info for protocol ", k, " not required"))
			continue
		}
		if k == "IBGP" || k == "EBGP" {
           continue		
		}
		pub_ipc := "ipc:///tmp/ribd_"+strings.ToLower(k)+"d.ipc"
		logger.Info(fmt.Sprintln("pub_ipc:", pub_ipc))
		pub := InitPublisher(pub_ipc)
		PublisherInfoMap[k] = PublisherMapInfo{pub_ipc, pub}
	}
	PublisherInfoMap["EBGP"] = PublisherInfoMap["BGP"]
	PublisherInfoMap["IBGP"] = PublisherInfoMap["BGP"]
	PublisherInfoMap["BFD"] = PublisherMapInfo{ribdCommonDefs.PUB_SOCKET_BFDD_ADDR,InitPublisher(ribdCommonDefs.PUB_SOCKET_BFDD_ADDR)}
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

func arpResolveCalled(key NextHopInfoKey) (bool) {
	if routeServiceHandler.NextHopInfoMap == nil {
		return false
	}
	info,ok := routeServiceHandler.NextHopInfoMap[key]
	if !ok || info.refCount == 0 {
		logger.Info(fmt.Sprintln("Arp resolve not called for ", key.nextHopIp))
		return false
	}
	return true
}
func updateNextHopMap(key NextHopInfoKey, op int) (count int){
	opStr := ""
	if op == add {
		opStr = "incrementing"
	} else if op == del {
		opStr = "decrementing"
	}
	logger.Info(fmt.Sprintln(opStr, " nextHop Map for ", key.nextHopIp))
	if routeServiceHandler.NextHopInfoMap == nil {
		return -1
	}
	info,ok := routeServiceHandler.NextHopInfoMap[key]
	if !ok {
		routeServiceHandler.NextHopInfoMap[key] = NextHopInfo{1}
		count = 1
	} else {
	    if op == add {
		    info.refCount++
	    } else if op == del {
		    info.refCount--
	    }
	    routeServiceHandler.NextHopInfoMap[key] = info
		count = info.refCount
	}
	logger.Info(fmt.Sprintln("Updated refcount = ", count))
	return count
}
func findElement(list []string, element string) (int) {
	index := -1
	for i :=0 ;i<len(list);i++ {
		if list[i]==element {
			logger.Info(fmt.Sprintln("Found element ", element, " at index ",i))
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
		return entity,err
	}
	entity.DestNetIp = destNetIp
	entity.NextHopIp = route.NextHopIp
	entity.RouteProtocol = ReverseRouteProtoTypeMapDB[int(route.Prototype)]
	if routeInfo.createType != Invalid {
		entity.CreatePath = true
	}
	if routeInfo.deleteType != Invalid {
		entity.DeletePath = true
	}
	return entity,err
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
func getPolicyRouteMapIndex(entity policy.PolicyEngineFilterEntityParams, policy string) (policyRouteIndex policy.PolicyEntityMapIndex) {
	logger.Println("getPolicyRouteMapIndex")
	policyRouteIndex = PolicyRouteIndex{destNetIP: entity.DestNetIp, policy: policy}
	return policyRouteIndex
}
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
	logger.Info(fmt.Sprintln("Adding policy ", policy, " to route %v", destNet))
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
func RedistributionNotificationSend(PUB *nanomsg.PubSocket, route ribdInt.Routes, evt int) {
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
	eventInfo = eventInfo + evtStr + " for route " + route.Ipaddr + " " + route.Mask + " type" + ReverseRouteProtoTypeMapDB[int(route.Prototype)]
	logger.Info(fmt.Sprintln("Adding ", evtStr, " for route ", route.Ipaddr, " ", route.Mask, " to notification channel"))
	routeServiceHandler.NotificationChannel <- NotificationMsg{PUB,buf,eventInfo}
}
func RouteReachabilityStatusNotificationSend(targetProtocol string, info RouteReachabilityStatusInfo) {
	logger.Info(fmt.Sprintln("RouteReachabilityStatusNotificationSend for protocol ", targetProtocol))
	publisherInfo,ok := PublisherInfoMap[targetProtocol]
	if !ok {
		logger.Info(fmt.Sprintln("Publisher not found for protocol ", targetProtocol))
		return
	}
	evt := ribdCommonDefs.NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE
	PUB := publisherInfo.pub_socket
	msgInfo := ribdCommonDefs.RouteReachabilityStatusMsgInfo{}
	msgInfo.Network = info.destNet
	if info.status == "Up" || info.status == "Updated"{
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
		eventInfo = eventInfo + " NextHop IP: " + info.nextHopIntf.NextHopIp + " IfType/Index: " + strconv.Itoa(int(info.nextHopIntf.NextHopIfType)) + "/" + strconv.Itoa(int(info.nextHopIntf.NextHopIfIndex ))
	}
	logger.Info(fmt.Sprintln("Adding  NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE with status ",info.status, " for network ", info.destNet, " to notification channel"))
	routeServiceHandler.NotificationChannel <- NotificationMsg{PUB,buf,eventInfo}
}
func RouteReachabilityStatusUpdate(targetProtocol string, info RouteReachabilityStatusInfo) {
	logger.Info(fmt.Sprintln("RouteReachabilityStatusUpdate targetProtocol ", targetProtocol))
    if targetProtocol != "NONE" {
	    RouteReachabilityStatusNotificationSend(targetProtocol,info)
	}
	var ipMask net.IP
	ip,ipNet,err := net.ParseCIDR(info.destNet)
	if err != nil {
		logger.Err(fmt.Sprintln("Error getting IP from cidr: ", info.destNet))
		return 
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	destIpPrefix,err := getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
	if err != nil {
		logger.Err(fmt.Sprintln("Error getting ip prefix for ip:", ipAddrStr, " mask:", ipMaskStr))
		return 
	}
	//check the TrackReachabilityMap to see if any other protocols are interested in receiving updates for this network 
	for k,list := range TrackReachabilityMap {
		prefix,err := getNetowrkPrefixFromStrings(k,ipMaskStr)
	    if err != nil {
		    logger.Err(fmt.Sprintln("Error getting ip prefix for ip:", k, " mask:", ipMaskStr))
		    return 
	    }
		if bytes.Equal(destIpPrefix,prefix) {
	        for idx := 0;idx <len(list);idx++{
		        logger.Info(fmt.Sprintln(" protocol ", list[idx], " interested in receving reachability updates for ipAddr ", info.destNet))
				info.destNet = k
		        RouteReachabilityStatusNotificationSend(list[idx],info)
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

func getNetworkPrefix(destNetIp net.IP, networkMask net.IP) (destNet patriciaDB.Prefix, err error) {
	prefixLen, err := getPrefixLen(networkMask)
	if err != nil {
		logger.Info(fmt.Sprintln("err when getting prefixLen, err= ", err))
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
