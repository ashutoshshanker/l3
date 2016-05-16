package rpc

import (
	"dhcpd"
	"fmt"
)

func (h *DHCPHandler) UpdateDhcpGlobalConfig(origConf *dhcpd.DhcpGlobalConfig, newConf *dhcpd.DhcpGlobalConfig, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original Dhcp global config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New Dhcp gloabl config attrs:", newConf))
	return true, nil
}

func (h *DHCPHandler) UpdateDhcpIntfConfig(origConf *dhcpd.DhcpIntfConfig, newConf *dhcpd.DhcpIntfConfig, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original Dhcp Intf config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New Dhcp Intf config attrs:", newConf))
	return true, nil
}
