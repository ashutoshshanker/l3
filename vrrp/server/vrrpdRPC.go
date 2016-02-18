package vrrpServer

import (
	"vrrpd"
)

func (h *VrrpServiceHandler) CreateVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	return true, nil
}
func (h *VrrpServiceHandler) UpdateVrrpIntfConfig(origconfig *vrrpd.VrrpIntfConfig,
	newconfig *vrrpd.VrrpIntfConfig, attrset []bool) (r bool, err error) {
	return true, nil
}

func (h *VrrpServiceHandler) DeleteVrrpIntfConfig(config *vrrpd.VrrpIntfConfig) (r bool, err error) {
	return true, nil
}
