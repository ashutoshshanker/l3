package rpc
import (
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"ribd"
	"strconv"
	"utils/logging"
	"l3/rib/server"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}
type RIBDServicesHandler struct {
	server *server.RIBDServer
	logger *logging.Writer
}
var logger *logging.Writer
func getClient(logger *logging.Writer, fileName string, process string) (*ClientJson, error) {
	var allClients []ClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Err(fmt.Sprintf("Failed to open RIBd config file:%s, err:%s", fileName, err))
		return nil, err
	}

	json.Unmarshal(data, &allClients)
	for _, client := range allClients {
		if client.Name == process {
			return &client, nil
		}
	}

	logger.Err(fmt.Sprintf("Did not find port for %s in config file:%s", process, fileName))
	return nil, nil
}
func NewRIBdHandler(loggerC *logging.Writer, server *server.RIBDServer) (*RIBDServicesHandler) {
	hdl := new(RIBDServicesHandler)
	hdl.server = server
	hdl.logger = loggerC
	logger = loggerC
	return hdl
}
func NewRIBdRPCServer(logger *logging.Writer, handler *RIBDServicesHandler, fileName string) () {
	var transport thrift.TServerTransport
	clientJson, err := getClient(logger, fileName+"clients.json", "ribd")
	if err != nil || clientJson == nil {
		return
	}
	var addr = "localhost:" + strconv.Itoa(clientJson.Port)//"localhost:5000"
	fmt.Println("Starting rib daemon at addr ", addr)

	transport, err = thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to create Socket with:", addr))
	}
	processor := ribd.NewRIBDServicesProcessor((handler))
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	logger.Println("Starting RIB daemon")
	server.Serve()
}