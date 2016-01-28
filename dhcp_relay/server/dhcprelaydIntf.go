// Dhcp Relay Agent Interface Handling
package relayServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type DHCPRELAYClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	DHCPRELAYClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

/*
 * Global Variable
 */
var (
	portInfoMap    map[string]int
	asicdClient    AsicdClient
	asicdSubSocket *nanomsg.SubSocket
)

func DhcpRelayAgentListenAsicUpdate(address string) error {
	var err error
	logger.Info("DRA: setting up asicd update listener")
	if asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to create ASIC subscribe socket, error:", err))
		return err
	}

	if err = asicdSubSocket.Subscribe(""); err != nil {
		logger.Err(fmt.Sprintln("DRA:Failed to subscribe to \"\" on ASIC subscribe socket, error:", err))
		return err
	}

	if _, err = asicdSubSocket.Connect(address); err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to connect to ASIC publisher socket, address:", address, "error:", err))
		return err
	}

	logger.Info(fmt.Sprintln("DRA: Connected to ASIC publisher at address:", address))
	if err = asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to set the buffer size for ASIC publisher socket, error:", err))
		return err
	}
	logger.Info("DRA: asicd update listener is set")
	return nil
}

func DhcpRelayAgentUpdateIntfPortMap(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	// @TODO: wait for vikram to add if_name to IPv4IntfNotifyMsg... until that
	// time we cannot listen to update notification
	//intfId := asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex)
	logger.Info(fmt.Sprintln("DRA: Got a ipv4 interface notification for:", msgType))
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE {
		//portInfoMap[msg.IfIndex] = intfId
		// Init DRA Global Handling for new interface....
		//DhcpRelayAgentInitGblHandling(msg.IfIndex, intfId)
	} else {
		// @TODO: jgheewala do we need to disable relay agent for the
		// interface which is deleted... or remove the entry from
		// portInfoMap??
		//delete(portInfoMap, msg.IfIndex)
	}
}

func DhcpRelayAgentUpdateL3IntfStateChange(msg asicdConstDefs.L3IntfStateNotifyMsg) {
	if msg.IfState == asicdConstDefs.INTF_STATE_UP {
		logger.Info(fmt.Sprintln("DRA: Got intf state up notification"))

	} else if msg.IfState == asicdConstDefs.INTF_STATE_DOWN {
		logger.Info(fmt.Sprintln("DRA: Got intf state down notification"))

	}
}
func DhcpRelayAsicdSubscriber() {
	for {
		logger.Info("DRA: Read on Asic Subscriber socket....")
		rxBuf, err := asicdSubSocket.Recv(0)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Recv on asicd Subscriber socket failed with error:", err))
			continue
		}
		logger.Info(fmt.Sprintln("DRA: asicd Subscriber recv returned:", rxBuf))
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Unable to Unmarshal asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
			var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("DRA: Unable to Unmarshal ipv4IntfNotifyMsg:", msg.Msg))
				continue
			}
			DhcpRelayAgentUpdateIntfPortMap(ipv4IntfNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
			//INTF_STATE_CHANGE
			var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("DRA: unable to Unmarshal l3 intf state change:", msg.Msg))
				continue
			}
			DhcpRelayAgentUpdateL3IntfStateChange(l3IntfStateNotifyMsg)
		}
	}
}

/*
 * DhcpRelayInitPortParams:
 *	    API to handle initialization of port parameter
 */
func DhcpRelayInitPortParams() error {
	logger.Info("DRA: initializing Port Parameters & Global Init")
	// constructing port configs...
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	hack := false // dra hack for running the code on localhost
	more := false
	objCount := 0
	portNum := 0
	if !asicdClient.IsConnected {
		logger.Info("DRA: is not connected to asicd.... is it bad?")
	}
	err := DhcpRelayAgentListenAsicUpdate(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go DhcpRelayAsicdSubscriber()
	}
	logger.Info("DRA calling asicd for port config")
	count := 10
	// for optimization initializing 25 interfaces map...
	//dhcprelayGblInfo = make(map[string]DhcpRelayAgentGlobalInfo, 25)
	dhcprelayGblInfo = make(map[int]DhcpRelayAgentGlobalInfo, 25)
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkPortConfig(
			int64(currMarker), int64(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk port config"+
				" from asicd failed with reason", err))
			//return err <--- DRA doesn't start as no bulk port
			//
			logger.Info("DRA: HACK For interface is invoked")
			hack = true
			//return nil
		}
		if hack == true {
			objCount = 1
			portNum = 1
		} else {
			objCount = int(bulkInfo.ObjCount)
			more = bool(bulkInfo.More)
			currMarker = int64(bulkInfo.NextMarker)
		}
		for i := 0; i < objCount; i++ {
			//var entry portInfo
			var ifName string
			if hack == true {
				portNum = 1
				ifName = "wlp2s0" //"enp1s0f0"
			} else {
				portNum = int(bulkInfo.PortConfigList[i].IfIndex)
				ifName = bulkInfo.PortConfigList[i].Name
			}
			portInfoMap[ifName] = portNum
			// Init DRA Global Handling for all interfaces....
			DhcpRelayAgentInitGblHandling(ifName, portNum)
		}
		if hack {
			logger.Info("DRA: HACK and hence creating clien/server right away")
			DhcpRelayAgentCreateClientServerConn()
		}
		if more == false {
			break
		}
	}
	logger.Info("DRA: initialized Port Parameters & Global Info successfully")
	return nil
}
