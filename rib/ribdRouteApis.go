package main

import (
	"arpd"
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

func getSelectedRoute(routeInfoRecordList RouteInfoRecordList) (routeInfoRecord RouteInfoRecord, err error) {
	if routeInfoRecordList.selectedRouteIdx == PROTOCOL_NONE {
		err = errors.New("No route selected")
	} else {
		routeInfoRecord = routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
	}
	return routeInfoRecord, err
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
	logger.Println("Getting ip intfs from portd")
	var currMarker int64
	var count int64
	count = 100
	for {
		logger.Printf("Getting %d objects from currMarker %d\n", count, currMarker)
		IPIntfBulk, err := asicdclnt.ClientHdl.GetBulkIPv4Intf(currMarker, count)
		if err != nil {
			logger.Println("GetBulkIPv4Intf with err ", err)
			return
		}
		if IPIntfBulk.ObjCount == 0 {
			logger.Println("0 objects returned from GetBulkIPv4Intf")
			return
		}
		logger.Printf("len(IPIntfBulk.IPv4IntfList)  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfList), IPIntfBulk.ObjCount)
		for i := 0; i < int(IPIntfBulk.ObjCount); i++ {
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
			_, err = routeServiceHandler.CreateV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribdCommonDefs.CONNECTED)// FIBAndRIB, ribd.Int(len(destNetSlice)))
			if err != nil {
				logger.Printf("Failed to create connected route for ip Addr %s/%s intfType %d intfId %d\n", ipAddrStr, ipMaskStr, ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(IPIntfBulk.IPv4IntfList[i].IfIndex)))
			}
		}
		if IPIntfBulk.More == false {
			logger.Println("more returned as false, so no more get bulks")
			return
		}
		currMarker = IPIntfBulk.NextMarker
	}
}

//thrift API definitions

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
	routes = &returnRouteGetInfo
	moreRoutes := true
	if destNetSlice == nil {
		logger.Println("destNetSlice not initialized")
		return routes, err
	}
	for ; ; i++ {
		logger.Printf("Fetching trie record for index %d\n", i+fromIndex)
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
		if prefixNode != nil {
			prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
			if prefixNodeRouteList.isPolicyBasedStateValid == false {
			    logger.Println("Route invalidated based on policy")
				continue
			}
			logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
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
			nextRoute.PolicyList = make([]string,0)
			for i:=0;i<len(prefixNodeRouteList.policyList);i++ {
				nextRoute.PolicyList = append(nextRoute.PolicyList, prefixNodeRouteList.policyList[i])
			}
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
	if routeInfoRecordList.selectedRouteIdx == PROTOCOL_NONE {
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
	route.Metric = routeInfoRecord.metric
	route.Prototype = ribd.Int(routeInfoRecord.protocol)
	return route, err
}

func SelectV4Route(destNetPrefix patriciaDB.Prefix,
	routeInfoRecordList RouteInfoRecordList,
	routeInfoRecord RouteInfoRecord,
	op ribd.Int,
	index int) (err error) {
	var routeInfoRecordNew RouteInfoRecord
	var routeInfoRecordOld RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	routeInfoRecordNew.protocol = PROTOCOL_NONE
	routeInfoRecordOld.protocol = PROTOCOL_NONE
	var i int8
	var deleteRoute bool
	policyRoute := ribd.Routes{Ipaddr: routeInfoRecord.destNetIp.String(), Mask: routeInfoRecord.networkMask.String(), NextHopIp: routeInfoRecord.nextHopIp.String(), NextHopIfType: ribd.Int(routeInfoRecord.nextHopIfType), IfIndex: routeInfoRecord.nextHopIfIndex, Metric: routeInfoRecord.metric, Prototype: ribd.Int(routeInfoRecord.protocol), IsPolicyBasedStateValid:routeInfoRecordList.isPolicyBasedStateValid}
	logger.Printf("Selecting the best Route for destNetPrefix %v, index = %d\n", destNetPrefix, index)
	if op == add {
		selectedRoute, err := getSelectedRoute(routeInfoRecordList)
		logger.Printf("Selected route protocol = %d, routeinforecord.protool=%d\n", selectedRoute.protocol, routeInfoRecord.protocol)
		if err == nil && ((selectedRoute.protocol == PROTOCOL_NONE && routeInfoRecord.protocol != PROTOCOL_NONE) || routeInfoRecord.protocol <= selectedRoute.protocol) {
			routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx] = selectedRoute
			routeInfoRecordOld = selectedRoute
			destNetSlice[routeInfoRecordOld.sliceIdx].isValid = false
			//destNetSlice is a slice of localDB maintained for a getBulk operations. An entry is created in this db when we create a new route
			if destNetSlice != nil && (len(destNetSlice) > int(routeInfoRecord.sliceIdx)) { //&& bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNet)) {
				if bytes.Equal(destNetSlice[routeInfoRecord.sliceIdx].prefix, destNetPrefix) == false {
					logger.Println("Unexpected destination network prefix found at the slice Idx ", routeInfoRecord.sliceIdx)
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
			logger.Printf("new selected route idx = %d\n", routeInfoRecordList.selectedRouteIdx)
		}
	} else if op == del {
		logger.Println(" in del index selectedrouteIndex", index, routeInfoRecordList.selectedRouteIdx)
		if len(routeInfoRecordList.routeInfoList) == 0 {
			logger.Println(" in del,numRoutes now 0, so delete the node")
			RouteInfoMap.Delete(destNetPrefix)
			//call asicd to del
			if asicdclnt.IsConnected {
				asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
			}
			var params RouteParams
			params.createType = Invalid
			params.destNetIp = routeInfoRecord.destNetIp.String()
			params.networkMask = routeInfoRecord.networkMask.String()
			policyRoute.PolicyList = routeInfoRecordList.policyList
	        PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export,params )
		    return nil
		}
		if destNetSlice == nil || int(routeInfoRecord.sliceIdx) >= len(destNetSlice) {
			logger.Println("Destination slice not found at the expected slice index ", routeInfoRecord.sliceIdx)
			return err
		}
		destNetSlice[routeInfoRecord.sliceIdx].isValid = false //invalidate this entry in the local db
		if int8(index) == routeInfoRecordList.selectedRouteIdx {
			logger.Println("Deleting the selected route")
			var dummyRouteInfoRecord RouteInfoRecord
			dummyRouteInfoRecord.protocol = PROTOCOL_NONE
			deleteRoute = true
			routeInfoRecord.protocol = PROTOCOL_NONE
			for i = 0; i < int8(len(routeInfoRecordList.routeInfoList)); i++ {
				routeInfoRecordTemp = routeInfoRecordList.routeInfoList[i]
				/*if i == int8(index) { //if(ok != true || i==routeInfoRecord.protocol) {
					continue
				}*/
				logger.Printf("temp protocol=%d, routeInfoRecord.protocol=%d\n", routeInfoRecordTemp.protocol, routeInfoRecord.protocol)
				if (routeInfoRecordTemp.protocol != PROTOCOL_NONE && routeInfoRecord.protocol != routeInfoRecordTemp.protocol && destNetSlice[routeInfoRecord.sliceIdx].isValid) {
					logger.Printf(" selceting protocol %d", routeInfoRecordTemp.protocol)
					routeInfoRecordList.routeInfoList[i] = routeInfoRecordTemp
					routeInfoRecordNew = routeInfoRecordTemp
					routeInfoRecordList.selectedRouteIdx = i
					logger.Println("routeRecordInfo.sliceIdx = ", routeInfoRecord.sliceIdx)
					logger.Println("routeRecordInfoNew.sliceIdx = ", routeInfoRecordNew.sliceIdx)
					routeInfoRecordNew.sliceIdx = routeInfoRecord.sliceIdx
					destNetSlice[routeInfoRecordNew.sliceIdx].isValid = true
					break
				}
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
	RouteInfoMap.Set(patriciaDB.Prefix(destNetPrefix), routeInfoRecordList)

	if deleteRoute == true || routeInfoRecordOld.protocol != PROTOCOL_NONE {
		if deleteRoute == true {
			logger.Println("Deleting the selected route, so call asicd to delete")
		}
		if routeInfoRecordOld.protocol != PROTOCOL_NONE {
			logger.Println("routeInfoRecordOld.protocol != PROTOCOL_NONE - adding a better route, so call asicd to delete")
		}
		//call asicd to del
		if asicdclnt.IsConnected {
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
		}
		var params RouteParams
		params.createType = Invalid
		params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
		policyRoute.PolicyList = routeInfoRecordList.policyList
	    PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export,params )
	}
	if routeInfoRecordNew.protocol != PROTOCOL_NONE {
		logger.Println("New route selected, call asicd to install a new route")
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
		var params RouteParams
		params.deleteType = Invalid
		params.destNetIp = routeInfoRecord.destNetIp.String()
		params.networkMask = routeInfoRecord.networkMask.String()
	    PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export,params )
	}
	return nil
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
	policyStateChange int,
	sliceIdx ribd.Int) (rc ribd.Int, err error) {
	logger.Printf("createV4Route for ip %s mask %s next hop ip %s addType %d\n", destNetIp, networkMask, nextHopIp, addType)

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
	var prefixNodeRouteList RouteInfoRecordList
	var prefixNodeRoute RouteInfoRecord
	routeInfoRecord := RouteInfoRecord{destNetIp: destNetIpAddr, networkMask: networkMaskAddr, protocol: routePrototype, nextHopIp: nextHopIpAddr, nextHopIfType: int8(nextHopIfType), nextHopIfIndex: nextHopIfIndex, metric: metric, sliceIdx: int(sliceIdx)}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if routeInfoRecordListItem == nil {
		if addType == FIBOnly {
			logger.Println("route record list not found in RIB")
			err = errors.New("Unexpected: route record list not found in RIB")
			return 0, err
		}
		var newRouteInfoRecordList RouteInfoRecordList
		newRouteInfoRecordList.routeInfoList = make([]RouteInfoRecord, 0)
		newRouteInfoRecordList.routeInfoList = append(newRouteInfoRecordList.routeInfoList, routeInfoRecord)
		newRouteInfoRecordList.selectedRouteIdx = 0
	    if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
	      newRouteInfoRecordList.isPolicyBasedStateValid = false
	   } else if  policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
	      newRouteInfoRecordList.isPolicyBasedStateValid = true
    }
		if ok := RouteInfoMap.Insert(destNet, newRouteInfoRecordList); ok != true {
			logger.Println(" return value not ok")
		}
		localDBRecord := localDB{prefix: destNet, isValid: true}
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
	    var params RouteParams
		params.destNetIp = destNetIp
		params.networkMask = networkMask
		params.routeType = routeType
		params.createType = addType 
		params.deleteType = Invalid
		policyRoute.IsPolicyBasedStateValid = newRouteInfoRecordList.isPolicyBasedStateValid
	    PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Export,params )
	} else {
		logger.Println("routeInfoRecordListItem not nil")
		routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
		found, _ := IsRoutePresent(routeInfoRecordList, routePrototype)
        if (found && (addType == FIBAndRIB)) {
			logger.Println("Trying to create a duplicate route")
			err = errors.New("Duplicate route creation")
			return 0, err
		} 
		if !found {
			if addType != FIBOnly {
				routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList, routeInfoRecord)
				logger.Printf("Fetching trie record for prefix %v\n", destNet)
				prefixNode := RouteInfoMap.Get(destNet)
				if prefixNode != nil {
					prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
					logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
					prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
					logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
				}
			}
		  }
	      if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
	         routeInfoRecordList.isPolicyBasedStateValid = false
	      } else if  policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
	        routeInfoRecordList.isPolicyBasedStateValid = true
          }
			err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, add, len(routeInfoRecordList.routeInfoList)-1)
			logger.Printf("Fetching trie record for prefix %v after selectv4route\n", destNet)
			prefixNode := RouteInfoMap.Get(destNet)
			if prefixNode != nil {
				prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
				logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
				prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
				logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
			}
		//}
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
	routeType ribd.Int) (rc ribd.Int, err error) {
	logger.Printf("Received create route request for ip %s mask %s\n", destNetIp, networkMask)
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return 0, err
	}

	policyRoute := ribd.Routes{Ipaddr: destNetIp, Mask: networkMask, NextHopIp: nextHopIp, NextHopIfType: nextHopIfType, IfIndex: nextHopIfIndex, Metric: metric, Prototype: routeType}
	params := RouteParams{destNetIp:destNetIp, networkMask:networkMask, nextHopIp:nextHopIp, nextHopIfType:nextHopIfType, nextHopIfIndex:nextHopIfIndex, metric:metric, routeType:routeType, sliceIdx:ribd.Int(len(destNetSlice)), createType:FIBAndRIB, deleteType:Invalid}
    logger.Println("createType = ", params.createType, "deleteType = ", params.deleteType)
	PolicyEngineFilter(policyRoute, ribdCommonDefs.PolicyPath_Import,params )

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
	routeType ribd.Int,
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
	if policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoInValid {
	   routeInfoRecordList.isPolicyBasedStateValid = false
	} else if  policyStateChange == ribdCommonDefs.RoutePolicyStateChangetoValid {
	   routeInfoRecordList.isPolicyBasedStateValid = true
    }
	routeInfoRecord := routeInfoRecordList.routeInfoList[i]
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
		routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList[:i], routeInfoRecordList.routeInfoList[i+1:]...)
	}
	logger.Printf("Fetching trie record for prefix after append%v\n", destNet)
	prefixNode = RouteInfoMap.Get(destNet)
	if prefixNode != nil {
		prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
		logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
		prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
		logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
	}
	err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, del, int(i)) //this function will invalidate the route in destNetSlice and also delete the entry in FIB (Asic)
	logger.Printf("Fetching trie record for prefix after selectv4route%v\n", destNet)
	prefixNode = RouteInfoMap.Get(destNet)
	if prefixNode != nil {
		prefixNodeRouteList = prefixNode.(RouteInfoRecordList)
		logger.Println("selectedRouteIdx = ", prefixNodeRouteList.selectedRouteIdx)
		prefixNodeRoute = prefixNodeRouteList.routeInfoList[prefixNodeRouteList.selectedRouteIdx]
		logger.Println("selected route's next hop = ", prefixNodeRoute.nextHopIp.String())
	}

	if routePrototype == ribdCommonDefs.CONNECTED { //PROTOCOL_CONNECTED {
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
	routeType ribd.Int) (rc ribd.Int, err error) {
	logger.Println("Received Route Delete request")
	if !acceptConfig {
		logger.Println("Not ready to accept config")
		//return 0,err
	}
	_, err = deleteV4Route(destNetIp, networkMask, routeType, FIBAndRIB,ribdCommonDefs.RoutePolicyStateChangetoInValid)
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
	if !acceptConfig {
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
