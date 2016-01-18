package rpc

import (
    "ospfd"
    "fmt"
    "l3/ospf/config"
    "errors"
//    "l3/ospf/server"
//    "log/syslog"
//    "net"
)

func (h *OSPFHandler) SendOspfGlobal(ospfGlobalConf *ospfd.OspfGlobalConf) bool {
    gConf := config.GlobalConf {
        RouterId:                   config.RouterId(ospfGlobalConf.RouterId),
        AdminStat:                  config.Status(ospfGlobalConf.AdminStat),
        ASBdrRtrStatus:             ospfGlobalConf.ASBdrRtrStatus,
        TOSSupport:                 ospfGlobalConf.TOSSupport,
        ExtLsdbLimit:               ospfGlobalConf.ExtLsdbLimit,
        MulticastExtensions:        ospfGlobalConf.MulticastExtensions,
        ExitOverflowInterval:       config.PositiveInteger(ospfGlobalConf.ExitOverflowInterval),
        RFC1583Compatibility:       ospfGlobalConf.RFC1583Compatibility,
        ReferenceBandwidth:         ospfGlobalConf.ReferenceBandwidth,
        RestartSupport:             config.RestartSupport(ospfGlobalConf.RestartSupport),
        RestartInterval:            ospfGlobalConf.RestartInterval,
        RestartStrictLsaChecking:   ospfGlobalConf.RestartStrictLsaChecking,
        StubRouterAdvertisement:    config.AdvertiseAction(ospfGlobalConf.StubRouterAdvertisement),
    }
    h.server.GlobalConfigCh <- gConf
    return true
}

func (h *OSPFHandler) SendOspfIfConf(ospfIfConf *ospfd.OspfIfConf) bool {
    ifConf := config.InterfaceConf {
        IfIpAddress:                config.IpAddress(ospfIfConf.IfIpAddress),
        AddressLessIf:              config.InterfaceIndexOrZero(ospfIfConf.AddressLessIf),
        IfAreaId:                   config.AreaId(ospfIfConf.IfAreaId),
        IfType:                     config.IfType(ospfIfConf.IfType),
        IfAdminStat:                config.Status(ospfIfConf.IfAdminStat),
        IfRtrPriority:              config.DesignatedRouterPriority(ospfIfConf.IfRtrPriority),
        IfTransitDelay:             config.UpToMaxAge(ospfIfConf.IfTransitDelay),
        IfRetransInterval:          config.UpToMaxAge(ospfIfConf.IfRetransInterval),
        IfHelloInterval:            config.HelloRange(ospfIfConf.IfHelloInterval),
        IfRtrDeadInterval:          config.PositiveInteger(ospfIfConf.IfRtrDeadInterval),
        IfPollInterval:             config.PositiveInteger(ospfIfConf.IfPollInterval),
        IfAuthKey:                  ospfIfConf.IfAuthKey,
        IfMulticastForwarding:      config.MulticastForwarding(ospfIfConf.IfMulticastForwarding),
        IfDemand:                   ospfIfConf.IfDemand,
        IfAuthType:                 config.AuthType(ospfIfConf.IfAuthType),
    }

    h.server.IntfConfigCh <- ifConf
    return true
}

func (h *OSPFHandler) CreateOspfGlobalConf(ospfGlobalConf *ospfd.OspfGlobalConf) (bool, error) {
    if ospfGlobalConf == nil {
        err := errors.New("Invalid Global Configuration Argument")
        return false, err
    }
    h.logger.Info(fmt.Sprintln("Create global config attrs:", ospfGlobalConf))
    return h.SendOspfGlobal(ospfGlobalConf), nil
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
    if ospfIfConf == nil {
        err := errors.New("Invalid Interface Configuration Argument")
        return false, err
    }
    h.logger.Info(fmt.Sprintln("Create interface config attrs:", ospfIfConf))
    return h.SendOspfIfConf(ospfIfConf), nil
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

