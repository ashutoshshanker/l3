package vrrpServer

import (
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log/syslog"
	"vrrpd"
)

func NewVrrpServer() *VrrpServiceHandler {
	return &VrrpServiceHandler{}
}

func VrrpAllocateMemoryToGlobalDS() {
	vrrpGblInfo = make(map[int32]VrrpGlobalInfo, 10)
}

func VrrpConnectToClient() {

}

func StartServer(log *syslog.Writer, handler *VrrpServiceHandler, addr string) error {
	logger = log
	logger.Info("VRRP: allocating memory to global ds")

	// Allocate memory to all the Data Structures
	VrrpAllocateMemoryToGlobalDS()

	params := flag.String("params", "", "Directory Location for config files")
	flag.Parse()
	paramsDir = *params

	// Initialize DB
	err := VrrpInitDB()
	if err != nil {
		logger.Err("VRRP: DB init failed")
	} else {
		VrrpReadDB()
	}

	// @TODO: Initialize port information and packet handler for vrrp using
	go VrrpConnectToClient()

	// create transport and protocol for server
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("VRRP: StartServer: NewTServerSocket "+
			"failed with error:", err))
		return err
	}
	processor := vrrpd.NewVRRPDServicesProcessor(handler)
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Failed to start the listener, err:", err))
		return err
	}
	return nil
}
