// bgp_client.go
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
	clientTransport, err := thrift.NewTSocket(CONF_IP + ":" + CONF_PORT)
	if err != nil {
		fmt.Println("NewTSocket failed with error:", err)
		return
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		fmt.Println("Failed to open the socket, error:", err)
	}

	client := bfdd.NewBFDDServicesClientFactory(clientTransport, protocolFactory)

	sessionConfigArgs := bfdd.NewBfdSessionConfig()
	sessionConfigArgs.IpAddr = "10.10.0.130"
	sessionConfigArgs.Owner = 1
	sessionConfigArgs.Operation = 1
	fmt.Println("Creating BFD Session: ", sessionConfigArgs)
	ret, err := client.CreateBfdSessionConfig(sessionConfigArgs)
	if !ret {
		fmt.Println("BfdSessionConfig FAILED, ret:", ret, "err:", err)
	} else {
		fmt.Println("Bfd session configured")
	}
}
