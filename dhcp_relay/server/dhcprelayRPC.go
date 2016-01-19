package relayServer

import (
	"dhcprelayd"
	_ "fmt"
)

/*
 * Global DataStructure for DHCP RELAY
 */
/*
 type DhcpRelayGlobal struct {
	BaseObj
	// This will tell whether DHCP RELAY is enabled/disabled
	// on the box right now or not.
	Enable bool `SNAPROUTE: "KEY"`
}
*/
/*
 * This DS will be used while adding/deleting Relay Agent.
 */
/*
type DhcpRelayConf struct {
	BaseObj
	IpSubnet string `SNAPROUTE: "KEY"`
	Netmask  string `SNAPROUTE: "KEY"`
	//@TODO: Need to check if_index type
	IfIndex string
	// Use below field for agent sub-type
	AgentSubType int32
	Enable       bool
	ServerIp     []string
}
*/
/******** Trift APIs *******/
/*
 * Add a relay agent
 */

func (h *DhcpRelayServiceHandler) CreateDhcpRelayGlobal(
	config *dhcprelayd.DhcpRelayGlobal) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayGlobal(
	origconfig *dhcprelayd.DhcpRelayGlobal,
	newconfig *dhcprelayd.DhcpRelayGlobal,
	attrset []bool) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayGlobal(
	config *dhcprelayd.DhcpRelayGlobal) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) CreateDhcpRelayConf(
	config *dhcprelayd.DhcpRelayConf) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayConf(
	origconfig *dhcprelayd.DhcpRelayConf,
	newconfig *dhcprelayd.DhcpRelayConf,
	attrset []bool) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayConf(
	config *dhcprelayd.DhcpRelayConf) (bool, error) {
	return true, nil
}
