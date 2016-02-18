package vrrpServer

import (
	_ "asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"log/syslog"
	"strconv"
	"time"
	"utils/ipcutils"
	"vrrpd"
)

func NewVrrpServer() *VrrpServiceHandler {
	return &VrrpServiceHandler{}
}

/*
	// The initial value is the same as Advertisement_Interval.
	MasterAdverInterval int32
	// (((256 - priority) * Master_Adver_Interval) / 256)
	SkewTime int32
	// (3 * Master_Adver_Interval) + Skew_time
	MasterDownInterval int32
	// IfIndex IpAddr which needs to be used if no Virtual Ip is specified
	IpAddr string
*/

func VrrpInitGblInfo(IfIndex int32, IfName string, IpAddr string) {
	gblInfo := vrrpGblInfo[IfIndex]
	gblInfo.MasterAdverInterval = 0
	gblInfo.SkewTime = 0
	gblInfo.MasterDownInterval = 0
	gblInfo.IpAddr = IpAddr
	gblInfo.IfName = IfName
	vrrpGblInfo[IfIndex] = gblInfo
}

func VrrpUpdateGblInfo(IfIndex int32) {
	gblInfo := vrrpGblInfo[IfIndex]
	gblInfo.MasterAdverInterval = gblInfo.IntfConfig.AdvertisementInterval
	if gblInfo.IntfConfig.Priority != 0 && gblInfo.MasterAdverInterval != 0 {
		gblInfo.SkewTime = ((256 - gblInfo.IntfConfig.Priority) * gblInfo.MasterAdverInterval) / 256
	}
	gblInfo.MasterDownInterval = (3 * gblInfo.MasterAdverInterval) + gblInfo.SkewTime
	vrrpGblInfo[IfIndex] = gblInfo
}

func VrrpConnectToAsicd(client VrrpClientJson) error {
	logger.Info(fmt.Sprintln("VRRP: Connecting to asicd at port",
		client.Port))
	var err error
	asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
	asicdClient.Transport, asicdClient.PtrProtocolFactory, err =
		ipcutils.CreateIPCHandles(asicdClient.Address)
	if asicdClient.Transport == nil ||
		asicdClient.PtrProtocolFactory == nil ||
		err != nil {
		logger.Err(fmt.Sprintln("VRRP: Connecting to",
			client.Name, "failed ", err))
		return err
	}
	asicdClient.ClientHdl =
		asicdServices.NewASICDServicesClientFactory(
			asicdClient.Transport,
			asicdClient.PtrProtocolFactory)
	asicdClient.IsConnected = true
	return nil
}

func VrrpConnectToUnConnectedClient(client VrrpClientJson) error {
	switch client.Name {
	case "asicd":
		return VrrpConnectToAsicd(client)
	default:
		return errors.New(VRRP_CLIENT_CONNECTION_NOT_REQUIRED)
	}
}

func VrrpConnectAndInitPortVlan() error {

	configFile := paramsDir + "/clients.json"
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP:Error while reading configuration file",
			configFile))
		return err
	}
	var unConnectedClients []VrrpClientJson
	err = json.Unmarshal(bytes, &unConnectedClients)
	if err != nil {
		logger.Err("VRRP: Error in Unmarshalling Json")
		return err
	}

	// connect to client
	for {
		time.Sleep(time.Millisecond * 500)
		for i := 0; i < len(unConnectedClients); i++ {
			err := VrrpConnectToUnConnectedClient(unConnectedClients[i])
			if err == nil {
				logger.Info("VRRP: Connected to " +
					unConnectedClients[i].Name)
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)

			} else if err.Error() == VRRP_CLIENT_CONNECTION_NOT_REQUIRED {
				logger.Info("VRRP: connection to " + unConnectedClients[i].Name +
					" not required")
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)
			}
		}
		if len(unConnectedClients) == 0 {
			logger.Info("VRRP: all clients connected successfully")
			break
		}
	}

	VrrpGetInfoFromAsicd()
	return err
}

func VrrpAllocateMemoryToGlobalDS() {
	vrrpGblInfo = make(map[int32]VrrpGlobalInfo, 10)
}

func StartServer(log *syslog.Writer, handler *VrrpServiceHandler, addr string) error {
	logger = log
	logger.Info("VRRP: allocating memory to global ds")

	// Allocate memory to all the Data Structures
	VrrpAllocateMemoryToGlobalDS()

	params := flag.String("params", "", "Directory Location for config files")
	flag.Parse()
	paramsDir = *params

	// Initialize DB
	err := VrrpInitDB()
	if err != nil {
		logger.Err("VRRP: DB init failed")
	} else {
		VrrpReadDB()
	}

	go VrrpConnectAndInitPortVlan()

	// create transport and protocol for server
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("VRRP: StartServer: NewTServerSocket "+
			"failed with error:", err))
		return err
	}
	processor := vrrpd.NewVRRPDServicesProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Failed to start the listener, err:", err))
		return err
	}
	return nil
}
