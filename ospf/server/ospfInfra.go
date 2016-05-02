package server

import (
	"asicd/asicdCommonDefs"
	"asicdServices"
	"errors"
	"net"
	"utils/commonDefs"
)

type PortProperty struct {
	Name     string
	VlanName string
	VlanId   uint16
	IpAddr   net.IP
	Mtu      int32
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
	Mtu     int32
}

//FIXME: Old ipv4intf notify msg format from asic. Needs to be cleaned up later
type IPv4IntfNotifyMsg struct {
	IpAddr string
	IfId   uint16
	IfType uint8
}

type IpProperty struct {
	IfId   uint16
	IfType uint8
}

func (server *OSPFServer) computeMinMTU(msg IPv4IntfNotifyMsg) int32 {
	var minMtu int32 = 10000                    //in bytes
	if msg.IfType == commonDefs.IfTypePort { // PHY
		ent, _ := server.portPropertyMap[int32(msg.IfId)]
		minMtu = ent.Mtu
	} else if msg.IfType == commonDefs.IfTypeVlan { // Vlan
		ent, _ := server.vlanPropertyMap[msg.IfId]
		for _, portNum := range ent.UntagPorts {
			entry, _ := server.portPropertyMap[portNum]
			if minMtu > entry.Mtu {
				minMtu = entry.Mtu
			}
		}
	}
	return minMtu
}

func (server *OSPFServer) updateIpPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	ipAddr, _, _ := net.ParseCIDR(msg.IpAddr)
	ip := convertAreaOrRouterIdUint32(ipAddr.String())
	if msgType == asicdCommonDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
		ent := server.ipPropertyMap[ip]
		ent.IfId = msg.IfId
		ent.IfType = msg.IfType
		server.ipPropertyMap[ip] = ent
	} else { // Delete IP
		delete(server.ipPropertyMap, ip)
	}
}

func (server *OSPFServer) updateIpInVlanPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdCommonDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
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

func (server *OSPFServer) updateIpInPortPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdCommonDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
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

func (server *OSPFServer) updateVlanPropertyMap(vlanNotifyMsg asicdCommonDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdCommonDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		ent := server.vlanPropertyMap[vlanNotifyMsg.VlanId]
		ent.Name = vlanNotifyMsg.VlanName
		ent.UntagPorts = vlanNotifyMsg.UntagPorts
		server.vlanPropertyMap[vlanNotifyMsg.VlanId] = ent
	} else { // Delete Vlan
		delete(server.vlanPropertyMap, vlanNotifyMsg.VlanId)
	}
}

func (server *OSPFServer) updatePortPropertyMap(vlanNotifyMsg asicdCommonDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdCommonDefs.NOTIFY_VLAN_CREATE { // Create Vlan
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

func (server *OSPFServer) BuildPortPropertyMap() {
	currMarker := asicdServices.Int(asicdCommonDefs.MIN_SYS_PORTS)
	if server.asicdClient.IsConnected {
		server.logger.Info("Calling asicd for getting port state")
		count := 10
		for {
			bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPortState(asicdServices.Int(currMarker), asicdServices.Int(count))
			if bulkInfo == nil {
				break
			}
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
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
				break
			}
		}
	}
	currMarker = asicdServices.Int(asicdCommonDefs.MIN_SYS_PORTS)
	if server.asicdClient.IsConnected {
		server.logger.Info("Calling asicd for getting the Port Config")
		count := 10
		for {
			bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPort(asicdServices.Int(currMarker), asicdServices.Int(count))
			if bulkInfo == nil {
				break
			}
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				portNum := bulkInfo.PortList[i].PortNum
				ent := server.portPropertyMap[portNum]
				ent.Mtu = bulkInfo.PortList[i].Mtu
				server.portPropertyMap[portNum] = ent
			}
			if more == false {
				break
			}
		}
	}
}

func (server *OSPFServer) getLinuxIntfName(ifId uint16, ifType uint8) (ifName string, err error) {
	if ifType == commonDefs.IfTypeVlan { // Vlan
		ifName = server.vlanPropertyMap[ifId].Name
	} else if ifType == commonDefs.IfTypePort { // PHY
		ifName = server.portPropertyMap[int32(ifId)].Name
	} else {
		ifName = ""
		err = errors.New("Invalid Interface Type")
	}
	return ifName, err
}

func getMacAddrIntfName(ifName string) (macAddr net.HardwareAddr, err error) {

	ifi, err := net.InterfaceByName(ifName)
	if err != nil {
		return macAddr, err
	}
	macAddr = ifi.HardwareAddr
	return macAddr, nil
}
