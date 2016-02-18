package vrrpServer

import (
	"errors"
	"fmt"
	"vrrpd"
)

/*
	IfIndex                 int32  `SNAPROUTE: "KEY", ACCESS:"w",  MULTIPLICITY:"*"`
	VRID                    int32  // no default for VRID
	Priority                int32  // default value is 100
	VirtualIPv4Addr         string // No Default for Virtual IPv4 addr.. Can support one or more
	AdvertisementInterval   int32  // Default is 100 centiseconds which is 1 SEC
	PreemptMode             bool   // False to prohibit preemption. Default is True.
	AcceptMode              bool   // The default is False.
	VirtualRouterMACAddress string // MAC address used for the source MAC address in VRRP advertisements
*/

func (h *VrrpServiceHandler) CreateVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	logger.Info(fmt.Sprintln("VRRP: Interface config create for ifindex ",
		config.IfIndex))
	gblInfo := vrrpGblInfo[config.IfIndex]
	if config.VRID == 0 {
		logger.Info("VRRP: Invalid VRID")
		return false, errors.New(VRRP_INVALID_VRID)
	}
	if config.Priority == 0 {
		logger.Info("VRRP: Setting default priority which is 100")
		gblInfo.IntfConfig.Priority = 100
	} else {
		gblInfo.IntfConfig.Priority = config.Priority
	}
	vrrpGblInfo[config.IfIndex] = gblInfo
	return true, nil
}
func (h *VrrpServiceHandler) UpdateVrrpIntfConfig(origconfig *vrrpd.VrrpIntfConfig,
	newconfig *vrrpd.VrrpIntfConfig, attrset []bool) (r bool, err error) {
	return true, nil
}

func (h *VrrpServiceHandler) DeleteVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	return true, nil
}
