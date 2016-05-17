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

package server

import (
	"asicd/asicdCommonDefs"
	"asicdInt"
	"asicdServices"
	"errors"
	"fmt"
	"net"
	"utils/commonDefs"
)

type PortProperty struct {
	Name string
	Mtu  int32
}

type VlanProperty struct {
	Name       string
	UntagPorts []int32
}

type IPv4IntfNotifyMsg struct {
	IpAddr string
	IfId   uint16
	IfType uint8
}

type IpProperty struct {
	IfId   uint16
	IfType uint8
	IpAddr string // CIDR Notation
}

type IPIntfProperty struct {
	IfName  string
	IpAddr  net.IP
	MacAddr net.HardwareAddr
	NetMask []byte
	Mtu     int32
}

func (server *OSPFServer) computeMinMTU(msg IPv4IntfNotifyMsg) int32 {
	var minMtu int32 = 10000                 //in bytes
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
		ent.IpAddr = msg.IpAddr
		server.ipPropertyMap[ip] = ent
	} else { // Delete IP
		delete(server.ipPropertyMap, ip)
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

func (server *OSPFServer) BuildOspfInfra() {
	server.constructPortInfra()
	server.constructVlanInfra()
	server.constructL3Infra()
}

func (server *OSPFServer) constructPortInfra() {
	server.getBulkPortState()
	server.getBulkPortConfig()
}

func (server *OSPFServer) constructVlanInfra() {
	curMark := 0
	server.logger.Info("Calling Asicd for getting Vlan Property")
	count := 100
	for {
		bulkVlanInfo, _ := server.asicdClient.ClientHdl.GetBulkVlan(asicdInt.Int(curMark), asicdInt.Int(count))
		// Get bulk on vlan state can re-use curMark and count used
		// by get bulk vlan, as there is a 1:1 mapping in terms of cfg/state objs
		bulkVlanStateInfo, _ := server.asicdClient.ClientHdl.GetBulkVlanState(asicdServices.Int(curMark), asicdServices.Int(count))
		if bulkVlanStateInfo == nil &&
			bulkVlanInfo == nil {
			break
		}
		objCnt := int(bulkVlanInfo.Count)
		more := bool(bulkVlanInfo.More)
		curMark = int(bulkVlanInfo.EndIdx)
		for i := 0; i < objCnt; i++ {
			vlanId := uint16(bulkVlanInfo.VlanList[i].VlanId)
			ent := server.vlanPropertyMap[vlanId]
			ent.UntagPorts = bulkVlanInfo.VlanList[i].UntagIfIndexList
			ent.Name = bulkVlanStateInfo.VlanStateList[i].VlanName
			server.vlanPropertyMap[vlanId] = ent
		}
		if more == false {
			break
		}
	}

}

func (server *OSPFServer) constructL3Infra() {
	curMark := 0
	server.logger.Info("Calling Asicd for getting L3 Interfaces")
	count := 100
	for {
		bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkIPv4IntfState(asicdServices.Int(curMark), asicdServices.Int(count))
		if bulkInfo == nil {
			break
		}

		objCnt := int(bulkInfo.Count)
		more := bool(bulkInfo.More)
		curMark = int(bulkInfo.EndIdx)
		for i := 0; i < objCnt; i++ {
			ipAddr, _, _ := net.ParseCIDR(bulkInfo.IPv4IntfStateList[i].IpAddr)
			ifIdx := bulkInfo.IPv4IntfStateList[i].IfIndex
			ifType := uint8(asicdCommonDefs.GetIntfTypeFromIfIndex(ifIdx))
			ifId := uint16(asicdCommonDefs.GetIntfIdFromIfIndex(ifIdx))
			ip := convertAreaOrRouterIdUint32(ipAddr.String())
			ent := server.ipPropertyMap[ip]
			ent.IfId = ifId
			ent.IfType = ifType
			ent.IpAddr = bulkInfo.IPv4IntfStateList[i].IpAddr
			var ipv4IntfMsg IPv4IntfNotifyMsg
			ipv4IntfMsg.IpAddr = ent.IpAddr
			ipv4IntfMsg.IfType = ifType
			ipv4IntfMsg.IfId = ifId
			mtu := server.computeMinMTU(ipv4IntfMsg)
			server.createIPIntfConfMap(ipv4IntfMsg, mtu, ifIdx, broadcast)
			server.ipPropertyMap[ip] = ent
		}
		if more == false {
			break
		}
	}
}

func (server *OSPFServer) getBulkPortState() {
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
				server.portPropertyMap[portNum] = ent
			}
			if more == false {
				break
			}
		}
	}
}

func (server *OSPFServer) getBulkPortConfig() {
	currMarker := asicdServices.Int(asicdCommonDefs.MIN_SYS_PORTS)
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

func (server *OSPFServer) UpdateVlanInfra(msg asicdCommonDefs.VlanNotifyMsg, msgType uint8) {
	server.updateVlanPropertyMap(msg, msgType)
}

func (server *OSPFServer) UpdateIPv4Infra(msg asicdCommonDefs.IPv4IntfNotifyMsg, msgType uint8) {
	var ipv4IntfMsg IPv4IntfNotifyMsg
	ipv4IntfMsg.IpAddr = msg.IpAddr
	ipv4IntfMsg.IfType = uint8(asicdCommonDefs.GetIntfTypeFromIfIndex(msg.IfIndex))
	ipv4IntfMsg.IfId = uint16(asicdCommonDefs.GetIntfIdFromIfIndex(msg.IfIndex))
	if msgType == asicdCommonDefs.NOTIFY_IPV4INTF_CREATE {
		server.logger.Info(fmt.Sprintln("Receive IPV4INTF_CREATE", ipv4IntfMsg))
		mtu := server.computeMinMTU(ipv4IntfMsg)
		// We need more information from Asicd about numbered/unnumbered p2p
		// or broadcast
		//Start
		/*
				ip, _, _ := net.ParseCIDR(ipv4IntfMsg.IpAddr)
				if ip.String() == "40.0.1.10" {
					server.createIPIntfConfMap(ipv4IntfMsg, mtu, msg.IfIndex, unnumberedP2P)
				} else if ip.String() == "40.0.1.15" {
					server.createIPIntfConfMap(ipv4IntfMsg, mtu, msg.IfIndex, numberedP2P)
				} else {
					server.createIPIntfConfMap(ipv4IntfMsg, mtu, msg.IfIndex, broadcast)
				}
			//End
		*/

		server.createIPIntfConfMap(ipv4IntfMsg, mtu, msg.IfIndex, broadcast)
		server.updateIpPropertyMap(ipv4IntfMsg, msgType)
	} else {
		server.logger.Info(fmt.Sprintln("Receive IPV4INTF_DELETE", ipv4IntfMsg))
		server.deleteIPIntfConfMap(ipv4IntfMsg, msg.IfIndex)
		server.updateIpPropertyMap(ipv4IntfMsg, msgType)
	}

}
