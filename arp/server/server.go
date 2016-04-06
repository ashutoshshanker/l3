package server

import (
	"asicd/asicdConstDefs"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"utils/ipcutils"
	"utils/logging"
	//"github.com/google/gopacket/pcap"
	"asicdServices"
	//"utils/commonDefs"
	"database/sql"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type ArpEntry struct {
	MacAddr string
	VlanId  int
	IfName  string
	L3IfIdx int
	Counter int
	//Valid           bool
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

type ArpConf struct {
	RefTimeout int
}

type ArpClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
}

type ARPServer struct {
	logger                  *logging.Writer
	arpCache                map[string]ArpEntry //Key: Dest IpAddr
	asicdClient             AsicdClient
	asicdSubSocket          *nanomsg.SubSocket
	asicdSubSocketCh        chan []byte
	asicdSubSocketErrCh     chan error
	dbHdl                   *sql.DB
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
	arpEntryMacMoveCh      chan asicdConstDefs.IPv4NbrMacMoveNotifyMsg
	arpEntryCntUpdateCh    chan int
	arpSliceRefreshStartCh chan bool
	arpSliceRefreshDoneCh  chan bool
	arpCounterUpdateCh     chan bool
	ResolveIPv4Ch          chan ResolveIPv4
	ArpConfCh              chan ArpConf
	dumpArpTable           bool
	InitDone	       chan bool
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
	arpServer.ResolveIPv4Ch = make(chan ResolveIPv4)
	arpServer.ArpConfCh = make(chan ArpConf)
	arpServer.InitDone = make(chan bool)
	return arpServer
}

func (server *ARPServer) initArpParams() {
	server.logger.Info("Calling initParams...")
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
	server.usrConfDbName = "UsrConfDb.db"
	server.dumpArpTable = false
}

func (server *ARPServer) connectToClients(paramsFile string) {
	server.logger.Info(fmt.Sprintln("Inside connectToClients...paramsFile", paramsFile))
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
			server.logger.Info("Arpd is connected to Asicd")
			server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
		}
	}
}

func (server *ARPServer) sigHandler(sigChan <-chan os.Signal) {
	server.logger.Info("Inside sigHandler....")
	signal := <-sigChan
	switch signal {
	case syscall.SIGHUP:
		server.logger.Info("Received SIGHUP signal")
		server.printArpEntries()
		server.logger.Info("Closing DB handler")
		if server.dbHdl != nil {
			server.dbHdl.Close()
		}
		os.Exit(0)
	default:
		server.logger.Info(fmt.Sprintln("Unhandled signal : ", signal))
	}
}

func (server *ARPServer) InitServer(paramDir string) {
	server.initArpParams()

	fileName := paramDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	server.logger.Info("Starting Arp Server")
	server.connectToClients(fileName)
	server.logger.Info("Listen for ASICd updates")
	server.listenForASICdUpdates(asicdConstDefs.PUB_SOCKET_ADDR)
	go server.createASICdSubscriber()
	server.buildArpInfra()

	if paramDir[len(paramDir)-1] != '/' {
		paramDir = paramDir + "/"
	}
	dbName := paramDir + server.usrConfDbName

	err := server.initiateDB(dbName)
	if err != nil {
		server.logger.Info(fmt.Sprintln("DB Initialization failure...", err))
	} else {
		server.logger.Info("ArpCache DB has been initiated successfully...")
		server.updateArpCacheFromDB()
		server.refreshArpDB()
	}

	sigChan := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChan, signalList...)
	go server.sigHandler(sigChan)
	go server.updateArpCache()
	go server.refreshArpSlice()
	server.processArpInfra()
	go server.arpCacheTimeout()
}

func (server *ARPServer) StartServer(paramDir string) {
	server.logger.Info(fmt.Sprintln("Inside Start Server...", paramDir))
	server.InitServer(paramDir)
	server.InitDone <- true
	for {
		select {
		case arpConf := <-server.ArpConfCh:
			server.processArpConf(arpConf)
		case rConf := <-server.ResolveIPv4Ch:
			server.processResolveIPv4(rConf)
		case asicdrxBuf := <-server.asicdSubSocketCh:
			server.processAsicdNotification(asicdrxBuf)
		case <-server.asicdSubSocketErrCh:
		}
	}
}
