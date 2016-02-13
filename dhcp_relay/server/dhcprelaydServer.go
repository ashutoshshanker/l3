// Main entry point for DHCP_RELAY
package relayServer

import (
	"asicdServices"
	"dhcprelayd"
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
)

/******* Local API Calls. *******/

func NewDhcpRelayServer() *DhcpRelayServiceHandler {
	return &DhcpRelayServiceHandler{}
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

func DhcpRelayConnectToAsicd(client ClientJson) error {
	logger.Info(fmt.Sprintln("DRA: Connecting to asicd at port",
		client.Port))
	var err error
	asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
	asicdClient.Transport, asicdClient.PtrProtocolFactory, err =
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
	return nil
}

/*
 *  ConnectToClients:
 *	    This API will accept configFile location and from that it will
 *	    connect to clients like asicd, etc..
 */
func DhcpRelayAgentConnectToClients(client ClientJson) error {
	logger.Info(fmt.Sprintln("DRA: Client name is", client.Name))
	switch client.Name {
	case "asicd":
		return DhcpRelayConnectToAsicd(client)
	default:
		return errors.New(CLIENT_CONNECTION_NOT_REQUIRED)
	}
}

/*
 *  InitDhcpRelayPktHandler:
 *	    This API is used to initialize all the data structures varialbe that
 *	    is needed by relay agent to perform its operation
 */
func InitDhcpRelayPortPktHandler() error {
	// connecting to asicd
	params_dir := flag.String("params", "",
		"Directory Location for config files")
	flag.Parse()
	configFile := *params_dir + "/clients.json"
	logger.Info(fmt.Sprintln("DRA: configFile is ", configFile))
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA:Error while reading configuration file",
			configFile))
		return err
	}
	var unConnectedClients []ClientJson
	err = json.Unmarshal(bytes, &unConnectedClients)
	if err != nil {
		logger.Err("DRA: Error in Unmarshalling Json")
		return err
	}

	logger.Info("DRA: Connecting to Clients")
	// connect to client
	for {
		time.Sleep(time.Millisecond * 500)
		for i := 0; i < len(unConnectedClients); i++ {
			err := DhcpRelayAgentConnectToClients(unConnectedClients[i])
			if err == nil {
				logger.Info("DRA: Connected to " +
					unConnectedClients[i].Name)
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)

			} else if err.Error() == CLIENT_CONNECTION_NOT_REQUIRED {
				logger.Info("DRA: connection to " + unConnectedClients[i].Name +
					" not required")
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)
			}
		}
		if len(unConnectedClients) == 0 {
			logger.Info("DRA: all clients connected successfully")
			break
		}
	}
	// Initialize port parameters after client is connected
	err = DhcpRelayInitPortParams()
	if err != nil {
		logger.Err("DRA: initializing port paramters failed")
		return err
	}
	// OS signal channel listener thread
	DhcpRelayAgentOSSignalHandle()

	return nil
}

func DhcpRelayAgentInitIntfServerState(serverIp string, id int32) {
	IntfId := int(id)
	key := strconv.Itoa(IntfId) + "_" + serverIp
	intfServerEntry := dhcprelayIntfServerStateMap[key]
	intfServerEntry.IntfId = id
	intfServerEntry.ServerIp = serverIp
	intfServerEntry.Request = 0
	intfServerEntry.Responses = 0
	dhcprelayIntfServerStateMap[key] = intfServerEntry
	dhcprelayIntfServerStateSlice = append(dhcprelayIntfServerStateSlice, key)
}

func DhcpRelayAgentInitIntfState(IntfId int32) {
	intfEntry := dhcprelayIntfStateMap[IntfId]
	intfEntry.IntfId = IntfId
	intfEntry.TotalDrops = 0
	intfEntry.TotalDhcpClientRx = 0
	intfEntry.TotalDhcpClientTx = 0
	intfEntry.TotalDhcpServerRx = 0
	intfEntry.TotalDhcpServerTx = 0
	dhcprelayIntfStateMap[IntfId] = intfEntry
	dhcprelayIntfStateSlice = append(dhcprelayIntfStateSlice, IntfId)
}

func DhcpRelayAgentInitGblHandling(ifNum int32) {
	logger.Info("DRA: Initializaing Global Info for " + strconv.Itoa(int(ifNum)))
	// Created a global Entry for Interface
	gblEntry := dhcprelayGblInfo[ifNum]
	// Setting up default values for globalEntry
	gblEntry.IpAddr = ""
	gblEntry.Netmask = ""
	gblEntry.IntfConfig.IfIndex = ifNum
	gblEntry.IntfConfig.Enable = false
	dhcprelayGblInfo[ifNum] = gblEntry
}

func DhcpRelayAgentInitVlanInfo(VlanName string, VlanId int32) {
	logger.Info(fmt.Sprintln("DRA: Vlan update message for ",
		VlanName, "vlan id is ", VlanId))
	var linuxInterface *net.Interface
	var err error
	linuxInterface, err = net.InterfaceByName(VlanName)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: getting interface by name failed", err))
		return
	}
	dhcprelayLogicalIntfId2LinuxIntId[linuxInterface.Index] = VlanId
}

func StartServer(log *syslog.Writer, handler *DhcpRelayServiceHandler, addr string) error {
	logger = log
	// Initialize port information and packet handler for dhcp
	go InitDhcpRelayPortPktHandler()
	dhcprelayEnable = false
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
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to start the listener, err:", err))
		return err
	}

	logger.Info("DRA:Started the Server successfully")
	return nil
}
