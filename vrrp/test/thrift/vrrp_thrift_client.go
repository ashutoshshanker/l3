package main

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"vrrpd"
)

const (
	IP   = "localhost"
	PORT = "10009"
)

func main() {
	fmt.Println("Starting VRRP Thrift client for Testing")
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

	client := vrrpd.NewVRRPDServicesClientFactory(clientTransport, protocolFactory)

	vrrpIntfConfig := vrrpd.NewVrrpIntf()
	vrrpIntfConfig.IfIndex = 123
	vrrpIntfConfig.VRID = 1
	vrrpIntfConfig.VirtualIPv4Addr = "172.16.0.1"
	vrrpIntfConfig.Priority = 100
	vrrpIntfConfig.PreemptMode = false
	vrrpIntfConfig.AcceptMode = false
	ret, err := client.CreateVrrpIntf(vrrpIntfConfig)
	if !ret {
		fmt.Println("Create Vrrp Intf Config Failed", err)
	} else {
		fmt.Println("Create Vrrp Intf Config Success")
	}
}
