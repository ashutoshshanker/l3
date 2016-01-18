package main

import (
	"dhcprelayd"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
)

func StartTestClient(addr string) error {
	// create transport and protocol for server
	fmt.Println("Request for starting Dhcp Relay Test Client")
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	socket, err := thrift.NewTSocket(addr)
	if err != nil {
		fmt.Println("Error Opening Socket at addr %s", addr)
		return err
	}
	transport := transportFactory.GetTransport(socket)
	defer transport.Close()
	if err := transport.Open(); err != nil {
		return err
	}
	fmt.Printf("Transport for Test Client created successfully\n")
	fmt.Println("client started at %s", addr)

	//Create a client for communicating with the server
	client := dhcprelayd.NewDhcpRelayServerClientFactory(transport, protocolFactory)
	fmt.Println("DHCP RELAY TEST Client Started")
	fmt.Println("Calling add relay agent")

	//Create dhcprelay configuration structure
	globalConfigArgs := dhcprelayd.NewDhcpRelayConf()
	globalConfigArgs.IpSubnet = "10.10.1.1"
	globalConfigArgs.IfIndex = "Ethernet1/1"

	// Call add relay agent api for the client with configuration
	err = client.AddRelayAgent(globalConfigArgs)
	if err != nil {
		fmt.Println("Add Relay Agent returned error")
		return err
	}
	fmt.Println("addition of relay agent success")
	fmt.Println("calling update relay agent")
	err = client.UpdRelayAgent()
	if err != nil {
		fmt.Println("Update Relay Agent returned error")
		return err
	}
	fmt.Println("updation of relay agent success")
	fmt.Println("calling delete relay agent")
	err = client.DelRelayAgent()
	if err != nil {
		fmt.Println("Delete Relay Agent returned error")
		return err
	}
	fmt.Println("deletion of relay agent successful")
	return nil
}

func main() {
	addr := "localhost:7000"
	err := StartTestClient(addr)
	if err != nil {
		fmt.Println("Failed to start test client.. Exiting!!!!")
		return
	}
}
