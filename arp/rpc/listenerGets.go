package rpc

import (
	"arpd"
	"fmt"
)

func (h *ARPHandler) GetArpEntry(ipAddr string) (*arpd.ArpEntry, error) {
	h.logger.Info(fmt.Sprintln("Get call for ArpEntry...", ipAddr))
	arpEntryResponse := arpd.NewArpEntry()
	return arpEntryResponse, nil
}
