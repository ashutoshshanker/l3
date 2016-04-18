// main.go
package main

import (
	"asicdServices"
	"bfdd"
	"flag"
	"fmt"
	bgppolicy "l3/bgp/policy"
	"l3/bgp/rpc"
	"l3/bgp/server"
	"l3/bgp/utils"
	"ribd"
	"utils/keepalive"
	"utils/logging"
)

const IP string = "localhost" //"10.0.2.15"
const BGPPort string = "179"
const CONF_PORT string = "2001"
const BGPConfPort string = "4050"
const RIBConfPort string = "5000"

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

	var asicdClient *asicdServices.ASICDServicesClient = nil
	asicdClientChan := make(chan *asicdServices.ASICDServicesClient)

	logger.Info("Connecting to ASICd")
	go rpc.StartAsicdClient(logger, fileName, asicdClientChan)
	asicdClient = <-asicdClientChan
	if asicdClient == nil {
		logger.Err("Failed to connect to ASICd")
		return
	} else {
		logger.Info("Connected to ASICd")
	}

	var ribdClient *ribd.RIBDServicesClient = nil
	ribdClientChan := make(chan *ribd.RIBDServicesClient)

	logger.Info("Connecting to RIBd")
	go rpc.StartRibdClient(logger, fileName, ribdClientChan)
	ribdClient = <-ribdClientChan
	if ribdClient == nil {
		logger.Err("Failed to connect to RIBd\n")
		return
	} else {
		logger.Info("Connected to RIBd")
	}

	var bfddClient *bfdd.BFDDServicesClient = nil
	bfddClientChan := make(chan *bfdd.BFDDServicesClient)

	logger.Info("Connecting to BFDd")
	go rpc.StartBfddClient(logger, fileName, bfddClientChan)
	bfddClient = <-bfddClientChan
	if bfddClient == nil {
		logger.Err("Failed to connect to BFDd\n")
		return
	} else {
		logger.Info("Connected to BFDd")
	}

	logger.Info(fmt.Sprintln("Starting BGP policy engine..."))
	bgpPolicyEng := bgppolicy.NewBGPPolicyEngine(logger)
	go bgpPolicyEng.StartPolicyEngine()

	logger.Info(fmt.Sprintln("Starting BGP Server..."))
	bgpServer := server.NewBGPServer(logger, bgpPolicyEng, ribdClient, bfddClient, asicdClient)
	go bgpServer.StartServer()

	logger.Info(fmt.Sprintln("Starting config listener..."))
	confIface := rpc.NewBGPHandler(bgpServer, bgpPolicyEng, logger, fileName)
	rpc.StartServer(logger, confIface, fileName)
}
