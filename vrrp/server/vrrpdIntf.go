package vrrpServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"strconv"
	"strings"
)

func (svr *VrrpServer) VrrpCreateIfIndexEntry(IfIndex int32, IpAddr string) {
	svr.vrrpIfIndexIpAddr[IfIndex] = IpAddr
	svr.logger.Info(fmt.Sprintln("VRRP: ip address for ifindex ", IfIndex,
		"is", IpAddr))
	for _, key := range svr.vrrpIntfStateSlice {
		startFsm := false
		splitString := strings.Split(key, "_")
		// splitString = { IfIndex, VRID }
		ifindex, _ := strconv.Atoi(splitString[0])
		if int32(ifindex) != IfIndex {
			// Key doesn't match
			continue
		}
		// If IfIndex matches then use that key and check if gblInfo is
		// created or not
		gblInfo, found := svr.vrrpGblInfo[key]
		if !found {
			svr.logger.Err("No entry found for Ifindex:" +
				splitString[0] + " VRID:" + splitString[1] +
				" hence not updating ip addr, " +
				"it will be updated during create")
			continue
		}
		gblInfo.IpAddr = IpAddr
		gblInfo.StateLock.Lock()
		if gblInfo.StateName == VRRP_UNINTIALIZE_STATE {
			startFsm = true
			gblInfo.StateName = VRRP_INITIALIZE_STATE
		}
		gblInfo.StateLock.Unlock()
		svr.vrrpGblInfo[key] = gblInfo
		if !svr.vrrpMacConfigAdded {
			svr.logger.Info("Adding protocol mac for punting packets to CPU")
			svr.VrrpUpdateProtocolMacEntry(true /*add vrrp protocol mac*/)
		}
		if startFsm {
			svr.vrrpFsmCh <- VrrpFsm{
				key: key,
			}
		}
	}
}

func (svr *VrrpServer) VrrpCreateVlanEntry(vlanId int, vlanName string) {
	entry := svr.vrrpVlanId2Name[vlanId]
	entry = vlanName
	svr.vrrpVlanId2Name[vlanId] = entry
}

func (svr *VrrpServer) VrrpGetPortList() {
	svr.logger.Info("VRRP: Get Port List")
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			svr.logger.Err(fmt.Sprintln("VRRP: getting bulk port config"+
				" from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			/*
				var ifName string
				var portNum int32
				portNum = bulkInfo.PortStateList[i].IfIndex
				ifName = bulkInfo.PortStateList[i].Name
				//svr.logger.Info("VRRP: interface global init for " + ifName)
				//VrrpInitGblInfo(portNum, ifName, "")
			*/
		}
		if more == false {
			break
		}
	}
}

func (svr *VrrpServer) VrrpGetIPv4IntfList() {
	svr.logger.Info("VRRP: Get IPv4 Interface List")
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkIPv4Intf(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			svr.logger.Err(fmt.Sprintln("DRA: getting bulk ipv4 intf config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			svr.VrrpCreateIfIndexEntry(bulkInfo.IPv4IntfList[i].IfIndex,
				bulkInfo.IPv4IntfList[i].IpAddr)
		}
		if more == false {
			break
		}
	}
}

func (svr *VrrpServer) VrrpGetVlanList() {
	svr.logger.Info("VRRP: Get Vlans")
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkVlan(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			svr.logger.Err(fmt.Sprintln("DRA: getting bulk vlan config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			svr.VrrpCreateVlanEntry(int(bulkInfo.VlanList[i].VlanId),
				bulkInfo.VlanList[i].VlanName)
		}
		if more == false {
			break
		}
	}
}

func (svr *VrrpServer) VrrpUpdateVlanGblInfo(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	svr.logger.Info(fmt.Sprintln("Vlan Update msg for", vlanNotifyMsg))
	switch msgType {
	case asicdConstDefs.NOTIFY_VLAN_CREATE:
		svr.VrrpCreateVlanEntry(int(vlanNotifyMsg.VlanId), vlanNotifyMsg.VlanName)
	case asicdConstDefs.NOTIFY_VLAN_DELETE:
		delete(svr.vrrpVlanId2Name, int(vlanNotifyMsg.VlanId))
	}
}

func (svr *VrrpServer) VrrpUpdateIPv4GblInfo(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	switch msgType {
	case asicdConstDefs.NOTIFY_IPV4INTF_CREATE:
		svr.VrrpCreateIfIndexEntry(msg.IfIndex, msg.IpAddr)
		go svr.VrrpMapIfIndexToLinuxIfIndex(msg.IfIndex)
	case asicdConstDefs.NOTIFY_IPV4INTF_DELETE:
		delete(svr.vrrpIfIndexIpAddr, msg.IfIndex)
	}
}

func (svr *VrrpServer) VrrpUpdateL3IntfStateChange(msg asicdConstDefs.L3IntfStateNotifyMsg) {
	switch msg.IfState {
	case asicdConstDefs.INTF_STATE_UP:
		svr.VrrpHandleIntfUpEvent(msg.IfIndex)
		svr.logger.Info("VRRP: Got Interface state up notification")
	case asicdConstDefs.INTF_STATE_DOWN:
		svr.VrrpHandleIntfShutdownEvent(msg.IfIndex)
		svr.logger.Info("VRRP: Got Interface state down notification")
	}
}

func (svr *VrrpServer) VrrpAsicdSubscriber() {
	for {
		svr.logger.Info("VRRP: Read on Asic Subscriber socket....")
		rxBuf, err := svr.asicdSubSocket.Recv(0)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("VRRP: Recv on asicd Subscriber",
				"socket failed with error:", err))
			continue
		}
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("VRRP: Unable to Unmarshal",
				"asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
			//Vlan Create Msg
			var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
			err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("VRRP: Unable to",
					"unmashal vlanNotifyMsg:", msg.Msg))
				return
			}
			svr.VrrpUpdateVlanGblInfo(vlanNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
			var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("VRRP: Unable to Unmarshal",
					"ipv4IntfNotifyMsg:", msg.Msg))
				continue
			}
			svr.VrrpUpdateIPv4GblInfo(ipv4IntfNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
			//INTF_STATE_CHANGE
			var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("VRRP: unable to Unmarshal l3 intf",
					"state change:", msg.Msg))
				continue
			}
			svr.VrrpUpdateL3IntfStateChange(l3IntfStateNotifyMsg)
		}
	}
}

func (svr *VrrpServer) VrrpRegisterWithAsicdUpdates(address string) error {
	var err error
	svr.logger.Info("setting up asicd update listener")
	if svr.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to create ASIC subscribe",
			"socket, error:", err))
		return err
	}

	if err = svr.asicdSubSocket.Subscribe(""); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on",
			"ASIC subscribe socket, error:",
			err))
		return err
	}

	if _, err = svr.asicdSubSocket.Connect(address); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to connect to ASIC",
			"publisher socket, address:", address, "error:", err))
		return err
	}

	if err = svr.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to set the buffer size for ",
			"ASIC publisher socket, error:", err))
		return err
	}
	svr.logger.Info("asicd update listener is set")
	return nil
}

func (svr *VrrpServer) VrrpGetInfoFromAsicd() error {
	svr.logger.Info("VRRP: Calling Asicd to initialize port properties")
	err := svr.VrrpRegisterWithAsicdUpdates(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go svr.VrrpAsicdSubscriber()
	}
	// Get Port List Most Likely Not needed...as we are only interested
	// in Ipv4Intf...
	//VrrpGetPortList()
	// Get Vlan List
	svr.VrrpGetVlanList()
	// Get IPv4 Interface List
	svr.VrrpGetIPv4IntfList()
	return nil
}
