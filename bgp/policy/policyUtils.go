// policyUtils.go
package policy

import (
	"bgpd"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"utils/patriciaDB"
)

const (
	add = iota
	del
	delAll
	invalidate
)
const (
	Invalid = -1
	Valid   = 0
)

type ApplyActionFunc func(*bgpd.BGPRoute, []string, interface{}, interface{}, interface{})

type RouteParams struct {
	DestNetIp     string
	PrefixLen     uint16
	NextHopIp     string
	CreateType    int
	DeleteType    int
	ActionFuncMap map[int]ApplyActionFunc
}

type PolicyRouteIndex struct {
	routeIP   string // patriciaDB.Prefix
	prefixLen uint16
	policy    string
}

type localDB struct {
	prefix     patriciaDB.Prefix
	isValid    bool
	precedence int
	nextHopIp  string
}
type ConditionsAndActionsList struct {
	conditionList []string
	actionList    []string
}
type PolicyStmtMap struct {
	policyStmtMap map[string]ConditionsAndActionsList
}

var PolicyRouteMap map[PolicyRouteIndex]PolicyStmtMap

func getIPInt(ip net.IP) (ipInt int, err error) {
	if ip == nil {
		fmt.Printf("ip address %v invalid\n", ip)
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
		fmt.Println("err when getting prefixLen, err= ", err)
		return destNet, err
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
	destNetIpAddr, err := getIP(ipAddr)
	if err != nil {
		fmt.Println("destNetIpAddr invalid")
		return prefix, err
	}
	networkMaskAddr, err := getIP(mask)
	if err != nil {
		fmt.Println("networkMaskAddr invalid")
		return prefix, err
	}
	prefix, err = getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		fmt.Println("err=", err)
		return prefix, err
	}
	return prefix, err
}
func getNetworkPrefixFromCIDR(ipAddr string) (ipPrefix patriciaDB.Prefix, err error) {
	//var ipMask net.IP
	_, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return ipPrefix, err
	}
	/*
		ipMask = make(net.IP, 4)
		copy(ipMask, ipNet.Mask)
		ipAddrStr := ip.String()
		ipMaskStr := net.IP(ipMask).String()
		ipPrefix, err = getNetowrkPrefixFromStrings(ipAddrStr, ipMaskStr)
	*/
	i := strings.IndexByte(ipAddr, '/')
	prefixLen, _ := strconv.Atoi(ipAddr[i+1:])
	numbytes := (prefixLen + 7) / 8
	destNet := make([]byte, numbytes)
	for i := 0; i < numbytes; i++ {
		destNet[i] = ipNet.IP[i]
	}

	return patriciaDB.Prefix(destNet), err
}
func deleteRoutePolicyStateAll(route *bgpd.BGPRoute) {
	fmt.Println("deleteRoutePolicyStateAll")
	route.PolicyList = nil
	return
}
func addRoutePolicyState(route *bgpd.BGPRoute, policy string, policyStmt string) {
	fmt.Println("addRoutePolicyState")
	if route.PolicyList == nil {
		route.PolicyList = make([]string, 0)
	}
	route.PolicyList = append(route.PolicyList, policy)
	return
}
func addPolicyRouteMapEntry(route *bgpd.BGPRoute, policy string, policyStmt string, conditionList []string, actionList []string) {
	fmt.Println("addPolicyRouteMapEntry")
	var policyStmtMap PolicyStmtMap
	var conditionsAndActionsList ConditionsAndActionsList
	if PolicyRouteMap == nil {
		PolicyRouteMap = make(map[PolicyRouteIndex]PolicyStmtMap)
	}
	policyRouteIndex := PolicyRouteIndex{routeIP: route.Network, prefixLen: uint16(route.CIDRLen), policy: policy}
	policyStmtMap, ok := PolicyRouteMap[policyRouteIndex]
	if !ok {
		policyStmtMap.policyStmtMap = make(map[string]ConditionsAndActionsList)
	}
	_, ok = policyStmtMap.policyStmtMap[policyStmt]
	if ok {
		fmt.Println("policy statement map for statement ", policyStmt, " already in place for policy ", policy)
		return
	}
	conditionsAndActionsList.conditionList = make([]string, 0)
	conditionsAndActionsList.actionList = make([]string, 0)
	for i := 0; conditionList != nil && i < len(conditionList); i++ {
		conditionsAndActionsList.conditionList = append(conditionsAndActionsList.conditionList, conditionList[i])
	}
	for i := 0; actionList != nil && i < len(actionList); i++ {
		conditionsAndActionsList.actionList = append(conditionsAndActionsList.actionList, actionList[i])
	}
	policyStmtMap.policyStmtMap[policyStmt] = conditionsAndActionsList
	PolicyRouteMap[policyRouteIndex] = policyStmtMap
}
func deleteRoutePolicyState(ipPrefix patriciaDB.Prefix, policyName string) {
	fmt.Println("deleteRoutePolicyState")
}
func deletePolicyRouteMapEntry(route *bgpd.BGPRoute, policy string) {
	fmt.Println("deletePolicyRouteMapEntry for policy ", policy, "route ", route.Network, "/", route.CIDRLen)
	if PolicyRouteMap == nil {
		fmt.Println("PolicyRouteMap empty")
		return
	}
	policyRouteIndex := PolicyRouteIndex{routeIP: route.Network, prefixLen: uint16(route.CIDRLen), policy: policy}
	//PolicyRouteMap[policyRouteIndex].policyStmtMap=nil
	delete(PolicyRouteMap, policyRouteIndex)
}

func updateRoutePolicyState(route *bgpd.BGPRoute, op int, policy string, policyStmt string) {
	fmt.Println("updateRoutePolicyState")
	if op == delAll {
		deleteRoutePolicyStateAll(route)
		deletePolicyRouteMapEntry(route, policy)
	} else if op == add {
		addRoutePolicyState(route, policy, policyStmt)
	}
}
