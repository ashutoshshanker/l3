package main

import (
	"dhcprelayd"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
)

const (
	IP   = "localhost"
	PORT = "10007"
)

func main() {
	fmt.Println("Starting Dhcp Relay Agent Thrift client for Testing")
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket(IP + ":" + PORT)
	if err != nil {
		fmt.Println("NewTSocket failed with error:", err)
		return
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		fmt.Println("Failed to open the socket, error:", err)
	}
	client := dhcprelayd.NewDHCPRELAYDServicesClientFactory(clientTransport,
		protocolFactory)

	gblConfig := dhcprelayd.NewDhcpRelayGlobal()
	gblConfig.Enable = true
	gblConfig.DhcpRelay = "dhcp_test"
	ret, err := client.CreateDhcpRelayGlobal(gblConfig)
	if !ret {
		fmt.Println("Create DHCP Relay Global Config Failed", err)
	} else {
		fmt.Println("Create DHCP Relay Global Config Success")
	}
}
