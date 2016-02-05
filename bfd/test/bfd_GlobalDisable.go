package main

import (
	"bfdd"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
)

const CONF_IP string = "localhost" //"10.0.2.15"
const CONF_PORT string = "9050"

func main() {
	fmt.Println("Starting the BFD thrift client...")
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket("localhost:" + CONF_PORT)
	if err != nil {
		fmt.Println("NewTSocket failed with error:", err)
		return
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		fmt.Println("Failed to open the socket, error:", err)
	}

	client := bfdd.NewBFDDServicesClientFactory(clientTransport, protocolFactory)

	globalConfigArgs := bfdd.NewBfdGlobalConfig()
	globalConfigArgs.Bfd = "Router BFD"
	globalConfigAgrs.Enable = false

	fmt.Println("calling Bfd disable with attr:", globalConfigArgs)
	ret, err := client.CreateBfdGlobalConfig(globalConfigArgs)
	if !ret {
		fmt.Println("BfdGlobal FAILED, ret:", ret, "err:", err)
	}
	fmt.Println("Bfd global disabled")
}
