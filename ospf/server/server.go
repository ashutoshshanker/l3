package server

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"container/list"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"l3/ospf/config"
	"ribd"
	"strconv"
	"sync"
	"time"
	"utils/ipcutils"
	"utils/logging"
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

type RoutingTblKey struct {
	AreaId uint32
}

type LsdbSliceEnt struct {
	AreaId uint32
	LSType uint8
	LSId   uint32
	AdvRtr uint32
}

type OSPFServer struct {
	logger          *logging.Writer
	ribdClient      RibdClient
	asicdClient     AsicdClient
	portPropertyMap map[int32]PortProperty
	vlanPropertyMap map[uint16]VlanProperty
	//IPIntfPropertyMap  map[string]IPIntfProperty
	ipPropertyMap      map[uint32]IpProperty
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
	NetworkDRChangeCh  chan DrChangeMsg
	FlushNetworkLSACh  chan NetworkLSAChangeMsg
	CreateNetworkLSACh chan ospfNbrMdata
	AdjOKEvtCh         chan AdjOKEvtMsg
	maxAgeLsaCh        chan maxAgeLsaMsg
	ExternalRouteNotif chan RouteMdata

	//	   connRoutesTimer         *time.Timer
	ribSubSocket      *nanomsg.SubSocket
	ribSubSocketCh    chan []byte
	ribSubSocketErrCh chan error

	asicdSubSocket        *nanomsg.SubSocket
	asicdSubSocketCh      chan []byte
	asicdSubSocketErrCh   chan error
	AreaConfMap           map[AreaConfKey]AreaConf
	IntfConfMap           map[IntfConfKey]IntfConf
	IntfTxMap             map[IntfConfKey]IntfTxHandle
	IntfRxMap             map[IntfConfKey]IntfRxHandle
	NeighborConfigMap     map[NeighborConfKey]OspfNeighborEntry
	NeighborListMap       map[IntfConfKey]list.List
	neighborConfMutex     sync.Mutex
	neighborHelloEventCh  chan IntfToNeighMsg
	neighborFSMCtrlCh     chan bool
	neighborConfCh        chan ospfNeighborConfMsg
	neighborConfStopCh    chan bool
	nbrFSMCtrlCh          chan bool
	neighborSliceRefCh    *time.Ticker
	neighborSliceStartCh  chan bool
	neighborBulkSlice     []NeighborConfKey
	neighborDBDEventCh    chan ospfNeighborDBDMsg
	neighborLSAReqEventCh chan ospfNeighborLSAreqMsg
	neighborLSAUpdEventCh chan ospfNeighborLSAUpdMsg
	neighborLSAACKEventCh chan ospfNeighborLSAAckMsg
	ospfNbrDBDSendCh      chan ospfNeighborDBDMsg
	ospfNbrLsaReqSendCh   chan ospfNeighborLSAreqMsg
	ospfNbrLsaUpdSendCh   chan ospfFloodMsg
	ospfNbrLsaAckSendCh   chan ospfNeighborAckTxMsg
	ospfRxNbrPktStopCh    chan bool
	ospfTxNbrPktStopCh    chan bool

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

	TempAreaRoutingTbl   map[AreaIdKey]AreaRoutingTbl
	GlobalRoutingTbl     map[RoutingTblEntryKey]GlobalRoutingTblEntry
	OldGlobalRoutingTbl  map[RoutingTblEntryKey]GlobalRoutingTblEntry
	TempGlobalRoutingTbl map[RoutingTblEntryKey]GlobalRoutingTblEntry

	SummaryLsDb map[LsdbKey]SummaryLsaMap

	StartCalcSPFCh chan bool
	DoneCalcSPFCh  chan bool
	AreaGraph      map[VertexKey]Vertex
	SPFTree        map[VertexKey]TreeVertex
	AreaStubs      map[VertexKey]StubVertex
}

func NewOSPFServer(logger *logging.Writer) *OSPFServer {
	ospfServer := &OSPFServer{}
	ospfServer.logger = logger
	ospfServer.GlobalConfigCh = make(chan config.GlobalConf)
	ospfServer.AreaConfigCh = make(chan config.AreaConf)
	ospfServer.IntfConfigCh = make(chan config.InterfaceConf)
	ospfServer.portPropertyMap = make(map[int32]PortProperty)
	ospfServer.vlanPropertyMap = make(map[uint16]VlanProperty)
	ospfServer.ipPropertyMap = make(map[uint32]IpProperty)
	ospfServer.AreaConfMap = make(map[AreaConfKey]AreaConf)
	ospfServer.IntfConfMap = make(map[IntfConfKey]IntfConf)
	ospfServer.IntfTxMap = make(map[IntfConfKey]IntfTxHandle)
	ospfServer.IntfRxMap = make(map[IntfConfKey]IntfRxHandle)
	ospfServer.AreaLsdb = make(map[LsdbKey]LSDatabase)
	ospfServer.AreaSelfOrigLsa = make(map[LsdbKey]SelfOrigLsa)
	ospfServer.IntfStateChangeCh = make(chan LSAChangeMsg)
	ospfServer.NetworkDRChangeCh = make(chan DrChangeMsg)
	ospfServer.CreateNetworkLSACh = make(chan ospfNbrMdata)
	ospfServer.FlushNetworkLSACh = make(chan NetworkLSAChangeMsg)
	ospfServer.ExternalRouteNotif = make(chan RouteMdata)
	ospfServer.LsdbSlice = []LsdbSliceEnt{}
	ospfServer.LsdbUpdateCh = make(chan LsdbUpdateMsg)
	ospfServer.LsaUpdateRetCodeCh = make(chan bool)
	ospfServer.AdjOKEvtCh = make(chan AdjOKEvtMsg)
	ospfServer.maxAgeLsaCh = make(chan maxAgeLsaMsg)
	ospfServer.NeighborConfigMap = make(map[NeighborConfKey]OspfNeighborEntry)
	ospfServer.NeighborListMap = make(map[IntfConfKey]list.List)
	ospfServer.neighborConfMutex = sync.Mutex{}
	ospfServer.neighborHelloEventCh = make(chan IntfToNeighMsg)
	ospfServer.neighborConfCh = make(chan ospfNeighborConfMsg)
	ospfServer.neighborConfStopCh = make(chan bool)
	ospfServer.neighborSliceStartCh = make(chan bool)
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
	ospfServer.neighborLSAReqEventCh = make(chan ospfNeighborLSAreqMsg, 2)
	ospfServer.neighborLSAUpdEventCh = make(chan ospfNeighborLSAUpdMsg, 2)
	ospfServer.neighborLSAACKEventCh = make(chan ospfNeighborLSAAckMsg, 2)
	ospfServer.ospfNbrDBDSendCh = make(chan ospfNeighborDBDMsg)
	ospfServer.ospfNbrLsaAckSendCh = make(chan ospfNeighborAckTxMsg, 2)
	ospfServer.ospfNbrLsaReqSendCh = make(chan ospfNeighborLSAreqMsg, 2)
	ospfServer.ospfNbrLsaUpdSendCh = make(chan ospfFloodMsg, 2)
	ospfServer.ospfRxNbrPktStopCh = make(chan bool)
	ospfServer.ospfTxNbrPktStopCh = make(chan bool)

	ospfServer.ribSubSocketCh = make(chan []byte)
	ospfServer.ribSubSocketErrCh = make(chan error)
	// ospfServer.connRoutesTimer = time.NewTimer(time.Duration(10) * time.Second)
	// ospfServer.connRoutesTimer.Stop()

	ospfServer.asicdSubSocketCh = make(chan []byte)
	ospfServer.asicdSubSocketErrCh = make(chan error)

	ospfServer.GlobalRoutingTbl = make(map[RoutingTblEntryKey]GlobalRoutingTblEntry)
	ospfServer.OldGlobalRoutingTbl = make(map[RoutingTblEntryKey]GlobalRoutingTblEntry)
	ospfServer.TempGlobalRoutingTbl = make(map[RoutingTblEntryKey]GlobalRoutingTblEntry)
	//ospfServer.OldRoutingTbl = make(map[AreaIdKey]AreaRoutingTbl)
	ospfServer.TempAreaRoutingTbl = make(map[AreaIdKey]AreaRoutingTbl)
	ospfServer.StartCalcSPFCh = make(chan bool)
	ospfServer.DoneCalcSPFCh = make(chan bool)

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
		//server.logger.Info("#### Client name is ")
		//server.logger.Info(client.Name)
		if client.Name == "asicd" {
			server.logger.Info(fmt.Sprintln("found asicd at port", client.Port))
			server.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.asicdClient.Address)
			if err != nil {
				server.logger.Info(fmt.Sprintln("Failed to connect to Asicd, retrying until connection is successful"))
				count := 0
				ticker := time.NewTicker(time.Duration(1000) * time.Millisecond)
				for _ = range ticker.C {
					server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.asicdClient.Address)
					if err == nil {
						ticker.Stop()
						break
					}
					count++
					if (count % 10) == 0 {
						server.logger.Info("Still can't connect to Asicd, retrying..")
					}
				}

			}
			server.logger.Info("Ospfd is connected to Asicd")
			server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
			server.asicdClient.IsConnected = true
			/*
				if server.asicdClient.Transport != nil && server.asicdClient.PtrProtocolFactory != nil {
					server.logger.Info("connecting to asicd")
					server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
					server.asicdClient.IsConnected = true
				}
			*/
		} else if client.Name == "ribd" {
			server.logger.Info(fmt.Sprintln("found ribd at port", client.Port))
			server.ribdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.ribdClient.Address)
			if err != nil {
				server.logger.Info(fmt.Sprintln("Failed to connect to Ribd, retrying until connection is successful"))
				count := 0
				ticker := time.NewTicker(time.Duration(1000) * time.Millisecond)
				for _ = range ticker.C {
					server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.ribdClient.Address)
					if err == nil {
						ticker.Stop()
						break
					}
					count++
					if (count % 10) == 0 {
						server.logger.Info("Still can't connect to Ribd, retrying..")
					}
				}
			}
			server.logger.Info("Ospfd is connected to Ribd")
			server.ribdClient.ClientHdl = ribd.NewRIBDServicesClientFactory(server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory)
			server.ribdClient.IsConnected = true
			/*
				if server.ribdClient.Transport != nil && server.ribdClient.PtrProtocolFactory != nil {
					server.logger.Info("connecting to ribd")
					server.ribdClient.ClientHdl = ribd.NewRouteServiceClientFactory(server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory)
					server.ribdClient.IsConnected = true
				}
			*/
		}
	}
}

func (server *OSPFServer) InitServer(paramFile string) {
	server.logger.Info(fmt.Sprintln("Starting Ospf Server"))
	server.ConnectToClients(paramFile)
	server.logger.Info("Listen for ASICd updates")
	server.listenForASICdUpdates(asicdCommonDefs.PUB_SOCKET_ADDR)
	go server.createASICdSubscriber()

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
	err := server.initAsicdForRxMulticastPkt()
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to initialize asicd for receiving multicast packets", err))
	}
	go server.spfCalculation()

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

		case ribrxBuf := <-server.ribSubSocketCh:
			server.processRibdNotification(ribrxBuf)
		/*
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
