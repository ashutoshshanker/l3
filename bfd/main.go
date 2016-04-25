package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"l3/bfd/rpc"
	"l3/bfd/server"
	"utils/keepalive"
	"utils/logging"
)

func SigHandler() {
	server.logger.Info(fmt.Sprintln("Starting SigHandler"))
	sigChan := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChan, signalList...)

	for {
		select {
		case signal := <-sigChan:
			switch signal {
			case syscall.SIGHUP:
				server.logger.Info("Received SIGHUP signal")
				//server.SendAdminDownToAllNeighbors()
				//time.Sleep(500 * time.Millisecond)
				server.SendDeleteToAllSessions()
				time.Sleep(500 * time.Millisecond)
				server.logger.Info("Exiting!!!")
				os.Exit(0)
			default:
				server.logger.Info(fmt.Sprintln("Unhandled signal : ", signal))
			}
		}
	}
}

func main() {
	fmt.Println("Starting bfd daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger(fileName, "bfdd", "BFD")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}
	go logger.ListenForLoggingNotifications()
	logger.Info("Started the logger successfully.")

	// Start keepalive routine
	go keepalive.InitKeepAlive("bfdd", fileName)

	// Start signal handler
	go SigHandler()

	dbName := fileName + "UsrConfDb.db"
	fmt.Println("BFDd opening Config DB: ", dbName)
	dbHdl, err := sql.Open("sqlite3", dbName)
	if err != nil {
		fmt.Println("Failed to open connection to DB. ", err, " Exiting!!")
		return
	}
	clientsFileName := fileName + "clients.json"

	logger.Info(fmt.Sprintln("Starting BFD Server..."))
	bfdServer := server.NewBFDServer(logger)
	go bfdServer.StartServer(clientsFileName, dbHdl)
	logger.Info(fmt.Sprintln("Waiting for BFD server to come up"))
	up := <-bfdServer.ServerUpCh
	dbHdl.Close()
	logger.Info(fmt.Sprintln("BFD server is up: ", up))
	if !up {
		logger.Err(fmt.Sprintln("Exiting!!"))
		return
	}

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewBFDHandler(logger, bfdServer)
	rpc.StartServer(logger, confIface, clientsFileName)
}
