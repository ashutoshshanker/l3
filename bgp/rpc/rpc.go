// server.go
package rpc

import (
    "fmt"
	"generated/src/bgpd"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log/syslog"
)

func StartServer(logger *syslog.Writer, handler *BgpHandler, port string) {
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	serverTransport, err := thrift.NewTServerSocket("localhost:" + port)
	if err != nil {
		logger.Info(fmt.Sprintln("StartServer: NewTServerSocket failed with error:", err))
		return
	}
	processor := bgpd.NewBgpServerProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	logger.Info(fmt.Sprintln("Starting the BGP config listener"))
	err = server.Serve()
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to start the listener"))
	}
	logger.Info(fmt.Sprintln("Start the listener successfully"))
	return
}
