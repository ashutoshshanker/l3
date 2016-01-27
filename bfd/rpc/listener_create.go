package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	//"l3/bfd/config"
	//"l3/bfd/server"
	//"log/syslog"
	//"net"
)

/*
func (h *OSPFHandler) SendOspfGlobal(ospfGlobalConf *ospfd.OspfGlobalConfig) bool {
	gConf := config.GlobalConf{
		RouterId:                 config.RouterId(ospfGlobalConf.RouterIdKey),
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

func (h *OSPFHandler) SendOspfIfConf(ospfIfConf *ospfd.OspfIfEntryConfig) bool {
	ifConf := config.InterfaceConf{
		IfIpAddress:           config.IpAddress(ospfIfConf.IfIpAddressKey),
		AddressLessIf:         config.InterfaceIndexOrZero(ospfIfConf.AddressLessIfKey),
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
*/

func (h *BFDHandler) CreateBfdGlobalConfig(bfdGlobalConf *bfdd.BfdGlobalConfig) (bool, error) {
	if bfdGlobalConf == nil {
		err := errors.New("Invalid Global Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bfdGlobalConf))
	return true, nil
}

func (h *BFDHandler) CreateBfdIntfConfig(bfdIfConf *bfdd.BfdIntfConfig) (bool, error) {
	if bfdIfConf == nil {
		err := errors.New("Invalid Interface Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create interface config attrs:", bfdIfConf))
	return true, nil
}
