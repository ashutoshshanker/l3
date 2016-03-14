// lahandler
package rpc

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"vxland"
	//"time"
	//"errors"
)

const DBName string = "UsrConfDb.db"

type VXLANDServiceHandler struct {
}

func NewVXLANDServiceHandler() *VXLANDServiceHandler {
	//lacp.LacpStartTime = time.Now()
	// link up/down events for now
	//startEvtHandler()
	return &VXLANDServiceHandler{}
}

func (v *VXLANDServiceHandler) CreateVxlanInstance(config *vxland.VxlanInstance) (bool, error) {
	fmt.Println("CreateVxlanConfigInstance %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanInstance(config *vxland.VxlanInstance) (bool, error) {
	fmt.Println("DeleteVxlanConfigInstance %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanInstance(origconfig *vxland.VxlanInstance, newconfig *vxland.VxlanInstance, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanConfigInstance orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanVtepInstances(config *vxland.VxlanVtepInstances) (bool, error) {
	fmt.Println("CreateVxlanVtepInstances %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanVtepInstances(config *vxland.VxlanVtepInstances) (bool, error) {
	fmt.Println("DeleteVxlanVtepInstances %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanVtepInstances(origconfig *vxland.VxlanVtepInstances, newconfig *vxland.VxlanVtepInstances, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanVtepInstances orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanStaticVxlanTunnelAddressFamilyBindVxlanId(config *vxland.VxlanStaticVxlanTunnelAddressFamilyBindVxlanId) (bool, error) {
	fmt.Println("CreateVxlanStaticVxlanTunnelAddressFamilyBindVxlanId %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanStaticVxlanTunnelAddressFamilyBindVxlanId(config *vxland.VxlanStaticVxlanTunnelAddressFamilyBindVxlanId) (bool, error) {
	fmt.Println("DeleteVxlanStaticVxlanTunnelAddressFamilyBindVxlanId %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanStaticVxlanTunnelAddressFamilyBindVxlanId(origconfig *vxland.VxlanStaticVxlanTunnelAddressFamilyBindVxlanId, newconfig *vxland.VxlanStaticVxlanTunnelAddressFamilyBindVxlanId, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanVxlanStaticVxlanTunnelAddressFamilyBindVxlanId orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (s *VXLANDServiceHandler) ReadConfigFromDB(filePath string) error {
	var dbPath string = filePath + DBName

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
