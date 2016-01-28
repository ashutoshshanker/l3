// Main entry point for DHCP_RELAY
package relayServer

import (
	"asicdServices"
	"dhcprelayd"
	"encoding/json"
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"golang.org/x/net/ipv4"
	"io/ioutil"
	"log/syslog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"utils/ipcutils"
)

/*
 * Global DS local to DHCP RELAY AGENT
 */
type DhcpRelayServiceHandler struct {
}

// type is similar to typedef in c
type DhcpOptionCode byte
type OpCode byte
type MessageType byte // Option 53

// For Debugging if_name, mac_address, dhcp server ip
type MacAddrServerIpKey string

// Map of DHCP options
type DhcpRelayAgentOptions map[DhcpOptionCode][]byte

// A DHCP packet
type DhcpRelayAgentPacket []byte

/*
 * DhcpRelayAgentStateInfo will maintain state from when a packet was recieved
 * until it is out
 */
type DhcpRelayAgentStateInfo struct {
	stats []string
}

/*
 * Global DRA Data Structure:
 *	    IntfConfig Info
 *	    PCAP Handler specific to Interface
 */
type DhcpRelayAgentGlobalInfo struct {
	IntfConfig           dhcprelayd.DhcpRelayIntfConfig
	dhcprelayConfigMutex sync.RWMutex
}
type Option struct {
	Code  DhcpOptionCode
	Value []byte
}

var (
	// map key would be if_name
	// When we receive a udp packet... we will get interface id and that can
	// be used to collect the global info...
	dhcprelayGblInfo    map[int]DhcpRelayAgentGlobalInfo
	StateDebugInfo      map[MacAddrServerIpKey]DhcpRelayAgentStateInfo
	dhcprelayClientConn *ipv4.PacketConn
	dhcprelayServerConn *ipv4.PacketConn
	logger              *syslog.Writer
	// PadddingToMinimumSize pads a packet so that when sent over UDP,
	// the entire packet, is 300 bytes (which is BOOTP/DHCP min)
	dhcprelayPadder [DHCP_PACKET_MIN_SIZE]byte
)

/******* Local API Calls. *******/

func NewDhcpRelayServer() *DhcpRelayServiceHandler {
	return &DhcpRelayServiceHandler{}
}

func DhcpRelayAgentUpdateStats(input string, info *DhcpRelayAgentGlobalInfo) {
	/*
		info.StateDebugInfo.stats = append(info.StateDebugInfo.stats, input)
		info.dhcprelayConfigMutex.RLock()
		dhcprelayGblInfo[info.IntfConfig.IfIndex] = *info
		info.dhcprelayConfigMutex.RUnlock()
	*/
}

/*
 *  ConnectToClients:
 *	    This API will accept configFile location and from that it will
 *	    connect to clients like asicd, etc..
 */
func DhcpRelayAgentConnectToClients(paramsFile string) error {
	var clientsList []ClientJson
	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA:Error while reading configuration file",
			paramsFile))
		return err
	}
	logger.Info("DRA: Connecting to Clients")
	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		logger.Err("DRA: Error in Unmarshalling Json")
		return err
	}

	// Start connection.. to only to those which is needed as of now
	for _, client := range clientsList {
		logger.Info(fmt.Sprintln("DRA: Client name is", client.Name))
		if client.Name == "asicd" {
			logger.Info(fmt.Sprintln("DRA: Connecting to asicd at port",
				client.Port))
			asicdClient.Address = "localhost:" +
				strconv.Itoa(client.Port)
			asicdClient.Transport,
				asicdClient.PtrProtocolFactory, err =
				ipcutils.CreateIPCHandles(asicdClient.Address)
			if asicdClient.Transport == nil ||
				asicdClient.PtrProtocolFactory == nil ||
				err != nil {
				logger.Err(fmt.Sprintln("DRA: Connecting to",
					client.Name, "failed ", err))
				return err
			}
			asicdClient.ClientHdl =
				asicdServices.NewASICDServicesClientFactory(
					asicdClient.Transport,
					asicdClient.PtrProtocolFactory)
			asicdClient.IsConnected = true
			logger.Info("DRA: is connected to asicd")
		}
	}
	logger.Info("DRA: successfully connected to clients")
	return nil
}

/*
 * DhcpRelaySignalHandler:
 *	This API will catch any os signals for DRA and if the signal is of
 *	SIGHUP type then it exit the process
 */
func DhcpRelaySignalHandler(sigChannel <-chan os.Signal) {
	signal := <-sigChannel // receive from sigChannel and assign it to signal
	switch signal {
	case syscall.SIGHUP:
		logger.Alert("DRA: Received SIGHUP SIGNAL")
		// @TODO: jgheewala clean up stuff on exit...
		os.Exit(0)
	default:
		logger.Info(fmt.Sprintln("DRA: Unhandled Signal : ", signal))
	}

}

func DhcpRelayAgentOSSignalHandle() {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	sigChannel := make(chan os.Signal, 1)
	// SIGHUP is a signal sent to a process when its controlling terminal is
	// closed and we need to handle that signal
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChannel, signalList...)
	// start a light weighted thread goroutine for signal handler
	go DhcpRelaySignalHandler(sigChannel)
}

/*
 *  InitDhcpRelayPktHandler:
 *	    This API is used to initialize all the data structures varialbe that
 *	    is needed by relay agent to perform its operation
 */
func InitDhcpRelayPortPktHandler() error {
	// Init port configs
	//portInfoMap = make(map[string]int, 30) //map[]portInfo)

	// connecting to asicd
	params_dir := flag.String("params", "",
		"Directory Location for config files")
	flag.Parse()
	configFile := *params_dir + "/clients.json"
	logger.Info(fmt.Sprintln("DRA: configFile is ", configFile))
	// connect to client
	err := DhcpRelayAgentConnectToClients(configFile)
	if err != nil {
		return err
	}
	// OS signal channel listener thread
	DhcpRelayAgentOSSignalHandle()

	// Initialize port parameters
	err = DhcpRelayInitPortParams()
	if err != nil {
		logger.Err("DRA: initializing port paramters failed")
		return err
	}

	return nil
}

//func DhcpRelayAgentInitGblHandling(ifName string, ifNum int) {
func DhcpRelayAgentInitGblHandling(ifNum int) {
	logger.Info("DRA: Initializaing Global Info for " + strconv.Itoa(ifNum))
	// Created a global Entry for Interface
	gblEntry := dhcprelayGblInfo[ifNum]
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IpSubnet = ""
	gblEntry.IntfConfig.Netmask = ""
	gblEntry.IntfConfig.IfIndex = strconv.Itoa(ifNum) //ifName
	gblEntry.IntfConfig.AgentSubType = 0
	gblEntry.IntfConfig.Enable = false
	gblEntry.dhcprelayConfigMutex = sync.RWMutex{}
	// Stats information
	//gblEntry.StateDebugInfo = make(map[MacAddrServerIpKey]DhcpRelayAgentStateInfo, 150)
	dhcprelayGblInfo[ifNum] = gblEntry

}

func StartServer(log *syslog.Writer, handler *DhcpRelayServiceHandler, addr string) error {
	logger = log
	// Initialize port information and packet handler for dhcp

	err := InitDhcpRelayPortPktHandler()
	if err != nil {
		return err
	}

	// create transport and protocol for server
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("DRA: StartServer: NewTServerSocket "+
			"failed with error:", err))
		return err
	}
	processor := dhcprelayd.NewDHCPRELAYDServicesProcessor(handler)
	fmt.Printf("Starting DHCP-RELAY daemon at %s\n", addr)
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to start the listener, err:", err))
		return err
	}

	logger.Info("DRA:Start the Server successfully")
	return nil
}
