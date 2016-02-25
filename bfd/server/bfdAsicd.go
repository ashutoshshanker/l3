package server

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"utils/commonDefs"
)

type AsicdClient struct {
	BfdClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

func (server *BFDServer) createASICdSubscriber() {
	for {
		server.logger.Info("Read on ASICd subscriber socket...")
		asicdrxBuf, err := server.asicdSubSocket.Recv(0)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Recv on ASICd subscriber socket failed with error:", err))
			server.asicdSubSocketErrCh <- err
			continue
		}
		server.logger.Info(fmt.Sprintln("ASIC subscriber recv returned:", asicdrxBuf))
		server.asicdSubSocketCh <- asicdrxBuf
	}
}

func (server *BFDServer) listenForASICdUpdates(address string) error {
	var err error
	if server.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to create ASICd subscribe socket, error:", err))
		return err
	}

	if _, err = server.asicdSubSocket.Connect(address); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to connect to ASICd publisher socket, address:", address, "error:", err))
		return err
	}

	if err = server.asicdSubSocket.Subscribe(""); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on ASICd subscribe socket, error:", err))
		return err
	}

	server.logger.Info(fmt.Sprintln("Connected to ASICd publisher at address:", address))
	if err = server.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to set the buffer size for ASICd publisher socket, error:", err))
		return err
	}
	return nil
}

func (server *BFDServer) processAsicdNotification(asicdrxBuf []byte) {
	var msg asicdConstDefs.AsicdNotification
	err := json.Unmarshal(asicdrxBuf, &msg)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to unmarshal asicdrxBuf:", asicdrxBuf))
		return
	}
	if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
		msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
		// IPV4INTF Create, Delete
		var NewIpv4IntfMsg asicdConstDefs.IPv4IntfNotifyMsg
		var ipv4IntfMsg IPv4IntfNotifyMsg
		err = json.Unmarshal(msg.Msg, &NewIpv4IntfMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmarshal msg:", msg.Msg))
			return
		}
		ipv4IntfMsg.IpAddr = NewIpv4IntfMsg.IpAddr
		ipv4IntfMsg.IfId = NewIpv4IntfMsg.IfIndex
		if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE {
			server.logger.Info(fmt.Sprintln("Receive IPV4INTF_CREATE", ipv4IntfMsg))
			server.createIPIntfConfMap(ipv4IntfMsg)
			if asicdConstDefs.GetIntfTypeFromIfIndex(ipv4IntfMsg.IfId) == commonDefs.L2RefTypePort { // PHY
				server.updateIpInPortPropertyMap(ipv4IntfMsg, msg.MsgType)
			} else if asicdConstDefs.GetIntfTypeFromIfIndex(ipv4IntfMsg.IfId) == commonDefs.L2RefTypeVlan { // Vlan
				server.updateIpInVlanPropertyMap(ipv4IntfMsg, msg.MsgType)
			}
		} else {
			server.logger.Info(fmt.Sprintln("Receive IPV4INTF_DELETE", ipv4IntfMsg))
			server.deleteIPIntfConfMap(ipv4IntfMsg)
			if asicdConstDefs.GetIntfTypeFromIfIndex(ipv4IntfMsg.IfId) == commonDefs.L2RefTypePort { // PHY
				server.updateIpInPortPropertyMap(ipv4IntfMsg, msg.MsgType)
			} else if asicdConstDefs.GetIntfTypeFromIfIndex(ipv4IntfMsg.IfId) == commonDefs.L2RefTypeVlan { // Vlan
				server.updateIpInVlanPropertyMap(ipv4IntfMsg, msg.MsgType)
			}
		}
	} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
		// L3INTF state change
	} else if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
		msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
		// VLAN Create, Delete
		var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
		err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal vlanNotifyMsg:", msg.Msg))
			return
		}
		server.updatePortPropertyMap(vlanNotifyMsg, msg.MsgType)
		server.updateVlanPropertyMap(vlanNotifyMsg, msg.MsgType)
	} else if msg.MsgType == asicdConstDefs.NOTIFY_LAG_CREATE ||
		msg.MsgType == asicdConstDefs.NOTIFY_LAG_DELETE {
		// LAG Create, Delete
	}
}

func (server *BFDServer) GetIPv4Interfaces() error {
	server.logger.Info("Getting IPv4 interfaces from asicd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		var ipv4IntfMsg IPv4IntfNotifyMsg
		server.logger.Info(fmt.Sprintf("Getting %d objects from currMarker %d\n", count, currMarker))
		IPIntfBulk, err := server.asicdClient.ClientHdl.GetBulkIPv4Intf(currMarker, count)
		if err != nil {
			server.logger.Info(fmt.Sprintln("GetBulkIPv4Intf with err ", err))
			return err
		}
		if IPIntfBulk.Count == 0 {
			server.logger.Info(fmt.Sprintln("0 objects returned from GetBulkIPv4Intf"))
			return nil
		}
		server.logger.Info(fmt.Sprintln("Got IPv4 interfaces - len  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfList), IPIntfBulk.Count))
		for i := 0; i < int(IPIntfBulk.Count); i++ {
			ipv4IntfMsg.IpAddr = IPIntfBulk.IPv4IntfList[i].IpAddr
			ipv4IntfMsg.IfId = IPIntfBulk.IPv4IntfList[i].IfIndex
			server.createIPIntfConfMap(ipv4IntfMsg)
			server.logger.Info(fmt.Sprintln("Created IPv4 interface (%d : %s)\n", ipv4IntfMsg.IfId, ipv4IntfMsg.IpAddr))
		}
		if IPIntfBulk.More == false {
			server.logger.Info(fmt.Sprintln("Get IPv4 interfaces - more returned as false, so no more get bulks"))
			return nil
		}
		currMarker = asicdServices.Int(IPIntfBulk.EndIdx)
	}
}
