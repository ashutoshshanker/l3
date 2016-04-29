package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"l3/rib/rpc"
	"l3/rib/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	var err error
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("ribd", "RIB", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("ribd", fileName)

	dbHdl, err := redis.Dial("tcp", ":6379")
	if err != nil {
		logger.Err("Failed to dial out to Redis server")
		return
	}
	routeServer := server.NewRIBDServicesHandler(dbHdl, logger)
	if routeServer == nil {
		logger.Println("routeServer nil")
		return
	}
	go routeServer.NotificationServer()
	go routeServer.StartNetlinkServer()
	go routeServer.StartAsicdServer()
	go routeServer.StartArpdServer()
	go routeServer.StartServer(*paramsDir)
	up := <-routeServer.ServerUpCh
	//dbHdl.Close()
	logger.Info(fmt.Sprintln("RIBD server is up: ", up))
	if !up {
		logger.Err(fmt.Sprintln("Exiting!!"))
		return
	}
	ribdServicesHandler := rpc.NewRIBdHandler(logger, routeServer)
	rpc.NewRIBdRPCServer(logger, ribdServicesHandler, fileName)
}
