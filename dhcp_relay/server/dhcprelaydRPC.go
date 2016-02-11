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
	ServerIp []string
	//ServerIp string
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
	// Copy over configuration into globalInfo
	ifNum, _ := strconv.Atoi(config.IfIndex)
	gblEntry, ok := dhcprelayGblInfo[int32(ifNum)]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.."))
		return ok, nil
	}
	gblEntry.IntfConfig.IpSubnet = config.IpSubnet
	gblEntry.IntfConfig.Netmask = config.Netmask
	gblEntry.IntfConfig.AgentSubType = config.AgentSubType
	gblEntry.IntfConfig.Enable = config.Enable
	logger.Info("DRA: ServerIp:")
	for idx := 0; idx < len(config.ServerIp); idx++ {
		logger.Info(fmt.Sprintln("DRA: Server", idx, ": ",
			config.ServerIp[idx]))
		gblEntry.IntfConfig.ServerIp = append(gblEntry.IntfConfig.ServerIp,
			config.ServerIp[idx])
		DhcpRelayAgentInitIntfServerState(config.IfIndex,
			config.ServerIp[idx], int32(ifNum))
	}
	gblEntry.IntfConfig.IfIndex = config.IfIndex
	dhcprelayGblInfo[int32(ifNum)] = gblEntry

	if dhcprelayEnable == false {
		logger.Err("DRA: Enable DHCP RELAY AGENT GLOBALLY")
	}
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayIntfConfig(
	origconfig *dhcprelayd.DhcpRelayIntfConfig,
	newconfig *dhcprelayd.DhcpRelayIntfConfig,
	attrset []bool) (bool, error) {
	logger.Info("DRA: Intf Config Update")
	logger.Info("DRA: Updating Dhcp Relay Config for interface")
	logger.Info("DRA: IpSubnet: " + origconfig.IpSubnet + " changed to " +
		newconfig.IpSubnet)
	logger.Info("DRA: Netmask: " + origconfig.Netmask + " changed to " +
		newconfig.Netmask)
	logger.Info("DRA: IF Index: " + origconfig.IfIndex + " changed to " +
		newconfig.IfIndex)
	if origconfig.IfIndex != newconfig.IfIndex {
		logger.Info(fmt.Sprintln("DRA: Interface Id cannot be different.",
			" Relay Agent will not accept this update for changing if id from",
			origconfig.IfIndex, "to", newconfig.IfIndex))
		return false, nil
	}
	logger.Info("DRA: AgentSubType: " + string(origconfig.AgentSubType) +
		" changed to " + string(newconfig.AgentSubType))
	logger.Info(fmt.Sprintln("DRA: Enable: ", origconfig.Enable, "changed to",
		newconfig.Enable))
	// Copy over configuration into globalInfo
	ifNum, _ := strconv.Atoi(origconfig.IfIndex)
	gblEntry, ok := dhcprelayGblInfo[int32(ifNum)]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.. and hence cannot update"))
		return ok, nil
	}
	gblEntry.IntfConfig.IpSubnet = newconfig.IpSubnet
	gblEntry.IntfConfig.Netmask = newconfig.Netmask
	gblEntry.IntfConfig.AgentSubType = newconfig.AgentSubType
	gblEntry.IntfConfig.Enable = newconfig.Enable
	gblEntry.IntfConfig.ServerIp = nil
	logger.Warning("DRA: Deleted Older DHCP Server IP's List and creating new")
	logger.Info("DRA: New ServerIp's:")
	for idx := 0; idx < len(newconfig.ServerIp); idx++ {
		logger.Info(fmt.Sprintln("DRA: Server", idx, ": ",
			newconfig.ServerIp[idx]))
		gblEntry.IntfConfig.ServerIp = append(gblEntry.IntfConfig.ServerIp,
			newconfig.ServerIp[idx])
		DhcpRelayAgentInitIntfServerState(newconfig.IfIndex,
			newconfig.ServerIp[idx], int32(ifNum))

	}
	gblEntry.IntfConfig.IfIndex = newconfig.IfIndex
	dhcprelayGblInfo[int32(ifNum)] = gblEntry

	if dhcprelayEnable == false {
		logger.Err("DRA: Enable DHCP RELAY AGENT GLOBALLY")
	}
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayIntfConfig(
	config *dhcprelayd.DhcpRelayIntfConfig) (bool, error) {
	logger.Info("DRA: deleting config for interface" + config.IfIndex)
	ifNum, _ := strconv.Atoi(config.IfIndex)
	gblEntry, ok := dhcprelayGblInfo[int32(ifNum)]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.."))
		return ok, nil
	}
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IpSubnet = ""
	gblEntry.IntfConfig.Netmask = ""
	gblEntry.IntfConfig.IfIndex = strconv.Itoa(ifNum)
	gblEntry.IntfConfig.AgentSubType = 0
	gblEntry.IntfConfig.Enable = false
	gblEntry.PcapHandle.Close()
	gblEntry.PcapHandle = nil
	dhcprelayGblInfo[int32(ifNum)] = gblEntry
	return true, nil
}

func (h *DhcpRelayServiceHandler) GetBulkDhcpRelayHostDhcpState(fromIndex dhcprelayd.Int,
	count dhcprelayd.Int) (hostEntry *dhcprelayd.DhcpRelayHostDhcpStateGetInfo, err error) {
	logger.Info(fmt.Sprintln("DRA: Get Bulk for Host Server State for ", count, " hosts"))

	var nextEntry *dhcprelayd.DhcpRelayHostDhcpState
	var finalList []*dhcprelayd.DhcpRelayHostDhcpState
	var returnBulk dhcprelayd.DhcpRelayHostDhcpStateGetInfo
	var endIdx int
	var more bool
	hostEntry = &returnBulk

	if dhcprelayHostServerStateSlice == nil {
		logger.Info("DRA: Host Server Slice is not initialized")
		return hostEntry, err
	}
	currIdx := int(fromIndex)
	cnt := int(count)
	length := len(dhcprelayHostServerStateSlice)

	if currIdx+cnt >= length {
		cnt = length - currIdx
		endIdx = 0
		more = false
	} else {
		endIdx = currIdx + cnt
		more = true
	}
	for i := 0; i < cnt; i++ {
		if len(finalList) == 0 {
			finalList = make([]*dhcprelayd.DhcpRelayHostDhcpState, 0)
		}
		key := dhcprelayHostServerStateSlice[i]
		entry := dhcprelayHostServerStateMap[key]
		nextEntry = &entry
		finalList = append(finalList, nextEntry)
	}
	hostEntry.DhcpRelayHostDhcpStateList = finalList
	hostEntry.StartIdx = fromIndex
	hostEntry.EndIdx = dhcprelayd.Int(endIdx)
	hostEntry.More = more
	hostEntry.Count = dhcprelayd.Int(cnt)

	return hostEntry, err
}

func (h *DhcpRelayServiceHandler) GetBulkDhcpRelayIntfState(fromIndex dhcprelayd.Int,
	count dhcprelayd.Int) (intfEntry *dhcprelayd.DhcpRelayIntfStateGetInfo, err error) {
	logger.Info(fmt.Sprintln("DRA: Get Bulk for Intf State for ", count, " interfaces"))
	var nextEntry *dhcprelayd.DhcpRelayIntfState
	var finalList []*dhcprelayd.DhcpRelayIntfState
	var returnIntfStatebulk dhcprelayd.DhcpRelayIntfStateGetInfo
	var endIdx int
	var more bool
	intfEntry = &returnIntfStatebulk

	if dhcprelayIntfStateSlice == nil {
		logger.Info("DRA: Interface Slice is not initialized")
		return intfEntry, err
	}
	currIdx := int(fromIndex)
	cnt := int(count)
	length := len(dhcprelayIntfStateSlice)

	if currIdx+cnt >= length {
		cnt = length - currIdx
		endIdx = 0
		more = false
	} else {
		endIdx = currIdx + cnt
		more = true
	}

	for i := 0; i < cnt; i++ {
		if len(finalList) == 0 {
			finalList = make([]*dhcprelayd.DhcpRelayIntfState, 0)
		}
		key := dhcprelayIntfStateSlice[i]
		entry := dhcprelayIntfStateMap[key]
		nextEntry = &entry
		finalList = append(finalList, nextEntry)
	}
	intfEntry.DhcpRelayIntfStateList = finalList
	intfEntry.StartIdx = fromIndex
	intfEntry.EndIdx = dhcprelayd.Int(endIdx)
	intfEntry.More = more
	intfEntry.Count = dhcprelayd.Int(cnt)

	return intfEntry, err
}

func (h *DhcpRelayServiceHandler) GetBulkDhcpRelayIntfServerState(fromIndex dhcprelayd.Int,
	count dhcprelayd.Int) (intfServerEntry *dhcprelayd.DhcpRelayIntfServerStateGetInfo, err error) {
	logger.Info(fmt.Sprintln("DRA: Get Bulk for Intf Server State for ", count, " combination"))
	var nextEntry *dhcprelayd.DhcpRelayIntfServerState
	var finalList []*dhcprelayd.DhcpRelayIntfServerState
	var returnBulk dhcprelayd.DhcpRelayIntfServerStateGetInfo
	var endIdx int
	var more bool
	intfServerEntry = &returnBulk

	if dhcprelayIntfServerStateSlice == nil {
		logger.Info("DRA: Interface Server Slice is not initialized")
		return intfServerEntry, err
	}
	currIdx := int(fromIndex)
	cnt := int(count)
	length := len(dhcprelayIntfServerStateSlice)

	if currIdx+cnt >= length {
		cnt = length - currIdx
		endIdx = 0
		more = false
	} else {
		endIdx = currIdx + cnt
		more = true
	}
	for i := 0; i < cnt; i++ {
		if len(finalList) == 0 {
			finalList = make([]*dhcprelayd.DhcpRelayIntfServerState, 0)
		}
		key := dhcprelayIntfServerStateSlice[i]
		entry := dhcprelayIntfServerStateMap[key]
		nextEntry = &entry
		finalList = append(finalList, nextEntry)
	}
	intfServerEntry.DhcpRelayIntfServerStateList = finalList
	intfServerEntry.StartIdx = fromIndex
	intfServerEntry.EndIdx = dhcprelayd.Int(endIdx)
	intfServerEntry.More = more
	intfServerEntry.Count = dhcprelayd.Int(cnt)

	return intfServerEntry, err
}
