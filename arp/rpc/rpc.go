package rpc

import (
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"log/syslog"
	"arpd"
	"strconv"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

func getClient(logger *syslog.Writer, fileName string, process string) (*ClientJson, error) {
	var allClients []ClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Err(fmt.Sprintf("Failed to open OSPFd config file:%s, err:%s", fileName, err))
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

func StartServer(logger *syslog.Writer, handler *ARPHandler, paramDir string) {
        fileName := paramDir
        if fileName[len(fileName) - 1] != '/' {
                fileName = fileName + "/"
        }
        fileName = fileName + "clients.json"

	clientJson, err := getClient(logger, fileName, "arpd")
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
	processor := arpd.NewARPDServicesProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to start the listener, err:", err))
	}
	logger.Info(fmt.Sprintln("Start the listener successfully"))
	return
}

/*

func createClientIPCHandles(logger *syslog.Writer, port string) (thrift.TTransport, thrift.TProtocolFactory, error) {
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket("localhost:" + port)
	if err != nil {
		logger.Err(fmt.Sprintln("NewTSocket failed with error:", err))
		return nil, nil, err
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	err = clientTransport.Open()
	return clientTransport, protocolFactory, err
}

func connectToClient(logger *syslog.Writer, clientTransport thrift.TTransport) error {
	return clientTransport.Open()
}


func StartClient(logger *syslog.Writer, fileName string, ribdClient chan *arpd.RouteServiceClient) {
	clientJson, err := getClient(logger, fileName, "asicd")
	if err != nil || clientJson == nil {
		ribdClient <- nil
		return
	}

	clientTransport, protocolFactory, err := createClientIPCHandles(logger, strconv.Itoa(clientJson.Port))
	if err != nil {
		logger.Info(fmt.Sprintf("Failed to connect to ASICd, retrying until connection is successful"))
		ticker := time.NewTicker(time.Duration(1000) * time.Millisecond)
		for _ = range ticker.C {
			err = connectToClient(logger, clientTransport)
			if err == nil {
				ticker.Stop()
				break
			}
		}
	}

	client := ribd.NewRouteServiceClientFactory(clientTransport, protocolFactory)
	ribdClient <- client
}
*/
