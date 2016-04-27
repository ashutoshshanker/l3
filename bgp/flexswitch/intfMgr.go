package FSMgr

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bgp/api"
	"l3/bgp/config"
	"l3/bgp/rpc"
	"strconv"
	"utils/logging"
)

/*  Interface manager is responsible for handling asicd notifications and hence
 *  we are creating asicd client
 */
func NewFSIntfMgr(logger *logging.Writer, fileName string) (*FSIntfMgr, error) {
	var asicdClient *asicdServices.ASICDServicesClient = nil
	asicdClientChan := make(chan *asicdServices.ASICDServicesClient)

	logger.Info("Connecting to ASICd")
	go rpc.StartAsicdClient(logger, fileName, asicdClientChan)
	asicdClient = <-asicdClientChan
	if asicdClient == nil {
		logger.Err("Failed to connect to ASICd")
		return nil, errors.New("Failed to connect to ASICd")
	} else {
		logger.Info("Connected to ASICd")
	}
	mgr := &FSIntfMgr{
		plugin:      "ovsdb",
		AsicdClient: asicdClient,
		logger:      logger,
	}
	return mgr, nil
}

/*  Do any necessary init. Called from server..
 */
func (mgr *FSIntfMgr) Start() {
	mgr.asicdL3IntfSubSocket, _ = mgr.setupSubSocket(asicdCommonDefs.PUB_SOCKET_ADDR)
	go mgr.listenForAsicdEvents()
}

/*  Create One way communication asicd sub-socket
 */
func (mgr *FSIntfMgr) setupSubSocket(address string) (*nanomsg.SubSocket, error) {
	var err error
	var socket *nanomsg.SubSocket
	if socket, err = nanomsg.NewSubSocket(); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to create subscribe socket %s, error:%s",
			address, err))
		return nil, err
	}

	if err = socket.Subscribe(""); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to subscribe to \"\" on subscribe socket %s,",
			"error:%s", address, err))
		return nil, err
	}

	if _, err = socket.Connect(address); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to connect to publisher socket %s, error:%s",
			address, err))
		return nil, err
	}

	mgr.logger.Info(fmt.Sprintf("Connected to publisher socker %s", address))
	if err = socket.SetRecvBuffer(1024 * 1024); err != nil {
		mgr.logger.Err(fmt.Sprintln("Failed to set the buffer size for subsriber socket %s,",
			"error:", address, err))
		return nil, err
	}
	return socket, nil
}

/*  listen for asicd events mainly L3 interface state change
 */
func (mgr *FSIntfMgr) listenForAsicdEvents() {
	for {
		mgr.logger.Info("Read on Asicd subscriber socket...")
		rxBuf, err := mgr.asicdL3IntfSubSocket.Recv(0)
		if err != nil {
			mgr.logger.Info(fmt.Sprintln("Error in receiving Asicd events", err))
			return
		}

		mgr.logger.Info(fmt.Sprintln("Asicd subscriber recv returned", rxBuf))
		event := asicdCommonDefs.AsicdNotification{}
		err = json.Unmarshal(rxBuf, &event)
		if err != nil {
			mgr.logger.Err(fmt.Sprintf("Unmarshal Asicd event failed with err %s", err))
			return
		}

		switch event.MsgType {
		case asicdCommonDefs.NOTIFY_L3INTF_STATE_CHANGE:
			var msg asicdCommonDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(event.Msg, &msg)
			if err != nil {
				mgr.logger.Err(fmt.Sprintf("Unmarshal Asicd L3INTF",
					"event failed with err %s", err))
				return
			}

			mgr.logger.Info(fmt.Sprintf("Asicd L3INTF event idx %d ip %s state %d\n",
				msg.IfIndex, msg.IpAddr, msg.IfState))
			if msg.IfState == asicdCommonDefs.INTF_STATE_DOWN {
				api.SendIntfNotification(msg.IfIndex, msg.IpAddr,
					config.INTF_STATE_DOWN)
			} else {
				api.SendIntfNotification(msg.IfIndex, msg.IpAddr,
					config.INTF_STATE_UP)
			}
		}
	}
}

func (mgr *FSIntfMgr) GetIPv4Information(ifIndex int32) (string, error) {
	ipv4IntfState, err := mgr.AsicdClient.GetIPv4IntfState(strconv.Itoa(int(ifIndex)))
	if err != nil {
		return "", nil
	}
	return ipv4IntfState.IpAddr, err
}

func (mgr *FSIntfMgr) GetIfIndex(ifIndex, ifType int) int32 {
	return asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(ifIndex, ifType)
}
