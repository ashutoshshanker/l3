// ribdEventHandler.go
package main

import (
	"asicd/asicdConstDefs"
	"encoding/json"
	"fmt"
	"github.com/op/go-nanomsg"
	"net"
	"ribd"
	"strconv"
	"utils/commonDefs"
)

func processAsicdEvents(sub *nanomsg.SubSocket) {

	logger.Println("in process Asicd events")
	logger.Info(fmt.Sprintln(" asicdConstDefs.NOTIFY_IPV4INTF_CREATE = ", asicdConstDefs.NOTIFY_IPV4INTF_CREATE, "asicdConstDefs.asicdConstDefs.NOTIFY_IPV4INTF_DELETE: ", asicdConstDefs.NOTIFY_IPV4INTF_DELETE))
	for {
		logger.Println("In for loop")
		rcvdMsg, err := sub.Recv(0)
		if err != nil {
			logger.Info(fmt.Sprintln("Error in receiving ", err))
			return
		}
		logger.Info(fmt.Sprintln("After recv rcvdMsg buf", rcvdMsg))
		Notif := asicdConstDefs.AsicdNotification{}
		err = json.Unmarshal(rcvdMsg, &Notif)
		if err != nil {
			logger.Println("Error in Unmarshalling rcvdMsg Json")
			return
		}
		switch Notif.MsgType {
		case asicdConstDefs.NOTIFY_VLAN_CREATE:
			logger.Println("asicdConstDefs.NOTIFY_VLAN_CREATE")
			var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
			err = json.Unmarshal(Notif.Msg, &vlanNotifyMsg)
			if err != nil {
				logger.Info(fmt.Sprintln("Unable to unmashal vlanNotifyMsg:", Notif.Msg))
				return
			}
			ifId := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(vlanNotifyMsg.VlanId), commonDefs.L2RefTypeVlan)
			logger.Info(fmt.Sprintln("vlanId ", vlanNotifyMsg.VlanId, " ifId:", ifId))
			if IntfIdNameMap == nil {
				IntfIdNameMap = make(map[int32]IntfEntry)
			}
			intfEntry := IntfEntry{name: vlanNotifyMsg.VlanName}
			IntfIdNameMap[int32(ifId)] = intfEntry
			break
		case asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE:
			logger.Println("NOTIFY_L3INTF_STATE_CHANGE event")
			var msg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(Notif.Msg, &msg)
			if err != nil {
				logger.Info(fmt.Sprintln("Error in reading msg ", err))
				return
			}
			logger.Info(fmt.Sprintln("Msg linkstatus = %d msg ifType = %d ifId = %d\n", msg.IfState, msg.IfIndex))
			if msg.IfState == asicdConstDefs.INTF_STATE_DOWN {
				//processLinkDownEvent(ribd.Int(msg.IfType), ribd.Int(msg.IfId))
				processL3IntfDownEvent(msg.IpAddr)
			} else {
				//processLinkUpEvent(ribd.Int(msg.IfType), ribd.Int(msg.IfId))
				processL3IntfUpEvent(msg.IpAddr)
			}
			break
		case asicdConstDefs.NOTIFY_IPV4INTF_CREATE:
			logger.Println("NOTIFY_IPV4INTF_CREATE event")
			var msg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(Notif.Msg, &msg)
			if err != nil {
				logger.Info(fmt.Sprintln("Error in reading msg ", err))
				return
			}
			logger.Info(fmt.Sprintln("Received ipv4 intf create with ipAddr ", msg.IpAddr, " ifIndex = ", msg.IfIndex, " ifType ", asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex), " ifId ", asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex)))
			var ipMask net.IP
			ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
			if err != nil {
				return
			}
			ipMask = make(net.IP, 4)
			copy(ipMask, ipNet.Mask)
			ipAddrStr := ip.String()
			ipMaskStr := net.IP(ipMask).String()
			logger.Info(fmt.Sprintln("Calling createv4Route with ipaddr ", ipAddrStr, " mask ", ipMaskStr))
			nextHopIfTypeStr := ""
			switch asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex) {
			case commonDefs.L2RefTypePort:
				nextHopIfTypeStr = "PHY"
				break
			case commonDefs.L2RefTypeVlan:
				nextHopIfTypeStr = "VLAN"
				break
			case commonDefs.IfTypeNull:
				nextHopIfTypeStr = "NULL"
				break
			case commonDefs.IfTypeLoopback:
				nextHopIfTypeStr = "Loopback"
				if IntfIdNameMap == nil {
					IntfIdNameMap = make(map[int32]IntfEntry)
				}
				intfEntry := IntfEntry{}
				IntfIdNameMap[msg.IfIndex] = intfEntry
				break
			}
			cfg := ribd.IPv4Route{
				DestinationNw:     ipAddrStr,
				Protocol:          "CONNECTED",
				OutgoingInterface: strconv.Itoa(int(asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex))),
				OutgoingIntfType:  nextHopIfTypeStr,
				Cost:              0,
				NetworkMask:       ipMaskStr,
				NextHopIp:         "0.0.0.0"}

			_, err = routeServiceHandler.CreateIPv4Route(&cfg) //ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex)), "CONNECTED")
			//_, err = createV4Route(ipAddrStr, ipMaskStr, 0, "0.0.0.0", ribd.Int(asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)), ribd.Int(asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex)), ribdCommonDefs.CONNECTED, FIBAndRIB, ribdCommonDefs.RoutePolicyStateChangetoValid,ribd.Int(len(destNetSlice)))
			if err != nil {
				logger.Info(fmt.Sprintln("Route create failed with err %s\n", err))
				return
			}
			break
		case asicdConstDefs.NOTIFY_IPV4INTF_DELETE:
			logger.Println("NOTIFY_IPV4INTF_DELETE  event")
			break
		}
	}
}
func processEvents(sub *nanomsg.SubSocket, subType ribd.Int) {
	logger.Info(fmt.Sprintln("in process events for sub ", subType))
	if subType == SUB_ASICD {
		logger.Println("process Asicd events")
		processAsicdEvents(sub)
	}
}
func SetupEventHandler(sub *nanomsg.SubSocket, address string, subtype ribd.Int) {
	logger.Info(fmt.Sprintln("Setting up event handlers for sub type ", subtype))
	sub, err := nanomsg.NewSubSocket()
	if err != nil {
		logger.Println("Failed to open sub socket")
		return
	}
	logger.Println("opened socket")
	ep, err := sub.Connect(address)
	if err != nil {
		logger.Info(fmt.Sprintln("Failed to connect to pub socket - ", ep))
		return
	}
	logger.Info(fmt.Sprintln("Connected to ", ep.Address))
	err = sub.Subscribe("")
	if err != nil {
		logger.Println("Failed to subscribe to all topics")
		return
	}
	logger.Println("Subscribed")
	err = sub.SetRecvBuffer(1024 * 1204)
	if err != nil {
		logger.Println("Failed to set recv buffer size")
		return
	}
	//processPortdEvents(sub)
	processEvents(sub, subtype)
}
