package server

import (
    "fmt"
    "time"
    "l3/ospf/config"
    //"l3/ospf/rpc"
//    "l3/rib/ribdCommonDefs"
    "asicd/pluginManager/pluginCommon"
    "net"
    "github.com/google/gopacket/pcap"
)

type IntfConfKey struct {
    IPAddr          config.IpAddress
    IntfIdx         config.InterfaceIndexOrZero
}

type IntfConf struct {
    IfAreaId                []byte
    IfType                  config.IfType
    IfAdminStat             config.Status
    IfRtrPriority           uint8
    IfTransitDelay          config.UpToMaxAge
    IfRetransInterval       config.UpToMaxAge
    IfHelloInterval         uint16
    IfRtrDeadInterval       uint32
    IfPollInterval          config.PositiveInteger
    IfAuthKey               []byte
    IfMulticastForwarding   config.MulticastForwarding
    IfDemand                bool
    IfAuthType              uint16
    HelloPktSendCh          chan bool
    HelloPktSendStatusCh    chan string
    HelloPktRecvCh          chan bool
    SendPcapHdl             *pcap.Handle
    RecvPcapHdl             *pcap.Handle
    SendHelloPktTimer       *time.Timer
    IfName                  string
    IfIpAddr                  net.IP
    IfMacAddr               net.HardwareAddr
    IfNetmask               []byte
}

func (server *OSPFServer)initDefaultIntfConf(key IntfConfKey, ipIntfProp IPIntfProperty) {
    ent, exist := server.IntfConfMap[key]
    if !exist {
        areaId := convertAreaOrRouterId("0.0.0.0")
        if areaId == nil {
            return
        }
        ent.IfAreaId = areaId
        ent.IfType = config.Broadcast
        ent.IfAdminStat = config.Enabled
        ent.IfRtrPriority = uint8(config.DesignatedRouterPriority(1))
        ent.IfTransitDelay = config.UpToMaxAge(1)
        ent.IfRetransInterval = config.UpToMaxAge(5)
        ent.IfHelloInterval = uint16(config.HelloRange(10))
        ent.IfRtrDeadInterval = uint32(config.PositiveInteger(40))
        ent.IfPollInterval = config.PositiveInteger(120)
        authKey := convertAuthKey("0.0.0.0.0.0.0.0")
        if authKey == nil {
            return
        }
        ent.IfAuthKey = authKey
        ent.IfMulticastForwarding = config.Blocked
        ent.IfDemand = false
        ent.IfAuthType = uint16(config.NoAuth)
        ent.HelloPktSendCh = make(chan bool)
        ent.HelloPktSendStatusCh = make(chan string)
        ent.HelloPktRecvCh = make(chan bool)
        ent.IfNetmask = ipIntfProp.NetMask
        ent.IfName = ipIntfProp.IfName
        ent.IfIpAddr = ipIntfProp.IpAddr
        ent.IfMacAddr = ipIntfProp.MacAddr
        sendHdl, err := pcap.OpenLive(ent.IfName, snapshot_len, promiscuous, timeout_pcap)
        if sendHdl == nil {
            server.logger.Err(fmt.Sprintln("SendHdl: No device found.", ent.IfName, err))
            return
        }
        ent.SendPcapHdl = sendHdl
        recvHdl, err := pcap.OpenLive(ent.IfName, snapshot_len, promiscuous, timeout_pcap)
        if recvHdl == nil {
            server.logger.Err(fmt.Sprintln("RecvHdl: No device found.", ent.IfName, err))
            return
        }
        ent.RecvPcapHdl = recvHdl
        server.IntfConfMap[key] = ent
        server.logger.Info(fmt.Sprintln("Intf Conf initialized", key))
    } else {
        server.logger.Info(fmt.Sprintln("Intf Conf is not initialized", key))
    }
}

func (server *OSPFServer)createIPIntfConfMap(msg pluginCommon.IPv4IntfNotifyMsg) {
    ip, ipNet, err := net.ParseCIDR(msg.IpAddr)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to parse IP address", msg.IpAddr))
        return
    }
    ifName, err := server.getLinuxIntfName(msg.IfId, msg.IfType)
    if err != nil {
        server.logger.Err("No Such Interface exists")
        return
    }
    server.logger.Info(fmt.Sprintln("create IPIntfConfMap for ", msg))

    // Set ifIdx = 0 for time being --- Need to be revisited
    intfConfKey := IntfConfKey {
        IPAddr:     config.IpAddress(ip.String()),
        //IntfIdx:    int(msg.IfIdx),
        IntfIdx:    config.InterfaceIndexOrZero(0),
    }
    macAddr, err := getMacAddrIntfName(ifName)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to get MacAddress of Interface exists", ifName))
        return
    }
    ipIntfProp := IPIntfProperty {
        IfName:     ifName,
        IpAddr:     ip,
        MacAddr:    macAddr,
        NetMask:    ipNet.Mask,
    }
    server.initDefaultIntfConf(intfConfKey, ipIntfProp)
}

func (server *OSPFServer)deleteIPIntfConfMap(msg pluginCommon.IPv4IntfNotifyMsg) {
    ip, _, err := net.ParseCIDR(msg.IpAddr)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to parse IP address", msg.IpAddr))
        return
    }

    server.logger.Info(fmt.Sprintln("delete IPIntfConfMap for ", msg))

    // Set ifIdx = 0 for time being --- Need to be revisited
    intfConfKey := IntfConfKey {
        IPAddr:     config.IpAddress(ip.String()),
        //IntfIdx:    int(msg.IfIdx),
        IntfIdx:    config.InterfaceIndexOrZero(0),
    }
    server.logger.Info(fmt.Sprintln("1:delete IPIntfConfMap for ", intfConfKey))
    delete(server.IntfConfMap, intfConfKey)
}

func (server *OSPFServer)updateIPIntfConfMap(ifConf config.InterfaceConf) {
    intfConfKey := IntfConfKey {
        IPAddr:             ifConf.IfIpAddress,
        IntfIdx:            config.InterfaceIndexOrZero(ifConf.AddressLessIf),
    }

    ent, exist := server.IntfConfMap[intfConfKey]
    //  we can update only when we already have entry
    if exist {
        areaId := convertAreaOrRouterId(string(ifConf.IfAreaId))
        if areaId == nil {
            server.logger.Err("Invalid areaId")
            return
        }
        ent.IfAreaId = areaId
        ent.IfType = ifConf.IfType
        ent.IfAdminStat = ifConf.IfAdminStat
        ent.IfRtrPriority = uint8(ifConf.IfRtrPriority)
        ent.IfTransitDelay = ifConf.IfTransitDelay
        ent.IfRetransInterval = ifConf.IfRetransInterval
        ent.IfHelloInterval = uint16(ifConf.IfHelloInterval)
        ent.IfRtrDeadInterval = uint32(ifConf.IfRtrDeadInterval)
        ent.IfPollInterval = ifConf.IfPollInterval
        authKey := convertAuthKey(string(ifConf.IfAuthKey))
        if authKey == nil {
            server.logger.Err("Invalid authKey")
            return
        }
        ent.IfAuthKey = authKey
        ent.IfMulticastForwarding = ifConf.IfMulticastForwarding
        ent.IfDemand = ifConf.IfDemand
        ent.IfAuthType = uint16(ifConf.IfAuthType)
        server.IntfConfMap[intfConfKey] = ent
        server.logger.Info(fmt.Sprintln("1:Update IPIntfConfMap for ", intfConfKey))
    }
}

func (server *OSPFServer)processIntfConfig(ifConf config.InterfaceConf) {
    intfConfKey := IntfConfKey {
        IPAddr:             ifConf.IfIpAddress,
        IntfIdx:            config.InterfaceIndexOrZero(ifConf.AddressLessIf),
    }
    ent, exist := server.IntfConfMap[intfConfKey]
    if !exist {
        server.logger.Err("No such L3 interface exists")
        return
    }
    if ent.IfAdminStat == config.Enabled &&
        server.ospfGlobalConf.AdminStat == config.Enabled {
        server.StopSendRecvHelloPkts(intfConfKey)
    }

    server.updateIPIntfConfMap(ifConf)

    server.logger.Info(fmt.Sprintln("InterfaceConf:", server.IntfConfMap))
    ent, _ = server.IntfConfMap[intfConfKey]
    if ent.IfAdminStat == config.Enabled &&
        server.ospfGlobalConf.AdminStat == config.Enabled {
        server.StartSendRecvHelloPkts(intfConfKey)
    }
}


