package main
import ("ribd"
        "asicdServices"
		"git.apache.org/thrift.git/lib/go/thrift"
        "errors"
        _ "net")

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
   destNet                 ribd.Int
   prefixLen               ribd.Int
   protocol                ribd.Int
   nextHop                 ribd.Int
   nextHopIfIndex          ribd.Int
   metric                  ribd.Int
   selected                bool
}

type RouteInfoMapIndex struct {
   destNetIdx                 ribd.Int
   prefixLenIdx               ribd.Int
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

func SelectV4Route( destNet     ribd.Int,         
                    prefixLen   ribd.Int,
					 routeInfoRecord RouteInfoRecord,
					 op          ribd.Int) (err error) {
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
	}
     return nil
}

func (m RouteServiceHandler) CreateV4Route( destNet         ribd.Int, 
                                            prefixLen       ribd.Int, 
                                            routeType       ribd.Int,
                                            nextHop         ribd.Int, 
                                            nextHopIfIndex  ribd.Int,
                                            metric          ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received create route request")
    routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet, prefixLenIdx:prefixLen}
	//routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet & prefixLen}
	logger.Printf("Creating/looking route request for network %d %d", routeInfoMapIndex.destNetIdx, routeInfoMapIndex.prefixLenIdx)
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return 0,err
	}
	logger.Printf("routePrototype %d for routeType %d", routePrototype, routeType)
    routeInfoRecord := RouteInfoRecord{destNet:destNet, prefixLen:prefixLen, protocol:routePrototype, nextHop:nextHop, nextHopIfIndex:nextHopIfIndex, metric:metric, selected:false}
    _,ok := RouteInfoMap[routeInfoMapIndex]
	if !ok {
		RouteInfoMap[routeInfoMapIndex] = make([] RouteInfoRecord, PROTOCOL_LAST)
		routeInfoRecord.selected = true
		RouteInfoMap[routeInfoMapIndex][routePrototype] = routeInfoRecord
		//call asicd 
		if(asicdclnt.IsConnected) {
	      asicdclnt.ClientHdl.CreateVlan(100,"1","2")
	   }
	} else {
	   RouteInfoMap[routeInfoMapIndex][routePrototype]=routeInfoRecord
	   //RouteInfoMap[routeInfoMapIndex] = append(RouteInfoMap[routeInfoMapIndex], routeInfoRecord)
       err = SelectV4Route(destNet, prefixLen, routeInfoRecord, add)
	}
    return 0, err
}

func (m RouteServiceHandler) DeleteV4Route( destNet         ribd.Int, 
                                            prefixLen       ribd.Int,
											  routeType       ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received Route Delete request")
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

func (m RouteServiceHandler) UpdateV4Route( destNet         ribd.Int, 
                                            prefixLen       ribd.Int, 
                                            routeType       ribd.Int,
                                            nextHop         ribd.Int, 
                                            nextHopIfIndex  ribd.Int,
                                            metric          ribd.Int) (err error) {
    logger.Println("Received update route request")
    routeInfoMapIndex := RouteInfoMapIndex{destNetIdx: destNet, prefixLenIdx:prefixLen}
	routePrototype,err := setProtocol(routeType)
	if(err != nil){
		return err
	}
    routeInfoRecord := RouteInfoRecord{destNet:destNet, prefixLen:prefixLen, protocol:routePrototype, nextHop:nextHop, nextHopIfIndex:nextHopIfIndex, metric:metric, selected:false}
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

func NewRouteServiceHandler () *RouteServiceHandler {
	asicdclnt.Address = "localhost:4000"
	asicdclnt.Transport, asicdclnt.PtrProtocolFactory = CreateIPCHandles(asicdclnt.Address)
	if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
		logger.Println("connecting to asicd")
		asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
       asicdclnt.IsConnected = true
	}
    return &RouteServiceHandler{}
}
