package rpc

import (
        "fmt"
        "arpd"
)

func (h *ARPHandler) UpdateArpConfig(origConf *arpd.ArpConfig, newConf *arpd.ArpConfig, attrset []bool) (bool, error) {
        h.logger.Info(fmt.Sprintln("Original Arp config attrs:", origConf))
        h.logger.Info(fmt.Sprintln("New Arp config attrs:", newConf))
        return true, nil
}
