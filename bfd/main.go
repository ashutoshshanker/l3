package main

import (
	"flag"
	"fmt"
	"l3/bfd/rpc"
	"l3/bfd/server"
	"log/syslog"
)

func main() {
	fmt.Println("Start the logger")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR BFD")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}

	logger.Info("Started the logger successfully.")

	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	logger.Info(fmt.Sprintln("Starting BFD Server..."))
	bfdServer := server.NewBFDServer(logger)
	go bfdServer.StartServer(fileName)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewBFDHandler(logger, bfdServer)
	rpc.StartServer(logger, confIface, fileName)
}
