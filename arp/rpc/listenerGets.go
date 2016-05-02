package rpc

import (
	"arpd"
	"fmt"
)

func (h *ARPHandler) GetArpEntryState(ipAddr string) (*arpd.ArpEntryState, error) {
	h.logger.Info(fmt.Sprintln("Get call for ArpEntry...", ipAddr))
	arpEntryResponse := arpd.NewArpEntryState()
	/* FIXME: When get is implemented return "Internal Vlan" for vlanId during display
	 * when vlan == asicdCommonDefs.SYS_RSVD_VLAN */
	arpEntry, err := h.server.GetArpEntry(ipAddr)
	if err != nil {
		return nil, err
	}
	arpEntryResponse = h.convertArpEntryToThrift(arpEntry)
	return arpEntryResponse, nil
}
