package rpc

import (
    "ospfd"
    "fmt"
    "l3/ospf/config"
    "l3/ospf/server"
    "log/syslog"
    "net"
)

type PeerConfigCommands struct {
    IP      net.IP
    Command int
}

type OSPFHandler struct {
    //PeerCommandCh chan PeerConfigCommands
    server        *server.OSPFServer
    logger        *syslog.Writer
}

func NewOSPFHandler(server *server.OSPFServer, logger *syslog.Writer) *OSPFHandler {
    h := new(OSPFHandler)
    //h.PeerCommandCh = make(chan PeerConfigCommands)
    h.server = server
    h.logger = logger
    return h
}

func (h *OSPFHandler) CreateOSPFGlobalConf(ospfGlobalConf *ospfd.OSPFGlobalConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create global config attrs:", ospfGlobalConf))

    gConf := config.GlobalConfig{
            RouterId: uint32(ospfGlobalConf.RouterId),
            RFC1583Compatibility: ospfGlobalConf.RFC1583Compatibility,
    }
    h.server.GlobalConfigCh <- gConf
    return true, nil
}

func (h *OSPFHandler) CreateOSPFAreaConf(ospfAreaConf *ospfd.OSPFAreaConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create area config attrs:", ospfAreaConf))
    ospfAddressRange := make([]config.AddressRange, len(ospfAreaConf.AddressRange))
    for i := 0; i < len(ospfAreaConf.AddressRange); i++ {
        ospfAddressRange[i].IP = net.ParseIP(ospfAreaConf.AddressRange[i].IP)
        ospfAddressRange[i].Mask = net.ParseIP(ospfAreaConf.AddressRange[i].Mask)
        ospfAddressRange[i].Status = ospfAreaConf.AddressRange[i].Status
    }
    areaConf := config.AreaConfig{
        AreaId:                     uint32(ospfAreaConf.AreaId),
        AddressRanges:              ospfAddressRange,
        ExternalRoutingCapability:  ospfAreaConf.ExternalRoutingCapability,
        StubDefaultCost:            uint32(ospfAreaConf.StubDefaultCost),
    }
    h.server.AreaConfigCh <- areaConf
    return true, nil
}
