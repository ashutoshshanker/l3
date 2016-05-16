package relayServer

import (
	"dhcprelayd"
	"fmt"
	_ "strconv"
	"sync"
)

/******** Trift APIs *******/
/*
 * Add a relay agent
 */

func (h *DhcpRelayServiceHandler) CreateDhcpRelayGlobal(
	config *dhcprelayd.DhcpRelayGlobal) (bool, error) {

	DhcpRelayGlobalInit(config.Enable)
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayGlobal(
	origconfig *dhcprelayd.DhcpRelayGlobal,
	newconfig *dhcprelayd.DhcpRelayGlobal,
	attrset []bool) (bool, error) {
	logger.Info(fmt.Sprintln("DRA: updating relay config to",
		newconfig.Enable))
	dhcprelayEnable = newconfig.Enable
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayGlobal(
	config *dhcprelayd.DhcpRelayGlobal) (bool, error) {
	logger.Info(fmt.Sprintln("DRA: deleting relay config to", config.Enable))
	dhcprelayEnable = config.Enable
	return true, nil
}

func (h *DhcpRelayServiceHandler) CreateDhcpRelayIntf(
	config *dhcprelayd.DhcpRelayIntf) (bool, error) {
	logger.Info(fmt.Sprintln("DRA: Intf Config Create for", config.IfIndex))
	// Copy over configuration into globalInfo
	ifNum := config.IfIndex
	gblEntry, ok := dhcprelayGblInfo[ifNum]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.."))
		return ok, nil
	}
	gblEntry.IntfConfig.Enable = config.Enable
	logger.Info("DRA: ServerIp:")
	for idx := 0; idx < len(config.ServerIp); idx++ {
		logger.Info(fmt.Sprintln("DRA: Server", idx, ": ",
			config.ServerIp[idx]))
		gblEntry.IntfConfig.ServerIp = append(gblEntry.IntfConfig.ServerIp,
			config.ServerIp[idx])
		DhcpRelayAgentInitIntfServerState(config.ServerIp[idx], ifNum)
	}
	gblEntry.IntfConfig.IfIndex = config.IfIndex
	dhcprelayGblInfo[ifNum] = gblEntry

	if dhcprelayEnable == false {
		logger.Err("DRA: Enable DHCP RELAY AGENT GLOBALLY")
	}
	if dhcprelayRefCountMutex == nil {
		dhcprelayRefCountMutex = &sync.RWMutex{}
		dhcprelayEnabledIntfRefCount = 0
	}
	if gblEntry.IntfConfig.Enable {
		dhcprelayRefCountMutex.Lock()
		dhcprelayEnabledIntfRefCount++
		dhcprelayRefCountMutex.Unlock()
	}
	return true, nil
}

func (h *DhcpRelayServiceHandler) UpdateDhcpRelayIntf(
	origconfig *dhcprelayd.DhcpRelayIntf,
	newconfig *dhcprelayd.DhcpRelayIntf,
	attrset []bool) (bool, error) {
	logger.Info("DRA: Intf Config Update")
	logger.Info("DRA: Updating Dhcp Relay Config for interface")
	if origconfig.IfIndex != newconfig.IfIndex {
		logger.Info(fmt.Sprintln("DRA: Interface Id cannot be different.",
			" Relay Agent will not accept this update for changing if id from",
			origconfig.IfIndex, "to", newconfig.IfIndex))
		return false, nil
	}
	logger.Info(fmt.Sprintln("DRA: Enable: ", origconfig.Enable, "changed to",
		newconfig.Enable))
	// Copy over configuration into globalInfo
	ifNum := origconfig.IfIndex
	gblEntry, ok := dhcprelayGblInfo[ifNum]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.. and hence cannot update"))
		return ok, nil
	}
	gblEntry.IntfConfig.Enable = newconfig.Enable
	gblEntry.IntfConfig.ServerIp = nil
	logger.Warning("DRA: Deleted Older DHCP Server IP's List and creating new")
	logger.Info("DRA: New ServerIp's:")
	for idx := 0; idx < len(newconfig.ServerIp); idx++ {
		logger.Info(fmt.Sprintln("DRA: Server", idx, ": ",
			newconfig.ServerIp[idx]))
		gblEntry.IntfConfig.ServerIp = append(gblEntry.IntfConfig.ServerIp,
			newconfig.ServerIp[idx])
		DhcpRelayAgentInitIntfServerState(newconfig.ServerIp[idx], ifNum)

	}
	gblEntry.IntfConfig.IfIndex = newconfig.IfIndex
	dhcprelayGblInfo[ifNum] = gblEntry

	if dhcprelayEnable == false {
		logger.Err("DRA: Enable DHCP RELAY AGENT GLOBALLY")
	}
	return true, nil
}

func (h *DhcpRelayServiceHandler) DeleteDhcpRelayIntf(
	config *dhcprelayd.DhcpRelayIntf) (bool, error) {
	logger.Info(fmt.Sprintln("DRA: deleting config for interface", config.IfIndex))
	ifNum := config.IfIndex
	gblEntry, ok := dhcprelayGblInfo[ifNum]
	if !ok {
		logger.Err(fmt.Sprintln("DRA: entry for ifNum", ifNum,
			" doesn't exist.."))
		return ok, nil
	}
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IfIndex = ifNum
	gblEntry.IntfConfig.Enable = false
	if gblEntry.PcapHandle != nil {
		gblEntry.PcapHandle.Close()
		gblEntry.PcapHandle = nil
	}
	dhcprelayGblInfo[ifNum] = gblEntry
	dhcprelayRefCountMutex.Lock()
	dhcprelayEnabledIntfRefCount--
	dhcprelayRefCountMutex.Unlock()
	logger.Info(fmt.Sprintln("DRA: deleted config for interface", config.IfIndex))
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

func (h *DhcpRelayServiceHandler) GetDhcpRelayHostDhcpState(macAddr string) (*dhcprelayd.DhcpRelayHostDhcpState, error) {
	logger.Info("Get State Info for host")
	response := dhcprelayd.NewDhcpRelayHostDhcpState()
	return response, nil
}

func (h *DhcpRelayServiceHandler) GetDhcpRelayIntfServerState(ifIndex int32) (*dhcprelayd.DhcpRelayIntfServerState, error) {
	logger.Info("Get State Info for interface server")
	response := dhcprelayd.NewDhcpRelayIntfServerState()
	return response, nil
}

func (h *DhcpRelayServiceHandler) GetDhcpRelayIntfState(ifIndex int32) (*dhcprelayd.DhcpRelayIntfState, error) {
	logger.Info("Get State Info for interface")
	response := dhcprelayd.NewDhcpRelayIntfState()
	return response, nil
}
