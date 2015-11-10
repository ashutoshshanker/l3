// main.go
package main

import (
    "fmt"
	"l3/bgp/rpc"
    "l3/bgp/server"
	"log/syslog"
)

const IP string = "localhost" //"10.0.2.15"
const BGPPort string = "179"
const CONF_PORT string = "2001"
const BGPConfPort string = "4050"
const RIBConfPort string = "9090"

func main() {
	fmt.Println("Start the logger")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR BGP")
	if err != nil	 {
		fmt.Println("Failed to start the logger. Exit!")
		return
	}
	logger.Info("Started the logger successfully.")

    logger.Info(fmt.Sprintln("Start connection to RIBd"))
	ribdClient, err := rpc.StartClient(logger, RIBConfPort)
	if err != nil {
		logger.Err("Failed to connect to RIBd\n")
		return
	}

    logger.Info(fmt.Sprintln("Start BGP Server"))
    bgpServer := server.NewBGPServer(logger, ribdClient)
    go bgpServer.StartServer()

    logger.Info(fmt.Sprintln("Start config listener"))
	confIface := rpc.NewBGPHandler(bgpServer, logger)
	rpc.StartServer(logger, confIface, BGPConfPort)
}

