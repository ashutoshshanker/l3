package main

import (
	"fmt"
	"l3/vrrp/server"
	"log/syslog"
)

const IP = "localhost"
const VRRP_PORT = "10000"

func main() {
	fmt.Printf("VRRP: Start the logger\n")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR VRRP")
	if err != nil {
		fmt.Println("Failed to start the logger... Exiting!!!")
		return
	}
	logger.Info("VRRP: Started the logger successfully.")
	var addr = IP + ":" + VRRP_PORT
	logger.Info("Starting VRRP....")
	// Create a handler
	handler := vrrpServer.NewVrrpServer()
	err = vrrpServer.StartServer(logger, handler, addr)
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Cannot start dhcp server", err))
		return
	}
	logger.Info("VRRP: started successfully")
}
