package vrrpServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"log/syslog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"utils/ipcutils"
	"vrrpd"
)

func NewVrrpServer() *VrrpServiceHandler {
	return &VrrpServiceHandler{}
}

func VrrpDumpIntfInfo(gblInfo VrrpGlobalInfo) {
	logger.Info(fmt.Sprintln("VRID:", gblInfo.IntfConfig.VRID))
	logger.Info(fmt.Sprintln("IpAddr:", gblInfo.IpAddr))
	logger.Info(fmt.Sprintln("IfIndex:", gblInfo.IntfConfig.IfIndex))
	logger.Info(fmt.Sprintln("Priority:", gblInfo.IntfConfig.Priority))
	logger.Info(fmt.Sprintln("Preempt Mode:", gblInfo.IntfConfig.PreemptMode))
	logger.Info(fmt.Sprintln("Virt Mac Addr:", gblInfo.IntfConfig.VirtualRouterMACAddress))
	logger.Info(fmt.Sprintln("VirtualIPv4Addr:", gblInfo.IntfConfig.VirtualIPv4Addr))
	logger.Info(fmt.Sprintln("AdvertisementTime:", gblInfo.IntfConfig.AdvertisementInterval))
	logger.Info(fmt.Sprintln("MasterAdverInterval:", gblInfo.MasterAdverInterval))
	logger.Info(fmt.Sprintln("Skew Time:", gblInfo.SkewTime))
	logger.Info(fmt.Sprintln("Master Down Interval:", gblInfo.MasterDownInterval))
}

func VrrpUpdateIntfIpAddr(gblInfo *VrrpGlobalInfo) bool {
	IpAddr, ok := vrrpIfIndexIpAddr[gblInfo.IntfConfig.IfIndex]
	if ok == false {
		logger.Err(fmt.Sprintln("missed ipv4 intf notification for IfIndex:",
			gblInfo.IntfConfig.IfIndex))
		return false
	}
	gblInfo.IpAddr = IpAddr
	return true
}

func VrrpPopulateIntfState(key string, entry *vrrpd.VrrpIntfState) {
	gblInfo, ok := vrrpGblInfo[key]
	if ok == false {
		logger.Err(fmt.Sprintln("Entry not found for", key))
		return
	}
	entry.IfIndex = gblInfo.IntfConfig.IfIndex
	entry.VRID = gblInfo.IntfConfig.VRID
	entry.IntfIpAddr = gblInfo.IpAddr
	entry.Priority = gblInfo.IntfConfig.Priority
	entry.VirtualIPv4Addr = gblInfo.IntfConfig.VirtualIPv4Addr
	entry.AdvertisementInterval = gblInfo.IntfConfig.AdvertisementInterval
	entry.PreemptMode = gblInfo.IntfConfig.PreemptMode
	entry.VirtualRouterMACAddress = gblInfo.IntfConfig.VirtualRouterMACAddress
	entry.SkewTime = gblInfo.SkewTime
	entry.MasterDownInterval = gblInfo.MasterDownInterval
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

func VrrpUpdateGblInfoTimers(key string) {
	gblInfo := vrrpGblInfo[key]
	gblInfo.MasterAdverInterval = gblInfo.IntfConfig.AdvertisementInterval
	if gblInfo.IntfConfig.Priority != 0 && gblInfo.MasterAdverInterval != 0 {
		gblInfo.SkewTime = ((256 - gblInfo.IntfConfig.Priority) *
			gblInfo.MasterAdverInterval) / 256
	}
	gblInfo.MasterDownInterval = (3 * gblInfo.MasterAdverInterval) + gblInfo.SkewTime

	if ok := VrrpUpdateIntfIpAddr(&gblInfo); ok == false {
		// If we miss Asic Notification then do one time get bulk for Ipv4
		// Interface... Once done then update Ip Addr again
		logger.Err("recalling get ipv4interface list")
		VrrpGetIPv4IntfList()
		VrrpUpdateIntfIpAddr(&gblInfo)
	}
	vrrpGblInfo[key] = gblInfo
	vrrpIntfStateSlice = append(vrrpIntfStateSlice, key)
	//VrrpDumpIntfInfo(gblInfo)
}

func VrrpMapIfIndexToLinuxIfIndex(IfIndex int32) {
	vlanId := asicdConstDefs.GetIntfIdFromIfIndex(IfIndex)
	vlanName, ok := vrrpVlanId2Name[vlanId]
	if ok == false {
		logger.Err(fmt.Sprintln("no mapping for vlan", vlanId))
		return
	}
	linuxInterface, err := net.InterfaceByName(vlanName)
	if err != nil {
		logger.Err(fmt.Sprintln("Getting linux If index for",
			"IfIndex:", IfIndex, "failed with ERROR:", err))
		return
	}
	logger.Info(fmt.Sprintln("Linux Id:", linuxInterface.Index,
		"maps to IfIndex:", IfIndex))
	//entry := vrrpLinuxIfIndex2AsicdIfIndex[linuxInterface.Index]
	//entry = IfIndex
	vrrpLinuxIfIndex2AsicdIfIndex[IfIndex] = linuxInterface
	//tempGblLinux = linuxInterface
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

func VrrpCloseAllPcapHandlers() {
	for i := 0; i < len(vrrpIntfStateSlice); i++ {
		key := vrrpIntfStateSlice[i]
		gblInfo := vrrpGblInfo[key]
		gblInfo.pHandle.Close()
	}
}

func VrrpSignalHandler(sigChannel <-chan os.Signal) {
	signal := <-sigChannel
	switch signal {
	case syscall.SIGHUP:
		logger.Alert("Received SIGHUP Signal")
		VrrpCloseAllPcapHandlers()
		VrrpDeAllocateMemoryToGlobalDS()
		logger.Info("Closed vrrp pkt handlers")
		os.Exit(0)
	default:
		logger.Info(fmt.Sprintln("Unhandled Signal:", signal))
	}
}

func VrrpOSSignalHandle() {
	sigChannel := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChannel, signalList...)
	go VrrpSignalHandler(sigChannel)
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

	// OS Signal channel listener thread
	VrrpOSSignalHandle()
	return err
}

func VrrpAllocateMemoryToGlobalDS() {
	vrrpGblInfo = make(map[string]VrrpGlobalInfo,
		VRRP_GLOBAL_INFO_DEFAULT_SIZE)
	vrrpIfIndexIpAddr = make(map[int32]string,
		VRRP_INTF_IPADDR_MAPPING_DEFAULT_SIZE)
	vrrpLinuxIfIndex2AsicdIfIndex = make(map[int32]*net.Interface,
		VRRP_LINUX_INTF_MAPPING_DEFAULT_SIZE)
	vrrpVlanId2Name = make(map[int]string,
		VRRP_VLAN_MAPPING_DEFAULT_SIZE)
}

func VrrpDeAllocateMemoryToGlobalDS() {
	vrrpGblInfo = nil
	vrrpIfIndexIpAddr = nil
	vrrpLinuxIfIndex2AsicdIfIndex = nil
	vrrpVlanId2Name = nil
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
