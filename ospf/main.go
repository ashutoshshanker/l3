// main.go
package main

import (
	"flag"
	"fmt"
	"l3/ospf/rpc"
	"l3/ospf/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting ospf daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger(fileName, "ospfd", "OSPF")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForSysdNotifications()
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("ospfd", fileName)

	fileName = fileName + "clients.json"

	logger.Info(fmt.Sprintln("Starting OSPF Server..."))
	ospfServer := server.NewOSPFServer(logger)
	go ospfServer.StartServer(fileName)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewOSPFHandler(ospfServer, logger)
	rpc.StartServer(logger, confIface, fileName)
}
