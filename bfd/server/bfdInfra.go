package server

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"errors"
	"fmt"
	"net"
	"utils/commonDefs"
)

type PortProperty struct {
	Name     string
	VlanName string
	VlanId   uint16
	IpAddr   net.IP
}

type VlanProperty struct {
	Name       string
	UntagPorts []int32
	IpAddr     net.IP
}

type IPIntfProperty struct {
	IfName  string
	IpAddr  net.IP
	MacAddr net.HardwareAddr
	NetMask []byte
}

type LagProperty struct {
	Links []int32
}

type IPv4IntfNotifyMsg struct {
	IpAddr string
	IfId   int32
}

func (server *BFDServer) updateIpInVlanPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
		ent := server.vlanPropertyMap[msg.IfId]
		ip, _, _ := net.ParseCIDR(msg.IpAddr)
		ent.IpAddr = ip
		server.vlanPropertyMap[msg.IfId] = ent
	} else { // Delete IP
		ent := server.vlanPropertyMap[msg.IfId]
		ent.IpAddr = nil
		server.vlanPropertyMap[msg.IfId] = ent
	}
}

func (server *BFDServer) updateIpInPortPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
		ent := server.portPropertyMap[int32(msg.IfId)]
		ip, _, _ := net.ParseCIDR(msg.IpAddr)
		ent.IpAddr = ip
		server.portPropertyMap[int32(msg.IfId)] = ent
	} else { // Delete IP
		ent := server.portPropertyMap[int32(msg.IfId)]
		ent.IpAddr = nil
		server.portPropertyMap[int32(msg.IfId)] = ent
	}
}

func (server *BFDServer) updateVlanPropertyMap(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		ent := server.vlanPropertyMap[int32(vlanNotifyMsg.VlanId)]
		ent.Name = vlanNotifyMsg.VlanName
		ent.UntagPorts = vlanNotifyMsg.UntagPorts
		server.vlanPropertyMap[int32(vlanNotifyMsg.VlanId)] = ent
	} else { // Delete Vlan
		delete(server.vlanPropertyMap, int32(vlanNotifyMsg.VlanId))
	}
}

func (server *BFDServer) updatePortPropertyMap(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		for _, portNum := range vlanNotifyMsg.UntagPorts {
			ent := server.portPropertyMap[portNum]
			ent.VlanId = vlanNotifyMsg.VlanId
			ent.VlanName = vlanNotifyMsg.VlanName
			server.portPropertyMap[portNum] = ent
		}
	} else { // Delete Vlan
		for _, portNum := range vlanNotifyMsg.UntagPorts {
			ent := server.portPropertyMap[portNum]
			ent.VlanId = 0
			ent.VlanName = ""
			server.portPropertyMap[portNum] = ent
		}
	}
}

func (server *BFDServer) BuildPortPropertyMap() error {
	currMarker := asicdServices.Int(asicdConstDefs.MIN_SYS_PORTS)
	if server.asicdClient.IsConnected {
		server.logger.Info("Calling asicd for port property")
		count := 10
		for {
			server.logger.Info(fmt.Sprintln("Calling bulkget port ", currMarker, count))
			bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPortState(asicdServices.Int(currMarker), asicdServices.Int(count))
			if bulkInfo == nil {
				server.logger.Info("Bulkget port got nothing")
				return nil
			}
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			server.logger.Info(fmt.Sprintln("Bulkget port got ", objCount, more))
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				portNum := bulkInfo.PortStateList[i].PortNum
				ent := server.portPropertyMap[portNum]
				ent.Name = bulkInfo.PortStateList[i].Name
				ent.VlanId = 0
				ent.VlanName = ""
				server.portPropertyMap[portNum] = ent
			}
			if more == false {
				return nil
			}
		}
	}
	return nil
}

func (server *BFDServer) BuildLagPropertyMap() error {
	server.logger.Info("Get configured lags ... TBD")
	return nil
}

func (server *BFDServer) BuildIPv4InterfacesMap() error {
	server.logger.Info("Getting IPv4 interfaces from asicd")
	var currMarker asicdServices.Int
	var count asicdServices.Int
	count = 100
	for {
		var ipv4IntfMsg IPv4IntfNotifyMsg
		server.logger.Info(fmt.Sprintf("Getting %d objects from currMarker %d\n", count, currMarker))
		IPIntfBulk, err := server.asicdClient.ClientHdl.GetBulkIPv4IntfState(currMarker, count)
		if err != nil {
			server.logger.Info(fmt.Sprintln("GetBulkIPv4IntfState with err ", err))
			return err
		}
		if IPIntfBulk.Count == 0 {
			server.logger.Info(fmt.Sprintln("0 objects returned from GetBulkIPv4IntfState"))
			return nil
		}
		server.logger.Info(fmt.Sprintf("Got IPv4 interfaces - len  = %d, num objects returned = %d\n", len(IPIntfBulk.IPv4IntfStateList), IPIntfBulk.Count))
		for i := 0; i < int(IPIntfBulk.Count); i++ {
			ipv4IntfMsg.IpAddr = IPIntfBulk.IPv4IntfStateList[i].IpAddr
			ipv4IntfMsg.IfId = IPIntfBulk.IPv4IntfStateList[i].IfIndex
			server.createIPIntfConfMap(ipv4IntfMsg)
			server.logger.Info(fmt.Sprintf("Created IPv4 interface (%d : %s)\n", ipv4IntfMsg.IfId, ipv4IntfMsg.IpAddr))
		}
		if IPIntfBulk.More == false {
			server.logger.Info(fmt.Sprintln("Get IPv4 interfaces - more returned as false, so no more get bulks"))
			return nil
		}
		currMarker = asicdServices.Int(IPIntfBulk.EndIdx)
	}
	return nil
}

func (server *BFDServer) updateLagPropertyMap(msg asicdConstDefs.LagNotifyMsg, msgType uint8) {
	_, exists := server.lagPropertyMap[msg.IfIndex]
	if msgType == asicdConstDefs.NOTIFY_LAG_CREATE { // Create LAG
		if exists {
			server.logger.Info(fmt.Sprintln("CreateLag: already exists", msg.IfIndex))
		} else {
			server.logger.Info(fmt.Sprintln("Creating lag ", msg.IfIndex))
			lagEntry := LagProperty{}
			lagEntry.Links = make([]int32, 0)
			for _, linkNum := range msg.IfIndexList {
				lagEntry.Links = append(lagEntry.Links, linkNum)
			}
			server.lagPropertyMap[msg.IfIndex] = lagEntry
		}
	} else if msgType == asicdConstDefs.NOTIFY_LAG_DELETE { // Delete Lag
		if exists {
			server.logger.Info(fmt.Sprintln("Deleting lag ", msg.IfIndex))
			delete(server.lagPropertyMap, msg.IfIndex)
		} else {
			server.logger.Info(fmt.Sprintln("DeleteLag: Does not exist ", msg.IfIndex))
		}
	}
}

func (server *BFDServer) getLinuxIntfName(ifIndex int32) (ifName string, err error) {
	ifType := asicdConstDefs.GetIntfTypeFromIfIndex(ifIndex)
	if ifType == commonDefs.L2RefTypeVlan { // Vlan
		ifName = server.vlanPropertyMap[ifIndex].Name
	} else if ifType == commonDefs.L2RefTypePort { // PHY
		ifName = server.portPropertyMap[int32(ifIndex)].Name
	} else {
		ifName = ""
		err = errors.New("Invalid Interface Type")
	}
	return ifName, err
}

func (server *BFDServer) getMacAddrFromIntfName(ifName string) (macAddr net.HardwareAddr, err error) {
	ifi, err := net.InterfaceByName(ifName)
	if err != nil {
		return macAddr, err
	}
	macAddr = ifi.HardwareAddr
	return macAddr, nil
}
