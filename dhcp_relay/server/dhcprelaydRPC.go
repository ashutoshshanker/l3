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
	DhcpRelay string `SNAPROUTE: "KEY"`
	Enable    bool
}

/*
 * This DS will be used while adding/deleting Relay Agent.
 */
type DhcpRelayIntfConfig struct {
	IpSubnet string `SNAPROUTE: "KEY"`
	Netmask  string `SNAPROUTE: "KEY"`
	//@TODO: Need to check if_index type
	IfIndex string `SNAPROUTE: "KEY"`
	// Use below field for agent sub-type
	AgentSubType int32
	Enable       bool
	// To make life easy for testing first pass lets have only 1 server
	//ServerIp     []string
	ServerIp string
}

/******** Trift APIs *******/
/*
 * Add a relay agent
 */

func (h *DhcpRelayServiceHandler) CreateDhcpRelayGlobalConfig(
	config *dhcprelayd.DhcpRelayGlobalConfig) (bool, error) {

	if config.Enable {
		fmt.Println("Enabling Dhcp Relay Global Config")
	} else {
		fmt.Println("Disabling Dhcp Relay Global Config")
	}
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayGlobalConfig(
	origconfig *dhcprelayd.DhcpRelayGlobalConfig,
	newconfig *dhcprelayd.DhcpRelayGlobalConfig,
	attrset []bool) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayGlobalConfig(
	config *dhcprelayd.DhcpRelayGlobalConfig) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) CreateDhcpRelayIntfConfig(
	config *dhcprelayd.DhcpRelayIntfConfig) (bool, error) {
	logger.Info("DRA: Intf Config Create")
	fmt.Println("Creating Dhcp Relay Config for interface")
	fmt.Println("IpSubnet:", config.IpSubnet)
	fmt.Println("Netmask:", config.Netmask)
	fmt.Println("IF Index:", config.IfIndex)
	fmt.Println("AgentSubType:", config.AgentSubType)
	fmt.Println("Enable:", config.Enable)
	fmt.Println("ServerIp:", config.ServerIp)
	// Copy over configuration into globalInfo
	gblEntry := dhcprelayGblInfo[config.IfIndex]
	// Acquire lock for updating configuration.
	gblEntry.dhcprelayConfigMutex.RLock()
	gblEntry.IntfConfig.IpSubnet = config.IpSubnet
	gblEntry.IntfConfig.Netmask = config.Netmask
	gblEntry.IntfConfig.AgentSubType = config.AgentSubType
	gblEntry.IntfConfig.Enable = config.Enable
	dhcprelayGblInfo[config.IfIndex] = gblEntry
	// Release lock after updation is done
	gblEntry.dhcprelayConfigMutex.RUnlock()

	// Stats information
	DhcpRelayAgentUpdateStats("dhcp relay config create request",
		gblEntry)
	go DhcpRelayAgentReceiveDhcpPkt(gblEntry)
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayIntfConfig(
	origconfig *dhcprelayd.DhcpRelayIntfConfig,
	newconfig *dhcprelayd.DhcpRelayIntfConfig,
	attrset []bool) (bool, error) {
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayIntfConfig(
	config *dhcprelayd.DhcpRelayIntfConfig) (bool, error) {
	return true, nil
}
