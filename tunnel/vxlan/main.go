// main
package main

import (
	"flag"
	"fmt"
	vxlan "l3/tunnel/vxlan/protocol"
	"l3/tunnel/vxlan/rpc"
	"utils/logging"
)

func main() {

	// lookup port
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	path := *paramsDir
	if path[len(path)-1] != '/' {
		path = path + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger(path, "vxland", "VXLAN")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForSysdNotifications()
	logger.Info("Started the logger successfully.")

	// create a new vxlan server
	server := vxlan.NewVXLANServer(logger, path)
	handler := rpc.NewVXLANDServiceHandler(server, logger)
	// blocking call
	handler.StartThriftServer()
}
