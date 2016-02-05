package relayServer

import (
	"dhcprelayd"
	"fmt"
	"strconv"
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
	IpSubnet string `SNAPROUTE: "KEY"` // Ip Address of the interface
	Netmask  string `SNAPROUTE: "KEY"` // NetMaks of the interface
	IfIndex  string `SNAPROUTE: "KEY"` // Unique If Id of the interface
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
		dhcprelayEnable = config.Enable
		if dhcprelayClientConn != nil {
			logger.Info("DRA: no need to create pcap as its already created")
			return true, nil
		} else {
			DhcpRelayAgentCreateClientServerConn()
		}
	} else {
		dhcprelayEnable = config.Enable
	}
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayGlobalConfig(
	origconfig *dhcprelayd.DhcpRelayGlobalConfig,
	newconfig *dhcprelayd.DhcpRelayGlobalConfig,
	attrset []bool) (bool, error) {
	logger.Info(fmt.Sprintln("DRA: updating relay config to",
		newconfig.Enable))
	dhcprelayEnable = newconfig.Enable
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayGlobalConfig(
	config *dhcprelayd.DhcpRelayGlobalConfig) (bool, error) {
	logger.Info(fmt.Sprintln("DRA: deleting relay config to", config.Enable))
	dhcprelayEnable = config.Enable
	return true, nil
}

func (h *DhcpRelayServiceHandler) CreateDhcpRelayIntfConfig(
	config *dhcprelayd.DhcpRelayIntfConfig) (bool, error) {
	logger.Info("DRA: Intf Config Create")
	logger.Info("DRA: Creating Dhcp Relay Config for interface")
	logger.Info("DRA: IpSubnet:" + config.IpSubnet)
	logger.Info("DRA: Netmask:" + config.Netmask)
	logger.Info("DRA: IF Index:" + config.IfIndex)
	logger.Info("DRA: AgentSubType:" + string(config.AgentSubType))
	logger.Info(fmt.Sprintln("DRA: Enable:", config.Enable))
	logger.Info("DRA: ServerIp:" + config.ServerIp)
	// Copy over configuration into globalInfo
	ifNum, _ := strconv.Atoi(config.IfIndex)
	gblEntry, ok := dhcprelayGblInfo[ifNum]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.."))
		return ok, nil
	}
	// Acquire lock for updating configuration.
	gblEntry.IntfConfig.IpSubnet = config.IpSubnet
	gblEntry.IntfConfig.Netmask = config.Netmask
	gblEntry.IntfConfig.AgentSubType = config.AgentSubType
	gblEntry.IntfConfig.Enable = config.Enable
	gblEntry.IntfConfig.ServerIp = config.ServerIp
	gblEntry.IntfConfig.IfIndex = config.IfIndex
	dhcprelayGblInfo[ifNum] = gblEntry
	if dhcprelayEnable == false {
		logger.Err("DRA: Enable DHCP RELAY AGENT GLOBALLY")
	}
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
	logger.Info("DRA: deleting config for interface" + config.IfIndex)
	ifNum, _ := strconv.Atoi(config.IfIndex)
	gblEntry, ok := dhcprelayGblInfo[ifNum]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.."))
		return ok, nil
	}
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IpSubnet = ""
	gblEntry.IntfConfig.Netmask = ""
	gblEntry.IntfConfig.IfIndex = strconv.Itoa(ifNum) //ifName
	gblEntry.IntfConfig.AgentSubType = 0
	gblEntry.IntfConfig.Enable = false
	gblEntry.PcapHandle.Close()
	gblEntry.PcapHandle = nil
	dhcprelayGblInfo[ifNum] = gblEntry
	return true, nil
}

func (h *DhcpRelayServiceHandler) GetBulkDhcpRelayHostDhcpState(fromIndex dhcprelayd.Int,
	count dhcprelayd.Int) (hostEntry *dhcprelayd.DhcpRelayHostDhcpStateGetInfo, err error) {

	return hostEntry, err
}

func (h *DhcpRelayServiceHandler) GetBulkDhcpRelayIntfState(fromIndex dhcprelayd.Int,
	count dhcprelayd.Int) (intfEntry *dhcprelayd.DhcpRelayIntfStateGetInfo, err error) {
	return intfEntry, err
}

func (h *DhcpRelayServiceHandler) GetBulkDhcpRelayIntfServerState(fromIndex dhcprelayd.Int,
	count dhcprelayd.Int) (intfServerEntry *dhcprelayd.DhcpRelayIntfServerStateGetInfo, err error) {
	return intfServerEntry, err
}
