// server.go
package rpc

import (
	"bgpd"
    "fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log/syslog"
	"ribd"
)

func StartServer(logger *syslog.Writer, handler *BGPHandler, port string) {
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	serverTransport, err := thrift.NewTServerSocket("localhost:" + port)
	if err != nil {
		logger.Info(fmt.Sprintln("StartServer: NewTServerSocket failed with error:", err))
		return
	}
	processor := bgpd.NewBGPServerProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	logger.Info(fmt.Sprintln("Starting the BGP config listener"))
	err = server.Serve()
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to start the listener"))
	}
	logger.Info(fmt.Sprintln("Start the listener successfully"))
	return
}

func StartClient(logger *syslog.Writer, port string) (*ribd.RouteServiceClient, error) {
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket("localhost:" + port)
	if err != nil {
		logger.Info(fmt.Sprintln("NewTSocket failed with error:", err))
		return nil, err
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		logger.Err(fmt.Sprintln("Failed to open the socket, error:", err))
		return nil, err
	}

	client := ribd.NewRouteServiceClientFactory(clientTransport, protocolFactory)
	return client, nil
}
