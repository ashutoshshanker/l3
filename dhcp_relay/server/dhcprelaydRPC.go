package relayServer

import (
	"dhcprelayd"
	"fmt"
	"net"
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
	logger.Info("DRA: Creating Dhcp Relay Config for interface")
	logger.Info("DRA: IpSubnet:" + config.IpSubnet)
	logger.Info("DRA: Netmask:" + config.Netmask)
	logger.Info("DRA: IF Index:" + config.IfIndex)
	logger.Info("DRA: AgentSubType:" + string(config.AgentSubType))
	logger.Info(fmt.Sprintln("DRA: Enable:", config.Enable))
	logger.Info("DRA: ServerIp:" + config.ServerIp)
	// Copy over configuration into globalInfo
	//gblEntry := dhcprelayGblInfo[config.IfIndex]
	ifNum, _ := strconv.Atoi(config.IfIndex) //portInfoMap[config.IfIndex]
	var err error
	var linuxInterface *net.Interface
	//@TODO: hack for if_index
	if ifNum == 33554441 {
		logger.Info(fmt.Sprintln("DRA: jgheewal:::: hack for ifNUm", ifNum))
		linuxInterface, err = net.InterfaceByName("SVI9")
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting interface by name failed", err))
		} else {
			//copy correct if_id
			ifNum = linuxInterface.Index
			logger.Info(fmt.Sprintln("DRA: jgheewal:::: Updated for ifNUm", ifNum))
		}

	} else if ifNum == 33554442 {
		logger.Info(fmt.Sprintln("DRA: jgheewal:::: hack for ifNUm", ifNum))
		linuxInterface, err = net.InterfaceByName("SVI10")
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting interface by name failed", err))
		} else {
			//copy correct if_id
			ifNum = linuxInterface.Index
			logger.Info(fmt.Sprintln("DRA: jgheewal:::: Updated for ifNUm", ifNum))
		}
	}
	gblEntry, ok := dhcprelayGblInfo[ifNum]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum, " doesn't exist.."))
		return ok, nil
	}
	// Acquire lock for updating configuration.
	gblEntry.dhcprelayConfigMutex.RLock()
	gblEntry.IntfConfig.IpSubnet = config.IpSubnet
	gblEntry.IntfConfig.Netmask = config.Netmask
	gblEntry.IntfConfig.AgentSubType = config.AgentSubType
	gblEntry.IntfConfig.Enable = config.Enable
	gblEntry.IntfConfig.ServerIp = config.ServerIp
	dhcprelayGblInfo[ifNum] = gblEntry
	// Release lock after updation is done
	gblEntry.dhcprelayConfigMutex.RUnlock()
	//@TODO: FIXME jgheewala
	// if entry is present then update DB with new info rather than
	// just writing it again...
	if dhcprelayClientConn != nil {
		logger.Info("DRA: no need to create pcap as its already created")
		return true, nil
	} else {
		logger.Info("DRA: len of global entries is " + string(len(dhcprelayGblInfo)))
		// Stats information
		//DhcpRelayAgentUpdateStats("dhcp relay config create request",
		//	&gblEntry)
		DhcpRelayAgentCreateClientServerConn()
		// Stats information
		StateDebugInfo = make(map[string]DhcpRelayAgentStateInfo, 150)
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
	return true, nil
}
