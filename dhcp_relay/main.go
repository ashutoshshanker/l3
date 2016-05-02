// Main entry point for DHCP_RELAY
package main

import (
	"flag"
	"fmt"
	"l3/dhcp_relay/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting dhcprelay daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fmt.Println("Start logger")
	logger, err := logging.NewLogger("dhcprelayd", "DRA", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("dhcprelayd", fileName)

	logger.Info(fmt.Sprintln("Starting DHCP RELAY...."))
	// Create a handler
	handler := relayServer.NewDhcpRelayServer()
	err = relayServer.StartServer(logger, handler, *paramsDir)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Cannot start dhcp server", err))
		return
	}
}
