// Main entry point for DHCP_RELAY
package relayServer

import (
	_ "asicd/asicdConstDefs"
	"asicdServices"
	"dhcprelayd"
	"encoding/json"
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	_ "github.com/google/gopacket/pcap"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"log/syslog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"utils/ipcutils"
)

/*
 * Global DS local to DHCP RELAY AGENT
 */
type DhcpRelayServiceHandler struct {
}

/*
 * DhcpRelayAgentStateInfo will maintain state from when a packet was recieved
 * until it is out
 */
type DhcpRelayAgentStateInfo struct {
	initDone string
}

/*
 * Global DRA Data Structure:
 *	    IntfConfig Info
 *	    PCAP Handler specific to Interface
 */
type DhcpRelayAgentGlobalInfo struct {
	IntfConfig     dhcprelayd.DhcpRelayIntfConfig
	StateDebugInfo DhcpRelayAgentStateInfo
}

var (
	// map key would be if_name
	dhcprelayGblInfo    map[int]DhcpRelayAgentGlobalInfo
	asicdSubSocket      *nanomsg.SubSocket
	logger              *syslog.Writer
	asicdSubSocketCh    chan []byte = make(chan []byte)
	asicdSubSocketErrCh chan error  = make(chan error)
)

/******* Local API Calls. *******/

func NewDhcpRelayServer() *DhcpRelayServiceHandler {
	return &DhcpRelayServiceHandler{}
}

// @TODO: cleanup if not needed....
func DhcpRelayAgentUpdateHandler() {
	for {
		select {
		case rxBuf := <-asicdSubSocketCh:
			dhcpRelayAgentProcessAsicdNotification(rxBuf)
		case <-asicdSubSocketErrCh:

		}
	}
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
		logger.Err(fmt.Sprintln("Error while reading configuration file",
			paramsFile))
		return err
	}
	logger.Info("DRA: Connecting to Clients")
	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		logger.Err("Error in Unmarshalling Json")
		return err
	}

	// Start connection.. to only to those which is needed as of now
	for _, client := range clientsList {
		logger.Info(fmt.Sprintln("DRA: Client name is", client.Name))
		if client.Name == "asicd" {
			logger.Info(fmt.Sprintln("Connecting to asicd at port",
				client.Port))
			asicdClient.Address = "localhost:" +
				strconv.Itoa(client.Port)
			asicdClient.Transport,
				asicdClient.PtrProtocolFactory, _ =
				ipcutils.CreateIPCHandles(asicdClient.Address)
			if asicdClient.Transport == nil ||
				asicdClient.PtrProtocolFactory == nil {
				logger.Err(fmt.Sprintln("Connecting to",
					client.Name+"failed"))
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

func DhcpRelayAgentListenForASICUpdate(address string) error {
	logger.Info("DRA: Setting up relay agent for Asic Update")
	var err error
	if asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		logger.Info(fmt.Sprintln("Failed to create ASIC subscribe "+
			"socket, error:", err))
		return err
	}

	if err = asicdSubSocket.Subscribe(""); err != nil {
		logger.Info(fmt.Sprintln("Failed to subscribe to \"\" on "+
			"ASIC subscribe socket, error:", err))
		return err
	}

	if _, err = asicdSubSocket.Connect(address); err != nil {
		logger.Err(fmt.Sprintln("Failed to connect to ASIC "+
			"publisher socket, address:", address, "error:", err))
		return err
	}

	logger.Info(fmt.Sprintln("Connected to ASIC publisher at address:",
		address))
	if err = asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		logger.Info(fmt.Sprintln("Failed to set the buffer size "+
			"for ASIC publisher socket, error:", err))
		return err
	}
	logger.Info("DRA: relay agent set for Asic Update successfully")
	return nil

}

// @TODO: Not used right now... clean it up later
func DhcpRelayAgentAsicdSubscriber() {
	for {
		logger.Info("DRA: Read on Asic subscriber socket...")
		rxBuf, err := asicdSubSocket.Recv(0)
		if err != nil {
			logger.Err(fmt.Sprintln("Recv on Asicd subscriber "+
				"socket failed with error:", err))
			asicdSubSocketErrCh <- err
			continue
		}
		logger.Info(fmt.Sprintln("Asicd subscriber recv returned:",
			rxBuf))
		asicdSubSocketCh <- rxBuf
	}
}

/*
 *  InitDhcpRelayPktHandler:
 *	    This API is used to initialize all the data structures varialbe that
 *	    is needed by relay agent to perform its operation
 */
func InitDhcpRelayPortPktHandler() error {
	// Init port configs
	portInfoMap = make(map[int]portInfo)

	// connecting to asicd
	params_dir := flag.String("params", "",
		"Directory Location for config files")
	flag.Parse()
	configFile := *params_dir + "/clients.json"
	logger.Info(fmt.Sprintln("configFile is ", configFile))
	// connect to client
	err := DhcpRelayAgentConnectToClients(configFile)
	if err != nil {
		return err
	}
	// @TODO: jgheewala: Do we need update handler...???
	//go DhcpRelayAgentUpdateHandler()
	// OS signal channel listener thread
	DhcpRelayAgentOSSignalHandle()
	// @TODO: jgheewala... DO we need a routine to listen to intf state
	// change???
	/*
		DhcpRelayAgentListenForASICUpdate(pluginCommon.PUB_SOCKET_ADDR)
		if err == nil {
			// asicd update listerner thread
			go DhcpRelayAgentAsicdSubscriber()
		}
	*/
	// Initialize port parameters
	err = DhcpRelayInitPortParams()
	if err != nil {
		logger.Err("DRA: initializing port paramters failed")
		return err
	}

	return nil
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
		logger.Info(fmt.Sprintln("StartServer: NewTServerSocket "+
			"failed with error:", err))
		return err
	}
	fmt.Println("%T", transport)
	processor := dhcprelayd.NewDHCPRELAYDServicesProcessor(handler)
	fmt.Printf("%T\n", transportFactory)
	fmt.Printf("%T\n", protocolFactory)
	fmt.Printf("Starting DHCP-RELAY daemon at %s\n", addr)
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to start the listener, err:", err))
		return err
	}

	logger.Info(fmt.Sprintln("Start the Server successfully"))
	return nil
}
