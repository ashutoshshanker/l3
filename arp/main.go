package main

import (
	"flag"
	"fmt"
	"l3/arp/rpc"
	"l3/arp/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting arp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("arpd", "ARP", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	logger.Info(fmt.Sprintln("Starting ARP server..."))
	arpServer := server.NewARPServer(logger)
	//go arpServer.StartServer(fileName)
	go arpServer.StartServer(*paramsDir)

	<-arpServer.InitDone

	// Start keepalive routine
	go keepalive.InitKeepAlive("arpd", fileName)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewARPHandler(arpServer, logger)
	rpc.StartServer(logger, confIface, *paramsDir)
}
