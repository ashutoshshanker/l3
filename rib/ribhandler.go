package main
import ("ribd"
        "l3/rib/ribdCommonDefs"
        "asicdServices"
	    "encoding/json"
		"utils/patriciaDB"
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
	PROTOCOL_NONE = iota
	PROTOCOL_CONNECTED
	PROTOCOL_STATIC 
	PROTOCOL_OSPF
	PROTOCOL_BGP
	PROTOCOL_LAST
)

const (
	add = iota
	del
)

type RouteInfoRecord struct {
   prefixLen                int
   destNetIp				 string
   networkMask              string
   protocol                ribd.Int
   nextHopIp               string
   nextHopIfIndex          ribd.Int 
   metric                  ribd.Int
   selected                bool
}
type RouteInfoRecordList struct {
   destNet                  patriciaDB.Prefix  //this is the index /prefix into the trie and is computed as dest IP & mask
   numRoutes                ribd.Int
   selectedRouteIdx         ribd.Int
   routeInfoList            [PROTOCOL_LAST] RouteInfoRecord	
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


func setProtocol(routeType    ribd.Int) (proto ribd.Int, err error) {
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
	}
	return proto,err
}

func getSelectedRoute(routeInfoRecordList RouteInfoRecordList) (routeInfoRecord RouteInfoRecord, err error) {
/*   for _,routeInfoRecord = range routeInfoRecordList.routeInfoList {
	   if(routeInfoRecord.selected) { 
	     return routeInfoRecord, nil
	 }
   }
   err = errors.New("No route selected")*/
   routeInfoRecord = routeInfoRecordList.routeInfoList[routeInfoRecordList.selectedRouteIdx]
   return routeInfoRecord, err
}

func SelectV4Route( destNet     patriciaDB.Prefix,         
                    routeInfoRecordList RouteInfoRecordList,
					 routeInfoRecord RouteInfoRecord,
					 op          ribd.Int) ( err error) {
	var routeInfoRecordNew RouteInfoRecord
	var routeInfoRecordOld RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	var i ribd.Int
 //    logger.Printf("Selecting the best Route for destNet %v\n", destNet)
     if(op == add) {
		selectedRoute, err := getSelectedRoute(routeInfoRecordList)
		if( err == nil && routeInfoRecord.protocol < selectedRoute.protocol) { 
			selectedRoute.selected = false
			routeInfoRecordList.routeInfoList[selectedRoute.protocol] = selectedRoute
			routeInfoRecordOld = selectedRoute
			routeInfoRecord.selected = true
			routeInfoRecordList.routeInfoList[routeInfoRecord.protocol] = routeInfoRecord
			routeInfoRecordNew = routeInfoRecord
			routeInfoRecordList.selectedRouteIdx = routeInfoRecord.protocol
	   }
	 }	else if(op == del) {
		logger.Println(" in del")
		if(routeInfoRecordList.numRoutes == 0) {
	  	   logger.Println(" in del,numRoutes now 0, so delete the node")
			RouteInfoMap.Delete(destNet)
			return nil
		}
		routeInfoRecordOld = routeInfoRecord
		if(routeInfoRecord.selected == true) {
		  for i =0; i<PROTOCOL_LAST; i++ {
			if(i==routeInfoRecord.protocol) {
				continue
			}
			routeInfoRecordTemp = routeInfoRecordList.routeInfoList[i]
			logger.Printf("temp protocol=%d", routeInfoRecordTemp.protocol)
			if(routeInfoRecordTemp.protocol != PROTOCOL_NONE) {
				logger.Printf(" selceting protocol %d", routeInfoRecordTemp.protocol)
				routeInfoRecordList.routeInfoList[i].selected = true
               routeInfoRecordNew = routeInfoRecordTemp
			    routeInfoRecordList.selectedRouteIdx = i
				break;
			}
		  }
		}
	 }
	//update the patriciaDB trie with the updated route info record list
	RouteInfoMap.Set(patriciaDB.Prefix(destNet), routeInfoRecordList)
	
    if(routeInfoRecordOld.protocol != PROTOCOL_NONE) {
		//call asicd to del
		if(asicdclnt.IsConnected) {
			asicdclnt.ClientHdl.DeleteIPv4Route(routeInfoRecord.destNetIp, routeInfoRecord.networkMask)
		}
	}
	if(routeInfoRecordNew.protocol != PROTOCOL_NONE) {
		//call asicd to add
		if(asicdclnt.IsConnected){
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp, routeInfoRecord.networkMask, routeInfoRecord.nextHopIp)
        //call arpd to resolve the ip 
	    }
	}
     return nil
}

func getIPInt(ipAddr string) (ipInt int, err error) {
	ip := net.ParseIP(ipAddr)
    if ip == nil {
        return -1, errors.New("Invalid destination network IP Address")
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

func getPrefixLen(networkMask string) (prefixLen int, err error) {
	ipInt, err := getIPInt(networkMask)
	if(err != nil) {
		return -1, err
	}
	for prefixLen = 0;ipInt !=0; ipInt >>=1 {
		prefixLen += ipInt & 1;
	}
	return prefixLen, nil
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
     for _,v := range rmapInfoList.routeInfoList {
	    if(v.protocol != PROTOCOL_NONE || v.selected == true) {
           nextHopIntf.NextHopIfIndex = v.nextHopIfIndex
		   nextHopIntf.NextHopIp = v.nextHopIp
		   nextHopIntf.Metric = v.metric
		   found = true
		   break
	   }
     }
   }
	
   if(found == false) {
	  logger.Printf("dest IP %s not reachable\n", destNet)
	  err = errors.New("dest ip address not reachable")
   }
   duration := time.Since(t1)
   logger.Printf("time to get longestPrefixLen = %d\n", duration.Nanoseconds())
   logger.Printf("next hop ip of the route = %s\n", nextHopIntf.NextHopIfIndex)
   return nextHopIntf, err
}

func getNetworkPrefix(destNetIp string, networkMask string) (destNet patriciaDB.Prefix, err error) {
	prefixLen, err := getPrefixLen(networkMask)
	if(err != nil) {
		return destNet, err
	}
    ip, err := getIP(destNetIp)
    if err != nil {
        logger.Println("Invalid destination network IP Address")
		return destNet, err
    }
    vdestMaskIp,err := getIP(networkMask)
    if err != nil {
        logger.Println("Invalid network mask")
		return destNet, err
    }
	 vdestMask := net.IPv4Mask(vdestMaskIp[0], vdestMaskIp[1], vdestMaskIp[2], vdestMaskIp[3])
     netIp := ip.Mask(vdestMask)
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
func (m RouteServiceHandler) CreateV4Route( destNetIp         string, 
                                            networkMask     string, 
                                            metric          ribd.Int,
                                            nextHopIp         string, 
                                            nextHopIfIndex  ribd.Int,
                                            routeType       ribd.Int) (rc ribd.Int, err error) {
    logger.Printf("Received create route request for ip %s mask %s\n", destNetIp, networkMask)
	prefixLen, err := getPrefixLen(networkMask)
	if(err != nil) {
		return -1, err
	}
    destNet, err := getNetworkPrefix(destNetIp, networkMask)
	if(err != nil) {
		return -1, err
	}
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return 0,err
	}
//	logger.Printf("routePrototype %d for routeType %d", routePrototype, routeType)
    routeInfoRecord := RouteInfoRecord{prefixLen:prefixLen, destNetIp:destNetIp, networkMask:networkMask, protocol:routePrototype, nextHopIp:nextHopIp, nextHopIfIndex:nextHopIfIndex, metric:metric, selected:false}
//    ok := RouteInfoMap.Match(destNet)
//	if !ok {
	routeInfoRecordListItem := RouteInfoMap.Get(destNet)
	if(routeInfoRecordListItem == nil) {
		var newRouteInfoList RouteInfoRecordList
		newRouteInfoList.destNet = destNet
		newRouteInfoList.numRoutes = 1
		newRouteInfoList.selectedRouteIdx = routePrototype
		routeInfoRecord.selected = true
		newRouteInfoList.routeInfoList[routePrototype] = routeInfoRecord
		if ok := RouteInfoMap.Insert(destNet, newRouteInfoList); ok != true {
			logger.Println(" return value not ok")
		}
		
		//call asicd 
		if(asicdclnt.IsConnected) {
			asicdclnt.ClientHdl.CreateIPv4Route(destNetIp, networkMask, nextHopIp)
	   }
	} else {
       routeInfoRecordList := routeInfoRecordListItem.(RouteInfoRecordList) //RouteInfoMap.Get(destNet).(RouteInfoRecordList)
	   if(routeInfoRecordList.routeInfoList[routePrototype].protocol == PROTOCOL_NONE) {
         routeInfoRecordList.numRoutes++		
	   }
	   routeInfoRecordList.routeInfoList[routePrototype]=routeInfoRecord
	  // RouteInfoMap.Set(destNet, routeInfoList)
       err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, add)
	}
    return 0, err
}

func (m RouteServiceHandler) DeleteV4Route( destNetIp         string, 
                                            networkMask       string,
											  routeType       ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received Route Delete request")
    destNet, err := getNetworkPrefix(destNetIp, networkMask)
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
    routeInfoRecordList := RouteInfoMap.Get(destNet).(RouteInfoRecordList)
	//delete from slice
	routeInfoRecord := routeInfoRecordList.routeInfoList[routePrototype]
	routeInfoRecordList.routeInfoList[routePrototype]=DummyRouteInfoRecord
	routeInfoRecordList.numRoutes--
	err = SelectV4Route(destNet, routeInfoRecordList, routeInfoRecord, del)
    return 0, err
}

func (m RouteServiceHandler) UpdateV4Route( destNetIp         string, 
                                            networkMask     string, 
                                            routeType       ribd.Int,
                                            nextHopIp       string, 
                                            nextHopIfIndex  ribd.Int,
                                            metric          ribd.Int) (err error) {
    logger.Println("Received update route request")
    destNet, err := getNetworkPrefix(destNetIp, networkMask)
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
    routeInfoRecord := RouteInfoRecord{protocol:routePrototype, nextHopIp:nextHopIp, nextHopIfIndex:nextHopIfIndex, metric:metric, selected:false}
    routeInfoRecordList := RouteInfoMap.Get(destNet).(RouteInfoRecordList)
	routeInfoRecord.selected = routeInfoRecordList.routeInfoList[routePrototype].selected
	routeInfoRecordList.routeInfoList[routePrototype] = routeInfoRecord
    RouteInfoMap.Set(destNet, routeInfoRecordList)
	if(routeInfoRecord.selected == true) {
		//call asicd to update info
	}
    return err
}

func printRoutesInfo(prefix patriciaDB.Prefix, item patriciaDB.Item) (err error) {
    rmapInfoList := item.(RouteInfoRecordList)
	for _,v := range rmapInfoList.routeInfoList {
	   if(v.protocol == PROTOCOL_NONE) {
		continue
	}
      logger.Printf("%v-> %s %s %d %d\n", prefix, v.destNetIp, v.networkMask, v.protocol, v.selected)
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
	configFile := "params/clients.json"
	ConnectToClients(configFile)
	//CreateRoutes("RouteSetup.json")
    return &RouteServiceHandler{}
}
