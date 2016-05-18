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

// ribdEventHandler.go
package server

import (
	"asicd/asicdCommonDefs"
	"encoding/json"
	"fmt"
	"github.com/op/go-nanomsg"
	"net"
	"ribd"
	"strconv"
	"utils/commonDefs"
)

func (ribdServiceHandler *RIBDServer) ProcessAsicdEvents(sub *nanomsg.SubSocket) {

	ribdServiceHandler.Logger.Info("in process Asicd events")
	ribdServiceHandler.Logger.Info(fmt.Sprintln(" asicdCommonDefs.NOTIFY_IPV4INTF_CREATE = ", asicdCommonDefs.NOTIFY_IPV4INTF_CREATE, "asicdCommonDefs.asicdCommonDefs.NOTIFY_IPV4INTF_DELETE: ", asicdCommonDefs.NOTIFY_IPV4INTF_DELETE))
	for {
		ribdServiceHandler.Logger.Info("In for loop")
		rcvdMsg, err := sub.Recv(0)
		if err != nil {
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Error in receiving ", err))
			return
		}
		ribdServiceHandler.Logger.Info(fmt.Sprintln("After recv rcvdMsg buf", rcvdMsg))
		Notif := asicdCommonDefs.AsicdNotification{}
		err = json.Unmarshal(rcvdMsg, &Notif)
		if err != nil {
			ribdServiceHandler.Logger.Info("Error in Unmarshalling rcvdMsg Json")
			return
		}
		switch Notif.MsgType {
		case asicdCommonDefs.NOTIFY_LOGICAL_INTF_CREATE:
			ribdServiceHandler.Logger.Info("NOTIFY_LOGICAL_INTF_CREATE received")
			var logicalIntfNotifyMsg asicdCommonDefs.LogicalIntfNotifyMsg
			err = json.Unmarshal(Notif.Msg, &logicalIntfNotifyMsg)
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Unable to unmashal logicalIntfNotifyMsg:", Notif.Msg))
				return
			}
			ifId := logicalIntfNotifyMsg.IfIndex
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: logicalIntfNotifyMsg.LogicalIntfName}
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Updating IntfIdMap at index ", ifId, " with name ", logicalIntfNotifyMsg.LogicalIntfName))
			IntfIdNameMap[int32(ifId)] = intfEntry
			if IfNameToIfIndex == nil {
				IfNameToIfIndex = make(map[string]int32)
			}
			IfNameToIfIndex[logicalIntfNotifyMsg.LogicalIntfName] = ifId
			break
		case asicdCommonDefs.NOTIFY_VLAN_CREATE:
			ribdServiceHandler.Logger.Info("asicdCommonDefs.NOTIFY_VLAN_CREATE")
			var vlanNotifyMsg asicdCommonDefs.VlanNotifyMsg
			err = json.Unmarshal(Notif.Msg, &vlanNotifyMsg)
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Unable to unmashal vlanNotifyMsg:", Notif.Msg))
				return
			}
			ifId := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(vlanNotifyMsg.VlanId), commonDefs.IfTypeVlan)
			ribdServiceHandler.Logger.Info(fmt.Sprintln("vlanId ", vlanNotifyMsg.VlanId, " ifId:", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: vlanNotifyMsg.VlanName}
			IntfIdNameMap[int32(ifId)] = intfEntry
			if IfNameToIfIndex == nil {
				IfNameToIfIndex = make(map[string]int32)
			}
			IfNameToIfIndex[vlanNotifyMsg.VlanName] = ifId
			break
		case asicdCommonDefs.NOTIFY_L3INTF_STATE_CHANGE:
			ribdServiceHandler.Logger.Info("NOTIFY_L3INTF_STATE_CHANGE event")
			var msg asicdCommonDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(Notif.Msg, &msg)
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Error in reading msg ", err))
				return
			}
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Msg linkstatus = %d msg ifType = %d ifId = %d\n", msg.IfState, msg.IfIndex))
			if msg.IfState == asicdCommonDefs.INTF_STATE_DOWN {
				//processLinkDownEvent(ribd.Int(msg.IfType), ribd.Int(msg.IfId))
				ribdServiceHandler.ProcessL3IntfDownEvent(msg.IpAddr)
			} else {
				//processLinkUpEvent(ribd.Int(msg.IfType), ribd.Int(msg.IfId))
				ribdServiceHandler.ProcessL3IntfUpEvent(msg.IpAddr)
			}
			break
		case asicdCommonDefs.NOTIFY_IPV4INTF_CREATE:
			ribdServiceHandler.Logger.Info("NOTIFY_IPV4INTF_CREATE event")
			var msg asicdCommonDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(Notif.Msg, &msg)
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Error in reading msg ", err))
				return
			}
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Received NOTIFY_IPV4INTF_CREATE ipAddr ", msg.IpAddr, " ifIndex = ", msg.IfIndex, " ifType ", asicdCommonDefs.GetIntfTypeFromIfIndex(msg.IfIndex), " ifId ", asicdCommonDefs.GetIntfIdFromIfIndex(msg.IfIndex)))
			var ipMask net.IP
			ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
			if err != nil {
				return
			}
			ipMask = make(net.IP, 4)
			copy(ipMask, ipNet.Mask)
			ipAddrStr := ip.String()
			ipMaskStr := net.IP(ipMask).String()
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Calling createv4Route with ipaddr ", ipAddrStr, " mask ", ipMaskStr, " nextHopIntRef: ",strconv.Itoa(int(msg.IfIndex) )))
			cfg := ribd.IPv4Route{
				DestinationNw: ipAddrStr,
				Protocol:      "CONNECTED",
				Cost:          0,
				NetworkMask:   ipMaskStr,
			}
			nextHop := ribd.NextHopInfo{
				NextHopIp:     "0.0.0.0",
				NextHopIntRef: strconv.Itoa(int(msg.IfIndex)),
			}
			cfg.NextHop = make([]*ribd.NextHopInfo, 0)
			cfg.NextHop = append(cfg.NextHop, &nextHop)

			_, err = ribdServiceHandler.ProcessRouteCreateConfig(&cfg) //ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(msg.IfIndex)), "CONNECTED")
			//_, err = createV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(msg.IfIndex)), ribdCommonDefs.CONNECTED, FIBAndRIB, ribdCommonDefs.RoutePolicyStateChangetoValid,ribd.Int(len(destNetSlice)))
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Route create failed with err %s\n", err))
				return
			}
			break
		case asicdCommonDefs.NOTIFY_IPV4INTF_DELETE:
			ribdServiceHandler.Logger.Info("NOTIFY_IPV4INTF_DELETE  event")
			var msg asicdCommonDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(Notif.Msg, &msg)
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Error in reading msg ", err))
				return
			}
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Received ipv4 intf delete with ipAddr ", msg.IpAddr, " ifIndex = ", msg.IfIndex, " ifType ", asicdCommonDefs.GetIntfTypeFromIfIndex(msg.IfIndex), " ifId ", asicdCommonDefs.GetIntfIdFromIfIndex(msg.IfIndex)))
			var ipMask net.IP
			ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
			if err != nil {
				return
			}
			ipMask = make(net.IP, 4)
			copy(ipMask, ipNet.Mask)
			ipAddrStr := ip.String()
			ipMaskStr := net.IP(ipMask).String()
			ribdServiceHandler.Logger.Info(fmt.Sprintln("Calling deletev4Route with ipaddr ", ipAddrStr, " mask ", ipMaskStr))
			cfg := ribd.IPv4Route{
				DestinationNw: ipAddrStr,
				Protocol:      "CONNECTED",
				Cost:          0,
				NetworkMask:   ipMaskStr,
			}
			nextHop := ribd.NextHopInfo{
				NextHopIp:     "0.0.0.0",
				NextHopIntRef: strconv.Itoa(int(msg.IfIndex)),
			}
			cfg.NextHop = make([]*ribd.NextHopInfo, 0)
			cfg.NextHop = append(cfg.NextHop, &nextHop)
			_, err = ribdServiceHandler.ProcessRouteDeleteConfig(&cfg) //ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdCommonDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdCommonDefs.GetIntfIdFromIfIndex(msg.IfIndex)), "CONNECTED")
			if err != nil {
				ribdServiceHandler.Logger.Info(fmt.Sprintln("Route delete failed with err %s\n", err))
				return
			}
			break
		}
	}
}
func (ribdServiceHandler *RIBDServer) ProcessEvents(sub *nanomsg.SubSocket, subType ribd.Int) {
	ribdServiceHandler.Logger.Info(fmt.Sprintln("in process events for sub ", subType))
	if subType == SUB_ASICD {
		ribdServiceHandler.Logger.Info("process Asicd events")
		ribdServiceHandler.ProcessAsicdEvents(sub)
	}
}
func (ribdServiceHandler *RIBDServer) SetupEventHandler(sub *nanomsg.SubSocket, address string, subtype ribd.Int) {
	ribdServiceHandler.Logger.Info(fmt.Sprintln("Setting up event handlers for sub type ", subtype))
	sub, err := nanomsg.NewSubSocket()
	if err != nil {
		ribdServiceHandler.Logger.Info("Failed to open sub socket")
		return
	}
	ribdServiceHandler.Logger.Info("opened socket")
	ep, err := sub.Connect(address)
	if err != nil {
		ribdServiceHandler.Logger.Info(fmt.Sprintln("Failed to connect to pub socket - ", ep))
		return
	}
	ribdServiceHandler.Logger.Info(fmt.Sprintln("Connected to ", ep.Address))
	err = sub.Subscribe("")
	if err != nil {
		ribdServiceHandler.Logger.Info("Failed to subscribe to all topics")
		return
	}
	ribdServiceHandler.Logger.Info("Subscribed")
	err = sub.SetRecvBuffer(1024 * 1204)
	if err != nil {
		ribdServiceHandler.Logger.Info("Failed to set recv buffer size")
		return
	}
	//processPortdEvents(sub)
	ribdServiceHandler.ProcessEvents(sub, subtype)
}
