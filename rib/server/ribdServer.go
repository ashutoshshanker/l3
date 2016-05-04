package server

import (
	"arpd"
	"asicd/asicdCommonDefs"
	"asicdServices"
	//	"database/sql"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/garyburd/redigo/redis"
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
	"utils/logging"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)

type UpdateRouteInfo struct {
	OrigRoute *ribd.IPv4Route
	NewRoute  *ribd.IPv4Route
	Attrset   []bool
}
type TrackReachabilityInfo struct {
	IpAddr   string
	Protocol string
	Op       string
}
type NextHopInfoKey struct {
	nextHopIp string
}
type NextHopInfo struct {
	refCount int //number of routes using this as a next hop
}
type RIBDServer struct {
	Logger                       *logging.Writer
	PolicyEngineDB               *policy.PolicyEngineDB
	TrackReachabilityCh          chan TrackReachabilityInfo
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
	DbHdl                        redis.Conn
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
var logger *logging.Writer
var AsicdSub *nanomsg.SubSocket
var RouteServiceHandler *RIBDServer
var IntfIdNameMap map[int32]IntfEntry
var PolicyEngineDB *policy.PolicyEngineDB
var PARAMSDIR string

func (ribdServiceHandler *RIBDServer) ProcessL3IntfDownEvent(ipAddr string) {
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

func (ribdServiceHandler *RIBDServer) ProcessL3IntfUpEvent(ipAddr string) {
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
			ifId := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(portNum), commonDefs.IfTypePort)
			logger.Info(fmt.Sprintln("portNum = ", portNum, "ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.PortStateList[i].Name}
			IntfIdNameMap[ifId] = intfEntry
		}
		if bulkInfo.More == false {
			logger.Info(fmt.Sprintln("more returned as false, so no more get bulks"))
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
func (ribdServiceHandler *RIBDServer) AcceptConfigActions() {
	logger.Println("AcceptConfigActions: Setting AcceptConfig to true")
	RouteServiceHandler.AcceptConfig = true
	getIntfInfo()
	getConnectedRoutes()
	ribdServiceHandler.UpdateRoutesFromDB()
	go ribdServiceHandler.SetupEventHandler(AsicdSub, asicdCommonDefs.PUB_SOCKET_ADDR, SUB_ASICD)
	ribdServiceHandler.ServerUpCh <- true
}
func (ribdServiceHandler *RIBDServer) connectToClient(client ClientJson) {
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
					ribdServiceHandler.AcceptConfigActions()
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
					ribdServiceHandler.AcceptConfigActions()
				}
				timer.Stop()
				return
			}
		}
	}
}
func (ribdServiceHandler *RIBDServer) ConnectToClients(paramsFile string) {
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
					ribdServiceHandler.AcceptConfigActions()
				}
			} else {
				go ribdServiceHandler.connectToClient(client)
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
					ribdServiceHandler.AcceptConfigActions()
				}
			} else {
				go ribdServiceHandler.connectToClient(client)
			}
		}
	}
}

func (ribdServiceHandler *RIBDServer) InitializePolicyDB() *policy.PolicyEngineDB {
	ribdServiceHandler.PolicyEngineDB = policy.NewPolicyEngineDB(logger)
	ribdServiceHandler.PolicyEngineDB.SetDefaultImportPolicyActionFunc(defaultImportPolicyEngineActionFunc)
	ribdServiceHandler.PolicyEngineDB.SetDefaultExportPolicyActionFunc(defaultExportPolicyEngineActionFunc)
	ribdServiceHandler.PolicyEngineDB.SetIsEntityPresentFunc(DoesRouteExist)
	ribdServiceHandler.PolicyEngineDB.SetEntityUpdateFunc(UpdateRouteAndPolicyDB)
	ribdServiceHandler.PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineRouteDispositionAction)
	ribdServiceHandler.PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionRedistribute)
	ribdServiceHandler.PolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise, policyEngineActionNetworkStatementAdvertise)
	ribdServiceHandler.PolicyEngineDB.SetActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionSetAdminDistance)
	ribdServiceHandler.PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineUndoRouteDispositionAction)
	ribdServiceHandler.PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionUndoRedistribute)
	ribdServiceHandler.PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionUndoSetAdminDistance)
	ribdServiceHandler.PolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise, policyEngineActionUndoNetworkStatemenAdvertiseAction)
	ribdServiceHandler.PolicyEngineDB.SetTraverseAndApplyPolicyFunc(policyEngineTraverseAndApply)
	ribdServiceHandler.PolicyEngineDB.SetTraverseAndReversePolicyFunc(policyEngineTraverseAndReverse)
	ribdServiceHandler.PolicyEngineDB.SetGetPolicyEntityMapIndexFunc(getPolicyRouteMapIndex)
	return ribdServiceHandler.PolicyEngineDB
}
func NewRIBDServicesHandler(dbHdl redis.Conn, loggerC *logging.Writer) *RIBDServer {
	fmt.Println("NewRIBDServicesHandler")
	RouteInfoMap = patriciaDB.NewTrie()
	ribdServicesHandler := &RIBDServer{}
	ribdServicesHandler.Logger = loggerC
	logger = loggerC
	localRouteEventsDB = make([]RouteEventInfo, 0)
	RedistributeRouteMap = make(map[string][]RedistributeRouteInfo)
	TrackReachabilityMap = make(map[string][]string)
	RouteProtocolTypeMapDB = make(map[string]int)
	ReverseRouteProtoTypeMapDB = make(map[int]string)
	ProtocolAdminDistanceMapDB = make(map[string]RouteDistanceConfig)
	PublisherInfoMap = make(map[string]PublisherMapInfo)
	ribdServicesHandler.NextHopInfoMap = make(map[NextHopInfoKey]NextHopInfo)
	ribdServicesHandler.TrackReachabilityCh = make(chan TrackReachabilityInfo, 1000)
	ribdServicesHandler.RouteCreateConfCh = make(chan *ribd.IPv4Route, 5000)
	ribdServicesHandler.RouteDeleteConfCh = make(chan *ribd.IPv4Route)
	ribdServicesHandler.RouteUpdateConfCh = make(chan UpdateRouteInfo)
	ribdServicesHandler.NetlinkAddRouteCh = make(chan RouteInfoRecord, 5000)
	ribdServicesHandler.NetlinkDelRouteCh = make(chan RouteInfoRecord, 100)
	ribdServicesHandler.AsicdAddRouteCh = make(chan RouteInfoRecord, 5000)
	ribdServicesHandler.AsicdDelRouteCh = make(chan RouteInfoRecord, 1000)
	ribdServicesHandler.ArpdResolveRouteCh = make(chan RouteInfoRecord, 5000)
	ribdServicesHandler.ArpdRemoveRouteCh = make(chan RouteInfoRecord, 1000)
	ribdServicesHandler.NotificationChannel = make(chan NotificationMsg, 5000)
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
	RouteServiceHandler = ribdServicesHandler
	//ribdServicesHandler.RouteInstallCh = make(chan RouteParams)
	BuildRouteProtocolTypeMapDB()
	BuildProtocolAdminDistanceMapDB()
	BuildPublisherMap()
	PolicyEngineDB = ribdServicesHandler.InitializePolicyDB()
	return ribdServicesHandler
}
func (ribdServiceHandler *RIBDServer) StartServer(paramsDir string) {
	fmt.Println("StartServer")
	DummyRouteInfoRecord.protocol = PROTOCOL_NONE
	configFile := paramsDir + "/clients.json"
	logger.Info(fmt.Sprintln("configfile = ", configFile))
	PARAMSDIR = paramsDir
	//RIBD_BGPD_PUB = InitPublisher(ribdCommonDefs.PUB_SOCKET_BGPD_ADDR)
	//CreateRoutes("RouteSetup.json")
	ribdServiceHandler.UpdatePolicyObjectsFromDB() //(paramsDir)
	ribdServiceHandler.ConnectToClients(configFile)
	logger.Println("Starting the server loop")
	for {
		if !RouteServiceHandler.AcceptConfig {
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
			ribdServiceHandler.ProcessRouteUpdateConfig(routeUpdateConf.OrigRoute, routeUpdateConf.NewRoute, routeUpdateConf.Attrset)
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
		case info := <-ribdServiceHandler.TrackReachabilityCh:
			logger.Info("received message on TrackReachabilityCh channel")
			ribdServiceHandler.TrackReachabilityStatus(info.IpAddr, info.Protocol, info.Op)
		}
	}
}
