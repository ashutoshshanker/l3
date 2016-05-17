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
	"asicd/asicdCommonDefs"
	//"asicdServices"
	//"encoding/json"
	"fmt"
	//"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	//"io/ioutil"
	"os"
	"os/signal"
	//"strconv"
	"syscall"
	"time"
	"utils/asicdClientManager"
	"utils/dbutils"
	//"utils/ipcutils"
	"utils/logging"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type ArpEntry struct {
	MacAddr   string
	VlanId    int
	IfName    string
	L3IfIdx   int
	Counter   int
	TimeStamp time.Time
	PortNum   int
	Type      bool //True : RIB False: RX
}

type ArpState struct {
	IpAddr         string
	MacAddr        string
	VlanId         int
	Intf           string
	ExpiryTimeLeft string
}

type ResolveIPv4 struct {
	TargetIP string
	IfType   int
	IfId     int
}

type DeleteResolvedIPv4 struct {
	IpAddr string
}

type ArpConf struct {
	RefTimeout int
}

type ActionType uint8

const (
	DeleteByIPAddr  ActionType = 1
	RefreshByIPAddr ActionType = 2
	DeleteByIfName  ActionType = 3
	RefreshByIfName ActionType = 4
)

type ArpActionMsg struct {
	Type ActionType
	Obj  string
}

/*
type ArpClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
}
*/

type ARPServer struct {
	logger   *logging.Writer
	arpCache map[string]ArpEntry //Key: Dest IpAddr
	//asicdClient             AsicdClient
	asicdSubSocket          *nanomsg.SubSocket
	asicdSubSocketCh        chan []byte
	asicdSubSocketErrCh     chan error
	dbHdl                   *dbutils.DBUtil
	snapshotLen             int32
	pcapTimeout             time.Duration
	promiscuous             bool
	confRefreshTimeout      int
	minRefreshTimeout       int
	timerGranularity        int
	timeout                 time.Duration
	timeoutCounter          int
	minCnt                  int
	retryCnt                int
	probeWait               int
	probeNum                int
	probeMax                int
	probeMin                int
	arpSliceRefreshTimer    *time.Timer
	arpSliceRefreshDuration time.Duration
	usrConfDbName           string
	l3IntfPropMap           map[int]L3IntfProperty //Key: IfIndex
	portPropMap             map[int]PortProperty   //Key: IfIndex
	vlanPropMap             map[int]VlanProperty   //Key: IfIndex
	lagPropMap              map[int]LagProperty    //Key:IfIndex
	arpSlice                []string
	arpEntryUpdateCh        chan UpdateArpEntryMsg
	arpEntryDeleteCh        chan DeleteArpEntryMsg
	//arpEntryCreateCh        chan CreateArpEntryMsg
	arpEntryMacMoveCh      chan asicdCommonDefs.IPv4NbrMacMoveNotifyMsg
	arpEntryCntUpdateCh    chan int
	arpSliceRefreshStartCh chan bool
	arpSliceRefreshDoneCh  chan bool
	arpCounterUpdateCh     chan bool
	arpActionProcessCh     chan ArpActionMsg
	ResolveIPv4Ch          chan ResolveIPv4
	DeleteResolvedIPv4Ch   chan DeleteResolvedIPv4
	ArpConfCh              chan ArpConf
	dumpArpTable           bool
	InitDone               chan bool

	ArpActionCh                chan ArpActionMsg
	arpDeleteArpEntryFromRibCh chan string

	plugin string
	//FSAsicd    *asicdClientManager.FSAsicdClientMgr
	//OvsDBAsicd *asicdClientManager.OvsDBAsicdClientMgr
	AsicdPlugin asicdClientManager.AsicdClientIntf
}

func NewARPServer(logger *logging.Writer) *ARPServer {
	arpServer := &ARPServer{}
	arpServer.logger = logger
	arpServer.arpCache = make(map[string]ArpEntry)
	arpServer.asicdSubSocketCh = make(chan []byte)
	arpServer.asicdSubSocketErrCh = make(chan error)
	arpServer.l3IntfPropMap = make(map[int]L3IntfProperty)
	arpServer.lagPropMap = make(map[int]LagProperty)
	arpServer.vlanPropMap = make(map[int]VlanProperty)
	arpServer.portPropMap = make(map[int]PortProperty)
	arpServer.arpSlice = make([]string, 0)
	arpServer.arpEntryUpdateCh = make(chan UpdateArpEntryMsg)
	arpServer.arpEntryDeleteCh = make(chan DeleteArpEntryMsg)
	//arpServer.arpEntryCreateCh = make(chan CreateArpEntryMsg)
	arpServer.arpEntryCntUpdateCh = make(chan int)
	arpServer.arpSliceRefreshStartCh = make(chan bool)
	arpServer.arpSliceRefreshDoneCh = make(chan bool)
	arpServer.arpCounterUpdateCh = make(chan bool)
	arpServer.arpActionProcessCh = make(chan ArpActionMsg)
	arpServer.arpDeleteArpEntryFromRibCh = make(chan string)
	arpServer.ResolveIPv4Ch = make(chan ResolveIPv4)
	arpServer.DeleteResolvedIPv4Ch = make(chan DeleteResolvedIPv4)
	arpServer.ArpConfCh = make(chan ArpConf)
	arpServer.InitDone = make(chan bool)
	arpServer.ArpActionCh = make(chan ArpActionMsg)
	return arpServer
}

func (server *ARPServer) initArpParams() {
	server.logger.Debug("Calling initParams...")
	server.snapshotLen = 65549
	server.promiscuous = false
	server.minCnt = 1
	server.retryCnt = 10
	server.pcapTimeout = time.Duration(1) * time.Second
	server.timerGranularity = 1
	server.confRefreshTimeout = 600 / server.timerGranularity
	server.minRefreshTimeout = 300 / server.timerGranularity
	server.timeout = time.Duration(server.timerGranularity) * time.Second
	server.timeoutCounter = 600
	server.retryCnt = 5
	server.minCnt = 1
	server.probeWait = 5
	server.probeNum = 5
	server.probeMax = 20
	server.probeMax = 10
	server.arpSliceRefreshDuration = time.Duration(10) * time.Minute
	server.dumpArpTable = false
}

/*
func (server *ARPServer) connectToServers(paramsFile string) {
	server.logger.Debug(fmt.Sprintln("Inside connectToServers...paramsFile", paramsFile))
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		server.logger.Err("Error in reading configuration file")
		return
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		server.logger.Err("Error in Unmarshalling Json")
		return
	}

	for _, client := range clientsList {
		if client.Name == "asicd" {
			server.logger.Debug(fmt.Sprintln("found asicd at port", client.Port))
			server.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.asicdClient.Address)
			if err != nil {
				server.logger.Err(fmt.Sprintln("Failed to connect to Asicd, retrying until connection is successful"))
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
						server.logger.Err("Still can't connect to Asicd, retrying..")
					}
				}

			}
			server.logger.Info("Arpd is connected to Asicd")
			server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
			//server.FSAsicd = &asicdClientManager.FSAsicdClientMgr{server.asicdClient.ClientHdl}
		}
	}
}
*/

func (server *ARPServer) sigHandler(sigChan <-chan os.Signal) {
	server.logger.Debug("Inside sigHandler....")
	signal := <-sigChan
	switch signal {
	case syscall.SIGHUP:
		server.logger.Debug("Received SIGHUP signal")
		server.printArpEntries()
		server.logger.Debug("Closing DB handler")
		if server.dbHdl != nil {
			server.dbHdl.Disconnect()
		}
		os.Exit(0)
	default:
		server.logger.Err(fmt.Sprintln("Unhandled signal : ", signal))
	}
}

func (server *ARPServer) InitServer(paramDir string, plugin string) {
	server.initArpParams()

	server.plugin = plugin
	fileName := paramDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	server.logger.Debug("Starting Arp Server")
	//server.connectToServers(fileName)
	//server.AsicdPlugin = asicdClientManager.NewAsicdClientInit(server.plugin, server.asicdClient.ClientHdl, 100)
	server.AsicdPlugin = asicdClientManager.NewAsicdClientInit(server.plugin, fileName, server.logger)
	if server.AsicdPlugin == nil {
		server.logger.Err("Unable to instantiate Asicd Interface")
		return
	}
	server.logger.Debug("Listen for ASICd updates")
	server.listenForASICdUpdates(asicdCommonDefs.PUB_SOCKET_ADDR)
	go server.createASICdSubscriber()
	server.buildArpInfra()
	server.processArpInfra()

	err := server.initiateDB()
	if err != nil {
		server.logger.Err(fmt.Sprintln("DB Initialization failure...", err))
	} else {
		server.logger.Debug("ArpCache DB has been initiated successfully...")
		server.updateArpCacheFromDB()
		server.refreshArpDB()
	}

	if server.dbHdl != nil {
		server.getArpGlobalConfig()
	}

	sigChan := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChan, signalList...)
	go server.sigHandler(sigChan)
	go server.updateArpCache()
	go server.refreshArpSlice()
	server.FlushLinuxArpCache()
	go server.arpCacheTimeout()
}

func (server *ARPServer) StartServer(paramDir string, plugin string) {
	server.logger.Debug(fmt.Sprintln("Inside Start Server...", paramDir))
	server.InitServer(paramDir, plugin)
	server.InitDone <- true
	for {
		select {
		case arpConf := <-server.ArpConfCh:
			server.processArpConf(arpConf)
		case rConf := <-server.ResolveIPv4Ch:
			server.processResolveIPv4(rConf)
		case rConf := <-server.DeleteResolvedIPv4Ch:
			server.processDeleteResolvedIPv4(rConf.IpAddr)
		case arpActionMsg := <-server.ArpActionCh:
			server.processArpAction(arpActionMsg)
		case asicdrxBuf := <-server.asicdSubSocketCh:
			server.processAsicdNotification(asicdrxBuf)
		case <-server.asicdSubSocketErrCh:
		}
	}
}
