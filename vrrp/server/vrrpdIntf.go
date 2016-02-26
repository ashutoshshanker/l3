package vrrpServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
)

func VrrpCreateIfIndexEntry(IfIndex int32, IpAddr string) {
	entry := vrrpIfIndexIpAddr[IfIndex]
	entry = IpAddr
	vrrpIfIndexIpAddr[IfIndex] = entry
	logger.Info(fmt.Sprintln("VRRP: ip address for ifindex ", IfIndex,
		"is", entry))
}

func VrrpCreateVlanEntry(vlanId int, vlanName string) {
	entry := vrrpVlanId2Name[vlanId]
	entry = vlanName
	vrrpVlanId2Name[vlanId] = entry
}

func VrrpGetPortList() {
	logger.Info("VRRP: Get Port List")
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			logger.Err(fmt.Sprintln("VRRP: getting bulk port config"+
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
				//logger.Info("VRRP: interface global init for " + ifName)
				//VrrpInitGblInfo(portNum, ifName, "")
			*/
		}
		if more == false {
			break
		}
	}
}

func VrrpGetIPv4IntfList() {
	logger.Info("VRRP: Get IPv4 Interface List")
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkIPv4Intf(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk ipv4 intf config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			VrrpCreateIfIndexEntry(bulkInfo.IPv4IntfList[i].IfIndex,
				bulkInfo.IPv4IntfList[i].IpAddr)
		}
		if more == false {
			break
		}
	}
}

func VrrpGetVlanList() {
	logger.Info("VRRP: Get Vlans")
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkVlan(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk vlan config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			VrrpCreateVlanEntry(int(bulkInfo.VlanList[i].VlanId),
				bulkInfo.VlanList[i].VlanName)
		}
		if more == false {
			break
		}
	}
}

func VrrpUpdateVlanGblInfo(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	logger.Info(fmt.Sprintln("Vlan Update msg for", vlanNotifyMsg))
	switch msgType {
	case asicdConstDefs.NOTIFY_VLAN_CREATE:
		VrrpCreateVlanEntry(int(vlanNotifyMsg.VlanId), vlanNotifyMsg.VlanName)
	case asicdConstDefs.NOTIFY_VLAN_DELETE:
		delete(vrrpVlanId2Name, int(vlanNotifyMsg.VlanId))
	}
}

func VrrpUpdateIPv4GblInfo(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	switch msgType {
	case asicdConstDefs.NOTIFY_IPV4INTF_CREATE:
		VrrpCreateIfIndexEntry(msg.IfIndex, msg.IpAddr)
		go VrrpMapIfIndexToLinuxIfIndex(msg.IfIndex)
	case asicdConstDefs.NOTIFY_IPV4INTF_DELETE:
		delete(vrrpIfIndexIpAddr, msg.IfIndex)
	}
}

func VrrpUpdateL3IntfStateChange(msg asicdConstDefs.L3IntfStateNotifyMsg) {
	switch msg.IfState {
	case asicdConstDefs.INTF_STATE_UP:
		logger.Info("VRRP: Got Interface state up notification")
	case asicdConstDefs.INTF_STATE_DOWN:
		logger.Info("VRRP: Got Interface state down notification")
	}
}

func VrrpAsicdSubscriber() {
	for {
		logger.Info("VRRP: Read on Asic Subscriber socket....")
		rxBuf, err := asicdSubSocket.Recv(0)
		if err != nil {
			logger.Err(fmt.Sprintln("VRRP: Recv on asicd Subscriber",
				"socket failed with error:", err))
			continue
		}
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			logger.Err(fmt.Sprintln("VRRP: Unable to Unmarshal",
				"asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
			//Vlan Create Msg
			var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
			err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("VRRP: Unable to",
					"unmashal vlanNotifyMsg:", msg.Msg))
				return
			}
			VrrpUpdateVlanGblInfo(vlanNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
			var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("VRRP: Unable to Unmarshal",
					"ipv4IntfNotifyMsg:", msg.Msg))
				continue
			}
			VrrpUpdateIPv4GblInfo(ipv4IntfNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
			//INTF_STATE_CHANGE
			var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("VRRP: unable to Unmarshal l3 intf",
					"state change:", msg.Msg))
				continue
			}
			VrrpUpdateL3IntfStateChange(l3IntfStateNotifyMsg)
		}
	}
}

func VrrpRegisterWithAsicdUpdates(address string) error {
	var err error
	logger.Info("setting up asicd update listener")
	if asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		logger.Err(fmt.Sprintln("Failed to create ASIC subscribe",
			"socket, error:", err))
		return err
	}

	if err = asicdSubSocket.Subscribe(""); err != nil {
		logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on",
			"ASIC subscribe socket, error:",
			err))
		return err
	}

	if _, err = asicdSubSocket.Connect(address); err != nil {
		logger.Err(fmt.Sprintln("Failed to connect to ASIC",
			"publisher socket, address:", address, "error:", err))
		return err
	}

	if err = asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		logger.Err(fmt.Sprintln("Failed to set the buffer size for ",
			"ASIC publisher socket, error:", err))
		return err
	}
	logger.Info("asicd update listener is set")
	return nil
}

func VrrpGetInfoFromAsicd() error {
	logger.Info("VRRP: Calling Asicd to initialize port properties")
	err := VrrpRegisterWithAsicdUpdates(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go VrrpAsicdSubscriber()
	}
	// Get Port List Most Likely Not needed...as we are only interested
	// in Ipv4Intf...
	//VrrpGetPortList()
	// Get Vlan List
	VrrpGetVlanList()
	// Get IPv4 Interface List
	VrrpGetIPv4IntfList()
	return nil
}
