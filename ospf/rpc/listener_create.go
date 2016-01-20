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

func (h *OSPFHandler) SendOspfGlobal(ospfGlobalConf *ospfd.OspfGlobalConfig) bool {
    gConf := config.GlobalConf {
        RouterId:                   config.RouterId(ospfGlobalConf.RouterIdKey),
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

func (h *OSPFHandler) SendOspfIfConf(ospfIfConf *ospfd.OspfIfEntryConfig) bool {
    ifConf := config.InterfaceConf {
        IfIpAddress:                config.IpAddress(ospfIfConf.IfIpAddressKey),
        AddressLessIf:              config.InterfaceIndexOrZero(ospfIfConf.AddressLessIfKey),
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

func (h *OSPFHandler) SendOspfAreaConf(ospfAreaConf *ospfd.OspfAreaEntryConfig) bool {
    areaConf := config.AreaConf {
        AreaId:                                 config.AreaId(ospfAreaConf.AreaIdKey),
        AuthType:                               config.AuthType(ospfAreaConf.AuthType),
        ImportAsExtern:                         config.ImportAsExtern(ospfAreaConf.ImportAsExtern),
        AreaSummary:                            config.AreaSummary(ospfAreaConf.AreaSummary),
        AreaNssaTranslatorRole:                 config.NssaTranslatorRole(ospfAreaConf.AreaNssaTranslatorRole),
        AreaNssaTranslatorStabilityInterval:    config.PositiveInteger(ospfAreaConf.AreaNssaTranslatorStabilityInterval),
    }

    h.server.AreaConfigCh <- areaConf
    return true

}

func (h *OSPFHandler) CreateOspfGlobalConfig(ospfGlobalConf *ospfd.OspfGlobalConfig) (bool, error) {
    if ospfGlobalConf == nil {
        err := errors.New("Invalid Global Configuration")
        return false, err
    }
    h.logger.Info(fmt.Sprintln("Create global config attrs:", ospfGlobalConf))
    return h.SendOspfGlobal(ospfGlobalConf), nil
}

func (h *OSPFHandler) CreateOspfAreaEntryConfig(ospfAreaConf *ospfd.OspfAreaEntryConfig) (bool, error) {
    if ospfAreaConf == nil {
        err := errors.New("Invalid Area Configuration")
        return false, err
    }
    h.logger.Info(fmt.Sprintln("Create Area config attrs:", ospfAreaConf))
    return h.SendOspfAreaConf(ospfAreaConf), nil
}

func (h *OSPFHandler) CreateOspfStubAreaEntryConfig(ospfStubAreaConf *ospfd.OspfStubAreaEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Stub Area config attrs:", ospfStubAreaConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfAreaRangeEntryConfig(ospfAreaRangeConf *ospfd.OspfAreaRangeEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create address range config attrs:", ospfAreaRangeConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfHostEntryConfig(ospfHostConf *ospfd.OspfHostEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create host config attrs:", ospfHostConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfIfEntryConfig(ospfIfConf *ospfd.OspfIfEntryConfig) (bool, error) {
    if ospfIfConf == nil {
        err := errors.New("Invalid Interface Configuration")
        return false, err
    }
    h.logger.Info(fmt.Sprintln("Create interface config attrs:", ospfIfConf))
    return h.SendOspfIfConf(ospfIfConf), nil
}

func (h *OSPFHandler) CreateOspfIfMetricEntryConfig(ospfIfMetricConf *ospfd.OspfIfMetricEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create interface metric config attrs:", ospfIfMetricConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfVirtIfEntryConfig(ospfVirtIfConf *ospfd.OspfVirtIfEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create virtual interface config attrs:", ospfVirtIfConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfNbrEntryConfig(ospfNbrConf *ospfd.OspfNbrEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Neighbor Config attrs:", ospfNbrConf))
    return true, nil
}

func (h *OSPFHandler) CreateOspfAreaAggregateEntryConfig(ospfAreaAggregateConf *ospfd.OspfAreaAggregateEntryConfig) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create Area Agggregate Config attrs:", ospfAreaAggregateConf))
    return true, nil
}

