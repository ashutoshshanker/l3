package rpc

import (
        "fmt"
        "arpd"
)

func (h *ARPHandler)GetArpEntry(arpdGlobal *arpd.ArpEntry) (*arpd.ArpEntry, error) {
        h.logger.Info(fmt.Sprintln("Get call for ArpEntry..."))
        arpEntryResponse := arpd.NewArpEntry()
        return arpEntryResponse, nil
}
