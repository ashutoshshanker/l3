package server

import (
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"container/list"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"l3/ospf/config"
	"log/syslog"
	"ribd"
	"strconv"
	"sync"
	"time"
	"utils/ipcutils"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type OspfClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type LsdbKey struct {
	AreaId uint32
}

type LsdbSliceEnt struct {
	AreaId uint32
	LSType uint8
	LSId   uint32
	AdvRtr uint32
}

type OSPFServer struct {
	logger             *syslog.Writer
	ribdClient         RibdClient
	asicdClient        AsicdClient
	portPropertyMap    map[int32]PortProperty
	vlanPropertyMap    map[uint16]VlanProperty
	IPIntfPropertyMap  map[string]IPIntfProperty
	ospfGlobalConf     GlobalConf
	GlobalConfigCh     chan config.GlobalConf
	AreaConfigCh       chan config.AreaConf
	IntfConfigCh       chan config.InterfaceConf
	AreaLsdb           map[LsdbKey]LSDatabase
	LsdbSlice          []LsdbSliceEnt
	LsdbStateTimer     *time.Timer
	AreaSelfOrigLsa    map[LsdbKey]SelfOrigLsa
	LsdbUpdateCh       chan LsdbUpdateMsg
	LsaUpdateRetCodeCh chan bool
	IntfStateChangeCh  chan LSAChangeMsg
	NetworkDRChangeCh  chan LSAChangeMsg
	FlushNetworkLSACh  chan NetworkLSAChangeMsg
	CreateNetworkLSACh chan ospfNbrMdata
	AdjOKEvtCh         chan AdjOKEvtMsg

	/*
	   connRoutesTimer         *time.Timer
	   ribSubSocket        *nanomsg.SubSocket
	   ribSubSocketCh      chan []byte
	   ribSubSocketErrCh   chan error
	*/
	asicdSubSocket        *nanomsg.SubSocket
	asicdSubSocketCh      chan []byte
	asicdSubSocketErrCh   chan error
	AreaConfMap           map[AreaConfKey]AreaConf
	IntfConfMap           map[IntfConfKey]IntfConf
	IntfTxMap             map[IntfConfKey]IntfTxHandle
	IntfRxMap             map[IntfConfKey]IntfRxHandle
	NeighborConfigMap     map[uint32]OspfNeighborEntry
	NeighborListMap       map[IntfConfKey]list.List
	neighborConfMutex     sync.Mutex
	neighborHelloEventCh  chan IntfToNeighMsg
	neighborFSMCtrlCh     chan bool
	neighborConfCh        chan ospfNeighborConfMsg
	neighborConfStopCh    chan bool
	nbrFSMCtrlCh          chan bool
	neighborSliceRefCh    *time.Ticker
	neighborBulkSlice     []uint32
	neighborDBDEventCh    chan ospfNeighborDBDMsg
	neighborLSAReqEventCh chan ospfNeighborLSAreqMsg
	neighborLSAUpdEventCh chan ospfNeighborLSAUpdMsg
	neighborLSAACKEventCh chan ospfNeighborLSAACKMsg
	ospfNbrDBDSendCh      chan ospfNeighborDBDMsg
	ospfNbrLsaSendCh      chan ospfNeighborLSAreqMsg
	ospfNbrLsaUpdSendCh   chan ospfLsdbToNbrMsg
	ospfRxTxNbrPktStopCh  chan bool

	//neighborDBDEventCh   chan IntfToNeighDbdMsg

	AreaStateTimer           *time.Timer
	AreaStateMutex           sync.RWMutex
	AreaStateMap             map[AreaConfKey]AreaState
	AreaStateSlice           []AreaConfKey
	AreaConfKeyToSliceIdxMap map[AreaConfKey]int
	IntfKeySlice             []IntfConfKey
	IntfKeyToSliceIdxMap     map[IntfConfKey]bool
	IntfStateTimer           *time.Timer
	IntfSliceRefreshCh       chan bool
	IntfSliceRefreshDoneCh   chan bool

	RefreshDuration time.Duration
}

func NewOSPFServer(logger *syslog.Writer) *OSPFServer {
	ospfServer := &OSPFServer{}
	ospfServer.logger = logger
	ospfServer.GlobalConfigCh = make(chan config.GlobalConf)
	ospfServer.AreaConfigCh = make(chan config.AreaConf)
	ospfServer.IntfConfigCh = make(chan config.InterfaceConf)
	ospfServer.portPropertyMap = make(map[int32]PortProperty)
	ospfServer.vlanPropertyMap = make(map[uint16]VlanProperty)
	ospfServer.AreaConfMap = make(map[AreaConfKey]AreaConf)
	ospfServer.IntfConfMap = make(map[IntfConfKey]IntfConf)
	ospfServer.IntfTxMap = make(map[IntfConfKey]IntfTxHandle)
	ospfServer.IntfRxMap = make(map[IntfConfKey]IntfRxHandle)
	ospfServer.AreaLsdb = make(map[LsdbKey]LSDatabase)
	ospfServer.AreaSelfOrigLsa = make(map[LsdbKey]SelfOrigLsa)
	ospfServer.IntfStateChangeCh = make(chan LSAChangeMsg)
	ospfServer.NetworkDRChangeCh = make(chan LSAChangeMsg)
	ospfServer.CreateNetworkLSACh = make(chan ospfNbrMdata)
	ospfServer.FlushNetworkLSACh = make(chan NetworkLSAChangeMsg)
	ospfServer.LsdbSlice = []LsdbSliceEnt{}
	ospfServer.LsdbUpdateCh = make(chan LsdbUpdateMsg)
	ospfServer.LsaUpdateRetCodeCh = make(chan bool)
	ospfServer.AdjOKEvtCh = make(chan AdjOKEvtMsg)
	ospfServer.NeighborConfigMap = make(map[uint32]OspfNeighborEntry)
	ospfServer.NeighborListMap = make(map[IntfConfKey]list.List)
	ospfServer.neighborConfMutex = sync.Mutex{}
	ospfServer.neighborHelloEventCh = make(chan IntfToNeighMsg)
	ospfServer.neighborConfCh = make(chan ospfNeighborConfMsg)
	ospfServer.neighborConfStopCh = make(chan bool)
	ospfServer.neighborSliceRefCh = time.NewTicker(time.Minute * 10)
	ospfServer.AreaStateMutex = sync.RWMutex{}
	ospfServer.AreaStateMap = make(map[AreaConfKey]AreaState)
	ospfServer.AreaStateSlice = []AreaConfKey{}
	ospfServer.AreaConfKeyToSliceIdxMap = make(map[AreaConfKey]int)
	ospfServer.IntfKeySlice = []IntfConfKey{}
	ospfServer.IntfKeyToSliceIdxMap = make(map[IntfConfKey]bool)
	ospfServer.IntfSliceRefreshCh = make(chan bool)
	ospfServer.IntfSliceRefreshDoneCh = make(chan bool)
	ospfServer.nbrFSMCtrlCh = make(chan bool)
	ospfServer.RefreshDuration = time.Duration(10) * time.Minute
	ospfServer.neighborDBDEventCh = make(chan ospfNeighborDBDMsg)
	ospfServer.neighborLSAReqEventCh = make(chan ospfNeighborLSAreqMsg)
	ospfServer.neighborLSAUpdEventCh = make(chan ospfNeighborLSAUpdMsg)
	ospfServer.neighborLSAACKEventCh = make(chan ospfNeighborLSAACKMsg)
	ospfServer.ospfNbrDBDSendCh = make(chan ospfNeighborDBDMsg)
	ospfServer.ospfNbrLsaSendCh = make(chan ospfNeighborLSAreqMsg)
	ospfServer.ospfNbrLsaUpdSendCh = make(chan ospfLsdbToNbrMsg)
	ospfServer.ospfRxTxNbrPktStopCh = make(chan bool)

	/*
	   ospfServer.ribSubSocketCh = make(chan []byte)
	   ospfServer.ribSubSocketErrCh = make(chan error)
	   ospfServer.connRoutesTimer = time.NewTimer(time.Duration(10) * time.Second)
	   ospfServer.connRoutesTimer.Stop()
	*/
	ospfServer.asicdSubSocketCh = make(chan []byte)
	ospfServer.asicdSubSocketErrCh = make(chan error)

	return ospfServer
}

func (server *OSPFServer) ConnectToClients(paramsFile string) {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		server.logger.Info("Error in reading configuration file")
		return
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		server.logger.Info("Error in Unmarshalling Json")
		return
	}

	for _, client := range clientsList {
		server.logger.Info("#### Client name is ")
		server.logger.Info(client.Name)
		if client.Name == "asicd" {
			server.logger.Info(fmt.Sprintln("found asicd at port", client.Port))
			server.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(server.asicdClient.Address)
			if server.asicdClient.Transport != nil && server.asicdClient.PtrProtocolFactory != nil {
				server.logger.Info("connecting to asicd")
				server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
				server.asicdClient.IsConnected = true
			}
		} else if client.Name == "ribd" {
			server.logger.Info(fmt.Sprintln("found ribd at port", client.Port))
			server.ribdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(server.ribdClient.Address)
			if server.ribdClient.Transport != nil && server.ribdClient.PtrProtocolFactory != nil {
				server.logger.Info("connecting to ribd")
				server.ribdClient.ClientHdl = ribd.NewRouteServiceClientFactory(server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory)
				server.ribdClient.IsConnected = true
			}
		}
	}
}

func (server *OSPFServer) InitServer(paramFile string) {
	server.logger.Info(fmt.Sprintln("Starting Ospf Server"))
	server.ConnectToClients(paramFile)
	server.BuildPortPropertyMap()
	server.initOspfGlobalConfDefault()
	server.logger.Info(fmt.Sprintln("GlobalConf:", server.ospfGlobalConf))
	server.initAreaConfDefault()
	server.logger.Info(fmt.Sprintln("AreaConf:", server.AreaConfMap))
	server.initIntfStateSlice()
	/*
	   server.logger.Info("Listen for RIBd updates")
	   server.listenForRIBUpdates(ribdCommonDefs.PUB_SOCKET_ADDR)
	   go createRIBSubscriber()
	   server.connRoutesTimer.Reset(time.Duration(10) * time.Second)
	*/
	server.logger.Info("Listen for ASICd updates")
	server.listenForASICdUpdates(pluginCommon.PUB_SOCKET_ADDR)
	go server.createASICdSubscriber()

}

func (server *OSPFServer) StartServer(paramFile string) {
	server.InitServer(paramFile)
	for {
		select {
		case gConf := <-server.GlobalConfigCh:
			server.processGlobalConfig(gConf)
		case areaConf := <-server.AreaConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Area Configuration", areaConf))
			server.processAreaConfig(areaConf)
		case ifConf := <-server.IntfConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Intf Configuration", ifConf))
			server.processIntfConfig(ifConf)
		case asicdrxBuf := <-server.asicdSubSocketCh:
			server.processAsicdNotification(asicdrxBuf)
		case <-server.asicdSubSocketErrCh:

			/*
			   case ribrxBuf := <-server.ribSubSocketCh:
			       server.processRibdNotification(ribdrxBuf)
			   case <-server.connRoutesTimer.C:
			       routes, _ := server.ribdClient.ClientHdl.GetConnectedRoutesInfo()
			       server.logger.Info(fmt.Sprintln("Received Connected Routes:", routes))
			       //server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))
			       //server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

			   case <-server.ribSubSocketErrCh:
			       ;
			*/
		case msg := <-server.IntfSliceRefreshCh:
			if msg == true {
				server.refreshIntfKeySlice()
				server.IntfSliceRefreshDoneCh <- true
			}

		}
	}
}
