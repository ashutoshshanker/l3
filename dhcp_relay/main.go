// Main entry point for DHCP_RELAY
package main

import (
	//"dhcprelayd"
	"fmt"
	//"git.apache.org/thrift.git/lib/go/thrift"
	"l3/dhcp_relay/server"
	"log/syslog"
)

const IP string = "localhost"
const DHCP_RELAY_PORT string = "7001"

func main() {
	fmt.Printf("Start the logger\n")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR DHCP RELAY")
	if err != nil {
		fmt.Println("Failed to start the logger... Exiting!!!")
		return
	}
	logger.Info("Started the logger successfully.")
	var addr = IP + ":" + DHCP_RELAY_PORT
	fmt.Println("DHCP RELAY address is %s", addr)
	logger.Info(fmt.Sprintln("Starting DHCP RELAY...."))
	// Create a handler
	handler := relayServer.NewDhcpRelayServer()
	err = relayServer.StartServer(logger, handler, addr)
	if err != nil {
		fmt.Println("Cannot start server")
		panic(err)
	}
	fmt.Printf("done\n")
}
