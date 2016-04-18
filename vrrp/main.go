package main

import (
	"flag"
	"fmt"
	"l3/vrrp/rpc"
	"l3/vrrp/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting vrrp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger(fileName, "vrrpd", "VRRP")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForSysdNotifications()
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("vrrpd", fileName)

	logger.Info("Starting VRRP server....")
	// Create vrrp server handler
	vrrpSvr := vrrpServer.VrrpNewServer(logger)
	// Until Server is connected to clients do not start with RPC
	vrrpSvr.VrrpStartServer(*paramsDir)
	// Create vrrp rpc handler
	vrrpHdl := vrrpRpc.VrrpNewHandler(vrrpSvr, logger)
	logger.Info("Starting VRRP RPC listener....")
	err = vrrpRpc.VrrpRpcStartServer(logger, vrrpHdl, *paramsDir)
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Cannot start vrrp server", err))
		return
	}
}
