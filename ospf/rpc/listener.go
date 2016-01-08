package rpc

import (
    "ospfd"
    "fmt"
//    "l3/ospf/config"
    "l3/ospf/server"
    "log/syslog"
//    "net"
)

type OSPFHandler struct {
    server        *server.OSPFServer
    logger        *syslog.Writer
}

func NewOSPFHandler(server *server.OSPFServer, logger *syslog.Writer) *OSPFHandler {
    h := new(OSPFHandler)
    h.server = server
    h.logger = logger
    return h
}

/*
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
*/

func (h *OSPFHandler) CreateOspfGlobalConf(ospfGlobalConf *ospfd.OspfGlobalConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create global config attrs:", ospfGlobalConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfAreaConf(ospfAreaConf *ospfd.OspfAreaConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Area config attrs:", ospfAreaConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfStubAreaConf(ospfStubAreaConf *ospfd.OspfStubAreaConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Stub Area config attrs:", ospfStubAreaConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfAreaRangeConf(ospfAreaRangeConf *ospfd.OspfAreaRangeConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create address range config attrs:", ospfAreaRangeConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfHostConf(ospfHostConf *ospfd.OspfHostConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create host config attrs:", ospfHostConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfIfConf(ospfIfConf *ospfd.OspfIfConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create interface config attrs:", ospfIfConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfIfMetricConf(ospfIfMetricConf *ospfd.OspfIfMetricConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create interface metric config attrs:", ospfIfMetricConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfVirtIfConf(ospfVirtIfConf *ospfd.OspfVirtIfConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create virtual interface config attrs:", ospfVirtIfConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfNbrConf(ospfNbrConf *ospfd.OspfNbrConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Neighbor Config attrs:", ospfNbrConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfAreaAggregateConf(ospfAreaAggregateConf *ospfd.OspfAreaAggregateConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Area Agggregate Config attrs:", ospfAreaAggregateConf))
    return true, nil
}



func (h *OSPFHandler) DeleteOspfGlobalConf(ospfGlobalConf *ospfd.OspfGlobalConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete global config attrs:", ospfGlobalConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfAreaConf(ospfAreaConf *ospfd.OspfAreaConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete Area Config attrs:", ospfAreaConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfStubAreaConf(ospfStubAreaConf *ospfd.OspfStubAreaConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete Stub Area Config attrs:", ospfStubAreaConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfAreaRangeConf(ospfAreaRangeConf *ospfd.OspfAreaRangeConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete address range config attrs:", ospfAreaRangeConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfHostConf(ospfHostConf *ospfd.OspfHostConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete host config attrs:", ospfHostConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfIfConf(ospfIfConf *ospfd.OspfIfConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete interface config attrs:", ospfIfConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfIfMetricConf(ospfIfMetricConf *ospfd.OspfIfMetricConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete interface metric config attrs:", ospfIfMetricConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfVirtIfConf(ospfVirtIfConf *ospfd.OspfVirtIfConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete virtual interface config attrs:", ospfVirtIfConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfNbrConf(ospfNbrConf *ospfd.OspfNbrConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete Neighbor config attrs:", ospfNbrConf))
    return true, nil
}

func (h *OSPFHandler) DeleteOspfAreaAggregateConf(ospfAreaAggregateConf *ospfd.OspfAreaAggregateConf) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete Area Agggregate config attrs:", ospfAreaAggregateConf))
    return true, nil
}
