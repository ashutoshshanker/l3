// bgp_client.go
package main

import (
	"ospfd"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
)

const CONF_IP string = "localhost" //"10.0.2.15"
const CONF_PORT string = "7000"

func main() {
	fmt.Println("Starting the OSPF thrift client...")
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

	client := ospfd.NewOSPFServerClientFactory(clientTransport, protocolFactory)

	ifConfigArgs := ospfd.NewOspfIfConf()
        ifConfigArgs.IfIpAddress = "40.1.1.1"
        ifConfigArgs.AddressLessIf = 0
        ifConfigArgs.IfAreaId = "0.0.0.1"
        ifConfigArgs.IfType = 1
        ifConfigArgs.IfAdminStat = 1
        ifConfigArgs.IfRtrPriority = 1
        ifConfigArgs.IfTransitDelay = 1
        ifConfigArgs.IfRetransInterval = 5
        ifConfigArgs.IfHelloInterval = 20
        ifConfigArgs.IfRtrDeadInterval = 60
        ifConfigArgs.IfPollInterval = 150
        ifConfigArgs.IfAuthKey = "0.0.0.0.0.0.0.1"
        ifConfigArgs.IfMulticastForwarding = 2
        ifConfigArgs.IfDemand = false
        ifConfigArgs.IfAuthType = 0

	fmt.Println("calling CreateOspfIf with attr:", ifConfigArgs)
	ret, err := client.CreateOspfIfConf(ifConfigArgs)
	if !ret {
		fmt.Println("CreateOspfIf FAILED, ret:", ret, "err:", err)
	}
	fmt.Println("Created Ospf interface conf")
}
