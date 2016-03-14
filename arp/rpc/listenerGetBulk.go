package rpc

import (
        "errors"
        "fmt"
        "l3/arp/server"
        "arpd"
)

func (h *ARPHandler) convertArpEntryToThrift(arpState server.ArpState) *arpd.ArpEntry {
        arpEnt := arpd.NewArpEntry()
        arpEnt.IpAddr = arpState.IpAddr
        arpEnt.MacAddr = arpState.MacAddr
        arpEnt.Vlan = int32(arpState.VlanId)
        arpEnt.Intf = arpState.Intf
        arpEnt.ExpiryTimeLeft = arpState.ExpiryTimeLeft
        return arpEnt
}

func (h *ARPHandler)GetBulkArpEntry(fromIdx arpd.Int, count arpd.Int) (*arpd.ArpEntryGetInfo, error) {
        h.logger.Info(fmt.Sprintln("GetBulk call for ArpEntries..."))
        nextIdx, currCount, arpEntries := h.server.GetBulkArpEntry(int(fromIdx), int(count))
        if arpEntries == nil {
                err := errors.New("Arp is busy refreshing the Arp Cache")
                return nil, err
        }
        arpEntryResponse := make([]*arpd.ArpEntry, len(arpEntries))
        for idx, item := range arpEntries {
                arpEntryResponse[idx] = h.convertArpEntryToThrift(item)
        }
        arpEntryBulk := arpd.NewArpEntryGetInfo()
        arpEntryBulk.Count = arpd.Int(currCount)
        arpEntryBulk.StartIdx = arpd.Int(fromIdx)
        arpEntryBulk.EndIdx = arpd.Int(nextIdx)
        arpEntryBulk.More = (nextIdx != 0)
        arpEntryBulk.ArpEntryList = arpEntryResponse
        return arpEntryBulk, nil
}
