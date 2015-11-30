package main
import ("ribd"
        "l3/rib/ribdCommonDefs"
        "asicdServices"
	    "encoding/json"
		"utils/patriciaDB"
//		"patricia"
	    "io/ioutil"
		"git.apache.org/thrift.git/lib/go/thrift"
        "errors"
		"strconv"
		"time"
         "net")

type RouteServiceHandler struct {
}

type RIBClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	RIBClientBase
	ClientHdl          *asicdServices.AsicdServiceClient
}

const (
	PROTOCOL_NONE = -1
	PROTOCOL_CONNECTED = 0
	PROTOCOL_STATIC =1
	PROTOCOL_OSPF =2
	PROTOCOL_BGP=3
	PROTOCOL_LAST =4
)

const (
	add = iota
	del
)

type RouteInfoRecord struct {
   destNetIp				 net.IP//string
   networkMask              net.IP//string
   nextHopIp               net.IP
   nextHopIfIndex          ribd.Int 
   metric                  ribd.Int
   protocol                int8
}
//implement priority queue of the routes
type RouteInfoRecordList struct {
   selectedRouteIdx         int8
   routeInfoList            [] RouteInfoRecord//map[int]RouteInfoRecord	
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
	OutgoingInterface string
	Protocol          string
}

var RouteInfoMap = patriciaDB.NewTrie()
var DummyRouteInfoRecord RouteInfoRecord//{destNet:0, prefixLen:0, protocol:0, nextHop:0, nextHopIfIndex:0, metric:0, selected:false}
var asicdclnt AsicdClient
var count int
var ConnectedRoutes []*ribd.Routes


func setProtocol(routeType    ribd.Int) (proto int8, err error) {
	err=nil
	switch(routeType) {
		case ribdCommonDefs.CONNECTED:
		proto = PROTOCOL_CONNECTED
		case ribdCommonDefs.STATIC:
		proto = PROTOCOL_STATIC
		case ribdCommonDefs.OSPF:
		proto = PROTOCOL_OSPF
		case ribdCommonDefs.BGP:
		proto = PROTOCOL_BGP
		default:
		err=errors.New("Not accepted protocol")
		proto = -1
	}
	return proto,err
}

func getSelectedRoute(routeInfoRecordList RouteInfoRecordList) (routeInfoRecord RouteInfoRecord, err error) {
   if(routeInfoRecordList.selectedRouteIdx == PROTOCOL_NONE) {
      err = errors.New("No route selected")
	} else {
		routeInfoRecord = routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
	}
   return routeInfoRecord, err
}

func SelectV4Route( destNet     patriciaDB.Prefix,         
                    routeInfoRecordList RouteInfoRecordList,
					 routeInfoRecord RouteInfoRecord,
					 op          ribd.Int,
					 index       int) ( err error) {
	var routeInfoRecordNew RouteInfoRecord
	var routeInfoRecordOld RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	var i int8
    logger.Printf("Selecting the best Route for destNet %v, index = %d\n", destNet, index)
     if(op == add) {
		selectedRoute, err := getSelectedRoute(routeInfoRecordList)
		if( err == nil && routeInfoRecord.protocol < selectedRoute.protocol) { 
			routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx] = selectedRoute
			routeInfoRecordOld = selectedRoute
			routeInfoRecordList.routeInfoList[index] = routeInfoRecord
			routeInfoRecordNew = routeInfoRecord
			routeInfoRecordList.selectedRouteIdx = int8(index)
			logger.Printf("new selected route idx = %d\n", routeInfoRecordList.selectedRouteIdx)
	   }
	 }	else if(op == del) {
		logger.Println(" in del index selectedrouteIndex", index, routeInfoRecordList.selectedRouteIdx)
		if(len(routeInfoRecordList.routeInfoList) == 0) {
	  	   logger.Println(" in del,numRoutes now 0, so delete the node")
			RouteInfoMap.Delete(destNet)
			return nil
		}
		routeInfoRecordOld = routeInfoRecord
		if(int8(index) == routeInfoRecordList.selectedRouteIdx) {
		  for i =0; i<int8(len(routeInfoRecordList.routeInfoList)); i++ {
			routeInfoRecordTemp = routeInfoRecordList.routeInfoList[i]
			if(i == int8(index)) {//if(ok != true || i==routeInfoRecord.protocol) {
				continue
			}
			logger.Printf("temp protocol=%d", routeInfoRecordTemp.protocol)
			if(routeInfoRecordTemp.protocol != PROTOCOL_NONE) {
				logger.Printf(" selceting protocol %d", routeInfoRecordTemp.protocol)
				routeInfoRecordList.routeInfoList[i] = routeInfoRecordTemp
               routeInfoRecordNew = routeInfoRecordTemp
			    routeInfoRecordList.selectedRouteIdx = i
				break;
			}
		  }
		} else {
			if(routeInfoRecordList.selectedRouteIdx > int8(index)) {
				routeInfoRecordList.selectedRouteIdx--
			}
		}
	 }
	//update the patriciaDB trie with the updated route info record list
	RouteInfoMap.Set(patriciaDB.Prefix(destNet), routeInfoRecordList)
	
    if(routeInfoRecordOld.protocol != PROTOCOL_NONE) {
		//call asicd to del
		if(asicdclnt.IsConnected) {
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String())
		}
	}
	if(routeInfoRecordNew.protocol != PROTOCOL_NONE) {
		//call asicd to add
		if(asicdclnt.IsConnected){
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String(), routeInfoRecord.nextHopIp.String())
        //call arpd to resolve the ip 
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
    parsedPrefixIP := int(ip[3]) | int(ip[2]) << 8 | int(ip[1]) << 16 | int(ip[0]) << 24
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
	if(err != nil) {
		return -1, err
	}
	for prefixLen = 0;ipInt !=0; ipInt >>=1 {
		prefixLen += ipInt & 1;
	}
	return prefixLen, nil
}

func (m RouteServiceHandler) GetConnectedRoutesInfo () (routes []*ribd.Routes, err error) {
   logger.Println("Received GetConnectedRoutesInfo")
	routes = ConnectedRoutes
	return routes, err
}
func (m RouteServiceHandler) GetRouteReachabilityInfo(destNet string) (nextHopIntf *ribd.NextHopInfo, err error) {
   t1 := time.Now()
   var retnextHopIntf ribd.NextHopInfo
   nextHopIntf = &retnextHopIntf
   var found bool
   destNetIp, err := getIP(destNet)
   if( err != nil) {
      return nextHopIntf, errors.New("Invalid dest ip address")	
  }
  rmapInfoListItem := RouteInfoMap.GetLongestPrefixNode(patriciaDB.Prefix(destNetIp))
  if(rmapInfoListItem != nil) {
     rmapInfoList := rmapInfoListItem.(RouteInfoRecordList)
	if(rmapInfoList.selectedRouteIdx != PROTOCOL_NONE) {
	  found = true
	  v := rmapInfoList.routeInfoList[rmapInfoList.selectedRouteIdx]
      nextHopIntf.NextHopIfIndex = v.nextHopIfIndex
	  nextHopIntf.NextHopIp = v.nextHopIp.String()
	  nextHopIntf.Metric = v.metric
	}
   }
	
   if(found == false) {
	  logger.Printf("dest IP %s not reachable\n", destNetIp)
	  err = errors.New("dest ip address not reachable")
   }
   duration := time.Since(t1)
   logger.Printf("time to get longestPrefixLen = %d\n", duration.Nanoseconds())
   logger.Printf("next hop ip of the route = %s\n", nextHopIntf.NextHopIfIndex)
   return nextHopIntf, err
}

func getNetworkPrefix(destNetIp net.IP, networkMask net.IP) (destNet patriciaDB.Prefix, err error) {
	prefixLen, err := getPrefixLen(networkMask)
	if(err != nil) {
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
	 if((prefixLen % 8) != 0) {
		numbytes++
	}
	 destNet = make([]byte, numbytes)
	 for i :=0;i<numbytes;i++ {
		destNet[i]=netIp[i]
	}
	return destNet, nil
}
func updateConnectedRoutes(destNetIPAddr string, networkMaskAddr string, nextHopIfIndex ribd.Int, op int) {
	var temproute ribd.Routes
	route := &temproute
	logger.Printf("number of connectd routes = %d\n", len(ConnectedRoutes))
	if(len(ConnectedRoutes) == 0) {
		if(op == del) {
			logger.Println("Cannot delete a non-existent connected route")
			return
		}
	   ConnectedRoutes = make([]*ribd.Routes,1)
	   route.Ipaddr = destNetIPAddr
	   route.Mask = networkMaskAddr
	   route.IfIndex = nextHopIfIndex
	   ConnectedRoutes[0] = route
	   return
    }
	for i := 0;i<len(ConnectedRoutes);i++ {
//		if(!strings.EqualFold(ConnectedRoutes[i].Ipaddr,destNetIPAddr) && !strings.EqualFold(ConnectedRoutes[i].Mask,networkMaskAddr)){
	    if(ConnectedRoutes[i].Ipaddr == destNetIPAddr && ConnectedRoutes[i].Mask == networkMaskAddr) {
			if(op == del) {
				ConnectedRoutes = append(ConnectedRoutes[:i], ConnectedRoutes[i+1:]...)
			}
			return 
		}
	}
	if(op == del) {
		return
	}
	route.Ipaddr = destNetIPAddr
	route.Mask = networkMaskAddr
	route.IfIndex = nextHopIfIndex
	ConnectedRoutes = append(ConnectedRoutes, route)
}
func IsRoutePresent( routeInfoRecordList RouteInfoRecordList,
					  routePrototype int8) (found bool, i int) {
   for i:=0; i<len(routeInfoRecordList.routeInfoList);i++ {
	logger.Printf("len = %d i=%d routePrototype=%d\n", len(routeInfoRecordList.routeInfoList), i, routeInfoRecordList.routeInfoList[i].protocol)
	   if(routeInfoRecordList.routeInfoList[i].protocol == routePrototype){
		  found = true
		  return true, i
	   }
	}
	logger.Printf("retiurnong i = %d\n", i)
	return found, i
}
func (m RouteServiceHandler) CreateV4Route( destNetIp         string, 
                                            networkMask     string, 
                                            metric          ribd.Int,
                                            nextHopIp        string, 
                                            nextHopIfIndex  ribd.Int,
                                            routeType       ribd.Int) (rc ribd.Int, err error) {
//    logger.Printf("Received create route request for ip %s mask %s\n", destNetIp, networkMask)
   destNetIpAddr, err := getIP(destNetIp)
	if(err != nil){
		return 0,err
	}
	networkMaskAddr, err := getIP(networkMask)
	if(err != nil){
		return 0,err
	}
	nextHopIpAddr, err := getIP(nextHopIp)
	if(err != nil){
		return 0,err
	}
/*	prefixLen, err := getPrefixLen(networkMaskAddr)
	if(err != nil) {
		return -1, err
	}*/
    destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if(err != nil) {
		return -1, err
	}
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return 0,err
	}
	logger.Printf("routePrototype %d for routeType %d", routePrototype, routeType)
    routeInfoRecord := RouteInfoRecord{ destNetIp:destNetIpAddr, networkMask:networkMaskAddr, protocol:routePrototype, nextHopIp:nextHopIpAddr, nextHopIfIndex:nextHopIfIndex, metric:metric}
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if(routeInfoRecordListItem == nil) {
		var newRouteInfoRecordList RouteInfoRecordList
		newRouteInfoRecordList.routeInfoList = make([]RouteInfoRecord, 0)
		newRouteInfoRecordList.routeInfoList = append(newRouteInfoRecordList.routeInfoList, routeInfoRecord)
		newRouteInfoRecordList.selectedRouteIdx = 0
		if ok := RouteInfoMap.Insert(destNet, newRouteInfoRecordList); ok != true {
			logger.Println(" return value not ok")
		}
		//call asicd 
		if(asicdclnt.IsConnected) {
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp.String(), routeInfoRecord.networkMask.String(), routeInfoRecord.nextHopIp.String())
	   }
	} else {
       routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
       found,_ := IsRoutePresent(routeInfoRecordList, routePrototype)
	   if(!found){
        routeInfoRecordList.routeInfoList = append(routeInfoRecordList.routeInfoList, routeInfoRecord)
        err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, add, len(routeInfoRecordList.routeInfoList) - 1)
       } 
	}
	if(routePrototype == PROTOCOL_CONNECTED){
		updateConnectedRoutes(destNetIp, networkMask, nextHopIfIndex,add)
	}
    return 0, err
}

func (m RouteServiceHandler) DeleteV4Route( destNetIp        string, 
                                            networkMask      string,
											  routeType       ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received Route Delete request")
   destNetIpAddr, err := getIP(destNetIp)
	if(err != nil){
		return 0,err
	}
	networkMaskAddr, err := getIP(networkMask)
	if(err != nil){
		return 0,err
	}
    destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if(err != nil) {
		return -1, err
	}
	logger.Printf("destNet = %v\n", destNet)
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return 0,err
	}
    ok := RouteInfoMap.Match(destNet)
	if(!ok) {
		return 0,nil
	}
    routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if(routeInfoRecordListItem == nil){
		return 0, err
	}
    routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found, i := IsRoutePresent(routeInfoRecordList, routePrototype)
	if(!found) {
		logger.Println("Route not found")
		return 0, err
	}
	routeInfoRecord := routeInfoRecordList.routeInfoList[i]
	routeInfoRecordList.routeInfoList =  append(routeInfoRecordList.routeInfoList[:i], routeInfoRecordList.routeInfoList[i+1:]...)
	err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, del, int(i))
	if(routePrototype == PROTOCOL_CONNECTED){
		updateConnectedRoutes(destNetIp, networkMask, 0,del)
	}
    return 0, err
}

func (m RouteServiceHandler) UpdateV4Route( destNetIp        string,
                                            networkMask     string, 
                                            routeType       ribd.Int,
                                            nextHopIp       string, 
                                            nextHopIfIndex  ribd.Int,
                                            metric          ribd.Int) (err error) {
    logger.Println("Received update route request")
   destNetIpAddr, err := getIP(destNetIp)
	if(err != nil){
		return err
	}
	networkMaskAddr, err := getIP(networkMask)
	if(err != nil){
		return err
	}
	nextHopIpAddr, err := getIP(nextHopIp)
	if(err != nil){
		return err
	}
    destNet, err := getNetworkPrefix(destNetIpAddr, networkMaskAddr)
	if(err != nil) {
		return err
	}
	logger.Printf("destNet = %v\n", destNet)
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return err
	}
    ok := RouteInfoMap.Match(destNet)
	if !ok {
       err = errors.New("No route found")
	   return err
	}
    routeInfoRecord := RouteInfoRecord{protocol:routePrototype, nextHopIp:nextHopIpAddr, nextHopIfIndex:nextHopIfIndex, metric:metric}
    routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if(routeInfoRecordListItem == nil) {
		logger.Println("No route for destination network")
		return err
	}
	routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList)
	found, i := IsRoutePresent(routeInfoRecordList, routePrototype)
	if(!found) {
		logger.Println("No entry present for this destination and protocol")
		return err
	}
	routeInfoRecordList.routeInfoList[i] = routeInfoRecord
    RouteInfoMap.Set(destNet, routeInfoRecordList)
	if(routeInfoRecordList.selectedRouteIdx == int8(i)) {
		//call asicd to update info
	}
    return err
}

func printRoutesInfo(prefix patriciaDB.Prefix, item patriciaDB.Item) (err error) {
    rmapInfoRecordList := item.(RouteInfoRecordList)
	for _,v := range rmapInfoRecordList.routeInfoList {
	   if(v.protocol == PROTOCOL_NONE) {
		continue
	}
   //   logger.Printf("%v-> %d %d %d %d\n", prefix, v.destNetIp, v.networkMask, v.protocol)
		count++
    }
	return nil
}

func (m RouteServiceHandler) PrintV4Routes() (err error) {
	count =0
   logger.Println("Received print route")
   RouteInfoMap.Visit(printRoutesInfo)
	logger.Printf("total count = %d\n", count)
   return nil
}

//
// This method gets Thrift related IPC handles.
//
func CreateIPCHandles(address string) (thrift.TTransport, *thrift.TBinaryProtocolFactory) {
	var transportFactory thrift.TTransportFactory
	var transport thrift.TTransport
	var protocolFactory *thrift.TBinaryProtocolFactory
	var err error

	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory = thrift.NewTTransportFactory()
	transport, err = thrift.NewTSocket(address)
	transport = transportFactory.GetTransport(transport)
	if err = transport.Open(); err != nil {
		logger.Println("Failed to Open Transport", transport, protocolFactory)
		return nil, nil
	}
	return transport, protocolFactory
}

func ConnectToClients(paramsFile string){
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
        if(client.Name == "asicd") {
			logger.Printf("found asicd at port %d", client.Port)
	        asicdclnt.Address = "localhost:"+strconv.Itoa(client.Port)
	        asicdclnt.Transport, asicdclnt.PtrProtocolFactory = CreateIPCHandles(asicdclnt.Address)
	        if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
		       logger.Println("connecting to asicd")
		       asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
               asicdclnt.IsConnected = true
	        }
			break;
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

func NewRouteServiceHandler () *RouteServiceHandler {
   DummyRouteInfoRecord.protocol = PROTOCOL_NONE
   configFile := "params/clients.json"
	ConnectToClients(configFile)
	//CreateRoutes("RouteSetup.json")
    return &RouteServiceHandler{}
}
