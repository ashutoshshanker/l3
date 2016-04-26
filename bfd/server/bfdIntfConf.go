package server

import (
	"fmt"
	"net"
)

func (server *BFDServer) initDefaultIntfConf(ifIndex int32, ipIntfProp IpIntfProperty) {
	_, exist := server.bfdGlobal.Interfaces[ifIndex]
	if !exist {
		intf := &BfdInterface{}
		intf.conf.InterfaceId = ifIndex
		intf.conf.LocalMultiplier = DEFAULT_DETECT_MULTI
		intf.conf.DesiredMinTxInterval = DEFAULT_DESIRED_MIN_TX_INTERVAL
		intf.conf.RequiredMinRxInterval = DEFAULT_REQUIRED_MIN_RX_INTERVAL
		intf.conf.RequiredMinEchoRxInterval = DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL
		intf.conf.DemandEnabled = false
		intf.conf.AuthenticationEnabled = false
		intf.conf.AuthenticationType = 0
		intf.conf.AuthenticationKeyId = 0
		intf.conf.AuthenticationData = ""
		intf.property.IpAddr = ipIntfProp.IpAddr
		intf.property.NetMask = ipIntfProp.NetMask
		server.bfdGlobal.Interfaces[ifIndex] = intf
		server.bfdGlobal.InterfacesIdSlice = append(server.bfdGlobal.InterfacesIdSlice, ifIndex)
		server.logger.Info(fmt.Sprintln("Intf initialized ", ifIndex))
	} else {
		server.logger.Info(fmt.Sprintln("Intf Conf is not initialized ", ifIndex))
	}
}

func (server *BFDServer) createIPIntfConfMap(msg IPv4IntfNotifyMsg) {
	ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
	if err != nil {
		server.logger.Err(fmt.Sprintln("Unable to parse IP address", msg.IpAddr))
		return
	}
	ipIntfProp := IpIntfProperty{
		IpAddr:  ip,
		NetMask: ipNet.Mask,
	}
	server.initDefaultIntfConf(msg.IfId, ipIntfProp)
	_, exist := server.bfdGlobal.Interfaces[msg.IfId]
	if !exist {
		server.logger.Err("No such inteface exists")
		return
	}
}

func (server *BFDServer) deleteIPIntfConfMap(msg IPv4IntfNotifyMsg) {
	var i int
	server.logger.Info(fmt.Sprintln("delete IPIntfConfMap for ", msg))

	_, exist := server.bfdGlobal.Interfaces[msg.IfId]
	if !exist {
		server.logger.Err("No such inteface exists")
		return
	}
	delete(server.bfdGlobal.Interfaces, msg.IfId)
	for i = 0; i < len(server.bfdGlobal.InterfacesIdSlice); i++ {
		if server.bfdGlobal.InterfacesIdSlice[i] == msg.IfId {
			break
		}
	}
	server.bfdGlobal.InterfacesIdSlice = append(server.bfdGlobal.InterfacesIdSlice[:i], server.bfdGlobal.InterfacesIdSlice[i+1:]...)
}

func (server *BFDServer) updateIPIntfConfMap(ifConf IntfConfig) {
	intf, exist := server.bfdGlobal.Interfaces[ifConf.InterfaceId]
	//  we can update only when we already have entry
	if exist {
		intf.conf.InterfaceId = ifConf.InterfaceId
		intf.conf.LocalMultiplier = ifConf.LocalMultiplier
		intf.conf.DesiredMinTxInterval = ifConf.DesiredMinTxInterval * 1000
		intf.conf.RequiredMinRxInterval = ifConf.RequiredMinRxInterval * 1000
		intf.conf.RequiredMinEchoRxInterval = ifConf.RequiredMinEchoRxInterval * 1000
		intf.conf.DemandEnabled = ifConf.DemandEnabled
		intf.conf.AuthenticationEnabled = ifConf.AuthenticationEnabled
		intf.conf.AuthenticationType = ifConf.AuthenticationType
		intf.conf.AuthenticationKeyId = ifConf.AuthenticationKeyId
		intf.conf.AuthenticationData = ifConf.AuthenticationData
		server.UpdateBfdSessionsOnInterface(intf.conf.InterfaceId)
	}
}
