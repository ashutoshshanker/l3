package server

import (
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/garyburd/redigo/redis"
	"github.com/google/gopacket/pcap"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"l3/bfd/bfddCommonDefs"
	"net"
	"os"
	"os/signal"
	"ribd"
	"strconv"
	"syscall"
	"time"
	"utils/ipcutils"
	"utils/logging"
)

var (
	bfdSnapshotLen  int32  = 65549                   // packet capture length
	bfdPromiscuous  bool   = false                   // mode
	bfdDedicatedMac string = "01:00:5E:90:00:01"     // Dest MAC perlink packets till neighbor's MAC is learned
	bfdPcapFilter   string = "udp and dst port 6784" // packet capture filter
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type BfdClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type RibdClient struct {
	ipcutils.IPCClientBase
	ClientHdl *ribd.RIBDServicesClient
}

type IpIntfProperty struct {
	IpAddr  net.IP
	NetMask []byte
}

type BfdInterface struct {
	Enabled     bool
	NumSessions int32
	conf        IntfConfig
	property    IpIntfProperty
}

type BfdSessionMgmt struct {
	DestIp    string
	ParamName string
	Interface string
	Protocol  bfddCommonDefs.BfdSessionOwner
	PerLink   bool
	ForceDel  bool
}

type BfdSession struct {
	state                       SessionState
	rxInterval                  int32
	sessionTimer                *time.Timer
	txInterval                  int32
	txTimer                     *time.Timer
	TxTimeoutCh                 chan int32
	txJitter                    int32
	SessionTimeoutCh            chan int32
	bfdPacket                   *BfdControlPacket
	bfdPacketBuf                []byte
	ReceivedPacketCh            chan *BfdControlPacket
	SessionStopClientCh         chan bool
	pollSequence                bool
	pollSequenceFinal           bool
	authEnabled                 bool
	authType                    AuthenticationType
	authSeqNum                  uint32
	authKeyId                   uint32
	authData                    string
	txConn                      net.Conn
	sendPcapHandle              *pcap.Handle
	recvPcapHandle              *pcap.Handle
	useDedicatedMac             bool
	intfConfigChanged           bool
	paramConfigChanged          bool
	stateChanged                bool
	isClientActive              bool
	remoteParamChanged          bool
	switchingToConfiguredTimers bool
	server                      *BFDServer
}

type BfdSessionParam struct {
	state SessionParamState
}

type BfdGlobal struct {
	Enabled                 bool
	NumInterfaces           uint32
	Interfaces              map[int32]*BfdInterface
	InterfacesIdSlice       []int32
	NumSessions             uint32
	Sessions                map[int32]*BfdSession
	SessionsIdSlice         []int32
	InactiveSessionsIdSlice []int32
	NumSessionParams        uint32
	SessionParams           map[string]*BfdSessionParam
	NumUpSessions           uint32
	NumDownSessions         uint32
	NumAdminDownSessions    uint32
}

type RecvedBfdPacket struct {
	IpAddr    string
	Len       int32
	PacketBuf []byte
}

type BFDServer struct {
	logger                *logging.Writer
	ServerStartedCh       chan bool
	ribdClient            RibdClient
	asicdClient           AsicdClient
	GlobalConfigCh        chan GlobalConfig
	asicdSubSocket        *nanomsg.SubSocket
	asicdSubSocketCh      chan []byte
	asicdSubSocketErrCh   chan error
	ribdSubSocket         *nanomsg.SubSocket
	ribdSubSocketCh       chan []byte
	ribdSubSocketErrCh    chan error
	portPropertyMap       map[int32]PortProperty
	vlanPropertyMap       map[int32]VlanProperty
	IPIntfPropertyMap     map[string]IPIntfProperty
	CreateSessionCh       chan BfdSessionMgmt
	DeleteSessionCh       chan BfdSessionMgmt
	AdminUpSessionCh      chan BfdSessionMgmt
	AdminDownSessionCh    chan BfdSessionMgmt
	SessionConfigCh       chan SessionConfig
	CreatedSessionCh      chan int32
	bfddPubSocket         *nanomsg.PubSocket
	lagPropertyMap        map[int32]LagProperty
	notificationCh        chan []byte
	FailedSessionClientCh chan int32
	BfdPacketRecvCh       chan RecvedBfdPacket
	SessionParamConfigCh  chan SessionParamConfig
	SessionParamDeleteCh  chan string
	bfdGlobal             BfdGlobal
}

func NewBFDServer(logger *logging.Writer) *BFDServer {
	bfdServer := &BFDServer{}
	bfdServer.logger = logger
	bfdServer.ServerStartedCh = make(chan bool)
	bfdServer.GlobalConfigCh = make(chan GlobalConfig)
	bfdServer.asicdSubSocketCh = make(chan []byte)
	bfdServer.asicdSubSocketErrCh = make(chan error)
	bfdServer.ribdSubSocketCh = make(chan []byte)
	bfdServer.ribdSubSocketErrCh = make(chan error)
	bfdServer.portPropertyMap = make(map[int32]PortProperty)
	bfdServer.vlanPropertyMap = make(map[int32]VlanProperty)
	bfdServer.lagPropertyMap = make(map[int32]LagProperty)
	bfdServer.SessionConfigCh = make(chan SessionConfig)
	bfdServer.notificationCh = make(chan []byte)
	bfdServer.SessionParamConfigCh = make(chan SessionParamConfig)
	bfdServer.SessionParamDeleteCh = make(chan string)
	bfdServer.bfdGlobal.Enabled = false
	bfdServer.bfdGlobal.NumInterfaces = 0
	bfdServer.bfdGlobal.Interfaces = make(map[int32]*BfdInterface)
	bfdServer.bfdGlobal.InterfacesIdSlice = []int32{}
	bfdServer.bfdGlobal.NumSessions = 0
	bfdServer.bfdGlobal.Sessions = make(map[int32]*BfdSession)
	bfdServer.bfdGlobal.SessionsIdSlice = []int32{}
	bfdServer.bfdGlobal.InactiveSessionsIdSlice = []int32{}
	bfdServer.bfdGlobal.NumSessionParams = 0
	bfdServer.bfdGlobal.SessionParams = make(map[string]*BfdSessionParam)
	bfdServer.bfdGlobal.NumUpSessions = 0
	bfdServer.bfdGlobal.NumDownSessions = 0
	bfdServer.bfdGlobal.NumAdminDownSessions = 0
	return bfdServer
}

func (server *BFDServer) SigHandler(dbHdl redis.Conn) {
	sigChan := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChan, signalList...)

	for {
		select {
		case signal := <-sigChan:
			switch signal {
			case syscall.SIGHUP:
				server.SendAdminDownToAllNeighbors()
				time.Sleep(500 * time.Millisecond)
				server.logger.Info("Sent admin_down to all neighbors")
				server.SendDeleteToAllSessions()
				time.Sleep(500 * time.Millisecond)
				server.logger.Info("Stopped all sessions")
				dbHdl.Close()
				server.logger.Info("Exting!!!")
				os.Exit(0)
			default:
			}
		}
	}
}

func (server *BFDServer) ConnectToServers(paramsFile string) {
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
			server.asicdClient.TTransport, server.asicdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.asicdClient.Address)
			if err != nil {
				server.logger.Info(fmt.Sprintf("Failed to connect to Asicd, retrying until connection is successful"))
				count := 0
				ticker := time.NewTicker(time.Duration(1000) * time.Millisecond)
				for _ = range ticker.C {
					server.asicdClient.TTransport, server.asicdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.asicdClient.Address)
					if err == nil {
						ticker.Stop()
						break
					}
					count++
					if (count % 10) == 0 {
						server.logger.Info("Still can't connect to Asicd, retrying...")
					}
				}
			}
			if server.asicdClient.TTransport != nil && server.asicdClient.PtrProtocolFactory != nil {
				server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.TTransport, server.asicdClient.PtrProtocolFactory)
				server.asicdClient.IsConnected = true
				server.logger.Info("Bfdd is connected to Asicd")
			}
		} else if client.Name == "ribd" {
			server.logger.Info(fmt.Sprintln("found ribd at port", client.Port))
			server.ribdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.ribdClient.TTransport, server.ribdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.ribdClient.Address)
			if err != nil {
				server.logger.Info(fmt.Sprintf("Failed to connect to Ribd, retrying until connection is successful"))
				count := 0
				ticker := time.NewTicker(time.Duration(1000) * time.Millisecond)
				for _ = range ticker.C {
					server.ribdClient.TTransport, server.ribdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.ribdClient.Address)
					if err == nil {
						ticker.Stop()
						break
					}
					count++
					if (count % 10) == 0 {
						server.logger.Info("Still can't connect to Ribd, retrying...")
					}
				}
			}
			if server.ribdClient.TTransport != nil && server.ribdClient.PtrProtocolFactory != nil {
				server.ribdClient.ClientHdl = ribd.NewRIBDServicesClientFactory(server.ribdClient.TTransport, server.ribdClient.PtrProtocolFactory)
				server.ribdClient.IsConnected = true
				server.logger.Info("Bfdd is connected to Ribd")
			}
		}
	}
}

func (server *BFDServer) InitPublisher(pub_str string) (pub *nanomsg.PubSocket) {
	server.logger.Info(fmt.Sprintln("Setting up ", pub_str, "publisher"))
	pub, err := nanomsg.NewPubSocket()
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed to open pub socket"))
		return nil
	}
	ep, err := pub.Bind(pub_str)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed to bind pub socket - ", ep))
		return nil
	}
	err = pub.SetSendBuffer(1024)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed to set send buffer size"))
		return nil
	}
	return pub
}

func (server *BFDServer) PublishSessionNotifications() {
	server.bfddPubSocket = server.InitPublisher(bfddCommonDefs.PUB_SOCKET_ADDR)
	for {
		select {
		case event := <-server.notificationCh:
			_, err := server.bfddPubSocket.Send(event, nanomsg.DontWait)
			if err == syscall.EAGAIN {
				server.logger.Err(fmt.Sprintln("Failed to publish event"))
			}
		}
	}
}

func (server *BFDServer) InitServer(paramFile string) {
	server.logger.Info(fmt.Sprintln("Starting Bfd Server"))
	server.ConnectToServers(paramFile)
	server.initBfdGlobalConfDefault()
	server.BuildPortPropertyMap()
	server.BuildLagPropertyMap()
	server.BuildIPv4InterfacesMap()
	server.createDefaultSessionParam()
}

func (server *BFDServer) StartServer(paramFile string, dbHdl redis.Conn) {
	// Initialize BFD server from params file
	server.InitServer(paramFile)
	// Start subcriber for ASICd events
	go server.CreateASICdSubscriber()
	// Start subcriber for RIBd events
	go server.CreateRIBdSubscriber()
	// Start session management handler
	go server.StartSessionHandler()
	// Initialize and run notification publisher
	go server.PublishSessionNotifications()

	server.ServerStartedCh <- true

	// Now, wait on below channels to process
	for {
		select {
		case gConf := <-server.GlobalConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Global Configuration", gConf))
			server.processGlobalConfig(gConf)
		case asicdrxBuf := <-server.asicdSubSocketCh:
			server.processAsicdNotification(asicdrxBuf)
		case <-server.asicdSubSocketErrCh:
		case ribdrxBuf := <-server.ribdSubSocketCh:
			server.processRibdNotification(ribdrxBuf)
		case <-server.ribdSubSocketErrCh:
		case sessionConfig := <-server.SessionConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Session Configuration", sessionConfig))
			server.processSessionConfig(sessionConfig)
		case sessionParamConfig := <-server.SessionParamConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Session Param Configuration", sessionParamConfig))
			server.processSessionParamConfig(sessionParamConfig)
		case paramName := <-server.SessionParamDeleteCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Session Param Delete", paramName))
			server.processSessionParamDelete(paramName)
		}
	}
}
