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

	client := ospfd.NewOSPFDServicesClientFactory(clientTransport, protocolFactory)

	globalConfigArgs := ospfd.NewOspfGlobalConfig()
        globalConfigArgs.RouterIdKey = "40.1.1.2"
        globalConfigArgs.AdminStat = 1
        globalConfigArgs.ASBdrRtrStatus = true
        globalConfigArgs.TOSSupport = true
        globalConfigArgs.ExtLsdbLimit = 10
        globalConfigArgs.MulticastExtensions = 2
        globalConfigArgs.ExitOverflowInterval = 100
        globalConfigArgs.DemandExtensions = true
        globalConfigArgs.RFC1583Compatibility = false
        globalConfigArgs.ReferenceBandwidth = 1000
        globalConfigArgs.RestartSupport = 1
        globalConfigArgs.RestartInterval = 10
        globalConfigArgs.RestartStrictLsaChecking = true
        globalConfigArgs.StubRouterAdvertisement = 1

	fmt.Println("calling CreateOspfGlobal with attr:", globalConfigArgs)
	ret, err := client.CreateOspfGlobalConfig(globalConfigArgs)
	if !ret {
		fmt.Println("CreateOspfGlobal FAILED, ret:", ret, "err:", err)
	}
	fmt.Println("Created Ospf global conf")
}
