package main

import (
	"arpd"
	"asicdServices"
//	"portdServices"
	"encoding/json"
	"l3/rib/ribdCommonDefs"
	"ribd"
	"utils/patriciaDB"
	//		"patricia"
//	"errors"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/op/go-nanomsg"
	"asicd/asicdConstDefs"
	"io/ioutil"
	"net"
	"strconv"
	"time"
//	"encoding/binary"
//	"bytes"
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

type AsicdClient struct {
	RIBClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type ArpdClient struct {
	RIBClientBase
	ClientHdl *arpd.ARPDServicesClient
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
var count int
var ConnectedRoutes []*ribd.Routes
var destNetSlice []localDB
var acceptConfig bool
var AsicdSub *nanomsg.SubSocket
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


func processL3IntfDownEvent(ipAddr string){
	logger.Println("processL3IntfDownEvent")
    var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return  
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	logger.Printf(" processL3IntfDownEvent for  ipaddr %s mask %s\n", ipAddrStr, ipMaskStr)
   for i:=0;i<len(ConnectedRoutes);i++ {
	  if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr {
//      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex)){		
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

func processL3IntfUpEvent(ipAddr string){
	logger.Println("processL3IntfUpEvent")
    var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return  
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	logger.Printf(" processL3IntfUpEvent for  ipaddr %s mask %s\n", ipAddrStr, ipMaskStr)
   for i:=0;i<len(ConnectedRoutes);i++ {
	  if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr {
//      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex)){		
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

func processLinkDownEvent(ifType ribd.Int, ifIndex ribd.Int){
	logger.Println("processLinkDownEvent")
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
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				getConnectedRoutes()
				if(arpdclnt.IsConnected == true) {
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
				arpdclnt.ClientHdl = arpd.NewARPDServicesClientFactory(arpdclnt.Transport, arpdclnt.PtrProtocolFactory)
				arpdclnt.IsConnected = true
				if(asicdclnt.IsConnected == true) {
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
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
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
				arpdclnt.ClientHdl = arpd.NewARPDServicesClientFactory(arpdclnt.Transport, arpdclnt.PtrProtocolFactory)
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
	  Notif := asicdConstDefs.AsicdNotification {}
	  err = json.Unmarshal(rcvdMsg, &Notif)
	  if err != nil {
		logger.Println("Error in Unmarshalling rcvdMsg Json")
		return
	  }
      switch Notif.MsgType {
        case asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE:
		   logger.Println("NOTIFY_L3INTF_STATE_CHANGE event")
           var msg asicdConstDefs.L3IntfStateNotifyMsg
	       err = json.Unmarshal(Notif.Msg, &msg)
           if err != nil {
    	     logger.Println("Error in reading msg ", err)
		     return	
           }
		    logger.Printf("Msg linkstatus = %d msg ifType = %d ifId = %d\n", msg.IfState,msg.IfId)
		    if(msg.IfState == asicdConstDefs.INTF_STATE_DOWN) {
				//processLinkDownEvent(ribd.Int(msg.IfType), ribd.Int(msg.IfId))		
				processL3IntfDownEvent(msg.IpAddr)
			} else {
				//processLinkUpEvent(ribd.Int(msg.IfType), ribd.Int(msg.IfId))
				processL3IntfUpEvent(msg.IpAddr)
			}
			break
		case asicdConstDefs.NOTIFY_IPV4INTF_CREATE:
		   logger.Println("NOTIFY_IPV4INTF_CREATE event")
		   var msg asicdConstDefs.IPv4IntfNotifyMsg
	       err = json.Unmarshal(Notif.Msg, &msg)
           if err != nil {
    	     logger.Println("Error in reading msg ", err)
		     return	
           }
		   logger.Printf("Received ipv4 intf create with ipAddr %s ifType %d ifId %d\n", msg.IpAddr, msg.IfType, msg.IfId)
            var ipMask net.IP
			ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
		    if err != nil {
			   return  
		    }
		    ipMask = make(net.IP, 4)
		    copy(ipMask, ipNet.Mask)
		    ipAddrStr := ip.String()
		    ipMaskStr := net.IP(ipMask).String()
			logger.Printf("Calling createv4Route with ipaddr %s mask %s\n", ipAddrStr, ipMaskStr)
		   _,err = createV4Route(ipAddrStr,ipMaskStr, 0, "0.0.0.0", ribd.Int(msg.IfType), ribd.Int(msg.IfId), ribdCommonDefs.CONNECTED,  FIBAndRIB, ribd.Int(len(destNetSlice)))
		   if(err != nil) {
			  logger.Printf("Route create failed with err %s\n", err)
			  return 
		}
       }
	}
}
func processEvents(sub *nanomsg.SubSocket, subType ribd.Int) {
	logger.Println("in process events for sub ", subType)
	if(subType == SUB_ASICD){
		logger.Println("process Asicd events")
		processAsicdEvents(sub)
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
	BuildRouteProtocolTypeMapDB()
	RIBD_PUB = InitPublisher()
	go setupEventHandler(AsicdSub, asicdConstDefs.PUB_SOCKET_ADDR, SUB_ASICD)
	//CreateRoutes("RouteSetup.json")
	UpdateRoutesFromDB(paramsDir)
	return &RouteServiceHandler{}
}
