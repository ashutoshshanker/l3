// main.go
package main

import (
	"bfdd"
	"flag"
	"fmt"
	"l3/bgp/policy"
	"l3/bgp/rpc"
	"l3/bgp/server"
	"l3/bgp/utils"
	"log/syslog"
	"ribd"
)

const IP string = "localhost" //"10.0.2.15"
const BGPPort string = "179"
const CONF_PORT string = "2001"
const BGPConfPort string = "4050"
const RIBConfPort string = "5000"

func main() {
	fmt.Println("SR BGP: Start the logger")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR BGP")
	if err != nil {
		fmt.Println("SR BGP: Failed to start the logger. Exit!")
		return
	}

	logger.Info("Started the logger successfully.")
	utils.SetLogger(logger)

	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	var ribdClient *ribd.RouteServiceClient = nil
	ribdClientChan := make(chan *ribd.RouteServiceClient)

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

	logger.Info(fmt.Sprintln("Starting BGP Server..."))
	bgpServer := server.NewBGPServer(logger, ribdClient, bfddClient)
	go bgpServer.StartServer()

	logger.Info(fmt.Sprintln("Starting BGP policy engine..."))
	bgpPolicyEng := policy.NewBGPPolicyEngine(logger)
	go bgpPolicyEng.StartPolicyEngine()

	logger.Info(fmt.Sprintln("Starting config listener..."))
	confIface := rpc.NewBGPHandler(bgpServer, bgpPolicyEng, logger, fileName)
	rpc.StartServer(logger, confIface, fileName)
}
