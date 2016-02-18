package vrrpServer

import (
	"errors"
	"vrrpd"
)

func (h *VrrpServiceHandler) CreateVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	//logger.Info(fmt.Sprintln("VRRP: Interface config create for ifindex "), config.IfIndex)
	/*
		2 : i32 VRID
		3 : i32 Priority
		4 : string IPv4Addr
		5 : i32 AdvertisementInterval
		6 : bool PreemptMode
		7 : bool AcceptMode
		8 : string VirtualRouterMACAddress
	*/
	gblInfo := vrrpGblInfo[0] // @TODO: Fix this once Hari have the fix for thrift file
	if config.VRID == 0 {
		logger.Info("VRRP: Invalid VRID")
		return false, errors.New(INVALID_VRID)
	}
	if config.Priority == 0 {
		logger.Info("VRRP: Setting default priority which is 100")
		gblInfo.IntfConfig.Priority = 100
	} else {
		gblInfo.IntfConfig.Priority = config.Priority
	}
	return true, nil
}
func (h *VrrpServiceHandler) UpdateVrrpIntfConfig(origconfig *vrrpd.VrrpIntfConfig,
	newconfig *vrrpd.VrrpIntfConfig, attrset []bool) (r bool, err error) {
	return true, nil
}

func (h *VrrpServiceHandler) DeleteVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	return true, nil
}
