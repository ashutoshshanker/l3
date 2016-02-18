package vrrpServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
)

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
			var ifName string
			var portNum int32
			portNum = bulkInfo.PortStateList[i].IfIndex
			ifName = bulkInfo.PortStateList[i].Name
			logger.Info("VRRP: interface global init for " + ifName)
			VrrpInitGblInfo(portNum, ifName, "")
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
			logger.Err(fmt.Sprintln("DRA: getting bulk vlan config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			VrrpInitGblInfo(bulkInfo.IPv4IntfList[i].IfIndex, "",
				bulkInfo.IPv4IntfList[i].IpAddr)
		}
		if more == false {
			break
		}
	}
}

func VrrpGetVlanList() {

}

func VrrpUpdateIPv4IntfInfo(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	gblInfo := vrrpGblInfo[msg.IfIndex]
	switch msgType {
	case asicdConstDefs.NOTIFY_IPV4INTF_CREATE:
		gblInfo.IpAddr = msg.IpAddr
	case asicdConstDefs.NOTIFY_IPV4INTF_DELETE:
		gblInfo.IpAddr = ""
	}
	vrrpGblInfo[msg.IfIndex] = gblInfo
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
			logger.Err(fmt.Sprintln("VRRP: Recv on asicd Subscriber socket failed with error:", err))
			continue
		}
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			logger.Err(fmt.Sprintln("VRRP: Unable to Unmarshal asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
			//Vlan Create Msg
			var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
			err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("VRRP: Unable to unmashal vlanNotifyMsg:", msg.Msg))
				return
			}
			//DhcpRelayAgentUpdateVlanInfo(vlanNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
			var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("VRRP: Unable to Unmarshal ipv4IntfNotifyMsg:", msg.Msg))
				continue
			}
			VrrpUpdateIPv4IntfInfo(ipv4IntfNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
			//INTF_STATE_CHANGE
			var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("VRRP: unable to Unmarshal l3 intf state change:", msg.Msg))
				continue
			}
			VrrpUpdateL3IntfStateChange(l3IntfStateNotifyMsg)
		}
	}
}

func VrrpRegisterWithAsicdUpdates(address string) error {
	var err error
	logger.Info("VRRP: setting up asicd update listener")
	if asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		logger.Err(fmt.Sprintln("VRRP: Failed to create ASIC subscribe socket, error:", err))
		return err
	}

	if err = asicdSubSocket.Subscribe(""); err != nil {
		logger.Err(fmt.Sprintln("VRRP:Failed to subscribe to \"\" on ASIC subscribe socket, error:",
			err))
		return err
	}

	if _, err = asicdSubSocket.Connect(address); err != nil {
		logger.Err(fmt.Sprintln("VRRP: Failed to connect to ASIC publisher socket, address:",
			address, "error:", err))
		return err
	}

	if err = asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		logger.Err(fmt.Sprintln("VRRP: Failed to set the buffer size for ",
			"ASIC publisher socket, error:", err))
		return err
	}
	logger.Info("VRRP: asicd update listener is set")
	return nil
}

func VrrpGetInfoFromAsicd() error {
	logger.Info("VRRP: Calling Asicd to initialize port properties")
	err := VrrpRegisterWithAsicdUpdates(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go VrrpAsicdSubscriber()
	}
	// Get Port List
	VrrpGetPortList()
	// Get IPv4 Interface List
	VrrpGetIPv4IntfList()
	// @TODO: do we need vlan??
	//VrrpGetVlanList()
	return nil
}
