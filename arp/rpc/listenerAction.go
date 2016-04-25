package rpc

import (
	"arpd"
	"fmt"
)

func (h *ARPHandler) ExecuteActionArpDeleteByIfName(config *arpd.ArpDeleteByIfName) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received ArpDeleteByIfName for", config))
	return true, nil
}

func (h *ARPHandler) ExecuteActionArpDeleteByIPv4Addr(config *arpd.ArpDeleteByIPv4Addr) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received ArpDeleteByIPv4Addr for", config))
	return true, nil
}
