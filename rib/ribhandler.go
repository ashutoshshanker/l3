package main
import ("ribd"
        "asicdServices"
	    "encoding/json"
	    "io/ioutil"
		"git.apache.org/thrift.git/lib/go/thrift"
        "errors"
		"strconv"
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
      CONNECTED  = 0
      STATIC     = 1
      OSPF       = 89
      BGP        = 8
)
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
   destNet                 int
   prefixLen               int
   destNetIp				  string
   networkMask             string
   protocol                ribd.Int
   nextHopIp               string
   nextHopIfIndex          ribd.Int
   metric                  ribd.Int
   selected                bool
}

type RouteInfoMapIndex struct {
   destNetIdx                 int
   prefixLenIdx               int
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var RouteInfoMap = make(map[RouteInfoMapIndex] [] RouteInfoRecord)
var DummyRouteInfoRecord RouteInfoRecord//{destNet:0, prefixLen:0, protocol:0, nextHop:0, nextHopIfIndex:0, metric:0, selected:false}
var asicdclnt AsicdClient


func setProtocol(routeType    ribd.Int) (proto ribd.Int, err error) {
	err=nil
	switch(routeType) {
		case CONNECTED:
		proto = PROTOCOL_CONNECTED
		case STATIC:
		proto = PROTOCOL_STATIC
		case OSPF:
		proto = PROTOCOL_OSPF
		case BGP:
		proto = PROTOCOL_BGP
		default:
		err=errors.New("Not accepted protocol")
	}
	return proto,err
}

func getSelectedRoute(routeInfoRecordList [] RouteInfoRecord) (routeInfoRecord RouteInfoRecord, err error) {
   for _,routeInfoRecord = range routeInfoRecordList {
	   if(routeInfoRecord.selected) { 
	     return routeInfoRecord, nil
	 }
   }
   err = errors.New("No route selected")
   return routeInfoRecord, err
}

func SelectV4Route( destNet     int,         
                    prefixLen   int,
					 routeInfoRecord RouteInfoRecord,
					 op          ribd.Int) ( err error) {
	var routeInfoRecordNew RouteInfoRecord
	var routeInfoRecordOld RouteInfoRecord
	var routeInfoRecordTemp RouteInfoRecord
	var i ribd.Int
     logger.Printf("Selecting the best Route for destNet %d prefix %d\n", destNet, prefixLen)
    routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet, prefixLenIdx:prefixLen}
	 routeInfoRecordList,_ := RouteInfoMap[routeInfoMapIndex]
     if(op == add) {
		selectedRoute, err := getSelectedRoute(routeInfoRecordList)
		if( err != nil) { //no routes selected, so make this the selected route
			routeInfoRecord.selected = true
			routeInfoRecordNew = routeInfoRecord
		} else if(routeInfoRecord.protocol < selectedRoute.protocol) {
			selectedRoute.selected = false
			routeInfoRecordList[selectedRoute.protocol] = selectedRoute
			routeInfoRecordOld = selectedRoute
			routeInfoRecord.selected = true
			routeInfoRecordList[routeInfoRecord.protocol] = routeInfoRecord
			routeInfoRecordNew = routeInfoRecord
		}	
	 }	 else if(op == del) {
		routeInfoRecordOld = routeInfoRecord
		logger.Println(" in del")
		for i =0; i<PROTOCOL_LAST; i++ {
			if(i==routeInfoRecord.protocol) {
				continue
			}
			routeInfoRecordTemp = routeInfoRecordList[i]
			logger.Printf("temp protocol=%d", routeInfoRecordTemp.protocol)
			if(routeInfoRecordTemp.protocol != PROTOCOL_NONE) {
				logger.Printf(" selceting protocol %d", routeInfoRecordTemp.protocol)
				routeInfoRecordList[i].selected = true
               routeInfoRecordNew = routeInfoRecordTemp
				break;
			}
		}
	 }
    if(routeInfoRecordOld.protocol != PROTOCOL_NONE) {
		//call asicd to del
	}
	if(routeInfoRecordNew.protocol != PROTOCOL_NONE) {
		//call asicd to add
			asicdclnt.ClientHdl.CreateIPv4Route(routeInfoRecord.destNetIp, routeInfoRecord.networkMask, routeInfoRecord.nextHopIp)
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
	destNet, err := getIPInt(destNetIp)
	if(err != nil) {
		return -1, err
	}
    routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet, prefixLenIdx:prefixLen}
	//routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet & prefixLen}
	logger.Printf("Creating/looking route request for network %d %d", routeInfoMapIndex.destNetIdx, routeInfoMapIndex.prefixLenIdx)
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return 0,err
	}
	logger.Printf("routePrototype %d for routeType %d", routePrototype, routeType)
    routeInfoRecord := RouteInfoRecord{destNet:destNet, prefixLen:prefixLen, destNetIp:destNetIp, networkMask:networkMask, protocol:routePrototype, nextHopIp:nextHopIp, nextHopIfIndex:nextHopIfIndex, metric:metric, selected:false}
    _,ok := RouteInfoMap[routeInfoMapIndex]
	if !ok {
		RouteInfoMap[routeInfoMapIndex] = make([] RouteInfoRecord, PROTOCOL_LAST)
		routeInfoRecord.selected = true
		RouteInfoMap[routeInfoMapIndex][routePrototype] = routeInfoRecord
		//call asicd 
		if(asicdclnt.IsConnected) {
			asicdclnt.ClientHdl.CreateIPv4Route(destNetIp, networkMask, nextHopIp)
	   }
	} else {
	   RouteInfoMap[routeInfoMapIndex][routePrototype]=routeInfoRecord
	   //RouteInfoMap[routeInfoMapIndex] = append(RouteInfoMap[routeInfoMapIndex], routeInfoRecord)
       err = SelectV4Route(destNet, prefixLen, routeInfoRecord, add)
	}
    return 0, err
}

func (m RouteServiceHandler) DeleteV4Route( destNetIp         string, 
                                            networkMask       string,
											  routeType       ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received Route Delete request")
	prefixLen, err := getPrefixLen(networkMask)
	if(err != nil) {
		return -1, err
	}
	logger.Printf("prefixLen=%d\n", prefixLen)
	destNet, err := getIPInt(destNetIp)
	if(err != nil) {
		return -1, err
	}
	logger.Printf("destNet = %d\n", destNet)
    routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet, prefixLenIdx:prefixLen}
    routeInfoRecordList,ok := RouteInfoMap[routeInfoMapIndex]
	if !ok {
       return 0, nil
	}
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return 0,err
	}
	//delete from slice
	routeInfoRecord := routeInfoRecordList[routePrototype]
	routeInfoRecordList[routePrototype]=DummyRouteInfoRecord
	if (routeInfoRecord.selected == true) {
		err = SelectV4Route(destNet, prefixLen, routeInfoRecord, del)
	}
    return 0, err
}

func (m RouteServiceHandler) UpdateV4Route( destNetIp         string, 
                                            networkMask     string, 
                                            routeType       ribd.Int,
                                            nextHopIp       string, 
                                            nextHopIfIndex  ribd.Int,
                                            metric          ribd.Int) (err error) {
    logger.Println("Received update route request")
	prefixLen, err := getPrefixLen(networkMask)
	if(err != nil) {
		return  err
	}
	destNet, err := getIPInt(destNetIp)
	if(err != nil) {
		return  err
	}
    routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet, prefixLenIdx:prefixLen}
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return err
	}
    routeInfoRecord := RouteInfoRecord{destNet:destNet, prefixLen:prefixLen, protocol:routePrototype, nextHopIp:nextHopIp, nextHopIfIndex:nextHopIfIndex, metric:metric, selected:false}
    routeInfoRecordList,ok := RouteInfoMap[routeInfoMapIndex]
	if !ok {
       err = errors.New("No route found")
	   return err
	}
	routeInfoRecord.selected = routeInfoRecordList[routePrototype].selected
	RouteInfoMap[routeInfoMapIndex][routePrototype]=routeInfoRecord
	if(routeInfoRecord.selected == true) {
		//call asicd to update info
	}
    return err
}

func (m RouteServiceHandler) PrintV4Routes() (err error) {
   logger.Println("Received print route")
   for k,rmapInfoList := range RouteInfoMap {
	for _,v := range rmapInfoList {
      logger.Printf("%d %d-> %d %d\n", k.destNetIdx, k.prefixLenIdx, v.protocol, v.selected)
    }
   }
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


func NewRouteServiceHandler () *RouteServiceHandler {
	configFile := "../../config/params/clients.json"
	ConnectToClients(configFile)
    return &RouteServiceHandler{}
}
