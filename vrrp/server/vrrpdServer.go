package vrrp

import (
	"log/syslog"
)

func NewVrrpServer() *VrrpServiceHandler {
	return &VrrpServiceHandler{}
}

func VrrpAllocateMemoryToGlobalDS() {
	vrrpGblInfo = make(map[int32]VrrpGlobalInfo, 10)
}

func StartServer(log *syslog.Writer, handler *VrrpServiceHandler, addr string) error {
	logger = log
	logger.Info("VRRP: allocating memory to global ds")
	VrrpAllocateMemoryToGlobalDS()
	// @TODO: Initialize DB

	// @TODO: Initialize port information and packet handler for vrrp using
	// go routine

	// create transport and protocol for server
	/*
		transportFactory := thrift.NewTBufferedTransportFactory(8192)
		protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
		transport, err := thrift.NewTServerSocket(addr)
		if err != nil {
			logger.Info(fmt.Sprintln("DRA: StartServer: NewTServerSocket "+
				"failed with error:", err))
			return err
		}
		processor := dhcprelayd.NewDHCPRELAYDServicesProcessor(handler)
		server := thrift.NewTSimpleServer4(processor, transport,
			transportFactory, protocolFactory)
		err = server.Serve()
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Failed to start the listener, err:", err))
			return err
		}
	*/
	logger.Info("VRRP: Started the Server successfully")
	return nil
}
