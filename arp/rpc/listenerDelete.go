package rpc

import (
	"arpd"
	"fmt"
	"l3/arp/server"
)

func (h *ARPHandler) SendDeleteResolveArpIPv4(NextHopIp string) {
	rConf := server.DeleteResolvedIPv4{
		IpAddr: NextHopIp,
	}

	h.server.DeleteResolvedIPv4Ch <- rConf
	return
}

func (h *ARPHandler) DeleteArpConfig(conf *arpd.ArpConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Arp config attrs:", conf))
	return true, nil
}

func (h *ARPHandler) DeleteResolveArpIPv4(NextHopIp string) error {
	h.logger.Info(fmt.Sprintln("Received DeleteResolveArpIPv4 call with NextHopIp:", NextHopIp))
	h.SendDeleteResolveArpIPv4(NextHopIp)
	return nil
}
