package main

import (
	"arpd"
	"asicdServices"
	"portdServices"
	"encoding/json"
	"l3/rib/ribdCommonDefs"
	"ribd"
	"utils/patriciaDB"
	//		"patricia"
	"errors"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/op/go-nanomsg"
	"asicd/asicdConstDefs"
	"infra/portd/portdCommonDefs"
	"io/ioutil"
	"net"
	"strconv"
	"time"
	"encoding/binary"
	"bytes"
	"utils/ipcutils"
)

type RouteServiceHandler struct {
}

type RIBClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type PortdClient struct {
	RIBClientBase
	ClientHdl *portdServices.PortServiceClient
}

type AsicdClient struct {
	RIBClientBase
	ClientHdl *asicdServices.AsicdServiceClient
}

type ArpdClient struct {
	RIBClientBase
	ClientHdl *arpd.ARPServiceClient
}

const (
	PROTOCOL_NONE      = -1
	PROTOCOL_CONNECTED = 0
	PROTOCOL_STATIC    = 1
	PROTOCOL_OSPF      = 2
	PROTOCOL_BGP       = 3
	PROTOCOL_LAST      = 4
)

const (
	add = iota
	del
	invalidate
)
const (
	FIBOnly = iota
	FIBAndRIB
	RIBOnly
)
const (
	SUB_PORTD = 0
	SUB_ASICD = 1
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
}

//implement priority queue of the routes
type RouteInfoRecordList struct {
	selectedRouteIdx int8
	routeInfoList    []RouteInfoRecord //map[int]RouteInfoRecord
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type IPRoute struct {
	DestinationNw     string 
	NetworkMask       string 
	Cost              int
	NextHopIp         string
	OutgoingIntfType  string
	OutgoingInterface string
	Protocol          string
}

type localDB struct{
	prefix           patriciaDB.Prefix
	isValid           bool
}
var RouteInfoMap = patriciaDB.NewTrie()
var DummyRouteInfoRecord RouteInfoRecord //{destNet:0, prefixLen:0, protocol:0, nextHop:0, nextHopIfIndex:0, metric:0, selected:false}
var asicdclnt AsicdClient
var arpdclnt ArpdClient
var portdclnt PortdClient
var count int
var ConnectedRoutes []*ribd.Routes
var destNetSlice []localDB
var acceptConfig bool
var AsicdSub *nanomsg.SubSocket
var PortdSub *nanomsg.SubSocket
var RIBD_PUB  *nanomsg.PubSocket
/*
func setProtocol(routeType ribd.Int) (proto int8, err error) {
	err = nil
	switch routeType {
	case ribdCommonDefs.CONNECTED:
		proto = PROTOCOL_CONNECTED
	case ribdCommonDefs.STATIC:
		proto = PROTOCOL_STATIC
	case ribdCommonDefs.OSPF:
		proto = PROTOCOL_OSPF
	case ribdCommonDefs.BGP:
		proto = PROTOCOL_BGP
	default:
		err = errors.New("Not accepted protocol")
		proto = -1
	}
	return proto, err
}
*/
func getSelectedRoute(routeInfoRecordList RouteInfoRecordList) (routeInfoRecord RouteInfoRecord, err error) {
	if routeInfoRecordList.selectedRouteIdx == PROTOCOL_NONE {
		err = errors.New("No route selected")
	} else {
		routeInfoRecord = routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
	}
	return routeInfoRecord, err
}

func SelectV4Route(destNetPrefix patriciaDB.Prefix,
	routeInfoRecordList RouteInfoRecordList,
	routeInfoRecord RouteInfoRecord,
	op ribd.Int,
	index int) (err error) {
	var routeInfoRecordNew RouteInfoRecord
	var routeInfoRecordOld RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	var i int8
	logger.Printf("Selecting the best Route for destNetPrefix %v, index = %d\n", destNetPrefix, index)
	if op == add {
		selectedRoute, err := getSelectedRoute(routeInfoRecordList)
		logger.Printf("Selected route protocol = %d, routeinforecord.protool=%d\n", selectedRoute.protocol, routeInfoRecord.protocol)
		if err == nil && ((selectedRoute.protocol == PROTOCOL_NONE && routeInfoRecord.protocol != PROTOCOL_NONE) ||routeInfoRecord.protocol <= selectedRoute.protocol) {
			routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx] = selectedRoute
			routeInfoRecordOld = selectedRoute
			destNetSlice[routeInfoRecordOld.sliceIdx].isValid = false
			//destNetSlice is a slice of localDB maintained for a getBulk operations. An entry is created in this db when we create a new route
			if(destNetSlice != nil && (len(destNetSlice) > int(routeInfoRecord.sliceIdx) ) ) { //&& bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNet)) {
				if(bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNetPrefix) == false) {
					logger.Println("Unexpected destination network prefix found at the slice Idx ", routeInfoRecord.sliceIdx)
					return err
				}
				//There is already an entry in the destNetSlice at the route index and was invalidated earlier because  of a link down of the nexthop intf of the route or if the route was deleted
				logger.Println("sliceIdx ", routeInfoRecord.sliceIdx)
				destNetSlice[routeInfoRecord.sliceIdx].isValid = true
			} else {		//this is a new route being added
			   routeInfoRecord.sliceIdx = len(destNetSlice)
               localDBRecord := localDB{prefix:destNetPrefix, isValid:true}
			   if(destNetSlice == nil) {
				 destNetSlice = make([]localDB, 0)
			   } 
			   destNetSlice = append(destNetSlice, localDBRecord)
			}
			routeInfoRecordList.routeInfoList[index] = routeInfoRecord
			routeInfoRecordNew = routeInfoRecord
			routeInfoRecordList.selectedRouteIdx = int8(index)
			logger.Printf("new selected route idx = %d\n", routeInfoRecordList.selectedRouteIdx)
		}
	} else if op == del {
		logger.Println(" in del index selectedrouteIndex", index, routeInfoRecordList.selectedRouteIdx)
		if len(routeInfoRecordList.routeInfoList) == 0 {
			logger.Println(" in del,numRoutes now 0, so delete the node")
			RouteInfoMap.Delete(destNetPrefix)
			return nil
		}
		routeInfoRecordOld = routeInfoRecord
		routeInfoRecord.protocol = PROTOCOL_NONE
		routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx] = routeInfoRecord
		if(destNetSlice == nil || int(routeInfoRecord.sliceIdx) >= len(destNetSlice)) {
			logger.Println("Destination slice not found at the expected slice index ", routeInfoRecord.sliceIdx)
			return err
		}
        destNetSlice[routeInfoRecord.sliceIdx].isValid = false   //invalidate this entry in the local db
		if int8(index) == routeInfoRecordList.selectedRouteIdx {
			for i = 0; i < int8(len(routeInfoRecordList.routeInfoList)); i++ {
				routeInfoRecordTemp = routeInfoRecordList.routeInfoList[i]
				if i == int8(index) { //if(ok != true || i==routeInfoRecord.protocol) {
					continue
				}
				logger.Printf("temp protocol=%d", routeInfoRecordTemp.protocol)
				if routeInfoRecordTemp.protocol != PROTOCOL_NONE {
					logger.Printf(" selceting protocol %d", routeInfoRecordTemp.protocol)
					routeInfoRecordList.routeInfoList[i] = routeInfoRecordTemp
					routeInfoRecordNew = routeInfoRecordTemp
					routeInfoRecordList.selectedRouteIdx = i
					destNetSlice[routeInfoRecordNew.sliceIdx].isValid = true
					break
				}
			}
		}
	}
	//update the patriciaDB trie with the updated route info record list
	RouteInfoMap.Set(patriciaDB.Prefix(destNetPrefix), routeInfoRecordList)

	if routeInfoRecordOld.protocol != PROTOCOL_NONE {
		//call asicd to del
		if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
		}
	}
	if routeInfoRecordNew.protocol != PROTOCOL_NONE {
		//call asicd to add
		if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String(), routeInfoRecord.nextHopIp.String())
		}
		if arpdclnt.IsConnected && routeInfoRecord.protocol != ribdCommonDefs.CONNECTED {
			//call arpd to resolve the ip
			logger.Println("### Sending ARP Resolve for ", routeInfoRecord.nextHopIp.String(), routeInfoRecord.nextHopIfType)
			arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.nextHopIp.String(), arpd.Int(routeInfoRecord.nextHopIfType), arpd.Int(routeInfoRecord.nextHopIfIndex))
			//arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.destNetIp.String(), arpd.Int(routeInfoRecord.nextHopIfIndex))
		}
	}
	return nil
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
func updateConnectedRoutes(destNetIPAddr string, networkMaskAddr string, nextHopIP string, nextHopIfIndex ribd.Int, nextHopIfType ribd.Int, op int, sliceIdx ribd.Int) {
	var temproute ribd.Routes
	route := &temproute
	logger.Printf("number of connectd routes = %d\n", len(ConnectedRoutes))
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
	routePrototype int8) (found bool, i int) {
	for i := 0; i < len(routeInfoRecordList.routeInfoList); i++ {
		logger.Printf("len = %d i=%d routePrototype=%d\n", len(routeInfoRecordList.routeInfoList), i, routeInfoRecordList.routeInfoList[i].protocol)
		if routeInfoRecordList.routeInfoList[i].protocol == routePrototype {
			found = true
			return true, i
		}
	}
	logger.Printf("returning i = %d\n", i)
	return found, i
}

func getConnectedRoutes() {
   var IPIntfList []*portdServices.IPIntf
   logger.Println("Getting ip intfs from portd")	
   IPIntfList, err := portdclnt.ClientHdl.GetIPIntfsAll()
   if(err != nil) {
      logger.Println("Failed to get ip intfs with err ", err)
	  return	
   }
   logger.Printf("Number of ip intfs  = %d\n", len(IPIntfList))
   for i:=0;i<len(IPIntfList);i++ {
      _, err := createV4Route(IPIntfList[i].IpAddr, IPIntfList[i].Mask, 0, "0.0.0.0", ribd.Int(IPIntfList[i].IntfType), ribd.Int(IPIntfList[i].IntfId), ribdCommonDefs.CONNECTED, FIBAndRIB,ribd.Int(len(destNetSlice)))	
	if(err != nil) {
		logger.Printf("Failed to create connected route for ip Addr %s/%s intfType %d intfId %d\n", IPIntfList[i].IpAddr, IPIntfList[i].Mask, ribd.Int(IPIntfList[i].IntfType), ribd.Int(IPIntfList[i].IntfId))
	}
   }
}

//thrift API definitions

func (m RouteServiceHandler) GetBulkRoutes( fromIndex ribd.Int, rcount ribd.Int) (routes *ribd.RoutesGetInfo, err error){//(routes []*ribd.Routes, err error) {
	logger.Println("GetBulkRoutes")
    var i, validCount, toIndex ribd.Int
	var temproute []ribd.Routes = make ([]ribd.Routes, rcount)
	var nextRoute *ribd.Routes
    var returnRoutes []*ribd.Routes
	var returnRouteGetInfo ribd.RoutesGetInfo
	var prefixNodeRouteList RouteInfoRecordList
	var prefixNodeRoute RouteInfoRecord
	i = 0
	routes = &returnRouteGetInfo
	moreRoutes := true
    if(destNetSlice == nil) {
		logger.Println("destNetSlice not initialized")
		return routes, err
	}
	for ;;i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
		if(i+fromIndex >= ribd.Int(len(destNetSlice))) {
			logger.Println("All the routes fetched")
			moreRoutes = false
			break
		}
		if(destNetSlice[i+fromIndex].isValid == false) {
			logger.Println("Invalid route")
			continue
		}
		if(validCount==rcount) {
			logger.Println("Enough routes fetched")
			break
		}
		logger.Printf("Fetching trie record for index %d and prefix %v\n", i+fromIndex, (destNetSlice[i+fromIndex].prefix))
		prefixNode := RouteInfoMap.Get(destNetSlice[i+fromIndex].prefix)
		if(prefixNode != nil) {
			prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
			prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
			nextRoute = &temproute[validCount]
		    nextRoute.Ipaddr = prefixNodeRoute.destNetIp.String()
		    nextRoute.Mask = prefixNodeRoute.networkMask.String()
		    nextRoute.NextHopIp = prefixNodeRoute.nextHopIp.String()
			nextRoute.NextHopIfType = ribd.Int(prefixNodeRoute.nextHopIfType)
			nextRoute.IfIndex = prefixNodeRoute.nextHopIfIndex
			nextRoute.Metric = prefixNodeRoute.metric
			nextRoute.Prototype = ribd.Int(prefixNodeRoute.protocol)
			nextRoute.IsValid = destNetSlice[i+fromIndex].isValid
			toIndex = ribd.Int(prefixNodeRoute.sliceIdx)
			if(len(returnRoutes) == 0){
				returnRoutes = make([]*ribd.Routes, 0)
			}
			returnRoutes = append(returnRoutes, nextRoute)
			validCount++
		}
	}
	logger.Printf("Returning %d list of routes\n", validCount)
	routes.RouteList = returnRoutes
	routes.StartIdx = fromIndex
	routes.EndIdx = toIndex+1
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
   for i:=0;i<len(ConnectedRoutes);i++ {
      if(ConnectedRoutes[i].IsValid == true) {		
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
		if rmapInfoList.selectedRouteIdx != PROTOCOL_NONE {
			found = true
			v := rmapInfoList.routeInfoList[rmapInfoList.selectedRouteIdx]
			nextHopIntf.NextHopIfType = ribd.Int(v.nextHopIfType)
			nextHopIntf.NextHopIfIndex = v.nextHopIfIndex
			nextHopIntf.NextHopIp = v.nextHopIp.String()
			nextHopIntf.Metric = v.metric
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
func (m RouteServiceHandler) GetRoute ( destNetIp string, networkMask string) (route *ribd.Routes, err error){
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
	if(routeInfoRecordListItem == nil) {
		logger.Println("No such route")
		err = errors.New("Route does not exist")
		return route, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
    if(routeInfoRecordList.selectedRouteIdx == PROTOCOL_NONE) {
		logger.Println("No selected route for this network")
		err = errors.New("No selected route for this network")
		return route, err
	}
	routeInfoRecord := routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
	route.Ipaddr = destNetIp
	route.Mask = networkMask
	route.NextHopIp = routeInfoRecord.nextHopIp.String()
	route.NextHopIfType = ribd.Int(routeInfoRecord.nextHopIfType)
	route.IfIndex = routeInfoRecord.nextHopIfIndex
	route.Metric =  routeInfoRecord.metric
	route.Prototype = ribd.Int(routeInfoRecord.protocol)
	return route, err
} 
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
	sliceIdx ribd.Int) (rc ribd.Int, err error) {
	logger.Printf("createV4Route for ip %s mask %s next hop ip %s addType %d\n", destNetIp, networkMask, nextHopIp,addType)

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
	logger.Printf("routePrototype %d for routeType %d", routePrototype, routeType)
	routeInfoRecord := RouteInfoRecord{destNetIp: destNetIpAddr, networkMask: networkMaskAddr, protocol: routePrototype, nextHopIp: nextHopIpAddr, nextHopIfType: int8(nextHopIfType), nextHopIfIndex: nextHopIfIndex, metric: metric, sliceIdx:int(sliceIdx)}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		if(addType == FIBOnly) {
			logger.Println("route record list not found in RIB")
			err  = errors.New("Unexpected: route record list not found in RIB")
			return 0, err
		}
		var newRouteInfoRecordList RouteInfoRecordList
		newRouteInfoRecordList.routeInfoList = make([]RouteInfoRecord, 0)
		newRouteInfoRecordList.routeInfoList = append(newRouteInfoRecordList.routeInfoList, routeInfoRecord)
		newRouteInfoRecordList.selectedRouteIdx = 0
		if ok := RouteInfoMap.Insert(destNet, newRouteInfoRecordList); ok != true {
			logger.Println(" return value not ok")
		}
		localDBRecord := localDB{prefix: destNet, isValid:true}
		if(destNetSlice == nil) {
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
	} else {
		routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
		found, _ := IsRoutePresent(routeInfoRecordList, routePrototype)
		if !found {
			if(addType != FIBOnly) {
			   routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList, routeInfoRecord)
			}
			err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, add, len(routeInfoRecordList.routeInfoList)-1)
		}
	}
	if addType != FIBOnly && routePrototype == ribdCommonDefs.CONNECTED { //PROTOCOL_CONNECTED {
		updateConnectedRoutes(destNetIp, networkMask, nextHopIp, nextHopIfIndex, nextHopIfType,add, sliceIdx)
	}
	return 0, err

}
func (m RouteServiceHandler) CreateV4Route(destNetIp string,
	networkMask string,
	metric ribd.Int,
	nextHopIp string,
	nextHopIfType ribd.Int,
	nextHopIfIndex ribd.Int,
	routeType ribd.Int) (rc ribd.Int, err error) {
	logger.Printf("Received create route request for ip %s mask %s\n", destNetIp, networkMask)
	if(!acceptConfig) {
		logger.Println("Not ready to accept config")
		//return 0, err
	}
    _,err = createV4Route(destNetIp, networkMask, metric, nextHopIp, nextHopIfType, nextHopIfIndex, routeType, FIBAndRIB, ribd.Int(len(destNetSlice)))
	
	if(err != nil) {
		logger.Println("creating v4 route failed with err ", err)
		return 0, err
	}
	
	//If this is not a connected route, then nothing more to do
	if(routeType != ribdCommonDefs.CONNECTED) {
	    logger.Println("This is not a connected route, nothing more to do")
		return 0, err
	}
	logger.Println("This is a connected route, so send a route add event")

	//Send a event
	route := ribd.Routes { Ipaddr : destNetIp, Mask : networkMask,	NextHopIp : nextHopIp, NextHopIfType: nextHopIfType, IfIndex : nextHopIfIndex, Metric : metric}

	msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo : route}
	msgbufbytes, err := json.Marshal( msgBuf)
    msg := ribdCommonDefs.RibdNotifyMsg {MsgType:ribdCommonDefs.NOTIFY_ROUTE_CREATED, MsgBuf: msgbufbytes}
	buf, err := json.Marshal( msg)
	if err != nil {
		logger.Println("Error in marshalling Json")
		return
	}
	logger.Println("buf", buf)
   	RIBD_PUB.Send(buf, nanomsg.DontWait)
	return 0, err
}
/**
   This function is called when:
   -  a user/protocol deletes a route - delType = FIBAndRIB
   - when a link goes down and we have connected routes on that link - delType = FIBOnly
**/
func deleteV4Route(destNetIp string,
	networkMask string,
	routeType ribd.Int,
	delType ribd.Int) (rc ribd.Int, err error) {
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
	routePrototype := int8(routeType)
/*	routePrototype, err := setProtocol(routeType)
	if err != nil {
		return 0, err
	}*/
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		return 0, nil
	}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		return 0, err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found, i := IsRoutePresent(routeInfoRecordList, routePrototype)
	if !found {
		logger.Println("Route not found")
		return 0, err
	}
	routeInfoRecord := routeInfoRecordList.routeInfoList[i]
	if(delType != FIBOnly) { //if this is not FIBOnly, then we have to delete this route from the RIB data base as well.
	   routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList[:i], routeInfoRecordList.routeInfoList[i+1:]...)
	}
	err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, del, int(i)) //this function will invalidate the route in destNetSlice and also delete the entry in FIB (Asic)

	if routePrototype == ribdCommonDefs.CONNECTED { //PROTOCOL_CONNECTED {
		if delType == FIBOnly { //link gone down, just invalidate the connected route
		   updateConnectedRoutes(destNetIp, networkMask, "",0, 0,invalidate,0)
		} else {
		   updateConnectedRoutes(destNetIp, networkMask, "",0, 0,del,0)
		}
	}
	return 0, err
}

func (m RouteServiceHandler) DeleteV4Route(destNetIp string,
	networkMask string,
	routeType ribd.Int) (rc ribd.Int, err error) {
	logger.Println("Received Route Delete request")
	if(!acceptConfig) {
		logger.Println("Not ready to accept config")
		//return 0,err
	}
	_,err = deleteV4Route(destNetIp, networkMask, routeType, FIBAndRIB)
	return 0, err
}
func (m RouteServiceHandler) UpdateV4Route(destNetIp string,
	networkMask string,
	routeType ribd.Int,
	nextHopIp string,
	//	nextHopIfType ribd.Int,
	nextHopIfIndex ribd.Int,
	metric ribd.Int) (err error) {
	logger.Println("Received update route request")
	if(!acceptConfig) {
		logger.Println("Not ready to accept config")
		//return err
	}
	destNetIpAddr, err := getIP(destNetIp)
	if err != nil {
		return err
	}
	networkMaskAddr, err := getIP(networkMask)
	if err != nil {
		return err
	}
	nextHopIpAddr, err := getIP(nextHopIp)
	if err != nil {
		return err
	}
	destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if err != nil {
		return err
	}
	logger.Printf("destNet = %v\n", destNet)
	routePrototype := int8(routeType)
/*	routePrototype, err := setProtocol(routeType)
	if err != nil {
		return err
	}*/
	ok := RouteInfoMap.Match(destNet)
	if !ok {
		err = errors.New("No route found")
		return err
	}
	routeInfoRecord := RouteInfoRecord{protocol: routePrototype, nextHopIp: nextHopIpAddr, nextHopIfIndex: nextHopIfIndex, metric: metric}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		logger.Println("No route for destination network")
		return err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found, i := IsRoutePresent(routeInfoRecordList, routePrototype)
	if !found {
		logger.Println("No entry present for this destination and protocol")
		return err
	}
	routeInfoRecordList.routeInfoList[i] = routeInfoRecord
	RouteInfoMap.Set(destNet, routeInfoRecordList)
	if routeInfoRecordList.selectedRouteIdx == int8(i) {
		//call asicd to update info
	}
	return err
}

func printRoutesInfo(prefix patriciaDB.Prefix, item patriciaDB.Item) (err error) {
	rmapInfoRecordList := item.(RouteInfoRecordList)
	for _, v := range rmapInfoRecordList.routeInfoList {
		if v.protocol == PROTOCOL_NONE {
			continue
		}
		//   logger.Printf("%v-> %d %d %d %d\n", prefix, v.destNetIp, v.networkMask, v.protocol)
		count++
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

func processLinkDownEvent(ifType ribd.Int, ifIndex ribd.Int){
	logger.Println("processLinkDown")
   for i:=0;i<len(ConnectedRoutes);i++ {
      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex)){		
	     logger.Printf("Delete this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask)	

		 //Send a event
	     msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo : *ConnectedRoutes[i]}
	     msgbufbytes, err := json.Marshal( msgBuf)
         msg := ribdCommonDefs.RibdNotifyMsg {MsgType:ribdCommonDefs.NOTIFY_ROUTE_DELETED, MsgBuf: msgbufbytes}
	     buf, err := json.Marshal( msg)
	     if err != nil {
		   logger.Println("Error in marshalling Json")
		   return
	     }
	     logger.Println("buf", buf)
   	     RIBD_PUB.Send(buf, nanomsg.DontWait)
		
         //Delete this route
		 deleteV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, 0, FIBOnly)
	  }	
   }
}

func processLinkUpEvent(ifType ribd.Int, ifIndex ribd.Int){
	logger.Println("processLinkUpEvent")
   for i:=0;i<len(ConnectedRoutes);i++ {
      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex) && ConnectedRoutes[i].IsValid == false){		
	     logger.Printf("Add this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask)	

         ConnectedRoutes[i].IsValid = true
		 //Send a event
	     msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo : *ConnectedRoutes[i]}
	     msgbufbytes, err := json.Marshal( msgBuf)
         msg := ribdCommonDefs.RibdNotifyMsg {MsgType:ribdCommonDefs.NOTIFY_ROUTE_CREATED, MsgBuf: msgbufbytes}
	     buf, err := json.Marshal( msg)
	     if err != nil {
		   logger.Println("Error in marshalling Json")
		   return
	     }
	     logger.Println("buf", buf)
   	     RIBD_PUB.Send(buf, nanomsg.DontWait)
		
         //Add this route
		 createV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, ConnectedRoutes[i].Metric,ConnectedRoutes[i].NextHopIp, ConnectedRoutes[i].NextHopIfType,ConnectedRoutes[i].IfIndex, ConnectedRoutes[i].Prototype,FIBOnly, ConnectedRoutes[i].SliceIdx)
	  }	
   }
}

func (m RouteServiceHandler) LinkDown(ifType ribd.Int, ifIndex ribd.Int) (err error){
	logger.Println("LinkDown")
	processLinkDownEvent(ifType,ifIndex)
	return nil
}

func (m RouteServiceHandler) LinkUp(ifType ribd.Int, ifIndex ribd.Int) (err error){
	logger.Println("LinkUp")
	processLinkUpEvent(ifType,ifIndex)
	return nil
}

func connectToClient(client ClientJson) {
	var timer *time.Timer
	logger.Printf("in go routine ConnectToClient for connecting to %s\n", client.Name)
	for {
		timer = time.NewTimer(time.Second * 10)
		<-timer.C
		if client.Name == "asicd" {
			//logger.Printf("found asicd at port %d", client.Port)
			asicdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				//logger.Println("connecting to asicd")
				asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				if(arpdclnt.IsConnected == true && portdclnt.IsConnected == true) {
					acceptConfig = true
				}
				timer.Stop()
				return
			}
		}
		if client.Name == "portd" {
			//logger.Printf("found portd at port %d", client.Port)
			portdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			portdclnt.Transport, portdclnt.PtrProtocolFactory = CreateIPCHandles(portdclnt.Address)
			if portdclnt.Transport != nil && portdclnt.PtrProtocolFactory != nil {
				//logger.Println("connecting to asicd")
				portdclnt.ClientHdl = portdServices.NewPortServiceClientFactory(portdclnt.Transport, portdclnt.PtrProtocolFactory)
				portdclnt.IsConnected = true
				getConnectedRoutes()
				portdclnt.ClientHdl.GetIPIntfsAll()
				if(arpdclnt.IsConnected == true && asicdclnt.IsConnected == true) {
					acceptConfig = true
				}
				timer.Stop()
				return
			}
		}
		if client.Name == "arpd" {
			//logger.Printf("found arpd at port %d", client.Port)
			arpdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			arpdclnt.Transport, arpdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(arpdclnt.Address)
			if arpdclnt.Transport != nil && arpdclnt.PtrProtocolFactory != nil {
				//logger.Println("connecting to arpd")
				arpdclnt.ClientHdl = arpd.NewARPServiceClientFactory(arpdclnt.Transport, arpdclnt.PtrProtocolFactory)
				arpdclnt.IsConnected = true
				if(asicdclnt.IsConnected == true && portdclnt.IsConnected == true) {
					acceptConfig = true
				}
				timer.Stop()
				return
			}
		}
	}
}
func ConnectToClients(paramsFile string) {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		logger.Println("Error in reading configuration file")
		return
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		logger.Println("Error in Unmarshalling Json")
		return
	}

	for _, client := range clientsList {
		logger.Println("#### Client name is ", client.Name)
		if client.Name == "asicd" {
			logger.Printf("found asicd at port %d", client.Port)
			asicdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				logger.Println("connecting to asicd")
				asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
			} else {
				go connectToClient(client)
			}
		}
		if client.Name == "portd" {
			logger.Printf("found portd at port %d", client.Port)
			portdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			portdclnt.Transport, portdclnt.PtrProtocolFactory = CreateIPCHandles(portdclnt.Address)
			if portdclnt.Transport != nil && portdclnt.PtrProtocolFactory != nil {
				logger.Println("connecting to port")
				portdclnt.ClientHdl = portdServices.NewPortServiceClientFactory(portdclnt.Transport, portdclnt.PtrProtocolFactory)
				portdclnt.IsConnected = true
				getConnectedRoutes()
			} else {
				go connectToClient(client)
			}
		}
		if client.Name == "arpd" {
			logger.Printf("found arpd at port %d", client.Port)
			arpdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			arpdclnt.Transport, arpdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(arpdclnt.Address)
			if arpdclnt.Transport != nil && arpdclnt.PtrProtocolFactory != nil {
				logger.Println("connecting to arpd")
				arpdclnt.ClientHdl = arpd.NewARPServiceClientFactory(arpdclnt.Transport, arpdclnt.PtrProtocolFactory)
				arpdclnt.IsConnected = true
			} else {
				go connectToClient(client)
			}
		}
	}
}

/*
func CreateRoutes(routeFile string){
	var routesList []IPRoute

	bytes, err := ioutil.ReadFile(routeFile)
	if err != nil {
		logger.Println("Error in reading route file")
		return
	}

	err = json.Unmarshal(bytes, &routesList)
	if err != nil {
		logger.Println("Error in Unmarshalling Json")
		return
	}

	for _, v4Route := range routesList {
		outIntf,_ :=strconv.Atoi(v4Route.OutgoingInterface)
		proto,_ :=strconv.Atoi(v4Route.Protocol)
		CreateV4Route(
			v4Route.DestinationNw, //ribd.Int(binary.BigEndian.Uint32(net.ParseIP(v4Route.DestinationNw).To4())),
			v4Route.NetworkMask,//ribd.Int(prefixLen),
			ribd.Int(v4Route.Cost),
			v4Route.NextHopIp,//ribd.Int(binary.BigEndian.Uint32(net.ParseIP(v4Route.NextHopIp).To4())),
			ribd.Int(outIntf),
			ribd.Int(proto))
   }
}
*/

func processAsicdEvents(sub *nanomsg.SubSocket) {
	
	logger.Println("in process Asicd events")
    for {
	  logger.Println("In for loop")
      rcvdMsg,err := sub.Recv(0)
	  if(err != nil) {
	     logger.Println("Error in receiving ", err)
		 return	
	  }
	  logger.Println("After recv rcvdMsg buf", rcvdMsg)
      buf := bytes.NewReader(rcvdMsg)
      var MsgType asicdConstDefs.AsicdNotifyMsg
      err = binary.Read(buf, binary.LittleEndian, &MsgType)
      if err != nil {
	     logger.Println("Error in reading msgtype ", err)
		 return	
      }
      switch MsgType {
        case asicdConstDefs.NOTIFY_LINK_STATE_CHANGE:
           var msg asicdConstDefs.LinkStateInfo
           err = binary.Read(buf, binary.LittleEndian, &msg)
           if err != nil {
    	     logger.Println("Error in reading msg ", err)
		     return	
           }
		    logger.Printf("Msg linkstatus = %d msg port = %d\n", msg.LinkStatus, msg.Port)
		    if(msg.LinkStatus == asicdConstDefs.LINK_STATE_DOWN) {
				processLinkDownEvent(portdCommonDefs.PHY, ribd.Int(msg.Port))		//asicd always sends out link state events for PHY ports
			} else {
				processLinkUpEvent(portdCommonDefs.PHY, ribd.Int(msg.Port))
			}
       }
	}
}

func processPortdEvents(sub *nanomsg.SubSocket) {
	
	logger.Println("in process Port events")
    for {
	  logger.Println("In for loop")
      rcvdMsg,err := sub.Recv(0)
	  if(err != nil) {
	     logger.Println("Error in receiving")
		 return	
	  }
	  logger.Println("After recv rcvdMsg buf", rcvdMsg)
	  MsgType := portdCommonDefs.PortdNotifyMsg {}
	  err = json.Unmarshal(rcvdMsg, &MsgType)
	  if err != nil {
		logger.Println("Error in Unmarshalling rcvdMsg Json")
		return
	  }
	  logger.Printf(" MsgTYpe=%v\n", MsgType)
      switch MsgType.MsgType {
        case portdCommonDefs.NOTIFY_LINK_STATE_CHANGE:
		   logger.Println("received link state change event")
           msg := portdCommonDefs.LinkStateInfo {}
	       err = json.Unmarshal(MsgType.MsgBuf, &msg)
	       if err != nil {
		      logger.Println("Error in Unmarshalling msgType Json")
		      return
	        }
		    logger.Printf("Msg linkstatus = %d msg linktype = %d linkId = %d\n", msg.LinkStatus, msg.LinkType, msg.LinkId)
		    if(msg.LinkStatus == portdCommonDefs.LINK_STATE_DOWN) {
				processLinkDownEvent(ribd.Int(msg.LinkType), ribd.Int(msg.LinkId))
			} else {
				processLinkUpEvent(ribd.Int(msg.LinkType), ribd.Int(msg.LinkId))
			}
      }
	}
}
func processEvents(sub *nanomsg.SubSocket, subType ribd.Int) {
	logger.Println("in process events for sub ", subType)
	if(subType == SUB_ASICD){
		logger.Println("process Asicd events")
		processAsicdEvents(sub)
	} else if(subType == SUB_PORTD){
		logger.Println("process portd events")
		processPortdEvents(sub)
	}
}
func setupEventHandler(sub *nanomsg.SubSocket, address string, subtype ribd.Int) {
	logger.Println("Setting up event handlers for sub type ", subtype)
	sub, err := nanomsg.NewSubSocket()
	 if err != nil {
        logger.Println("Failed to open sub socket")
        return
    }
	logger.Println("opened socket")
	ep, err := sub.Connect(address)
	if err != nil {
        logger.Println("Failed to connect to pub socket - ", ep)
        return
    }
	logger.Println("Connected to ", ep.Address)
	err = sub.Subscribe("")
	if(err != nil) {
		logger.Println("Failed to subscribe to all topics")
		return 
	}
	logger.Println("Subscribed")
	err = sub.SetRecvBuffer(1024 * 1204)
    if err != nil {
        logger.Println("Failed to set recv buffer size")
        return
    }
		//processPortdEvents(sub)
	processEvents(sub, subtype)
}
func InitPublisher()(pub *nanomsg.PubSocket) {
	pub, err := nanomsg.NewPubSocket()
    if err != nil {
        logger.Println("Failed to open pub socket")
        return nil
    }
    ep, err := pub.Bind(ribdCommonDefs.PUB_SOCKET_ADDR)
    if err != nil {
        logger.Println("Failed to bind pub socket - ", ep)
        return nil
    }
    err = pub.SetSendBuffer(1024*1024)
    if err != nil {
        logger.Println("Failed to set send buffer size")
        return nil
    }
	return pub
}

func NewRouteServiceHandler(paramsDir string) *RouteServiceHandler {
	DummyRouteInfoRecord.protocol = PROTOCOL_NONE
	configFile := paramsDir + "/clients.json"
	logger.Println("configfile = ", configFile)
	ConnectToClients(configFile)
	RIBD_PUB = InitPublisher()
	go setupEventHandler(AsicdSub, asicdConstDefs.PUB_SOCKET_ADDR, SUB_ASICD)
	go setupEventHandler(PortdSub, portdCommonDefs.PUB_SOCKET_ADDR, SUB_PORTD)
	//CreateRoutes("RouteSetup.json")
	UpdateRoutesFromDB(paramsDir)
	return &RouteServiceHandler{}
}
