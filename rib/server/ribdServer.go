//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//

package server

import (
	"arpd"
	"asicd/asicdCommonDefs"
	"asicdServices"
	//	"database/sql"
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
	"utils/dbutils"
	"utils/ipcutils"
	"utils/logging"
	"utils/patriciaDB"
	"utils/policy"
	"utils/policy/policyCommonDefs"
)

type RouteConfigInfo struct {
	OrigRoute *ribd.IPv4Route
	NewRoute  *ribd.IPv4Route
	Attrset   []bool
	Op        string   //"add"/"del"/"update"
}
type RIBdServerConfig struct {
	OrigConfigObject interface{}
	NewConfigObject  interface{}
	AttrSet          []bool
	Op               string   //"add"/"del"/"update"
	PatchOp          []*ribd.PatchOpInfo
}
/*type PatchUpdateRouteInfo struct {
	OrigRoute *ribd.IPv4Route
	NewRoute  *ribd.IPv4Route
	Op        []*ribd.PatchOpInfo
}*/
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
type ApplyPolicyInfo struct {
	Source     string
	Policy     string
	Action     string
	Conditions []*ribdInt.ConditionInfo
}
type RIBDServer struct {
	Logger                       *logging.Writer
	PolicyEngineDB               *policy.PolicyEngineDB
	GlobalPolicyEngineDB         *policy.PolicyEngineDB
	TrackReachabilityCh          chan TrackReachabilityInfo
	RouteConfCh                  chan RIBdServerConfig
	AsicdRouteCh                 chan RIBdServerConfig
	ArpdRouteCh                  chan RIBdServerConfig
	NotificationChannel          chan NotificationMsg
	NextHopInfoMap               map[NextHopInfoKey]NextHopInfo
	PolicyConditionConfCh        chan RIBdServerConfig
	PolicyActionConfCh           chan RIBdServerConfig
	PolicyStmtConfCh             chan RIBdServerConfig
	PolicyDefinitionConfCh       chan RIBdServerConfig
	PolicyApplyCh                chan ApplyPolicyInfo
	PolicyUpdateApplyCh          chan ApplyPolicyInfo
	DBRouteCh                    chan RIBdServerConfig
	AcceptConfig                 bool
	ServerUpCh                   chan bool
	DbHdl                        *dbutils.DBUtil
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
	SUB_ASICD = 0
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
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
var IfNameToIfIndex map[string]int32
var GlobalPolicyEngineDB *policy.PolicyEngineDB
var PolicyEngineDB *policy.PolicyEngineDB
var PARAMSDIR string

/*
    Handle Interface down event
*/
func (ribdServiceHandler *RIBDServer) ProcessL3IntfDownEvent(ipAddr string) {
	logger.Debug("processL3IntfDownEvent")
	var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	logger.Info(fmt.Sprintln(" processL3IntfDownEvent for  ipaddr ", ipAddrStr, " mask ", ipMaskStr))
	for i := 0; i < len(ConnectedRoutes); i++ {
		if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr {
			logger.Info(fmt.Sprintln("Delete this route with destAddress = ",ConnectedRoutes[i].Ipaddr," nwMask = ", ConnectedRoutes[i].Mask))
			deleteV4Route(ConnectedRoutes[i].Ipaddr, ConnectedRoutes[i].Mask, "CONNECTED", ConnectedRoutes[i].NextHopIp, FIBOnly, ribdCommonDefs.RoutePolicyStateChangeNoChange)
		}
	}
}

/*
    Handle Interface up event
*/
func (ribdServiceHandler *RIBDServer) ProcessL3IntfUpEvent(ipAddr string) {
	logger.Debug("processL3IntfUpEvent")
	var ipMask net.IP
	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return
	}
	ipMask = make(net.IP, 4)
	copy(ipMask, ipNet.Mask)
	ipAddrStr := ip.String()
	ipMaskStr := net.IP(ipMask).String()
	logger.Info(fmt.Sprintln(" processL3IntfUpEvent for  ipaddr ",ipAddrStr, " mask ",  ipMaskStr))
	for i := 0; i < len(ConnectedRoutes); i++ {
		logger.Info(fmt.Sprintln("Current state of this connected route is ", ConnectedRoutes[i].IsValid))
		if ConnectedRoutes[i].Ipaddr == ipAddrStr && ConnectedRoutes[i].Mask == ipMaskStr && ConnectedRoutes[i].IsValid == false {
			logger.Info(fmt.Sprintln("Add this route with destAddress = ",ConnectedRoutes[i].Ipaddr," nwMask = " , ConnectedRoutes[i].Mask))

			ConnectedRoutes[i].IsValid = true
			policyRoute := ribdInt.Routes{Ipaddr: ConnectedRoutes[i].Ipaddr, Mask: ConnectedRoutes[i].Mask, NextHopIp: ConnectedRoutes[i].NextHopIp, IfIndex: ConnectedRoutes[i].IfIndex, Metric: ConnectedRoutes[i].Metric, Prototype: ConnectedRoutes[i].Prototype}
			params := RouteParams{destNetIp: ConnectedRoutes[i].Ipaddr, networkMask: ConnectedRoutes[i].Mask, nextHopIp: ConnectedRoutes[i].NextHopIp, nextHopIfIndex: ribd.Int(ConnectedRoutes[i].IfIndex), metric: ribd.Int(ConnectedRoutes[i].Metric), routeType: ribd.Int(ConnectedRoutes[i].Prototype), sliceIdx: ribd.Int(ConnectedRoutes[i].SliceIdx), createType: FIBOnly, deleteType: Invalid}
			PolicyEngineFilter(policyRoute, policyCommonDefs.PolicyPath_Import, params)
		}
	}
}

func getLogicalIntfInfo() {
	logger.Debug("Getting Logical Interfaces from asicd")
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
			logger.Info("0 objects returned from GetBulkLogicalIntfState")
			return
		}
		logger.Info(fmt.Sprintln("len(bulkInfo.GetBulkLogicalIntfState)  = ", len(bulkInfo.LogicalIntfStateList), " num objects returned = ", bulkInfo.Count))
		for i := 0; i < int(bulkInfo.Count); i++ {
			ifId := (bulkInfo.LogicalIntfStateList[i].IfIndex)
			logger.Info(fmt.Sprintln("logical interface = ", bulkInfo.LogicalIntfStateList[i].Name, "ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.LogicalIntfStateList[i].Name}
			IntfIdNameMap[ifId] = intfEntry
			if IfNameToIfIndex == nil {
				IfNameToIfIndex = make(map[string]int32)
			}
			IfNameToIfIndex[bulkInfo.LogicalIntfStateList[i].Name] = ifId
		}
		if bulkInfo.More == false {
			logger.Info("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(bulkInfo.EndIdx)
	}
}
func getVlanInfo() {
	logger.Debug("Getting vlans from asicd")
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
			logger.Info("0 objects returned from GetBulkVlan")
			return
		}
		logger.Info(fmt.Sprintln("len(bulkInfo.GetBulkVlan)  = ",len(bulkInfo.VlanStateList)," num objects returned = " , bulkInfo.Count))
		for i := 0; i < int(bulkInfo.Count); i++ {
			ifId := (bulkInfo.VlanStateList[i].IfIndex)
			logger.Info(fmt.Sprintln("vlan = ", bulkInfo.VlanStateList[i].VlanId, "ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.VlanStateList[i].VlanName}
			IntfIdNameMap[ifId] = intfEntry
			if IfNameToIfIndex == nil {
				IfNameToIfIndex = make(map[string]int32)
			}
			IfNameToIfIndex[bulkInfo.VlanStateList[i].VlanName] = ifId
		}
		if bulkInfo.More == false {
			logger.Info("more returned as false, so no more get bulks")
			return
		}
		currMarker = asicdServices.Int(bulkInfo.EndIdx)
	}
}
func getPortInfo() {
	logger.Debug("Getting ports from asicd")
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
		logger.Info(fmt.Sprintln("len(bulkInfo.PortStateList)  = ",len(bulkInfo.PortStateList), " num objects returned = ",  bulkInfo.Count))
		for i := 0; i < int(bulkInfo.Count); i++ {
			ifId := bulkInfo.PortStateList[i].IfIndex
			logger.Info(fmt.Sprintln("ifId = ", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: bulkInfo.PortStateList[i].Name}
			IntfIdNameMap[ifId] = intfEntry
			if IfNameToIfIndex == nil {
				IfNameToIfIndex = make(map[string]int32)
			}
			IfNameToIfIndex[bulkInfo.PortStateList[i].Name] = ifId
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
	logger.Info("AcceptConfigActions: Setting AcceptConfig to true")
	RouteServiceHandler.AcceptConfig = true
	getIntfInfo()
	getConnectedRoutes()
	ribdServiceHandler.UpdateRoutesFromDB()
	go ribdServiceHandler.SetupEventHandler(AsicdSub, asicdCommonDefs.PUB_SOCKET_ADDR, SUB_ASICD)
	logger.Info("All set to signal start the RIBd server")
	ribdServiceHandler.ServerUpCh <- true
}
func (ribdServiceHandler *RIBDServer) connectToClient(client ClientJson) {
	var timer *time.Timer
	logger.Info(fmt.Sprintln("in go routine ConnectToClient for connecting to %s\n", client.Name))
	for {
		timer = time.NewTimer(time.Second * 10)
		<-timer.C
		if client.Name == "asicd" {
			logger.Info(fmt.Sprintln("found asicd at port ", client.Port))
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
			logger.Info(fmt.Sprintln("found arpd at port ", client.Port))
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
		logger.Info("Error in reading configuration file")
		return
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		logger.Info("Error in Unmarshalling Json")
		return
	}

	for _, client := range clientsList {
		logger.Info(fmt.Sprintln("#### Client name is ", client.Name))
		if client.Name == "asicd" {
			logger.Info(fmt.Sprintln("found asicd at port ", client.Port))
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
			logger.Info(fmt.Sprintln("found arpd at port ", client.Port))
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

func (ribdServiceHandler *RIBDServer) InitializeGlobalPolicyDB() *policy.PolicyEngineDB {
	ribdServiceHandler.GlobalPolicyEngineDB = policy.NewPolicyEngineDB(logger)
	ribdServiceHandler.GlobalPolicyEngineDB.SetDefaultImportPolicyActionFunc(defaultImportPolicyEngineActionFunc)
	ribdServiceHandler.GlobalPolicyEngineDB.SetDefaultExportPolicyActionFunc(defaultExportPolicyEngineActionFunc)
	ribdServiceHandler.GlobalPolicyEngineDB.SetIsEntityPresentFunc(DoesRouteExist)
	ribdServiceHandler.GlobalPolicyEngineDB.SetEntityUpdateFunc(UpdateRouteAndPolicyDB)
	ribdServiceHandler.GlobalPolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineRouteDispositionAction)
	ribdServiceHandler.GlobalPolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionRedistribute)
	ribdServiceHandler.GlobalPolicyEngineDB.SetActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise, policyEngineActionNetworkStatementAdvertise)
	ribdServiceHandler.GlobalPolicyEngineDB.SetActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionSetAdminDistance)
	ribdServiceHandler.GlobalPolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteDisposition, policyEngineUndoRouteDispositionAction)
	ribdServiceHandler.GlobalPolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeRouteRedistribute, policyEngineActionUndoRedistribute)
	ribdServiceHandler.GlobalPolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PoilcyActionTypeSetAdminDistance, policyEngineActionUndoSetAdminDistance)
	ribdServiceHandler.GlobalPolicyEngineDB.SetUndoActionFunc(policyCommonDefs.PolicyActionTypeNetworkStatementAdvertise, policyEngineActionUndoNetworkStatemenAdvertiseAction)
	ribdServiceHandler.GlobalPolicyEngineDB.SetTraverseAndApplyPolicyFunc(policyEngineTraverseAndApply)
	ribdServiceHandler.GlobalPolicyEngineDB.SetTraverseAndReversePolicyFunc(policyEngineTraverseAndReverse)
	ribdServiceHandler.GlobalPolicyEngineDB.SetGetPolicyEntityMapIndexFunc(getPolicyRouteMapIndex)
	return ribdServiceHandler.GlobalPolicyEngineDB
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
func NewRIBDServicesHandler(dbHdl *dbutils.DBUtil, loggerC *logging.Writer) *RIBDServer {
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
	ribdServicesHandler.RouteConfCh = make(chan RIBdServerConfig, 5000)
	ribdServicesHandler.AsicdRouteCh = make(chan RIBdServerConfig, 5000)
	ribdServicesHandler.ArpdRouteCh = make(chan RIBdServerConfig, 5000)
	ribdServicesHandler.NotificationChannel = make(chan NotificationMsg, 5000)
	ribdServicesHandler.PolicyConditionConfCh = make(chan RIBdServerConfig)
	ribdServicesHandler.PolicyActionConfCh = make(chan RIBdServerConfig)
	ribdServicesHandler.PolicyStmtConfCh = make(chan RIBdServerConfig)
	ribdServicesHandler.PolicyDefinitionConfCh = make(chan RIBdServerConfig)
	ribdServicesHandler.PolicyApplyCh = make(chan ApplyPolicyInfo, 100)
	ribdServicesHandler.PolicyUpdateApplyCh = make(chan ApplyPolicyInfo, 100)
	ribdServicesHandler.DBRouteCh = make(chan RIBdServerConfig)
	ribdServicesHandler.ServerUpCh = make(chan bool)
	ribdServicesHandler.DbHdl = dbHdl
	RouteServiceHandler = ribdServicesHandler
	//ribdServicesHandler.RouteInstallCh = make(chan RouteParams)
	BuildRouteProtocolTypeMapDB()
	BuildProtocolAdminDistanceMapDB()
	BuildPublisherMap()
	PolicyEngineDB = ribdServicesHandler.InitializePolicyDB()
	GlobalPolicyEngineDB = ribdServicesHandler.InitializeGlobalPolicyDB()
	return ribdServicesHandler
}
func (ribdServiceHandler *RIBDServer) StartServer(paramsDir string) {
	DummyRouteInfoRecord.protocol = PROTOCOL_NONE
	configFile := paramsDir + "/clients.json"
	logger.Debug(fmt.Sprintln("configfile = ", configFile))
	PARAMSDIR = paramsDir
	ribdServiceHandler.UpdatePolicyObjectsFromDB() //(paramsDir)
	ribdServiceHandler.ConnectToClients(configFile)
	logger.Debug("Starting the server loop")
	count := 0
	for {
		if !RouteServiceHandler.AcceptConfig {
			if count%1000 == 0 {
				logger.Debug("RIBD not ready to accept config")
			}
			count++
			continue
		}
		select {
		case routeConf := <-ribdServiceHandler.RouteConfCh:
			logger.Debug(fmt.Sprintln("received message on RouteConfCh channel, op: ", routeConf.Op))
			if routeConf.Op == "add" {
			    ribdServiceHandler.ProcessRouteCreateConfig(routeConf.OrigConfigObject.(*ribd.IPv4Route))
			} else if routeConf.Op == "del" {
				ribdServiceHandler.ProcessRouteDeleteConfig(routeConf.OrigConfigObject.(*ribd.IPv4Route))
			} else if routeConf.Op == "update" {
				if routeConf.PatchOp == nil || len(routeConf.PatchOp) == 0 {
                      ribdServiceHandler.ProcessRouteUpdateConfig(routeConf.OrigConfigObject.(*ribd.IPv4Route), routeConf.NewConfigObject.(*ribd.IPv4Route), routeConf.AttrSet)
				} else {
                     ribdServiceHandler.ProcessRoutePatchUpdateConfig(routeConf.OrigConfigObject.(*ribd.IPv4Route), routeConf.NewConfigObject.(*ribd.IPv4Route), routeConf.PatchOp)
				}
			}
		case info := <-ribdServiceHandler.PolicyApplyCh:
			logger.Debug("received message on PolicyApplyCh channel")
			//update the local policyEngineDB
			ribdServiceHandler.UpdateApplyPolicy(info, true, PolicyEngineDB)
			ribdServiceHandler.PolicyUpdateApplyCh <- info
		case info := <-ribdServiceHandler.TrackReachabilityCh:
			logger.Debug("received message on TrackReachabilityCh channel")
			ribdServiceHandler.TrackReachabilityStatus(info.IpAddr, info.Protocol, info.Op)
		}
	}
}
