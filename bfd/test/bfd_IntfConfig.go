// bgp_client.go
package main

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"ospfd"
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

	client := ospfd.NewBFDServerClientFactory(clientTransport, protocolFactory)

	ifConfigArgs := ospfd.NewBfdIntfConfig()

	fmt.Println("calling BfdIntfConfig with attr:", ifConfigArgs)
	ret, err := client.CreateBfdIntfConfig(ifConfigArgs)
	if !ret {
		fmt.Println("BfdIntfConfig FAILED, ret:", ret, "err:", err)
	}
	fmt.Println("Bfd interface configured")
}
