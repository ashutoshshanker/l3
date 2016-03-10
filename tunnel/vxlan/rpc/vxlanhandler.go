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

func (v *VXLANDServiceHandler) CreateVxlanVxlanInstanceAccessTypeL3interfaceL3interface(config *vxland.VxlanVxlanInstanceAccessTypeL3interfaceL3interface) (bool, error) {
	fmt.Println("CreateVxlanVxlanInstanceAccessTypeL3interfaceL3interface %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanVxlanInstanceAccessTypeL3interfaceL3interface(config *vxland.VxlanVxlanInstanceAccessTypeL3interfaceL3interface) (bool, error) {
	fmt.Println("DeleteVxlanVxlanInstanceAccessTypeL3interfaceL3interface %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanVxlanInstanceAccessTypeL3interfaceL3interface(origconfig *vxland.VxlanVxlanInstanceAccessTypeL3interfaceL3interface, newconfig *vxland.VxlanVxlanInstanceAccessTypeL3interfaceL3interface, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanVxlanInstanceAccessTypeL3interfaceL3interface orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanVxlanInstanceVxlanEvpnVpnTargets(config *vxland.VxlanVxlanInstanceVxlanEvpnVpnTargets) (bool, error) {
	fmt.Println("CreateVxlanVxlanInstanceVxlanEvpnVpnTargets %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanVxlanInstanceVxlanEvpnVpnTargets(config *vxland.VxlanVxlanInstanceVxlanEvpnVpnTargets) (bool, error) {
	fmt.Println("DeleteVxlanVxlanInstanceVxlanEvpnVpnTargets %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanVxlanInstanceVxlanEvpnVpnTargets(origconfig *vxland.VxlanVxlanInstanceVxlanEvpnVpnTargets, newconfig *vxland.VxlanVxlanInstanceVxlanEvpnVpnTargets, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanVxlanInstanceVxlanEvpnVpnTargets orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanVxlanInstanceAccessTypeMac(config *vxland.VxlanVxlanInstanceAccessTypeMac) (bool, error) {
	fmt.Println("CreateVxlanVxlanInstanceAccessTypeMac %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanVxlanInstanceAccessTypeMac(config *vxland.VxlanVxlanInstanceAccessTypeMac) (bool, error) {
	fmt.Println("DeleteVxlanVxlanInstanceAccessTypeMac %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanVxlanInstanceAccessTypeMac(origconfig *vxland.VxlanVxlanInstanceAccessTypeMac, newconfig *vxland.VxlanVxlanInstanceAccessTypeMac, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanVxlanInstanceAccessTypeMac orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId(config *vxland.VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId) (bool, error) {
	fmt.Println("CreateVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId(config *vxland.VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId) (bool, error) {
	fmt.Println("DeleteVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId(origconfig *vxland.VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId, newconfig *vxland.VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanInterfacesInterfaceVtepInstancesBindVxlanId(config *vxland.VxlanInterfacesInterfaceVtepInstancesBindVxlanId) (bool, error) {
	fmt.Println("CreateVxlanInterfacesInterfaceVtepInstancesBindVxlanId %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanInterfacesInterfaceVtepInstancesBindVxlanId(config *vxland.VxlanInterfacesInterfaceVtepInstancesBindVxlanId) (bool, error) {
	fmt.Println("DeleteVxlanInterfacesInterfaceVtepInstancesBindVxlanId %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanInterfacesInterfaceVtepInstancesBindVxlanId(origconfig *vxland.VxlanInterfacesInterfaceVtepInstancesBindVxlanId, newconfig *vxland.VxlanInterfacesInterfaceVtepInstancesBindVxlanId, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanInterfacesInterfaceVtepInstancesBindVxlanId orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) CreateVxlanVxlanInstanceAccessTypeVlanVlanList(config *vxland.VxlanVxlanInstanceAccessTypeVlanVlanList) (bool, error) {
	fmt.Println("CreateVxlanVxlanInstanceAccessTypeVlanVlanList %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) DeleteVxlanVxlanInstanceAccessTypeVlanVlanList(config *vxland.VxlanVxlanInstanceAccessTypeVlanVlanList) (bool, error) {
	fmt.Println("DeleteVxlanVxlanInstanceAccessTypeVlanVlanList %#v", config)
	return true, nil
}

func (v *VXLANDServiceHandler) UpdateVxlanVxlanInstanceAccessTypeVlanVlanList(origconfig *vxland.VxlanVxlanInstanceAccessTypeVlanVlanList, newconfig *vxland.VxlanVxlanInstanceAccessTypeVlanVlanList, attrset []bool) (bool, error) {
	fmt.Println("UpdateVxlanVxlanInstanceAccessTypeVlanVlanList orig[%#v] new[%#v]", origconfig, newconfig)
	return true, nil
}

func (v *VXLANDServiceHandler) GetBulkVxlanStateVxlanInstanceVxlanEvpnVpnTargets(fromIndex vxland.Int, count vxland.Int) (obj *vxland.VxlanStateVxlanInstanceVxlanEvpnVpnTargetsGetInfo, err error) {
	return obj, err
}

func (v *VXLANDServiceHandler) GetBulkVxlanStateStaticVxlanTunnelAddressFamilyBindVxlanId(fromIndex vxland.Int, count vxland.Int) (obj *vxland.VxlanStateStaticVxlanTunnelAddressFamilyBindVxlanIdGetInfo, err error) {
	return obj, err
}

func (v *VXLANDServiceHandler) GetBulkVxlanStateVxlanInstanceAccessVlan(fromIndex vxland.Int, count vxland.Int) (obj *vxland.VxlanStateVxlanInstanceAccessVlanGetInfo, err error) {
	return obj, err
}

func (v *VXLANDServiceHandler) GetBulkVxlanStateVxlanInstanceMapL3interface(fromIndex vxland.Int, count vxland.Int) (obj *vxland.VxlanStateVxlanInstanceMapL3interfaceGetInfo, err error) {
	return obj, err
}

func (v *VXLANDServiceHandler) GetBulkVxlanStateVtepInstanceBindVxlanId(fromIndex vxland.Int, count vxland.Int) (obj *vxland.VxlanStateVtepInstanceBindVxlanIdGetInfo, err error) {
	return obj, err
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
