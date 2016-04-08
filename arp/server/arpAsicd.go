package server

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	//"net"
	nanomsg "github.com/op/go-nanomsg"
	//"utils/commonDefs"
)

type AsicdClient struct {
	ArpClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

func (server *ARPServer) createASICdSubscriber() {
	for {
		server.logger.Info("Read on ASICd subscriber socket...")
		asicdrxBuf, err := server.asicdSubSocket.Recv(0)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Recv on ASICd subscriber socket failed with error:", err))
			server.asicdSubSocketErrCh <- err
			continue
		}
		//server.logger.Info(fmt.Sprintln("ASIC subscriber recv returned:", asicdrxBuf))
		server.asicdSubSocketCh <- asicdrxBuf
	}
}

func (server *ARPServer) listenForASICdUpdates(address string) error {
	var err error
	if server.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to create ASICd subscribe socket, error:", err))
		return err
	}

	if err = server.asicdSubSocket.Subscribe(""); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on ASICd subscribe socket, error:", err))
		return err
	}

	if _, err = server.asicdSubSocket.Connect(address); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to connect to ASICd publisher socket, address:", address, "error:", err))
		return err
	}

	server.logger.Info(fmt.Sprintln("Connected to ASICd publisher at address:", address))
	if err = server.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to set the buffer size for ASICd publisher socket, error:", err))
		return err
	}
	return nil
}

func (server *ARPServer) processAsicdNotification(asicdrxBuf []byte) {
	var rxMsg asicdConstDefs.AsicdNotification
	err := json.Unmarshal(asicdrxBuf, &rxMsg)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to unmarshal asicdrxBuf:", asicdrxBuf))
		return
	}
	if rxMsg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
		rxMsg.MsgType == asicdConstDefs.NOTIFY_VLAN_UPDATE ||
		rxMsg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
		//Vlan Create Msg
		server.logger.Info("Recvd VLAN notification")
		var vlanMsg asicdConstDefs.VlanNotifyMsg
		err = json.Unmarshal(rxMsg.Msg, &vlanMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal vlanNotifyMsg:", rxMsg.Msg))
			return
		}
		server.updateVlanInfra(vlanMsg, rxMsg.MsgType)
	} else if rxMsg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
		rxMsg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
		server.logger.Info("Recvd IPV4INTF notification")
		var v4Msg asicdConstDefs.IPv4IntfNotifyMsg
		err = json.Unmarshal(rxMsg.Msg, &v4Msg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal ipv4IntfNotifyMsg:", rxMsg.Msg))
			return
		}
		server.updateIpv4Infra(v4Msg, rxMsg.MsgType)
	} else if rxMsg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
		//L3_INTF_STATE_CHANGE
		server.logger.Info("Recvd INTF_STATE_CHANGE notification")
		var l3IntfMsg asicdConstDefs.L3IntfStateNotifyMsg
		err = json.Unmarshal(rxMsg.Msg, &l3IntfMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal l3IntfStateNotifyMsg:", rxMsg.Msg))
			return
		}
		server.processL3StateChange(l3IntfMsg)
	} else if rxMsg.MsgType == asicdConstDefs.NOTIFY_L2INTF_STATE_CHANGE {
		//L2_INTF_STATE_CHANGE
		server.logger.Info("Recvd INTF_STATE_CHANGE notification")
		var l2IntfMsg asicdConstDefs.L2IntfStateNotifyMsg
		err = json.Unmarshal(rxMsg.Msg, &l2IntfMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal l2IntfStateNotifyMsg:", rxMsg.Msg))
			return
		}
		//server.processL2StateChange(l2IntfMsg)
	} else if rxMsg.MsgType == asicdConstDefs.NOTIFY_LAG_CREATE ||
		rxMsg.MsgType == asicdConstDefs.NOTIFY_LAG_UPDATE ||
		rxMsg.MsgType == asicdConstDefs.NOTIFY_LAG_DELETE {
		server.logger.Info("Recvd NOTIFY_LAG notification")
		var lagMsg asicdConstDefs.LagNotifyMsg
		err = json.Unmarshal(rxMsg.Msg, &lagMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal lagNotifyMsg:", rxMsg.Msg))
			return
		}
		server.updateLagInfra(lagMsg, rxMsg.MsgType)
	} else if rxMsg.MsgType == asicdConstDefs.NOTIFY_IPV4NBR_MAC_MOVE {
		//IPv4 Neighbor mac move
		server.logger.Info("Recvd IPv4NBR_MAC_MOVE notification")
		var macMoveMsg asicdConstDefs.IPv4NbrMacMoveNotifyMsg
		err = json.Unmarshal(rxMsg.Msg, &macMoveMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal macMoveNotifyMsg:", rxMsg.Msg))
			return
		}
		server.processIPv4NbrMacMove(macMoveMsg)
	}
}
