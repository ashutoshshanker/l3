package rpc

import (
	"errors"
	"fmt"
	"l3/ospf/config"
	"ospfd"
	//    "l3/ospf/server"
	//    "utils/logging"
	//    "net"
)

func (h *OSPFHandler) SendOspfGlobal(ospfGlobalConf *ospfd.OspfGlobal) bool {
	gConf := config.GlobalConf{
		RouterId:                 config.RouterId(ospfGlobalConf.RouterId),
		AdminStat:                config.Status(ospfGlobalConf.AdminStat),
		ASBdrRtrStatus:           ospfGlobalConf.ASBdrRtrStatus,
		TOSSupport:               ospfGlobalConf.TOSSupport,
		ExtLsdbLimit:             ospfGlobalConf.ExtLsdbLimit,
		MulticastExtensions:      ospfGlobalConf.MulticastExtensions,
		ExitOverflowInterval:     config.PositiveInteger(ospfGlobalConf.ExitOverflowInterval),
		RFC1583Compatibility:     ospfGlobalConf.RFC1583Compatibility,
		ReferenceBandwidth:       ospfGlobalConf.ReferenceBandwidth,
		RestartSupport:           config.RestartSupport(ospfGlobalConf.RestartSupport),
		RestartInterval:          ospfGlobalConf.RestartInterval,
		RestartStrictLsaChecking: ospfGlobalConf.RestartStrictLsaChecking,
		StubRouterAdvertisement:  config.AdvertiseAction(ospfGlobalConf.StubRouterAdvertisement),
	}
	h.server.GlobalConfigCh <- gConf
	return true
}

func (h *OSPFHandler) SendOspfIfConf(ospfIfConf *ospfd.OspfIfEntry) bool {
	ifConf := config.InterfaceConf{
		IfIpAddress:           config.IpAddress(ospfIfConf.IfIpAddress),
		AddressLessIf:         config.InterfaceIndexOrZero(ospfIfConf.AddressLessIf),
		IfAreaId:              config.AreaId(ospfIfConf.IfAreaId),
		IfType:                config.IfType(ospfIfConf.IfType),
		IfAdminStat:           config.Status(ospfIfConf.IfAdminStat),
		IfRtrPriority:         config.DesignatedRouterPriority(ospfIfConf.IfRtrPriority),
		IfTransitDelay:        config.UpToMaxAge(ospfIfConf.IfTransitDelay),
		IfRetransInterval:     config.UpToMaxAge(ospfIfConf.IfRetransInterval),
		IfHelloInterval:       config.HelloRange(ospfIfConf.IfHelloInterval),
		IfRtrDeadInterval:     config.PositiveInteger(ospfIfConf.IfRtrDeadInterval),
		IfPollInterval:        config.PositiveInteger(ospfIfConf.IfPollInterval),
		IfAuthKey:             ospfIfConf.IfAuthKey,
		IfMulticastForwarding: config.MulticastForwarding(ospfIfConf.IfMulticastForwarding),
		IfDemand:              ospfIfConf.IfDemand,
		IfAuthType:            config.AuthType(ospfIfConf.IfAuthType),
	}

	h.server.IntfConfigCh <- ifConf
	return true
}

func (h *OSPFHandler) SendOspfAreaConf(ospfAreaConf *ospfd.OspfAreaEntry) bool {
	areaConf := config.AreaConf{
		AreaId:                              config.AreaId(ospfAreaConf.AreaId),
		AuthType:                            config.AuthType(ospfAreaConf.AuthType),
		ImportAsExtern:                      config.ImportAsExtern(ospfAreaConf.ImportAsExtern),
		AreaSummary:                         config.AreaSummary(ospfAreaConf.AreaSummary),
		AreaNssaTranslatorRole:              config.NssaTranslatorRole(ospfAreaConf.AreaNssaTranslatorRole),
		AreaNssaTranslatorStabilityInterval: config.PositiveInteger(ospfAreaConf.AreaNssaTranslatorStabilityInterval),
	}

	h.server.AreaConfigCh <- areaConf
	return true

}

func (h *OSPFHandler) CreateOspfGlobal(ospfGlobalConf *ospfd.OspfGlobal) (bool, error) {
	if ospfGlobalConf == nil {
		err := errors.New("Invalid Global Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create global config attrs:", ospfGlobalConf))
	return h.SendOspfGlobal(ospfGlobalConf), nil
}

func (h *OSPFHandler) CreateOspfAreaEntry(ospfAreaConf *ospfd.OspfAreaEntry) (bool, error) {
	if ospfAreaConf == nil {
		err := errors.New("Invalid Area Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create Area config attrs:", ospfAreaConf))
	return h.SendOspfAreaConf(ospfAreaConf), nil
}

func (h *OSPFHandler) CreateOspfStubAreaEntry(ospfStubAreaConf *ospfd.OspfStubAreaEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create Stub Area config attrs:", ospfStubAreaConf))
	return true, nil
}

func (h *OSPFHandler) CreateOspfAreaRangeEntry(ospfAreaRangeConf *ospfd.OspfAreaRangeEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create address range config attrs:", ospfAreaRangeConf))
	return true, nil
}

func (h *OSPFHandler) CreateOspfHostEntry(ospfHostConf *ospfd.OspfHostEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create host config attrs:", ospfHostConf))
	return true, nil
}

func (h *OSPFHandler) CreateOspfIfEntry(ospfIfConf *ospfd.OspfIfEntry) (bool, error) {
	if ospfIfConf == nil {
		err := errors.New("Invalid Interface Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create interface config attrs:", ospfIfConf))
	return h.SendOspfIfConf(ospfIfConf), nil
}

func (h *OSPFHandler) CreateOspfIfMetricEntry(ospfIfMetricConf *ospfd.OspfIfMetricEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create interface metric config attrs:", ospfIfMetricConf))
	return true, nil
}

func (h *OSPFHandler) CreateOspfVirtIfEntry(ospfVirtIfConf *ospfd.OspfVirtIfEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create virtual interface config attrs:", ospfVirtIfConf))
	return true, nil
}

func (h *OSPFHandler) CreateOspfNbrEntry(ospfNbrConf *ospfd.OspfNbrEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create Neighbor Config attrs:", ospfNbrConf))
	return true, nil
}

func (h *OSPFHandler) CreateOspfAreaAggregateEntry(ospfAreaAggregateConf *ospfd.OspfAreaAggregateEntry) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create Area Agggregate Config attrs:", ospfAreaAggregateConf))
	return true, nil
}
