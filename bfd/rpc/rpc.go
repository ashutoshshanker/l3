package rpc

import (
	"bfdd"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"strconv"
	"utils/logging"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

func getClient(logger *logging.Writer, fileName string, process string) (*ClientJson, error) {
	var allClients []ClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Err(fmt.Sprintf("Failed to open BFDd config file:%s, err:%s", fileName, err))
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

func StartServer(logger *logging.Writer, handler *BFDHandler, fileName string) {
	clientJson, err := getClient(logger, fileName, "bfdd")
	if err != nil || clientJson == nil {
		return
	}

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	serverTransport, err := thrift.NewTServerSocket("localhost:" + strconv.Itoa(clientJson.Port))
	if err != nil {
		logger.Info(fmt.Sprintln("StartServer: NewTServerSocket failed with error:", err))
		return
	}
	processor := bfdd.NewBFDDServicesProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	logger.Info(fmt.Sprintln("StartServer: Starting"))
	err = server.Serve()
	logger.Info(fmt.Sprintln("StartServer: Started"))
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to start the listener, err:", err))
	}
	logger.Info(fmt.Sprintln("Started the listener successfully"))
	return
}
