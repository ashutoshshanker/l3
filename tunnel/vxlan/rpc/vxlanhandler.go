// lahandler
package rpc

import (
	"database/sql"
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	_ "github.com/mattn/go-sqlite3"
	vxlan "l3/tunnel/vxlan/protocol"
	"utils/logging"
	"vxland"
)

const DBName string = "UsrConfDb.db"

type VXLANDServiceHandler struct {
	server *vxlan.VXLANServer
	logger *logging.Writer
}

func NewVXLANDServiceHandler(server *vxlan.VXLANServer, logger *logging.Writer) *VXLANDServiceHandler {
	//lacp.LacpStartTime = time.Now()
	// link up/down events for now
	//startEvtHandler()
	handler := &VXLANDServiceHandler{
		server: server,
		logger: logger,
	}

	// lets read the current config and re-play the config
	handler.ReadConfigFromDB()

	return handler
}

func (v *VXLANDServiceHandler) StartThriftServer() {

	var transport thrift.TServerTransport
	var err error

	fileName := v.server.Paramspath + "clients.json"
	port := vxlan.GetClientPort(fileName, "vxland")
	if port != 0 {
		addr := fmt.Sprintf("localhost:%d", port)
		transport, err = thrift.NewTServerSocket(addr)
		if err != nil {
			panic(fmt.Sprintf("Failed to create Socket with:", addr))
		}

		processor := vxland.NewVXLANDServicesProcessor(v)
		transportFactory := thrift.NewTBufferedTransportFactory(8192)
		protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
		thriftserver := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)

		err = thriftserver.Serve()
		panic(err)
	}
	panic(errors.New("Unable to find vxland port"))
}

func (v *VXLANDServiceHandler) CreateVxlanInstance(config *vxland.VxlanInstance) (bool, error) {
	v.logger.Info(fmt.Sprintf("CreateVxlanConfigInstance %#v", config))
	v.server.Configchans.Vxlancreate <- *config
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanInstance(config *vxland.VxlanInstance) (bool, error) {
	v.logger.Info(fmt.Sprintf("DeleteVxlanConfigInstance %#v", config))
	v.server.Configchans.Vxlandelete <- *config
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanInstance(origconfig *vxland.VxlanInstance, newconfig *vxland.VxlanInstance, attrset []bool) (bool, error) {
	v.logger.Info(fmt.Sprintf("UpdateVxlanConfigInstance orig[%#v] new[%#v]", origconfig, newconfig))
	update := vxlan.VxlanUpdate{
		Oldconfig: *origconfig,
		Newconfig: *newconfig,
		Attr:      attrset,
	}
	v.server.Configchans.Vxlanupdate <- update
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanVtepInstances(config *vxland.VxlanVtepInstances) (bool, error) {
	v.logger.Info(fmt.Sprintf("CreateVxlanVtepInstances %#v", config))
	v.server.Configchans.Vtepcreate <- *config
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanVtepInstances(config *vxland.VxlanVtepInstances) (bool, error) {
	v.logger.Info(fmt.Sprintf("DeleteVxlanVtepInstances %#v", config))
	v.server.Configchans.Vtepdelete <- *config
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanVtepInstances(origconfig *vxland.VxlanVtepInstances, newconfig *vxland.VxlanVtepInstances, attrset []bool) (bool, error) {
	v.logger.Info(fmt.Sprintf("UpdateVxlanVtepInstances orig[%#v] new[%#v]", origconfig, newconfig))
	update := vxlan.VtepUpdate{
		Oldconfig: *origconfig,
		Newconfig: *newconfig,
		Attr:      attrset,
	}
	v.server.Configchans.Vtepupdate <- update
	return true, nil
}

func (v *VXLANDServiceHandler) ReadConfigFromDB() error {
	var dbPath string = v.server.Paramspath + DBName

	dbHdl, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		//h.logger.Err(fmt.Sprintf("Failed to open the DB at %s with error %s", dbPath, err))
		//stp.StpLogger("ERROR", fmt.Sprintf("Failed to open the DB at %s with error %s", dbPath, err))
		return err
	}

	defer dbHdl.Close()
	/*
		if err := s.HandleDbReadDot1dStpBridgeConfig(dbHdl); err != nil {
			stp.StpLogger("ERROR", "Error getting All Dot1dStpBridgeConfig objects")
			return err
		}

		if err = s.HandleDbReadDot1dStpPortEntryConfig(dbHdl); err != nil {
			stp.StpLogger("ERROR", "Error getting All Dot1dStpPortEntryConfig objects")
			return err
		}
	*/

	return nil
}
