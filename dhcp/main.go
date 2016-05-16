package main

import (
	"flag"
	"fmt"
	"l3/dhcp/rpc"
	"l3/dhcp/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting dhcp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("dhcpd", "DHCP", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("dhcpd", fileName)

	logger.Info(fmt.Sprintln("Starting DHCP server..."))
	dhcpServer := server.NewDHCPServer(logger)
	go dhcpServer.StartServer(*paramsDir)

	<-dhcpServer.InitDone

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewDHCPHandler(dhcpServer, logger)
	rpc.StartServer(logger, confIface, *paramsDir)
}
