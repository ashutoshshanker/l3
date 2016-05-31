//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//

// lahandler
package rpc

import (
	//"database/sql"
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	//_ "github.com/mattn/go-sqlite3"
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
	//handler.ReadConfigFromDB()

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

	c, err := v.server.ConvertVxlanInstanceToVxlanConfig(config)
	if err == nil {
		v.server.Configchans.Vxlancreate <- *c
		return true, nil
	}
	return false, err
}

func (v *VXLANDServiceHandler) DeleteVxlanInstance(config *vxland.VxlanInstance) (bool, error) {
	v.logger.Info(fmt.Sprintf("DeleteVxlanConfigInstance %#v", config))
	c, err := v.server.ConvertVxlanInstanceToVxlanConfig(config)
	if err == nil {
		v.server.Configchans.Vxlandelete <- *c
		return true, nil
	}
	return false, err
}

func (v *VXLANDServiceHandler) UpdateVxlanInstance(origconfig *vxland.VxlanInstance, newconfig *vxland.VxlanInstance, attrset []bool, op []*vxland.PatchOpInfo) (bool, error) {
	v.logger.Info(fmt.Sprintf("UpdateVxlanConfigInstance orig[%#v] new[%#v]", origconfig, newconfig))
	oc, _ := v.server.ConvertVxlanInstanceToVxlanConfig(origconfig)
	nc, err := v.server.ConvertVxlanInstanceToVxlanConfig(newconfig)
	if err == nil {
		update := vxlan.VxlanUpdate{
			Oldconfig: *oc,
			Newconfig: *nc,
			Attr:      attrset,
		}
		v.server.Configchans.Vxlanupdate <- update
		return true, nil
	}
	return false, err
}

func (v *VXLANDServiceHandler) CreateVxlanVtepInstances(config *vxland.VxlanVtepInstances) (bool, error) {
	v.logger.Info(fmt.Sprintf("CreateVxlanVtepInstances %#v", config))
	c, err := v.server.ConvertVxlanVtepInstanceToVtepConfig(config)
	if err == nil {
		v.server.Configchans.Vtepcreate <- *c
		return true, err
	}
	return false, err
}

func (v *VXLANDServiceHandler) DeleteVxlanVtepInstances(config *vxland.VxlanVtepInstances) (bool, error) {
	v.logger.Info(fmt.Sprintf("DeleteVxlanVtepInstances %#v", config))
	c, err := v.server.ConvertVxlanVtepInstanceToVtepConfig(config)
	if err == nil {
		v.server.Configchans.Vtepdelete <- *c
		return true, nil
	}
	return false, err
}

func (v *VXLANDServiceHandler) UpdateVxlanVtepInstances(origconfig *vxland.VxlanVtepInstances, newconfig *vxland.VxlanVtepInstances, attrset []bool, op []*vxland.PatchOpInfo) (bool, error) {
	v.logger.Info(fmt.Sprintf("UpdateVxlanVtepInstances orig[%#v] new[%#v]", origconfig, newconfig))
	oc, _ := v.server.ConvertVxlanVtepInstanceToVtepConfig(origconfig)
	nc, err := v.server.ConvertVxlanVtepInstanceToVtepConfig(newconfig)
	if err == nil {
		update := vxlan.VtepUpdate{
			Oldconfig: *oc,
			Newconfig: *nc,
			Attr:      attrset,
		}
		v.server.Configchans.Vtepupdate <- update
		return true, nil
	}

	return false, err
}

/*
func (v *VXLANDServiceHandler) HandleDbReadVxlanInstance(dbHdl *sql.DB) error {
	dbCmd := "select * from VxlanInstance"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		fmt.Println(fmt.Sprintf("DB method Query failed for 'VxlanInstance' with error ", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(vxland.VxlanInstance)
		if err = rows.Scan(&object.VxlanId, &object.McDestIp, &object.VlanId, &object.Mtu); err != nil {

			fmt.Println("Db method Scan failed when interating over VxlanInstance")
		}
		_, err = v.CreateVxlanInstance(object)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *VXLANDServiceHandler) HandleDbReadVxlanVtepInstances(dbHdl *sql.DB) error {
	dbCmd := "select * from VxlanVtepInstances"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		fmt.Println(fmt.Sprintf("DB method Query failed for 'VxlanVtepInstances' with error ", dbCmd, err))
		return err
	}

	defer rows.Close()

	for rows.Next() {

		object := new(vxland.VxlanVtepInstances)
		if err = rows.Scan(&object.VtepId, &object.VxlanId, &object.VtepName, &object.SrcIfIndex, &object.UDP, &object.TTL, &object.TOS, &object.InnerVlanHandlingMode, &object.Learning, &object.Rsc, &object.L2miss, &object.L3miss, &object.DstIp, &object.DstMac, &object.VlanId); err != nil {

			fmt.Println("Db method Scan failed when interating over VxlanVtepInstances")
		}
		_, err = v.CreateVxlanVtepInstances(object)
		if err != nil {
			return err
		}
	}
	return nil
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

	if err := v.HandleDbReadVxlanInstance(dbHdl); err != nil {
		//stp.StpLogger("ERROR", "Error getting All VxlanInstance objects")
		return err
	}

	if err = v.HandleDbReadVxlanVtepInstances(dbHdl); err != nil {
		//stp.StpLogger("ERROR", "Error getting All VxlanVtepInstance objects")
		return err
	}

	return nil
}
*/
