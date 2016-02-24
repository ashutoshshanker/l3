package server

import (
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"database/sql"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"l3/bfd/bfddCommonDefs"
	"log/syslog"
	"net"
	"ribd"
	"strconv"
	"time"
	"utils/ipcutils"
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
	BfdClientBase
	ClientHdl *ribd.RouteServiceClient
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
	DestIp   string
	Protocol bfddCommonDefs.BfdSessionOwner
}

type BfdSession struct {
	state             SessionState
	sessionTimer      *time.Timer
	txTimer           *time.Timer
	TxTimeoutCh       chan int32
	SessionTimeoutCh  chan int32
	bfdPacket         *BfdControlPacket
	SessionDeleteCh   chan bool
	pollSequence      bool
	pollSequenceFinal bool
	authEnabled       bool
	authType          AuthenticationType
	authSeqNum        uint32
	authKeyId         uint32
	authData          string
}

type BfdGlobal struct {
	Enabled              bool
	NumInterfaces        uint32
	Interfaces           map[int32]*BfdInterface
	InterfacesIdSlice    []int32
	NumSessions          uint32
	Sessions             map[int32]*BfdSession
	SessionsIdSlice      []int32
	NumUpSessions        uint32
	NumDownSessions      uint32
	NumAdminDownSessions uint32
}

type BFDServer struct {
	logger              *syslog.Writer
	ribdClient          RibdClient
	asicdClient         AsicdClient
	GlobalConfigCh      chan GlobalConfig
	IntfConfigCh        chan IntfConfig
	IntfConfigDeleteCh  chan int32
	asicdSubSocket      *nanomsg.SubSocket
	asicdSubSocketCh    chan []byte
	asicdSubSocketErrCh chan error
	portPropertyMap     map[int32]PortProperty
	vlanPropertyMap     map[int32]VlanProperty
	IPIntfPropertyMap   map[string]IPIntfProperty
	CreateSessionCh     chan BfdSessionMgmt
	DeleteSessionCh     chan BfdSessionMgmt
	AdminUpSessionCh    chan BfdSessionMgmt
	AdminDownSessionCh  chan BfdSessionMgmt
	SessionConfigCh     chan SessionConfig
	bfddPubSocket       *nanomsg.PubSocket
	bfdGlobal           BfdGlobal
}

func NewBFDServer(logger *syslog.Writer) *BFDServer {
	bfdServer := &BFDServer{}
	bfdServer.logger = logger
	bfdServer.GlobalConfigCh = make(chan GlobalConfig)
	bfdServer.IntfConfigCh = make(chan IntfConfig)
	bfdServer.IntfConfigDeleteCh = make(chan int32)
	bfdServer.asicdSubSocketCh = make(chan []byte)
	bfdServer.asicdSubSocketErrCh = make(chan error)
	bfdServer.portPropertyMap = make(map[int32]PortProperty)
	bfdServer.vlanPropertyMap = make(map[int32]VlanProperty)
	bfdServer.SessionConfigCh = make(chan SessionConfig)
	bfdServer.bfdGlobal.Enabled = false
	bfdServer.bfdGlobal.NumInterfaces = 0
	bfdServer.bfdGlobal.Interfaces = make(map[int32]*BfdInterface)
	bfdServer.bfdGlobal.InterfacesIdSlice = []int32{}
	bfdServer.bfdGlobal.NumSessions = 0
	bfdServer.bfdGlobal.Sessions = make(map[int32]*BfdSession)
	bfdServer.bfdGlobal.SessionsIdSlice = []int32{}
	bfdServer.bfdGlobal.NumUpSessions = 0
	bfdServer.bfdGlobal.NumDownSessions = 0
	bfdServer.bfdGlobal.NumAdminDownSessions = 0
	return bfdServer
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
			server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.asicdClient.Address)
			if err != nil {
				server.logger.Info(fmt.Sprintf("Failed to connect to Asicd, retrying until connection is successful"))
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
						server.logger.Info("Still can't connect to Asicd, retrying...")
					}
				}
			}
			if server.asicdClient.Transport != nil && server.asicdClient.PtrProtocolFactory != nil {
				server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
				server.asicdClient.IsConnected = true
				server.logger.Info("Bfdd is connected to Asicd")
			}
		} else if client.Name == "ribd" {
			server.logger.Info(fmt.Sprintln("found ribd at port", client.Port))
			server.ribdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory, err = ipcutils.CreateIPCHandles(server.ribdClient.Address)
			if err != nil {
				server.logger.Info(fmt.Sprintf("Failed to connect to Ribd, retrying until connection is successful"))
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
						server.logger.Info("Still can't connect to Ribd, retrying...")
					}
				}
			}
			if server.ribdClient.Transport != nil && server.ribdClient.PtrProtocolFactory != nil {
				server.ribdClient.ClientHdl = ribd.NewRouteServiceClientFactory(server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory)
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

func (server *BFDServer) ReadGlobalConfigFromDB(dbHdl *sql.DB) error {
	server.logger.Info("Reading BfdGlobalConfig")
	dbCmd := "SELECT * FROM BfdGlobalConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to query DB - BfdGlobalConfig: ", err))
		dbHdl.Close()
		return err
	}

	for rows.Next() {
		var rtrBfd string
		var enable int
		var enableBool bool
		err = rows.Scan(&rtrBfd, &enable)
		if err != nil {
			server.logger.Info(fmt.Sprintln("Unable to scan entries from DB - BfdGlobalConfig: ", err))
			dbHdl.Close()
			return err
		}
		if enable == 1 {
			enableBool = true
		} else {
			enableBool = false
		}
		server.logger.Info(fmt.Sprintln("BfdGlobalConfig - Enable: ", enableBool))
		if !enableBool {
			gConf := GlobalConfig{
				Enable: enableBool,
			}
			server.GlobalConfigCh <- gConf
		}
	}
	return nil
}

func (server *BFDServer) ReadIntfConfigFromDB(dbHdl *sql.DB) error {
	server.logger.Info("Reading BfdIntfConfig")
	dbCmd := "SELECT * FROM BfdIntfConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to query DB - BfdIntfConfig: ", err))
		dbHdl.Close()
		return err
	}

	for rows.Next() {
		var interfaceId int32
		var localMultiplier int32
		var desiredMinTxInterval int32
		var requiredMinRxInterval int32
		var requiredMinEchoRxInterval int32
		var demandEnabled int
		var demandEnabledBool bool
		var authenticationEnabled int
		var authenticationEnabledBool bool
		var authenticationType string
		var authenticationKeyId int32
		var authenticationData string
		err = rows.Scan(&interfaceId, &localMultiplier, &desiredMinTxInterval, &requiredMinRxInterval, &requiredMinEchoRxInterval, &demandEnabled, &authenticationEnabled, &authenticationType, &authenticationKeyId, &authenticationData)
		if err != nil {
			server.logger.Info(fmt.Sprintln("Unable to scan entries from DB - BfdIntfConfig: ", err))
			dbHdl.Close()
			return err
		}
		if demandEnabled == 1 {
			demandEnabledBool = true
		} else {
			demandEnabledBool = false
		}
		if demandEnabled == 1 {
			demandEnabledBool = true
		} else {
			demandEnabledBool = false
		}
		ifConf := IntfConfig{
			InterfaceId:               interfaceId,
			LocalMultiplier:           localMultiplier,
			DesiredMinTxInterval:      desiredMinTxInterval,
			RequiredMinRxInterval:     requiredMinRxInterval,
			RequiredMinEchoRxInterval: requiredMinEchoRxInterval,
			DemandEnabled:             demandEnabledBool,
			AuthenticationEnabled:     authenticationEnabledBool,
			AuthenticationType:        server.ConvertBfdAuthTypeStrToVal(authenticationType),
			AuthenticationKeyId:       authenticationKeyId,
			AuthenticationData:        authenticationData,
		}
		server.logger.Info(fmt.Sprintln("BfdIntfConfig - ", ifConf))
		server.IntfConfigCh <- ifConf
	}
	return nil
}

func (server *BFDServer) ReadSessionConfigFromDB(dbHdl *sql.DB) error {
	server.logger.Info("Reading BfdSessionConfig")
	dbCmd := "SELECT * FROM BfdSessionConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to query DB - BfdSessionConfig: ", err))
		dbHdl.Close()
		return err
	}

	for rows.Next() {
		var dstIp string
		var perLink int
		var perLinkBool bool
		var owner string
		var operation string
		err = rows.Scan(&dstIp, &owner, &operation)
		if err != nil {
			server.logger.Info(fmt.Sprintln("Unable to scan entries from DB - BfdSessionConfig: ", err))
			dbHdl.Close()
			return err
		}
		if perLink == 1 {
			perLinkBool = true
		} else {
			perLinkBool = false
		}
		sessionConf := SessionConfig{
			DestIp:    dstIp,
			PerLink:   perLinkBool,
			Protocol:  bfddCommonDefs.USER,
			Operation: bfddCommonDefs.CREATE,
		}
		server.logger.Info(fmt.Sprintln("BfdSessionConfig : ", sessionConf))
		server.SessionConfigCh <- sessionConf
	}
	return nil
}

func (server *BFDServer) ReadConfigFromDB(dbHdl *sql.DB) error {
	var err error
	// BfdGlobalConfig
	err = server.ReadGlobalConfigFromDB(dbHdl)
	if err != nil {
		return err
	}
	// BfdIntfConfig
	err = server.ReadIntfConfigFromDB(dbHdl)
	if err != nil {
		return err
	}
	// BfdSessionConfig
	err = server.ReadSessionConfigFromDB(dbHdl)
	if err != nil {
		return err
	}
	dbHdl.Close()
	return nil
}

func (server *BFDServer) InitServer(paramFile string) {
	server.logger.Info(fmt.Sprintln("Starting Bfd Server"))
	server.ConnectToServers(paramFile)
	server.initBfdGlobalConfDefault()
	server.BuildPortPropertyMap()
	server.GetIPv4Interfaces()
	/*
		server.logger.Info("Listen for RIBd updates")
		server.listenForRIBUpdates(ribdCommonDefs.PUB_SOCKET_ADDR)
		go createRIBSubscriber()
		server.connRoutesTimer.Reset(time.Duration(10) * time.Second)
	*/
}

func (server *BFDServer) StartServer(paramFile string, dbHdl *sql.DB) {
	server.InitServer(paramFile)
	server.logger.Info("Listen for ASICd updates")
	server.listenForASICdUpdates(pluginCommon.PUB_SOCKET_ADDR)
	go server.createASICdSubscriber()
	// Start session handler
	go server.StartSessionHandler()
	// Initialize publisher
	server.bfddPubSocket = server.InitPublisher(bfddCommonDefs.PUB_SOCKET_ADDR)
	go server.ReadConfigFromDB(dbHdl)
	for {
		select {
		case gConf := <-server.GlobalConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Global Configuration", gConf))
			server.processGlobalConfig(gConf)
		case ifConf := <-server.IntfConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Intf Configuration", ifConf))
			server.processIntfConfig(ifConf)
		case ifIndex := <-server.IntfConfigDeleteCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Intf delete", ifIndex))
			server.processIntfConfigDelete(ifIndex)
		case asicdrxBuf := <-server.asicdSubSocketCh:
			server.processAsicdNotification(asicdrxBuf)
		case <-server.asicdSubSocketErrCh:
		case sessionConfig := <-server.SessionConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Session Configuration", sessionConfig))
			server.processSessionConfig(sessionConfig)
			/*
				case ribrxBuf := <-server.ribSubSocketCh:
					server.processRibdNotification(ribdrxBuf)
				case <-server.connRoutesTimer.C:
					routes, _ := server.ribdClient.ClientHdl.GetConnectedRoutesInfo()
					server.logger.Info(fmt.Sprintln("Received Connected Routes:", routes))
					//server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))
					//server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

				case <-server.ribSubSocketErrCh:
			*/
		}
	}
}
