package main

import (
	"database/sql"
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"ribd"
	"strconv"
	"utils/keepalive"
	"utils/logging"
	"utils/policy"
	"io/ioutil"
	"encoding/json"
)

var logger *logging.Writer
var routeServiceHandler *RIBDServicesHandler
var PARAMSDIR string
var PolicyEngineDB *policy.PolicyEngineDB

func getClient(logger *logging.Writer, fileName string, process string) (*ClientJson, error) {
	var allClients []ClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Err(fmt.Sprintf("Failed to open RIBd config file:%s, err:%s", fileName, err))
		return nil, err
	}

	json.Unmarshal(data, &allClients)
	for _, client := range allClients {
		if client.Name == process {
			return &client, nil
		}
	}

	logger.Err(fmt.Sprintf("Did not find port for %s in config file:%s", process, fileName))
	return nil, nil
}

func main() {
	var transport thrift.TServerTransport
	var err error
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
	clientJson, err := getClient(logger, fileName+"clients.json", "ribd")
	if err != nil || clientJson == nil {
		return
	}
	var addr = "localhost:" + strconv.Itoa(clientJson.Port)//"localhost:5000"
	fmt.Println("Starting rib daemon at addr ", addr)

	transport, err = thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to create Socket with:", addr))
	}
	handler := NewRIBDServicesHandler(dbHdl)
	if handler == nil {
		logger.Println("handler nill")
		return
	}
	routeServiceHandler = handler
	go routeServiceHandler.NotificationServer()
	go routeServiceHandler.StartNetlinkServer()
	go routeServiceHandler.StartAsicdServer()
	go routeServiceHandler.StartArpdServer()
	go routeServiceHandler.StartServer(*paramsDir)
	up := <-routeServiceHandler.ServerUpCh
	dbHdl.Close()
	logger.Info(fmt.Sprintln("RIBD server is up: ", up))
	if !up {
		logger.Err(fmt.Sprintln("Exiting!!"))
		return
	}
	processor := ribd.NewRIBDServicesProcessor((routeServiceHandler))
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	logger.Println("Starting RIB daemon")
	server.Serve()
}
