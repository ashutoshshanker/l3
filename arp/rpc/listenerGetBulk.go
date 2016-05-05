package rpc

import (
	"arpd"
	"asicd/asicdCommonDefs"
	"errors"
	"fmt"
	"l3/arp/server"
	"strconv"
)

func (h *ARPHandler) convertArpEntryToThrift(arpState server.ArpState) *arpd.ArpEntryState {
	arpEnt := arpd.NewArpEntryState()
	arpEnt.IpAddr = arpState.IpAddr
	arpEnt.MacAddr = arpState.MacAddr
	if arpState.VlanId == asicdCommonDefs.SYS_RSVD_VLAN {
		arpEnt.Vlan = "Internal Vlan"
	} else {
		arpEnt.Vlan = strconv.Itoa((arpState.VlanId))
	}
	arpEnt.Intf = arpState.Intf
	arpEnt.ExpiryTimeLeft = arpState.ExpiryTimeLeft
	return arpEnt
}

func (h *ARPHandler) GetBulkArpEntryState(fromIdx arpd.Int, count arpd.Int) (*arpd.ArpEntryStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("GetBulk call for ArpEntries..."))
	nextIdx, currCount, arpEntries := h.server.GetBulkArpEntry(int(fromIdx), int(count))
	if arpEntries == nil {
		err := errors.New("Arp is busy refreshing the Arp Cache")
		return nil, err
	}
	arpEntryResponse := make([]*arpd.ArpEntryState, len(arpEntries))
	for idx, item := range arpEntries {
		arpEntryResponse[idx] = h.convertArpEntryToThrift(item)
	}
	arpEntryBulk := arpd.NewArpEntryStateGetInfo()
	arpEntryBulk.Count = arpd.Int(currCount)
	arpEntryBulk.StartIdx = arpd.Int(fromIdx)
	arpEntryBulk.EndIdx = arpd.Int(nextIdx)
	arpEntryBulk.More = (nextIdx != 0)
	arpEntryBulk.ArpEntryStateList = arpEntryResponse
	return arpEntryBulk, nil
}
