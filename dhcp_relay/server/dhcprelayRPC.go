package relayServer

import (
	"dhcprelayd"
	"fmt"
	//"log/syslog"
)

type DhcpRelayServiceHandler struct {
}

/******** Trift APIs *******/
/*
 * Add a relay agent
 */
func (h *DhcpRelayServiceHandler) AddRelayAgent(dhcprelayGlobal *dhcprelayd.DhcpRelayConf) error {
	fmt.Println("Add Relay Agent")
	fmt.Println("Ip address is", dhcprelayGlobal.IpSubnet)
	fmt.Println("if_index is %s", dhcprelayGlobal.IfIndex)
	return nil
}

func (h *DhcpRelayServiceHandler) DelRelayAgent() error {
	fmt.Println("Del Relay Agent")
	return nil
}

func (h *DhcpRelayServiceHandler) UpdRelayAgent() error {
	fmt.Println("Update Relay Agent")
	return nil
}
