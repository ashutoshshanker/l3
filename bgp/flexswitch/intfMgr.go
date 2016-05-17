//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

package FSMgr

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"fmt"
	"l3/bgp/api"
	"l3/bgp/config"
	"l3/bgp/rpc"
	"strconv"
	"utils/logging"

	nanomsg "github.com/op/go-nanomsg"
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

			mgr.logger.Info(fmt.Sprintf("Asicd L3INTF event idx %d ip %s state %d\n", msg.IfIndex, msg.IpAddr,
				msg.IfState))
			if msg.IfState == asicdCommonDefs.INTF_STATE_DOWN {
				api.SendIntfNotification(msg.IfIndex, msg.IpAddr, config.INTF_STATE_DOWN)
			} else {
				api.SendIntfNotification(msg.IfIndex, msg.IpAddr, config.INTF_STATE_UP)
			}

		case asicdCommonDefs.NOTIFY_IPV4INTF_CREATE, asicdCommonDefs.NOTIFY_IPV4INTF_DELETE:
			var msg asicdCommonDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(event.Msg, &msg)
			if err != nil {
				mgr.logger.Err(fmt.Sprintf("Unmarshal Asicd IPV4INTF event failed with err %s", err))
				return
			}

			mgr.logger.Info(fmt.Sprintf("Asicd IPV4INTF event idx %d ip %s\n", msg.IfIndex, msg.IpAddr))
			if event.MsgType == asicdCommonDefs.NOTIFY_IPV4INTF_CREATE {
				api.SendIntfNotification(msg.IfIndex, msg.IpAddr, config.INTF_CREATED)
			} else {
				api.SendIntfNotification(msg.IfIndex, msg.IpAddr, config.INTF_DELETED)
			}
		}
	}
}

func (mgr *FSIntfMgr) GetIPv4Intfs() []*config.IntfStateInfo {
	var currMarker asicdServices.Int
	var count asicdServices.Int
	intfs := make([]*config.IntfStateInfo, 0)
	count = 100
	for {
		mgr.logger.Info(fmt.Sprintln("Getting ", count,
			"IPv4IntfState objects from currMarker", currMarker))
		getBulkInfo, err := mgr.AsicdClient.GetBulkIPv4IntfState(currMarker, count)
		if err != nil {
			mgr.logger.Info(fmt.Sprintln("GetBulkIPv4IntfState failed with error", err))
			break
		}
		if getBulkInfo.Count == 0 {
			mgr.logger.Info("0 objects returned from GetBulkIPv4IntfState")
			break
		}
		mgr.logger.Info(fmt.Sprintln("len(getBulkInfo.IPv4IntfStateList)  =", len(getBulkInfo.IPv4IntfStateList),
			"num objects returned =", getBulkInfo.Count))
		for _, intfState := range getBulkInfo.IPv4IntfStateList {
			intf := config.NewIntfStateInfo(intfState.IfIndex, intfState.IpAddr, config.INTF_CREATED)
			intfs = append(intfs, intf)
		}
		if getBulkInfo.More == false {
			mgr.logger.Info("more returned as false, so no more get bulks")
			break
		}
		currMarker = getBulkInfo.EndIdx
	}

	return intfs
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
