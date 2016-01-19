// Main entry point for DHCP_RELAY
package relayServer

import (
	"dhcprelayd"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log/syslog"
)

type DhcpRelayServiceHandler struct {
}

func NewDhcpRelayServer() *DhcpRelayServiceHandler {
	return &DhcpRelayServiceHandler{}
}

func StartServer(logger *syslog.Writer, handler *DhcpRelayServiceHandler, addr string) error {
	// create transport and protocol for server
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("StartServer: NewTServerSocket failed with error:", err))
		return err
	}
	fmt.Println("%T", transport)
	processor := dhcprelayd.NewDHCPRELAYDServicesProcessor(handler)
	fmt.Printf("%T\n", transportFactory)
	fmt.Printf("%T\n", protocolFactory)
	fmt.Printf("Starting DHCP-RELAY daemon at %s\n", addr)
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to start the listener, err:", err))
		return err
	}

	logger.Info(fmt.Sprintln("Start the Server successfully"))
	return nil
}
