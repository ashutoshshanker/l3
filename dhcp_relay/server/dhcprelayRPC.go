package relayServer

import (
	"dhcprelayd"
	"fmt"
)

/*
 * Global DataStructure for DHCP RELAY
 */
type DhcpRelayGlobalConfig struct {
	// This will tell whether DHCP RELAY is enabled/disabled
	// on the box right now or not.
	Enable bool `SNAPROUTE: "KEY"`
}

/*
 * This DS will be used while adding/deleting Relay Agent.
 */
type DhcpRelayIntfConfig struct {
	IpSubnet string `SNAPROUTE: "KEY"`
	Netmask  string `SNAPROUTE: "KEY"`
	//@TODO: Need to check if_index type
	IfIndex string
	// Use below field for agent sub-type
	AgentSubType int32
	Enable       bool
	ServerIp     []string
}

/******** Trift APIs *******/
/*
 * Add a relay agent
 */

func (h *DhcpRelayServiceHandler) CreateDhcpRelayGlobal(
	config *dhcprelayd.DhcpRelayGlobalConfig) (bool, error) {
	fmt.Println("Dhcp Relay %d", config.Enable)
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayGlobal(
	origconfig *dhcprelayd.DhcpRelayGlobalConfig,
	newconfig *dhcprelayd.DhcpRelayGlobalConfig,
	attrset []bool) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayGlobal(
	config *dhcprelayd.DhcpRelayGlobalConfig) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) CreateDhcpRelayConf(
	config *dhcprelayd.DhcpRelayIntfConfig) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayConf(
	origconfig *dhcprelayd.DhcpRelayIntfConfig,
	newconfig *dhcprelayd.DhcpRelayIntfConfig,
	attrset []bool) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayConf(
	config *dhcprelayd.DhcpRelayIntfConfig) (bool, error) {
	return true, nil
}
