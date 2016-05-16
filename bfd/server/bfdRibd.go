package server

import (
	"asicd/asicdCommonDefs"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/rib/ribdCommonDefs"
)

func (server *BFDServer) CreateRIBdSubscriber() {
	server.logger.Info("Listen for RIBd updates")
	server.listenForRIBdUpdates(ribdCommonDefs.PUB_SOCKET_BFDD_ADDR)
	for {
		server.logger.Info("Read on RIBd subscriber socket...")
		rxBuf, err := server.ribdSubSocket.Recv(0)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Recv on RIBd subscriber socket failed with error:", err))
			server.ribdSubSocketErrCh <- err
			continue
		}
		server.logger.Info(fmt.Sprintln("RIB subscriber recv returned:", rxBuf))
		server.ribdSubSocketCh <- rxBuf
	}
}

func (server *BFDServer) listenForRIBdUpdates(address string) error {
	var err error
	if server.ribdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to create RIBd subscribe socket, error:", err))
		return err
	}

	if _, err = server.ribdSubSocket.Connect(address); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to connect to RIBd publisher socket, address:", address, "error:", err))
		return err
	}

	if err = server.ribdSubSocket.Subscribe(""); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on RIBd subscribe socket, error:", err))
		return err
	}

	server.logger.Info(fmt.Sprintln("Connected to RIBd publisher at address:", address))
	if err = server.ribdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to set the buffer size for RIBd publisher socket, error:", err))
		return err
	}
	return nil
}

func (server *BFDServer) processRibdNotification(rxBuf []byte) error {
	var msg ribdCommonDefs.RibdNotifyMsg
	err := json.Unmarshal(rxBuf, &msg)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to unmarshal rxBuf:", rxBuf))
		return err
	}
	switch msg.MsgType {
	case ribdCommonDefs.NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE:
		server.logger.Info(fmt.Sprintln("Received NOTIFY_ROUTE_REACHABILITY_STATUS_UPDATE"))
		var msgInfo ribdCommonDefs.RouteReachabilityStatusMsgInfo
		err = json.Unmarshal(msg.MsgBuf, &msgInfo)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Unable to unmarshal msg:", msg.MsgBuf))
			return err
		}
		server.logger.Info(fmt.Sprintln(" IP ", msgInfo.Network, " reachabilityStatus: ", msgInfo.IsReachable))
		if msgInfo.IsReachable {
			server.logger.Info(fmt.Sprintln(" NextHop IP:", msgInfo.NextHopIntf.NextHopIp, " IntfType:IntfId ", msgInfo.NextHopIntf.NextHopIfType, ":", msgInfo.NextHopIntf.NextHopIfIndex))
			ifIndex := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(msgInfo.NextHopIntf.NextHopIfType), int(msgInfo.NextHopIntf.NextHopIfIndex))
			server.HandleNextHopChange(msgInfo.Network, ifIndex)
		} else {
			server.logger.Info(fmt.Sprintln(" NextHop IP:", msgInfo.NextHopIntf.NextHopIp, " is not reachable "))
			server.HandleNextHopChange(msgInfo.Network, 0)
		}
		break
	default:
		break
	}
	return nil
}
