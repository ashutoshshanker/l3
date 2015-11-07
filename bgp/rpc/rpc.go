// server.go
package rpc

import (
    "fmt"
	"generated/src/bgpd"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func StartServer(handler *BgpHandler, port string) {
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	serverTransport, err := thrift.NewTServerSocket("localhost:" + port)
	if err != nil {
		fmt.Println("StartServer: NewTServerSocket failed with error:", err)
		return
	}
	processor := bgpd.NewBgpServerProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)
	fmt.Println("Starting the BGP config listener")
	err = server.Serve()
	if err != nil {
		fmt.Println("Failed to start the listener")
	}
	fmt.Println("Start the listener successfully")
	return
}
