package main

import (
	"arpd"
	"asicdServices"
	//	"portdServices"
	"encoding/json"
	"l3/rib/ribdCommonDefs"
	"utils/policy"
	 "utils/policy/policyCommonDefs"
	"ribd"
	"utils/patriciaDB"
	//		"patricia"
	//	"errors"
	"asicd/asicdConstDefs"
	"utils/commonDefs"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/op/go-nanomsg"
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
	delAll
	invalidate
)
const (
	Invalid = -1
	FIBOnly = 0
	FIBAndRIB = 1
	RIBOnly = 2
)
const (
	SUB_PORTD = 0
	SUB_ASICD = 1
)

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

type localDB struct {
	prefix  patriciaDB.Prefix
	isValid bool
	precedence int
	nextHopIp string
}
type IntfEntry struct {
	name string
}
var asicdclnt AsicdClient
var arpdclnt ArpdClient
var count int
var ConnectedRoutes []*ribd.Routes
var acceptConfig bool
var AsicdSub *nanomsg.SubSocket
var RIBD_PUB *nanomsg.PubSocket
var RIBD_BGPD_PUB *nanomsg.PubSocket
var IntfIdNameMap map[int32]IntfEntry
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

func processL3IntfDownEvent(ipAddr string) {
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
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr {
			//      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex)){
			logger.Printf("Delete this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask)

/*			//Send a event
			msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo: *ConnectedRoutes[i]}
			msgbufbytes, err := json.Marshal(msgBuf)
			msg := ribdCommonDefs.RibdNotifyMsg{MsgType: ribdCommonDefs.NOTIFY_ROUTE_DELETED, MsgBuf: msgbufbytes}
			buf, err := json.Marshal(msg)
			if err != nil {
				logger.Println("Error in marshalling Json")
				return
			}
			logger.Println("buf", buf)
			RIBD_PUB.Send(buf, nanomsg.DontWait)
*/
			//Delete this route
			deleteV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask,"CONNECTED", ConnectedRoutes[i].NextHopIp,FIBOnly,ribdCommonDefs.RoutePolicyStateChangeNoChange)
		}
	}
}

func processL3IntfUpEvent(ipAddr string) {
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
	for i := 0; i < len(ConnectedRoutes); i++ {
		logger.Println("Current state of this connected route is ", ConnectedRoutes[i].IsValid)
		if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr && ConnectedRoutes[i].IsValid == false {
			//      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex)){
			logger.Printf("Add this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask)

			ConnectedRoutes[i].IsValid = true
	        policyRoute := ribd.Routes{Ipaddr: ConnectedRoutes[i].Ipaddr, Mask: ConnectedRoutes[i].Mask, NextHopIp: ConnectedRoutes[i].NextHopIp, NextHopIfType: ConnectedRoutes[i].NextHopIfType, IfIndex: ConnectedRoutes[i].IfIndex, Metric: ConnectedRoutes[i].Metric, Prototype: ConnectedRoutes[i].Prototype}
	        params := RouteParams{destNetIp:ConnectedRoutes[i].Ipaddr, networkMask:ConnectedRoutes[i].Mask, nextHopIp:ConnectedRoutes[i].NextHopIp, nextHopIfType:ConnectedRoutes[i].NextHopIfType, nextHopIfIndex:ConnectedRoutes[i].IfIndex, metric:ConnectedRoutes[i].Metric, routeType:ConnectedRoutes[i].Prototype,sliceIdx: ConnectedRoutes[i].SliceIdx, createType:FIBOnly, deleteType:Invalid}
	        PolicyEngineFilter(policyRoute, policyCommonDefs.PolicyPath_Import, params)
/*
			//Send a event
			msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo: *ConnectedRoutes[i]}
			msgbufbytes, err := json.Marshal(msgBuf)
			msg := ribdCommonDefs.RibdNotifyMsg{MsgType: ribdCommonDefs.NOTIFY_ROUTE_CREATED, MsgBuf: msgbufbytes}
			buf, err := json.Marshal(msg)
			if err != nil {
				logger.Println("Error in marshalling Json")
				return
			}
			logger.Println("buf", buf)
			RIBD_PUB.Send(buf, nanomsg.DontWait)
*/
			//Add this route
//			createV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, ConnectedRoutes[i].Metric, ConnectedRoutes[i].NextHopIp, ConnectedRoutes[i].NextHopIfType, ConnectedRoutes[i].IfIndex, ConnectedRoutes[i].Prototype, FIBOnly, ConnectedRoutes[i].SliceIdx)
		}
	}
}

func processLinkDownEvent(ifType ribd.Int, ifIndex ribd.Int) {
	logger.Println("processLinkDownEvent")
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex) {
			logger.Printf("Delete this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask)
			//Send a event
			msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo: *ConnectedRoutes[i]}
			msgbufbytes, err := json.Marshal(msgBuf)
			msg := ribdCommonDefs.RibdNotifyMsg{MsgType: ribdCommonDefs.NOTIFY_ROUTE_DELETED, MsgBuf: msgbufbytes}
			buf, err := json.Marshal(msg)
			if err != nil {
				logger.Println("Error in marshalling Json")
				return
			}
			logger.Println("buf", buf)
			RIBD_PUB.Send(buf, nanomsg.DontWait)

			//Delete this route
			deleteV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, "CONNECTED", ConnectedRoutes[i].NextHopIp,FIBOnly,ribdCommonDefs.RoutePolicyStateChangeNoChange)
		}
	}
}

func processLinkUpEvent(ifType ribd.Int, ifIndex ribd.Int) {
	logger.Println("processLinkUpEvent")
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex) && ConnectedRoutes[i].IsValid == false {
			logger.Printf("Add this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask)

			ConnectedRoutes[i].IsValid = true
			//Send a event
			msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo: *ConnectedRoutes[i]}
			msgbufbytes, err := json.Marshal(msgBuf)
			msg := ribdCommonDefs.RibdNotifyMsg{MsgType: ribdCommonDefs.NOTIFY_ROUTE_CREATED, MsgBuf: msgbufbytes}
			buf, err := json.Marshal(msg)
			if err != nil {
				logger.Println("Error in marshalling Json")
				return
			}
			logger.Println("buf", buf)
			RIBD_PUB.Send(buf, nanomsg.DontWait)

			//Add this route
			createV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, ConnectedRoutes[i].Metric, ConnectedRoutes[i].NextHopIp, ConnectedRoutes[i].NextHopIfType, ConnectedRoutes[i].IfIndex, ConnectedRoutes[i].Prototype, FIBOnly, ribdCommonDefs.RoutePolicyStateChangeNoChange,ConnectedRoutes[i].SliceIdx)
		}
	}
}

func (m RouteServiceHandler) LinkDown(ifType ribd.Int, ifIndex ribd.Int) (err error) {
	logger.Println("LinkDown")
	processLinkDownEvent(ifType, ifIndex)
	return nil
}

func (m RouteServiceHandler) LinkUp(ifType ribd.Int, ifIndex ribd.Int) (err error) {
	logger.Println("LinkUp")
	processLinkUpEvent(ifType, ifIndex)
	return nil
}

func (m RouteServiceHandler) IntfDown(ipAddr string) (err error) {
	logger.Println("IntfDown")
	processL3IntfDownEvent(ipAddr)
	return nil
}

func (m RouteServiceHandler) IntfUp(ipAddr string) (err error) {
	logger.Println("IntfUp")
	processL3IntfUpEvent(ipAddr)
	return nil
}

func getIntfInfo() {
	logger.Println("Getting intfs(ports) from asicd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		logger.Printf("Getting %d objects from currMarker %d\n", count, currMarker)
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(currMarker, count)
		if err != nil {
			logger.Println("GetBulkPortState with err ", err)
			return
		}
		if bulkInfo.Count == 0 {
			logger.Println("0 objects returned from GetBulkPortState")
			return
		}
		logger.Printf("len(bulkInfo.PortStateList)  = %d, num objects returned = %d\n", len(bulkInfo.PortStateList), bulkInfo.Count)
		for i := 0; i < int(bulkInfo.Count); i++ {
			portNum := bulkInfo.PortStateList[i].PortNum
			ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(portNum),commonDefs.L2RefTypePort)
             logger.Println("portNum = ", portNum, "ifId = ", ifId)
			if IntfIdNameMap==nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name:bulkInfo.PortStateList[i].Name}
			IntfIdNameMap[ifId] = intfEntry
		}
		if bulkInfo.More == false {
			logger.Println("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(bulkInfo.EndIdx)
	}
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
				getIntfInfo()
				if arpdclnt.IsConnected == true {
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
				if asicdclnt.IsConnected == true {
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
				getIntfInfo()
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
	logger.Println(" asicdConstDefs.NOTIFY_IPV4INTF_CREATE = ", asicdConstDefs.NOTIFY_IPV4INTF_CREATE, "asicdConstDefs.asicdConstDefs.NOTIFY_IPV4INTF_DELETE: ", asicdConstDefs.NOTIFY_IPV4INTF_DELETE)
	for {
		logger.Println("In for loop")
		rcvdMsg, err := sub.Recv(0)
		if err != nil {
			logger.Println("Error in receiving ", err)
			return
		}
		logger.Println("After recv rcvdMsg buf", rcvdMsg)
		Notif := asicdConstDefs.AsicdNotification{}
		err = json.Unmarshal(rcvdMsg, &Notif)
		if err != nil {
			logger.Println("Error in Unmarshalling rcvdMsg Json")
			return
		}
		switch Notif.MsgType {
		case asicdConstDefs.NOTIFY_VLAN_CREATE:
		 logger.Println("asicdConstDefs.NOTIFY_VLAN_CREATE")
		 var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
		 err = json.Unmarshal(Notif.Msg, &vlanNotifyMsg)
		 if err != nil {
			logger.Println("Unable to unmashal vlanNotifyMsg:", Notif.Msg)
			return
		}
		ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(vlanNotifyMsg.VlanId), commonDefs.L2RefTypeVlan)
		logger.Println("vlanId ", vlanNotifyMsg.VlanId, " ifId:", ifId)
		if IntfIdNameMap == nil {
			IntfIdNameMap = make(map[int32]IntfEntry)
		}
		intfEntry := IntfEntry{name:vlanNotifyMsg.VlanName}
		IntfIdNameMap[int32(ifId)] = intfEntry
		break
		case asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE:
			logger.Println("NOTIFY_L3INTF_STATE_CHANGE event")
			var msg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(Notif.Msg, &msg)
			if err != nil {
				logger.Println("Error in reading msg ", err)
				return
			}
			logger.Printf("Msg linkstatus = %d msg ifType = %d ifId = %d\n", msg.IfState, msg.IfIndex)
			if msg.IfState == asicdConstDefs.INTF_STATE_DOWN {
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
			logger.Printf("Received ipv4 intf create with ipAddr %s ifIndex = %d ifType %d ifId %d\n", msg.IpAddr, msg.IfIndex, asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex), asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex))
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
			_,err = routeServiceHandler.CreateV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex)), "CONNECTED")
			//_, err = createV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex)), ribdCommonDefs.CONNECTED, FIBAndRIB, ribdCommonDefs.RoutePolicyStateChangetoValid,ribd.Int(len(destNetSlice)))
			if err != nil {
				logger.Printf("Route create failed with err %s\n", err)
				return
			}
			break
		case asicdConstDefs.NOTIFY_IPV4INTF_DELETE:
		   logger.Println("NOTIFY_IPV4INTF_DELETE  event")
		   break
		}
	}
}
func processEvents(sub *nanomsg.SubSocket, subType ribd.Int) {
	logger.Println("in process events for sub ", subType)
	if subType == SUB_ASICD {
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
	if err != nil {
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
func InitPublisher(pub_str string) (pub *nanomsg.PubSocket) {
	logger.Println("Setting up %s", pub_str, "publisher")
	pub, err := nanomsg.NewPubSocket()
	if err != nil {
		logger.Println("Failed to open pub socket")
		return nil
	}
	ep, err := pub.Bind(pub_str)
	if err != nil {
		logger.Println("Failed to bind pub socket - ", ep)
		return nil
	}
	err = pub.SetSendBuffer(1024 * 1024)
	if err != nil {
		logger.Println("Failed to set send buffer size")
		return nil
	}
	return pub
}
func InitializePolicyDB() {
	PolicyEngineDB = policy.NewPolicyEngineDB()
	PolicyEngineDB.SetDefaultImportPolicyActionFunc(defaultImportPolicyEngineActionFunc)
	PolicyEngineDB.SetDefaultExportPolicyActionFunc(defaultExportPolicyEngineActionFunc)
	PolicyEngineDB.SetIsEntityPresentFunc(DoesRouteExist)
	PolicyEngineDB.SetEntityUpdateFunc(UpdateRouteAndPolicyDB)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineRouteDispositionAction)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionRedistribute)
    PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise,policyEngineActionNetworkStatementAdvertise)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionSetAdminDistance)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineUndoRouteDispositionAction)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionUndoRedistribute)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionUndoSetAdminDistance)
    PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise,policyEngineActionUndoNetworkStatemenAdvertiseAction)
	PolicyEngineDB.SetTraverseAndApplyPolicyFunc(policyEngineTraverseAndApply)
	PolicyEngineDB.SetTraverseAndReversePolicyFunc(policyEngineTraverseAndReverse)
	PolicyEngineDB.SetGetPolicyEntityMapIndexFunc(getPolicyRouteMapIndex)
}
func NewRouteServiceHandler(paramsDir string) *RouteServiceHandler {
	DummyRouteInfoRecord.protocol = PROTOCOL_NONE
	PARAMSDIR = paramsDir
	localRouteEventsDB = make([]RouteEventInfo,0)
	configFile := paramsDir + "/clients.json"
	logger.Println("configfile = ", configFile)
	ConnectToClients(configFile)
	BuildRouteProtocolTypeMapDB()
	BuildProtocolAdminDistanceMapDB()
	RIBD_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_ADDR)
	RIBD_BGPD_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_BGPD_ADDR)
	go setupEventHandler(AsicdSub, asicdConstDefs.PUB_SOCKET_ADDR, SUB_ASICD)
	//CreateRoutes("RouteSetup.json")
	InitializePolicyDB()
	UpdateFromDB()//(paramsDir)
	return &RouteServiceHandler{}
}