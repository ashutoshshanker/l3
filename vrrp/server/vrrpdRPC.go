package vrrpServer

import (
	"errors"
	"fmt"
	"strconv"
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

	gblInfo.IntfConfig.IfIndex = config.IfIndex
	if config.VRID == 0 {
		logger.Info("VRRP: Invalid VRID")
		return false, errors.New(VRRP_INVALID_VRID)
	}
	gblInfo.IntfConfig.VRID = config.VRID

	if config.Priority == 0 {
		logger.Info("VRRP: Setting default priority which is 100")
		gblInfo.IntfConfig.Priority = 100
	} else {
		gblInfo.IntfConfig.Priority = config.Priority
	}

	gblInfo.IntfConfig.VirtualIPv4Addr = config.VirtualIPv4Addr

	if config.AdvertisementInterval == 0 {
		logger.Info("VRRP: Setting default advertisment interval to 1 sec")
		gblInfo.IntfConfig.AdvertisementInterval = 1
	} else {
		gblInfo.IntfConfig.AdvertisementInterval = config.AdvertisementInterval
	}

	/* @TODO: hack for setting it to TRUE right until default is taken care
	* off
		* if config.PreemptMode == false {
			gblInfo.IntfConfig.PreemptMode = false
		} else {

			gblInfo.IntfConfig.PreemptMode = true
		}
	*/
	gblInfo.IntfConfig.PreemptMode = true

	if config.AcceptMode == true {
		gblInfo.IntfConfig.AcceptMode = true
	} else {
		gblInfo.IntfConfig.AcceptMode = false
	}

	if config.VirtualRouterMACAddress != "" {
		gblInfo.IntfConfig.VirtualRouterMACAddress = config.VirtualRouterMACAddress
	} else {
		if gblInfo.IntfConfig.VRID < 10 {
			gblInfo.IntfConfig.VirtualRouterMACAddress = "00-00-5E-00-01-0" +
				strconv.Itoa(int(gblInfo.IntfConfig.VRID))

		} else {
			gblInfo.IntfConfig.VirtualRouterMACAddress = "00-00-5E-00-01-" +
				strconv.Itoa(int(gblInfo.IntfConfig.VRID))
		}
	}

	vrrpGblInfo[config.IfIndex] = gblInfo
	go VrrpUpdateGblInfoTimers(config.IfIndex)
	go VrrpInitPacketListener()
	return true, nil
}
func (h *VrrpServiceHandler) UpdateVrrpIntfConfig(origconfig *vrrpd.VrrpIntfConfig,
	newconfig *vrrpd.VrrpIntfConfig, attrset []bool) (r bool, err error) {
	return true, nil
}

func (h *VrrpServiceHandler) DeleteVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	return true, nil
}
