// main.go
package main

import (
	"asicdServices"
	"bfdd"
	"errors"
	"flag"
	"fmt"
	_ "reflect"
	"utils/keepalive"
	"utils/logging"

	// Bgp packages
	"l3/bgp/ovsdbHandler"
	bgppolicy "l3/bgp/policy"
	"l3/bgp/rpc"
	"l3/bgp/server"
	"l3/bgp/utils"

	// Ribd package
	"ribd"
)

const (
	IP          string = "10.1.10.229" //"localhost"
	BGPPort     string = "179"
	CONF_PORT   string = "2001"
	BGPConfPort string = "4050"
	RIBConfPort string = "5000"

	OVSDB_PLUGIN = "ovsdb"
)

func main() {
	fmt.Println("Starting bgp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fmt.Println("Start logger")
	logger, err := logging.NewLogger(fileName, "bgpd", "BGP")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForSysdNotifications()
	logger.Info("Started the logger successfully.")
	utils.SetLogger(logger)

	// Start keepalive routine
	go keepalive.InitKeepAlive("bgpd", fileName)

	// @FIXME: Plugin name should come for json readfile...
	plugin := OVSDB_PLUGIN

	switch plugin {
	case OVSDB_PLUGIN:
		// if plugin used is ovs db then lets start ovsdb client listener
		quit := make(chan bool)
		ovsdbManager, err := ovsdbHandler.NewBGPOvsdbHandler()
		if err != nil {
			fmt.Println("Starting OVDB client failed ERROR:", err)
			return
		}
		err = ovsdbManager.BGPOvsdbServe()
		if err != nil {
			fmt.Println("OVSDB Serve failed ERROR:", err)
			return
		}
		<-quit
	default:
		// flexswitch plugin lets connect to clients first and then
		// start flexswitch client listener
		asicdClient, ribdClient, bfddClient, err :=
			bgpConnectToFlexSwitchClients(logger, fileName)
		if err != nil {
			return
		}

		// Connection to clients sucess, starting bgp policy engine...
		logger.Info(fmt.Sprintln("Starting BGP policy engine..."))
		bgpPolicyEng := bgppolicy.NewBGPPolicyEngine(logger)
		go bgpPolicyEng.StartPolicyEngine()

		// Connection to clients success, lets start bgp backend server
		logger.Info(fmt.Sprintln("Starting BGP Server..."))
		bgpServer := server.NewBGPServer(logger, bgpPolicyEng, ribdClient,
			bfddClient, asicdClient)
		go bgpServer.StartServer()

		logger.Info(fmt.Sprintln("Starting config listener..."))
		confIface := rpc.NewBGPHandler(bgpServer, bgpPolicyEng, logger, fileName)
		rpc.StartServer(logger, confIface, fileName)
	}
}

/* If FlexSwitch plugin, then connect to flexswitch dameons like ribd,
 * asicd, bfd. Only if connection is successful start the server
 */
func bgpConnectToFlexSwitchClients(logger *logging.Writer,
	fileName string) (*asicdServices.ASICDServicesClient,
	*ribd.RIBDServicesClient, *bfdd.BFDDServicesClient, error) {

	var asicdClient *asicdServices.ASICDServicesClient = nil
	var ribdClient *ribd.RIBDServicesClient = nil
	var bfddClient *bfdd.BFDDServicesClient = nil

	asicdClientChan := make(chan *asicdServices.ASICDServicesClient)

	logger.Info("Connecting to ASICd")
	go rpc.StartAsicdClient(logger, fileName, asicdClientChan)
	asicdClient = <-asicdClientChan
	if asicdClient == nil {
		logger.Err("Failed to connect to ASICd")
		return nil, nil, nil, errors.New("Failed to connect to ASICd")
	} else {
		logger.Info("Connected to ASICd")
	}

	ribdClientChan := make(chan *ribd.RIBDServicesClient)

	logger.Info("Connecting to RIBd")
	go rpc.StartRibdClient(logger, fileName, ribdClientChan)
	ribdClient = <-ribdClientChan
	if ribdClient == nil {
		logger.Err("Failed to connect to RIBd\n")
		return nil, nil, nil, errors.New("Failed to connect to RIBd")
	} else {
		logger.Info("Connected to RIBd")
	}

	bfddClientChan := make(chan *bfdd.BFDDServicesClient)

	logger.Info("Connecting to BFDd")
	go rpc.StartBfddClient(logger, fileName, bfddClientChan)
	bfddClient = <-bfddClientChan
	if bfddClient == nil {
		logger.Err("Failed to connect to BFDd\n")
		return nil, nil, nil, errors.New("Failed to connect to BFDd")
	} else {
		logger.Info("Connected to BFDd")
	}
	return asicdClient, ribdClient, bfddClient, nil
}
