package main

import (
	"arpd"
	"asicd/asicdConstDefs"
	"asicdInt"
	"asicdServices"
	"database/sql"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/op/go-nanomsg"
	"io/ioutil"
	"l3/rib/ribdCommonDefs"
	"net"
	"ribd"
	"ribdInt"
	"strconv"
	"time"
	"utils/commonDefs"
	"utils/ipcutils"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)

type UpdateRouteInfo struct {
	origRoute *ribd.IPv4Route
	newRoute  *ribd.IPv4Route
	attrset   []bool
}
type NextHopInfoKey struct {
	nextHopIp string
}
type NextHopInfo struct {
	refCount     int     //number of routes using this as a next hop
}
type RIBDServicesHandler struct {
	RouteCreateConfCh            chan *ribd.IPv4Route
	RouteDeleteConfCh            chan *ribd.IPv4Route
	RouteUpdateConfCh            chan UpdateRouteInfo
	NetlinkAddRouteCh            chan RouteInfoRecord
	NetlinkDelRouteCh            chan RouteInfoRecord
	AsicdAddRouteCh              chan RouteInfoRecord
	AsicdDelRouteCh              chan RouteInfoRecord
	ArpdResolveRouteCh           chan RouteInfoRecord
	ArpdRemoveRouteCh            chan RouteInfoRecord
	NotificationChannel          chan NotificationMsg
	NextHopInfoMap               map[NextHopInfoKey]NextHopInfo
	PolicyConditionCreateConfCh  chan *ribd.PolicyCondition
	PolicyConditionDeleteConfCh  chan *ribd.PolicyCondition
	PolicyConditionUpdateConfCh  chan *ribd.PolicyCondition
	PolicyActionCreateConfCh     chan *ribd.PolicyAction
	PolicyActionDeleteConfCh     chan *ribd.PolicyAction
	PolicyActionUpdateConfCh     chan *ribd.PolicyAction
	PolicyStmtCreateConfCh       chan *ribd.PolicyStmt
	PolicyStmtDeleteConfCh       chan *ribd.PolicyStmt
	PolicyStmtUpdateConfCh       chan *ribd.PolicyStmt
	PolicyDefinitionCreateConfCh chan *ribd.PolicyDefinition
	PolicyDefinitionDeleteConfCh chan *ribd.PolicyDefinition
	PolicyDefinitionUpdateConfCh chan *ribd.PolicyDefinition
	AcceptConfig                 bool
	ServerUpCh                   chan bool
	DbHdl                        *sql.DB
	//RouteInstallCh                 chan RouteParams
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
	Invalid   = -1
	FIBOnly   = 0
	FIBAndRIB = 1
	RIBOnly   = 2
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
	prefix     patriciaDB.Prefix
	isValid    bool
	precedence int
	nextHopIp  string
}
type IntfEntry struct {
	name string
}

var asicdclnt AsicdClient
var arpdclnt ArpdClient
var count int
var ConnectedRoutes []*ribdInt.Routes
var AsicdSub *nanomsg.SubSocket
var RIBD_PUB *nanomsg.PubSocket

//var RIBD_BGPD_PUB *nanomsg.PubSocket
var IntfIdNameMap map[int32]IntfEntry

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
	logger.Info(fmt.Sprintln(" processL3IntfDownEvent for  ipaddr %s mask %s\n", ipAddrStr, ipMaskStr))
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr {
			logger.Info(fmt.Sprintln("Delete this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask))
			deleteV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, "CONNECTED", ConnectedRoutes[i].NextHopIp, FIBOnly, ribdCommonDefs.RoutePolicyStateChangeNoChange)
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
	logger.Info(fmt.Sprintln(" processL3IntfUpEvent for  ipaddr %s mask %s\n", ipAddrStr, ipMaskStr))
	for i := 0; i < len(ConnectedRoutes); i++ {
		logger.Info(fmt.Sprintln("Current state of this connected route is ", ConnectedRoutes[i].IsValid))
		if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr && ConnectedRoutes[i].IsValid == false {
			//      if(ConnectedRoutes[i].NextHopIfType == ribd.Int(ifType) && ConnectedRoutes[i].IfIndex == ribd.Int(ifIndex)){
			logger.Info(fmt.Sprintln("Add this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask))

			ConnectedRoutes[i].IsValid = true
			policyRoute := ribdInt.Routes{Ipaddr: ConnectedRoutes[i].Ipaddr, Mask: ConnectedRoutes[i].Mask, NextHopIp: ConnectedRoutes[i].NextHopIp, NextHopIfType: ConnectedRoutes[i].NextHopIfType, IfIndex: ConnectedRoutes[i].IfIndex, Metric: ConnectedRoutes[i].Metric, Prototype: ConnectedRoutes[i].Prototype}
			params := RouteParams{destNetIp: ConnectedRoutes[i].Ipaddr, networkMask: ConnectedRoutes[i].Mask, nextHopIp: ConnectedRoutes[i].NextHopIp, nextHopIfType: ribd.Int(ConnectedRoutes[i].NextHopIfType), nextHopIfIndex: ribd.Int(ConnectedRoutes[i].IfIndex), metric: ribd.Int(ConnectedRoutes[i].Metric), routeType: ribd.Int(ConnectedRoutes[i].Prototype), sliceIdx: ribd.Int(ConnectedRoutes[i].SliceIdx), createType: FIBOnly, deleteType: Invalid}
			PolicyEngineFilter(policyRoute, policyCommonDefs.PolicyPath_Import, params)
		}
	}
}

func processLinkDownEvent(ifType ribd.Int, ifIndex ribd.Int) {
	logger.Println("processLinkDownEvent")
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].NextHopIfType == ribdInt.Int(ifType) && ConnectedRoutes[i].IfIndex == ribdInt.Int(ifIndex) {
			logger.Info(fmt.Sprintln("Delete this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask))
			//Send a event
			msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo: *ConnectedRoutes[i]}
			msgbufbytes, err := json.Marshal(msgBuf)
			msg := ribdCommonDefs.RibdNotifyMsg{MsgType: ribdCommonDefs.NOTIFY_ROUTE_DELETED, MsgBuf: msgbufbytes}
			buf, err := json.Marshal(msg)
			if err != nil {
				logger.Println("Error in marshalling Json")
				return
			}
			logger.Info(fmt.Sprintln("buf", buf))
			RIBD_PUB.Send(buf, nanomsg.DontWait)

			//Delete this route
			deleteV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, "CONNECTED", ConnectedRoutes[i].NextHopIp, FIBOnly, ribdCommonDefs.RoutePolicyStateChangeNoChange)
		}
	}
}

func processLinkUpEvent(ifType ribd.Int, ifIndex ribd.Int) {
	logger.Println("processLinkUpEvent")
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].NextHopIfType == ribdInt.Int(ifType) && ConnectedRoutes[i].IfIndex == ribdInt.Int(ifIndex) && ConnectedRoutes[i].IsValid == false {
			logger.Info(fmt.Sprintln("Add this route with destAddress = %s, nwMask = %s\n", ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask))

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
			logger.Info(fmt.Sprintln("buf", buf))
			RIBD_PUB.Send(buf, nanomsg.DontWait)

			//Add this route - should call install
			createV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, ribd.Int(ConnectedRoutes[i].Metric), ConnectedRoutes[i].NextHopIp, ribd.Int(ConnectedRoutes[i].NextHopIfType), ribd.Int(ConnectedRoutes[i].IfIndex), ribd.Int(ConnectedRoutes[i].Prototype), FIBOnly, ribdCommonDefs.RoutePolicyStateChangeNoChange, ribd.Int(ConnectedRoutes[i].SliceIdx))
		}
	}
}

func (m RIBDServicesHandler) LinkDown(ifType ribdInt.Int, ifIndex ribdInt.Int) (err error) {
	logger.Println("LinkDown")
	processLinkDownEvent(ribd.Int(ifType), ribd.Int(ifIndex))
	return nil
}

func (m RIBDServicesHandler) LinkUp(ifType ribdInt.Int, ifIndex ribdInt.Int) (err error) {
	logger.Println("LinkUp")
	processLinkUpEvent(ribd.Int(ifType), ribd.Int(ifIndex))
	return nil
}

func (m RIBDServicesHandler) IntfDown(ipAddr string) (err error) {
	logger.Println("IntfDown")
	processL3IntfDownEvent(ipAddr)
	return nil
}

func (m RIBDServicesHandler) IntfUp(ipAddr string) (err error) {
	logger.Println("IntfUp")
	processL3IntfUpEvent(ipAddr)
	return nil
}

//used at init time after connecting to ASICD so as to install any routes (static) read from DB in ASICd
func installRoutesInASIC() {
	logger.Println("installRoutesinASIC")
	if destNetSlice == nil {
		logger.Println("No routes installed in RIB")
		return
	}
	for i := 0; i < len(destNetSlice); i++ {
		if destNetSlice[i].isValid == false {
			logger.Println("Invalid route")
			continue
		}
		prefixNode := RouteInfoMap.Get(destNetSlice[i].prefix)
		if prefixNode != nil {
			prefixNodeRouteList := prefixNode.(RouteInfoRecordList)
			if prefixNodeRouteList.routeInfoProtocolMap == nil || prefixNodeRouteList.selectedRouteProtocol == "INVALID" || prefixNodeRouteList.routeInfoProtocolMap[prefixNodeRouteList.selectedRouteProtocol] == nil {
				logger.Println("selected route not valid")
				continue
			}
			routeInfoList := prefixNodeRouteList.routeInfoProtocolMap[prefixNodeRouteList.selectedRouteProtocol]
			for sel := 0; sel < len(routeInfoList); sel++ {
				routeInfoRecord := routeInfoList[sel]
				asicdclnt.ClientHdl.OnewayCreateIPv4Route([]*asicdInt.IPv4Route{
					&asicdInt.IPv4Route{
						routeInfoRecord.destNetIp.String(),
						routeInfoRecord.networkMask.String(),
						routeInfoRecord.nextHopIp.String(),
						int32(routeInfoRecord.nextHopIfType),
					},
				})
			}
		}

	}
}
func getLogicalIntfInfo() {
	logger.Println("Getting Logical Interfaces from asicd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		logger.Info(fmt.Sprintln("Getting ", count, "GetBulkLogicalIntf objects from currMarker:", currMarker))
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkLogicalIntfState(currMarker, count)
		if err != nil {
			logger.Info(fmt.Sprintln("GetBulkLogicalIntfState with err ", err))
			return
		}
		if bulkInfo.Count == 0 {
			logger.Println("0 objects returned from GetBulkLogicalIntfState")
			return
		}
		logger.Info(fmt.Sprintln("len(bulkInfo.GetBulkLogicalIntfState)  = %d, num objects returned = %d\n", len(bulkInfo.LogicalIntfStateList), bulkInfo.Count))
		for i := 0; i < int(bulkInfo.Count); i++ {
			ifId := (bulkInfo.LogicalIntfStateList[i].IfIndex)
			logger.Info(fmt.Sprintln("logical interface = ", bulkInfo.LogicalIntfStateList[i].Name, "ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.LogicalIntfStateList[i].Name}
			IntfIdNameMap[ifId] = intfEntry
		}
		if bulkInfo.More == false {
			logger.Println("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(bulkInfo.EndIdx)
	}
}
func getVlanInfo() {
	logger.Println("Getting vlans from asicd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		logger.Info(fmt.Sprintln("Getting ", count, "GetBulkVlan objects from currMarker:", currMarker))
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkVlanState(currMarker, count)
		if err != nil {
			logger.Info(fmt.Sprintln("GetBulkVlan with err ", err))
			return
		}
		if bulkInfo.Count == 0 {
			logger.Println("0 objects returned from GetBulkVlan")
			return
		}
		logger.Info(fmt.Sprintln("len(bulkInfo.GetBulkVlan)  = %d, num objects returned = %d\n", len(bulkInfo.VlanStateList), bulkInfo.Count))
		for i := 0; i < int(bulkInfo.Count); i++ {
			ifId := (bulkInfo.VlanStateList[i].IfIndex)
			logger.Info(fmt.Sprintln("vlan = ", bulkInfo.VlanStateList[i].VlanId, "ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.VlanStateList[i].VlanName}
			IntfIdNameMap[ifId] = intfEntry
		}
		if bulkInfo.More == false {
			logger.Println("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(bulkInfo.EndIdx)
	}
}
func getPortInfo() {
	logger.Println("Getting ports from asicd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		logger.Info(fmt.Sprintln("Getting ", count, "objects from currMarker:", currMarker))
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(currMarker, count)
		if err != nil {
			logger.Info(fmt.Sprintln("GetBulkPortState with err ", err))
			return
		}
		if bulkInfo.Count == 0 {
			logger.Println("0 objects returned from GetBulkPortState")
			return
		}
		logger.Info(fmt.Sprintln("len(bulkInfo.PortStateList)  = %d, num objects returned = %d\n", len(bulkInfo.PortStateList), bulkInfo.Count))
		for i := 0; i < int(bulkInfo.Count); i++ {
			portNum := bulkInfo.PortStateList[i].PortNum
			ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(portNum), commonDefs.IfTypePort)
			logger.Info(fmt.Sprintln("portNum = ", portNum, "ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.PortStateList[i].Name}
			IntfIdNameMap[ifId] = intfEntry
		}
		if bulkInfo.More == false {
			logger.Println("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(bulkInfo.EndIdx)
	}
}
func getIntfInfo() {
	getPortInfo()
	getVlanInfo()
	getLogicalIntfInfo()
}
func AcceptConfigActions() {
	logger.Println("AcceptConfigActions: Setting AcceptConfig to true")
	routeServiceHandler.AcceptConfig = true
	getIntfInfo()
	getConnectedRoutes()
	UpdateRoutesFromDB()
	go SetupEventHandler(AsicdSub, asicdConstDefs.PUB_SOCKET_ADDR, SUB_ASICD)
	routeServiceHandler.ServerUpCh <- true
}
func connectToClient(client ClientJson) {
	var timer *time.Timer
	logger.Info(fmt.Sprintln("in go routine ConnectToClient for connecting to %s\n", client.Name))
	for {
		timer = time.NewTimer(time.Second * 10)
		<-timer.C
		if client.Name == "asicd" {
			logger.Info(fmt.Sprintln("found asicd at port %d", client.Port))
			asicdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				logger.Info(fmt.Sprintln("connecting to asicd,arpdclnt.IsConnected:", arpdclnt.IsConnected))
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				if arpdclnt.IsConnected == true {
					logger.Info(fmt.Sprintln(" Connected to all clients: call AcceptConfigActions"))
					AcceptConfigActions()
				}
				timer.Stop()
				return
			}
		}
		if client.Name == "arpd" {
			logger.Info(fmt.Sprintln("found arpd at port %d", client.Port))
			arpdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			arpdclnt.Transport, arpdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(arpdclnt.Address)
			if arpdclnt.Transport != nil && arpdclnt.PtrProtocolFactory != nil {
				logger.Info(fmt.Sprintln("connecting to arpd,asicdclnt.IsConnected:", asicdclnt.IsConnected))
				arpdclnt.ClientHdl = arpd.NewARPDServicesClientFactory(arpdclnt.Transport, arpdclnt.PtrProtocolFactory)
				arpdclnt.IsConnected = true
				if asicdclnt.IsConnected == true {
					logger.Info(fmt.Sprintln(" Connected to all clients: call AcceptConfigActions"))
					AcceptConfigActions()
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
		logger.Info(fmt.Sprintln("#### Client name is ", client.Name))
		if client.Name == "asicd" {
			logger.Info(fmt.Sprintln("found asicd at port %d", client.Port))
			asicdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				logger.Info(fmt.Sprintln("connecting to asicd,arpdclnt.IsConnected:", arpdclnt.IsConnected))
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				if arpdclnt.IsConnected == true {
					logger.Info(fmt.Sprintln(" Connected to all clients: call AcceptConfigActions"))
					AcceptConfigActions()
				}
			} else {
				go connectToClient(client)
			}
		}
		if client.Name == "arpd" {
			logger.Info(fmt.Sprintln("found arpd at port %d", client.Port))
			arpdclnt.Address = "localhost:" + strconv.Itoa(client.Port)
			arpdclnt.Transport, arpdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(arpdclnt.Address)
			if arpdclnt.Transport != nil && arpdclnt.PtrProtocolFactory != nil {
				logger.Info(fmt.Sprintln("connecting to arpd,asicdclnt.IsConnected:", asicdclnt.IsConnected))
				arpdclnt.ClientHdl = arpd.NewARPDServicesClientFactory(arpdclnt.Transport, arpdclnt.PtrProtocolFactory)
				arpdclnt.IsConnected = true
				if asicdclnt.IsConnected == true {
					logger.Info(fmt.Sprintln(" Connected to all clients: call AcceptConfigActions"))
					AcceptConfigActions()
				}
			} else {
				go connectToClient(client)
			}
		}
	}
}

func InitPublisher(pub_str string) (pub *nanomsg.PubSocket) {
	logger.Info(fmt.Sprintln("Setting up %s", pub_str, "publisher"))
	pub, err := nanomsg.NewPubSocket()
	if err != nil {
		logger.Println("Failed to open pub socket")
		return nil
	}
	ep, err := pub.Bind(pub_str)
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to bind pub socket - ", ep))
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
	PolicyEngineDB = policy.NewPolicyEngineDB(logger)
	PolicyEngineDB.SetDefaultImportPolicyActionFunc(defaultImportPolicyEngineActionFunc)
	PolicyEngineDB.SetDefaultExportPolicyActionFunc(defaultExportPolicyEngineActionFunc)
	PolicyEngineDB.SetIsEntityPresentFunc(DoesRouteExist)
	PolicyEngineDB.SetEntityUpdateFunc(UpdateRouteAndPolicyDB)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineRouteDispositionAction)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionRedistribute)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise, policyEngineActionNetworkStatementAdvertise)
	PolicyEngineDB.SetActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionSetAdminDistance)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineUndoRouteDispositionAction)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionUndoRedistribute)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionUndoSetAdminDistance)
	PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise, policyEngineActionUndoNetworkStatemenAdvertiseAction)
	PolicyEngineDB.SetTraverseAndApplyPolicyFunc(policyEngineTraverseAndApply)
	PolicyEngineDB.SetTraverseAndReversePolicyFunc(policyEngineTraverseAndReverse)
	PolicyEngineDB.SetGetPolicyEntityMapIndexFunc(getPolicyRouteMapIndex)
}
func NewRIBDServicesHandler(dbHdl *sql.DB) *RIBDServicesHandler {
	RouteInfoMap = patriciaDB.NewTrie()
	ribdServicesHandler := &RIBDServicesHandler{}
	RedistributeRouteMap = make(map[string][]RedistributeRouteInfo)
	TrackReachabilityMap = make(map[string][]string)
	RouteProtocolTypeMapDB = make(map[string]int)
	ReverseRouteProtoTypeMapDB = make(map[int]string)
	ProtocolAdminDistanceMapDB = make(map[string]RouteDistanceConfig)
	PublisherInfoMap = make(map[string]PublisherMapInfo)
	ribdServicesHandler.NextHopInfoMap = make(map[NextHopInfoKey]NextHopInfo)
	ribdServicesHandler.RouteCreateConfCh = make(chan *ribd.IPv4Route,5000)
	ribdServicesHandler.RouteDeleteConfCh = make(chan *ribd.IPv4Route)
	ribdServicesHandler.RouteUpdateConfCh = make(chan UpdateRouteInfo)
	ribdServicesHandler.NetlinkAddRouteCh = make(chan RouteInfoRecord,5000)
	ribdServicesHandler.NetlinkDelRouteCh = make(chan RouteInfoRecord,100)
	ribdServicesHandler.AsicdAddRouteCh = make(chan RouteInfoRecord,5000)
	ribdServicesHandler.AsicdDelRouteCh = make(chan RouteInfoRecord,1000)
	ribdServicesHandler.ArpdResolveRouteCh = make(chan RouteInfoRecord,5000)
	ribdServicesHandler.ArpdRemoveRouteCh = make(chan RouteInfoRecord,1000)
	ribdServicesHandler.NotificationChannel = make(chan NotificationMsg,5000)
	ribdServicesHandler.PolicyConditionCreateConfCh = make(chan *ribd.PolicyCondition)
	ribdServicesHandler.PolicyConditionDeleteConfCh = make(chan *ribd.PolicyCondition)
	ribdServicesHandler.PolicyConditionUpdateConfCh = make(chan *ribd.PolicyCondition)
	ribdServicesHandler.PolicyActionCreateConfCh = make(chan *ribd.PolicyAction)
	ribdServicesHandler.PolicyActionDeleteConfCh = make(chan *ribd.PolicyAction)
	ribdServicesHandler.PolicyActionUpdateConfCh = make(chan *ribd.PolicyAction)
	ribdServicesHandler.PolicyStmtCreateConfCh = make(chan *ribd.PolicyStmt)
	ribdServicesHandler.PolicyStmtDeleteConfCh = make(chan *ribd.PolicyStmt)
	ribdServicesHandler.PolicyStmtUpdateConfCh = make(chan *ribd.PolicyStmt)
	ribdServicesHandler.PolicyDefinitionCreateConfCh = make(chan *ribd.PolicyDefinition)
	ribdServicesHandler.PolicyDefinitionDeleteConfCh = make(chan *ribd.PolicyDefinition)
	ribdServicesHandler.PolicyDefinitionUpdateConfCh = make(chan *ribd.PolicyDefinition)
	ribdServicesHandler.ServerUpCh = make(chan bool)
	ribdServicesHandler.DbHdl = dbHdl
	//ribdServicesHandler.RouteInstallCh = make(chan RouteParams)
	return ribdServicesHandler
}
func (ribdServiceHandler *RIBDServicesHandler) StartServer(paramsDir string) {
	DummyRouteInfoRecord.protocol = PROTOCOL_NONE
	PARAMSDIR = paramsDir
	localRouteEventsDB = make([]RouteEventInfo, 0)
	configFile := paramsDir + "/clients.json"
	logger.Info(fmt.Sprintln("configfile = ", configFile))
	BuildRouteProtocolTypeMapDB()
	BuildProtocolAdminDistanceMapDB()
	BuildPublisherMap()
	//RIBD_BGPD_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_BGPD_ADDR)
	//CreateRoutes("RouteSetup.json")
	InitializePolicyDB()
	UpdatePolicyObjectsFromDB() //(paramsDir)
	ConnectToClients(configFile)
	logger.Println("Starting the server loop")
	for {
		if !routeServiceHandler.AcceptConfig {
			logger.Println("Not ready to accept config")
			continue
		}
		select {
		case routeCreateConf := <-ribdServiceHandler.RouteCreateConfCh:
			logger.Info("received message on RouteCreateConfCh channel")
			ribdServiceHandler.ProcessRouteCreateConfig(routeCreateConf)
		case routeDeleteConf := <-ribdServiceHandler.RouteDeleteConfCh:
			logger.Info("received message on RouteDeleteConfCh channel")
			ribdServiceHandler.ProcessRouteDeleteConfig(routeDeleteConf)
		case routeUpdateConf := <-ribdServiceHandler.RouteUpdateConfCh:
			logger.Info("received message on RouteUpdateConfCh channel")
			ribdServiceHandler.ProcessRouteUpdateConfig(routeUpdateConf.origRoute, routeUpdateConf.newRoute, routeUpdateConf.attrset)
			/*		case routeInfo := <-ribdServiceHandler.RouteInstallCh:
			    logger.Println("received message on RouteInstallConfCh channel")
				ribdServiceHandler.ProcessRouteInstall(routeInfo)*/
		case condCreateConf := <-ribdServiceHandler.PolicyConditionCreateConfCh:
			logger.Info("received message on PolicyConditionCreateConfCh channel")
			ribdServiceHandler.ProcessPolicyConditionConfigCreate(condCreateConf)
		case condDeleteConf := <-ribdServiceHandler.PolicyConditionDeleteConfCh:
			logger.Info("received message on PolicyConditionDeleteConfCh channel")
			ribdServiceHandler.ProcessPolicyConditionConfigDelete(condDeleteConf)
		case actionCreateConf := <-ribdServiceHandler.PolicyActionCreateConfCh:
			logger.Info("received message on PolicyActionCreateConfCh channel")
			ribdServiceHandler.ProcessPolicyActionConfigCreate(actionCreateConf)
		case actionDeleteConf := <-ribdServiceHandler.PolicyActionDeleteConfCh:
			logger.Info("received message on PolicyActionDeleteConfCh channel")
			ribdServiceHandler.ProcessPolicyActionConfigDelete(actionDeleteConf)
		case stmtCreateConf := <-ribdServiceHandler.PolicyStmtCreateConfCh:
			logger.Info("received message on PolicyStmtCreateConfCh channel")
			ribdServiceHandler.ProcessPolicyStmtConfigCreate(stmtCreateConf)
		case stmtDeleteConf := <-ribdServiceHandler.PolicyStmtDeleteConfCh:
			logger.Info("received message on PolicyStmtDeleteConfCh channel")
			ribdServiceHandler.ProcessPolicyStmtConfigDelete(stmtDeleteConf)
		case policyCreateConf := <-ribdServiceHandler.PolicyDefinitionCreateConfCh:
			logger.Info("received message on PolicyDefinitionCreateConfCh channel")
			ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(policyCreateConf)
		case policyDeleteConf := <-ribdServiceHandler.PolicyDefinitionDeleteConfCh:
			logger.Info("received message on PolicyDefinitionDeleteConfCh channel")
			ribdServiceHandler.ProcessPolicyDefinitionConfigDelete(policyDeleteConf)
		}
	}
}
