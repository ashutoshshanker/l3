// ribdUtils.go
package main

import (
	"ribd"
	"encoding/json"
	"github.com/op/go-nanomsg"
	"net"
	"errors"
	"utils/patriciaDB"
	"l3/rib/ribdCommonDefs"
)

var RouteProtocolTypeMapDB = make(map[string]int)
var ReverseRouteProtoTypeMapDB = make(map[int]string)

func BuildRouteProtocolTypeMapDB() {
	RouteProtocolTypeMapDB["Connected"] = ribdCommonDefs.CONNECTED
	RouteProtocolTypeMapDB["BGP"]       = ribdCommonDefs.BGP
	RouteProtocolTypeMapDB["Static"]       = ribdCommonDefs.STATIC
	
	//reverse
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.CONNECTED] = "Connected"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.BGP] = "BGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.STATIC] = "Static"
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
		return prefix, err
	}
	return prefix, err
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
	routeInfoRecordList.policyList = append(routeInfoRecordList.policyList[:0])
	RouteInfoMap.Set(destNet,routeInfoRecordList)
	return
}
func addRoutePolicyState(route ribd.Routes, policy string) {
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
		routeInfoRecordList.policyList = make([]string, 0)
	}
	routeInfoRecordList.policyList = append(routeInfoRecordList.policyList, policy)
	RouteInfoMap.Set(destNet,routeInfoRecordList)
	return
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
func updateRoutePolicyState(route ribd.Routes, op int, policy string) {
	logger.Println("updateRoutePolicyState")
	if op == delAll {
		deleteRoutePolicyStateAll(route)
	} else if op == add {
		addRoutePolicyState(route, policy)
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
   	logger.Println("buf", buf)
   	PUB.Send(buf, nanomsg.DontWait)
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
	return destNet, nil
}
