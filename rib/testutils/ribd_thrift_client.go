// ribd_thrift_client.go
package testutils

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"ribd"
)

const (
	IP = "localhost"
	PORT = "10002"
)
func main() {
	
}
func GetRIBdClient() *ribd.RIBDServicesClient {
	fmt.Println("Starting RIBd Thrift client for Testing")
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket(IP + ":" + PORT)
	if err != nil {
		fmt.Println("NewTSocket failed with error:", err)
		return nil
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		fmt.Println("Failed to open the socket, error:", err)
	}

	fmt.Println("### Calling client ", clientTransport, protocolFactory, err)
	client := ribd.NewRIBDServicesClientFactory(clientTransport, protocolFactory)
	return client
}
