package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"l3/bfd/rpc"
	"l3/bfd/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting bfd daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("bfdd", "BFD", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	dbHdl, err := redis.Dial("tcp", ":6379")
	if err != nil {
		logger.Err("Failed to dial out to Redis server")
		return
	}

	clientsFileName := fileName + "clients.json"

	logger.Info(fmt.Sprintln("Starting BFD Server..."))
	bfdServer := server.NewBFDServer(logger)
	// Start signal handler
	go bfdServer.SigHandler(dbHdl)
	// Start bfd server
	go bfdServer.StartServer(clientsFileName, dbHdl)

	<-bfdServer.ServerStartedCh
	logger.Info(fmt.Sprintln("BFD Server started"))

	// Start keepalive routine
	go keepalive.InitKeepAlive("bfdd", fileName)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewBFDHandler(logger, bfdServer)
	// Read BFD configurations already present in DB
	confIface.ReadConfigFromDB(dbHdl)
	rpc.StartServer(logger, confIface, clientsFileName)
}
