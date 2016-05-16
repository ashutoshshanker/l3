package rpc

import (
	"dhcpd"
	"fmt"
)

func (h *DHCPHandler) DeleteDhcpGlobalConfig(conf *dhcpd.DhcpGlobalConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Dhcp global config attrs:", conf))
	return true, nil
}

func (h *DHCPHandler) DeleteDhcpIntfConfig(conf *dhcpd.DhcpIntfConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Dhcp Intf:", conf))
	return true, nil
}
