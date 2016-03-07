package main

import (
	"flag"
	"fmt"
	"l3/vrrp/rpc"
	"l3/vrrp/server"
	"log/syslog"
)

func main() {
	fmt.Printf("VRRP: Start the logger\n")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR VRRP")
	if err != nil {
		fmt.Println("Failed to start the logger... Exiting!!!")
		return
	}
	logger.Info("VRRP: Started the logger successfully.")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	logger.Info("Starting VRRP server....")
	// Create vrrp server handler
	vrrpSvr := vrrpServer.VrrpNewServer(logger)
	go vrrpSvr.StartServer(*paramsDir)

	logger.Info("Starting VRRP Rpc listener....")
	// Create vrrp rpc handler
	vrrpHdl := vrrpRpc.VrrpNewHandler(vrrpSvr, logger)
	err = vrrpRpc.StartServer(logger, vrrpHdl, *paramsDir)
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Cannot start vrrp server", err))
		return
	}
	logger.Info("Started Successfully")
}
