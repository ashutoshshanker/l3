package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
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

	dbName := *paramsDir + "/UsrConfDb.db"
	logger.Info(fmt.Sprintln("Opening Config DB: ", dbName))
	dbHdl, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Println("Failed to open connection to DB. ", err)
		logger.Err("Exiting!!")
		return
	}

	logger.Info(fmt.Sprintln("Starting BFD Server..."))
	bfdServer := server.NewBFDServer(logger)
	go bfdServer.StartServer(fileName, dbHdl)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewBFDHandler(logger, bfdServer)
	rpc.StartServer(logger, confIface, fileName)
}
