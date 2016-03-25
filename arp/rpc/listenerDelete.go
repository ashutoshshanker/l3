package rpc

import (
        "fmt"
        "arpd"
)

func (h *ARPHandler) DeleteArpConfig(conf *arpd.ArpConfig) (bool, error) {
        h.logger.Info(fmt.Sprintln("Delete Arp config attrs:", conf))
        return true, nil
}
