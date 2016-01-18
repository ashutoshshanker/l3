package server

import (
    "net"
    "utils/commonDefs"
    "asicd/asicdConstDefs"
    "asicd/pluginManager/pluginCommon"
    "errors"
)

type PortProperty struct {
    Name        string
    VlanName    string
    VlanId      uint16
    IpAddr      net.IP
}

type VlanProperty struct {
    Name        string
    UntagPorts  []int32
    IpAddr      net.IP
}

type IPIntfProperty struct {
    IfName      string
    IpAddr      net.IP
    MacAddr     net.HardwareAddr
    NetMask     []byte
}

func (server *OSPFServer) updateIpInVlanPropertyMap(msg pluginCommon.IPv4IntfNotifyMsg, msgType uint8) {
    if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
        ent := server.vlanPropertyMap[msg.IfId]
        ip, _, _:= net.ParseCIDR(msg.IpAddr)
        ent.IpAddr = ip
        server.vlanPropertyMap[msg.IfId] = ent
    } else { // Delete IP
        ent := server.vlanPropertyMap[msg.IfId]
        ent.IpAddr = nil
        server.vlanPropertyMap[msg.IfId] = ent
    }
}

func (server *OSPFServer) updateIpInPortPropertyMap(msg pluginCommon.IPv4IntfNotifyMsg, msgType uint8) {
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

func (server *OSPFServer) updateVlanPropertyMap(vlanNotifyMsg pluginCommon.VlanNotifyMsg, msgType uint8) {
    if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
        ent := server.vlanPropertyMap[vlanNotifyMsg.VlanId]
        ent.Name = vlanNotifyMsg.VlanName
        ent.UntagPorts = vlanNotifyMsg.UntagPorts
        server.vlanPropertyMap[vlanNotifyMsg.VlanId] = ent
    } else { // Delete Vlan
        delete(server.vlanPropertyMap, vlanNotifyMsg.VlanId)
    }
}

func (server *OSPFServer) updatePortPropertyMap(vlanNotifyMsg pluginCommon.VlanNotifyMsg, msgType uint8) {
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

func (server *OSPFServer) BuildPortPropertyMap() {
    currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
    if server.asicdClient.IsConnected {
        server.logger.Info("Calling asicd for port property")
        count := 10
        for {
            bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPortConfig(int64(currMarker), int64(count))
            if bulkInfo == nil {
                return
            }
            objCount := int(bulkInfo.ObjCount)
            more := bool(bulkInfo.More)
            currMarker = bulkInfo.NextMarker
            for i := 0; i < objCount; i++ {
                portNum := bulkInfo.PortConfigList[i].PortNum
                ent := server.portPropertyMap[portNum]
                ent.Name = bulkInfo.PortConfigList[i].Name
                ent.VlanId = 0
                ent.VlanName = ""
                server.portPropertyMap[portNum] = ent
            }
            if more == false {
                return
            }
        }
    }
}


func (server *OSPFServer)getLinuxIntfName(ifId uint16, ifType uint8) (ifName string, err error) {
    if ifType == commonDefs.L2RefTypeVlan { // Vlan
        ifName = server.vlanPropertyMap[ifId].Name
    } else if ifType == commonDefs.L2RefTypePort { // PHY
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



