package server

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"utils/ipcutils"
)

type AsicdClient struct {
	ipcutils.IPCClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

func (server *BFDServer) CreateASICdSubscriber() {
	server.logger.Info("Listen for ASICd updates")
	server.listenForASICdUpdates(asicdCommonDefs.PUB_SOCKET_ADDR)
	for {
		server.logger.Debug("Read on ASICd subscriber socket...")
		asicdrxBuf, err := server.asicdSubSocket.Recv(0)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Recv on ASICd subscriber socket failed with error:", err))
			server.asicdSubSocketErrCh <- err
			continue
		}
		server.logger.Debug(fmt.Sprintln("ASIC subscriber recv returned:", asicdrxBuf))
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
	var msg asicdCommonDefs.AsicdNotification
	err := json.Unmarshal(asicdrxBuf, &msg)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to unmarshal asicdrxBuf:", asicdrxBuf))
		return
	}
	if msg.MsgType == asicdCommonDefs.NOTIFY_VLAN_CREATE ||
		msg.MsgType == asicdCommonDefs.NOTIFY_VLAN_DELETE {
		// VLAN Create, Delete
		var vlanNotifyMsg asicdCommonDefs.VlanNotifyMsg
		err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal vlanNotifyMsg:", msg.Msg))
			return
		}
		server.updatePortPropertyMap(vlanNotifyMsg, msg.MsgType)
		server.updateVlanPropertyMap(vlanNotifyMsg, msg.MsgType)
	} else if msg.MsgType == asicdCommonDefs.NOTIFY_LAG_CREATE ||
		msg.MsgType == asicdCommonDefs.NOTIFY_LAG_DELETE {
		// LAG Create, Delete
		server.logger.Info("Recvd NOTIFY_LAG notification")
		var lagNotifyMsg asicdCommonDefs.LagNotifyMsg
		err = json.Unmarshal(msg.Msg, &lagNotifyMsg)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmashal lagNotifyMsg:", msg.Msg))
			return
		}
		server.updateLagPropertyMap(lagNotifyMsg, msg.MsgType)
	}
}
