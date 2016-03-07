package vrrpRpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"l3/vrrp/server"
	"log/syslog"
	"strconv"
	"vrrpd"
)

type VrrpHandler struct {
	server *vrrpServer.VrrpServer
	logger *syslog.Writer
}
type VrrpClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

func (h *VrrpHandler) CreateVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	h.logger.Info(fmt.Sprintln("VRRP: Interface config create for ifindex ",
		config.IfIndex))
	if config.VRID == 0 {
		h.logger.Info("VRRP: Invalid VRID")
		return false, errors.New(vrrpServer.VRRP_INVALID_VRID)
	}
	h.server.VrrpIntfConfigCh <- *config
	return true, nil
}
func (h *VrrpHandler) UpdateVrrpIntfConfig(origconfig *vrrpd.VrrpIntfConfig,
	newconfig *vrrpd.VrrpIntfConfig, attrset []bool) (r bool, err error) {
	return true, nil
}

func (h *VrrpHandler) DeleteVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	go h.server.VrrpAddMacEntry(false /*delete vrrp protocol mac*/)
	return true, nil
}

func (h *VrrpHandler) GetBulkVrrpIntfState(fromIndex vrrpd.Int,
	count vrrpd.Int) (intfEntry *vrrpd.VrrpIntfStateGetInfo, err error) {
	nextIdx, currCount, vrrpIntfStates := h.server.VrrpGetBulkVrrpIntfStates(
		int(fromIndex), int(count))
	if vrrpIntfStates == nil {
		return nil, errors.New("Interface Slice is not initialized")
	}
	intfEntry.VrrpIntfStateList = vrrpIntfStates
	intfEntry.StartIdx = fromIndex
	intfEntry.EndIdx = vrrpd.Int(nextIdx)
	intfEntry.Count = vrrpd.Int(currCount)
	intfEntry.More = (nextIdx != 0)
	return intfEntry, nil
}

func VrrpNewHandler(vrrpSvr *vrrpServer.VrrpServer, logger *syslog.Writer) *VrrpHandler {
	hdl := new(VrrpHandler)
	hdl.server = vrrpSvr
	hdl.logger = logger
	return hdl
}

func VrrpRpcGetClient(logger *syslog.Writer, fileName string, process string) (*VrrpClientJson, error) {
	var allClients []VrrpClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Err(fmt.Sprintf("Failed to open OSPFd config file:%s, err:%s", fileName, err))
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

func StartServer(log *syslog.Writer, handler *VrrpHandler, paramsDir string) error {
	logger := log
	fileName := paramsDir

	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	clientJson, err := VrrpRpcGetClient(logger, fileName, "vrrpd")
	if err != nil || clientJson == nil {
		return err
	}
	// create transport and protocol for server
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket("localhost:" + strconv.Itoa(clientJson.Port))
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
