package main

import (
	"flag"
	"fmt"
	"l3/arp/rpc"
	"l3/arp/server"
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
	logger, err := logging.NewLogger(fileName, "arpd", "ARP")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForSysdNotifications()
	logger.Info("Started the logger successfully.")

	logger.Info(fmt.Sprintln("Starting ARP server..."))
	arpServer := server.NewARPServer(logger)
	//go arpServer.StartServer(fileName)
	go arpServer.StartServer(*paramsDir)

	<-arpServer.InitDone

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewARPHandler(arpServer, logger)
	rpc.StartServer(logger, confIface, *paramsDir)
}
