package main

import (
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"ribd"
	"utils/logging"
	"utils/policy"
)

var logger *logging.Writer
var routeServiceHandler *RouteServiceHandler
var PARAMSDIR string
var PolicyEngineDB *policy.PolicyEngineDB

func main() {
	var transport thrift.TServerTransport
	var err error
	var addr = "localhost:5000"

	fmt.Println("Starting rib daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err = logging.NewLogger(fileName, "ribd", "RIB")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForSysdNotifications()
	logger.Info("Started the logger successfully.")

	transport, err = thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to create Socket with:", addr))
	}
	handler := NewRouteServiceHandler(*paramsDir)
	if handler == nil {
		logger.Println("handler nill")
		return
	}
	routeServiceHandler = handler
	UpdateFromDB() //(paramsDir)
	processor := ribd.NewRouteServiceProcessor(handler)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	logger.Println("Starting RIB daemon")
	server.Serve()
}
