package rpc

import (
	"arpd"
	"fmt"
	"l3/arp/server"
)

func (h *ARPHandler) ExecuteActionArpDeleteByIfName(config *arpd.ArpDeleteByIfName) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received ArpDeleteByIfName for", config))
	msg := server.ArpActionMsg{
		Type: server.DeleteByIfName,
		Obj:  config.IfName,
	}
	h.server.ArpActionCh <- msg
	return true, nil
}

func (h *ARPHandler) ExecuteActionArpDeleteByIPv4Addr(config *arpd.ArpDeleteByIPv4Addr) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received ArpDeleteByIPv4Addr for", config))
	msg := server.ArpActionMsg{
		Type: server.DeleteByIPAddr,
		Obj:  config.IpAddr,
	}
	h.server.ArpActionCh <- msg
	return true, nil
}

func (h *ARPHandler) ExecuteActionArpRefreshByIfName(config *arpd.ArpRefreshByIfName) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received ArpRefreshByIfName for", config))
	msg := server.ArpActionMsg{
		Type: server.RefreshByIfName,
		Obj:  config.IfName,
	}
	h.server.ArpActionCh <- msg
	return true, nil
}

func (h *ARPHandler) ExecuteActionArpRefreshByIPv4Addr(config *arpd.ArpRefreshByIPv4Addr) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received ArpRefreshByIPv4Addr for", config))
	msg := server.ArpActionMsg{
		Type: server.RefreshByIPAddr,
		Obj:  config.IpAddr,
	}
	h.server.ArpActionCh <- msg
	return true, nil
}
