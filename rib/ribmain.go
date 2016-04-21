package main

import (
	"database/sql"
	"flag"
	"fmt"
	"utils/keepalive"
	"utils/logging"
	"l3/rib/rpc"
	"l3/rib/server"
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
	logger, err := logging.NewLogger(fileName, "ribd", "RIB")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForLoggingNotifications()
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("ribd", fileName)

	dbName := fileName + "UsrConfDb.db"
	fmt.Println("RIBd opening Config DB: ", dbName)
	dbHdl, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Println("Failed to open connection to DB. ", err, " Exiting!!")
		return
	}
	if err = dbHdl.Ping(); err != nil {
		fmt.Println(fmt.Sprintln("Failed to keep DB connection alive"))
		return
	}
	routeServer := server.NewRIBDServicesHandler(dbHdl,logger)
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
	dbHdl.Close()
	logger.Info(fmt.Sprintln("RIBD server is up: ", up))
	if !up {
		logger.Err(fmt.Sprintln("Exiting!!"))
		return
	}
	ribdServicesHandler := rpc.NewRIBdHandler(logger,routeServer)
	rpc.NewRIBdRPCServer(logger,ribdServicesHandler,fileName)
}
