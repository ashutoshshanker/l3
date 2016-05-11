package rpc

import (
	"arpd"
	"fmt"
)

func (h *ARPHandler) DeleteArpConfig(conf *arpd.ArpConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Arp config attrs:", conf))
	return true, nil
}
