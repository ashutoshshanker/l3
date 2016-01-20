package rpc

import (
    "ospfd"
    "fmt"
//    "l3/ospf/config"
//    "l3/ospf/server"
//    "log/syslog"
//    "net"
)

func (h *OSPFHandler) UpdateOspfGlobalConfig(origConf *ospfd.OspfGlobalConfig, newConf *ospfd.OspfGlobalConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original global config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New global config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfAreaEntryConfig(origConf *ospfd.OspfAreaEntryConfig, newConf *ospfd.OspfAreaEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original area config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New area config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfStubAreaEntryConfig(origConf *ospfd.OspfStubAreaEntryConfig, newConf *ospfd.OspfStubAreaEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original stub area config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New stub area config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfAreaRangeEntryConfig(origConf *ospfd.OspfAreaRangeEntryConfig, newConf *ospfd.OspfAreaRangeEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original address range config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New address range config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfHostEntryConfig(origConf *ospfd.OspfHostEntryConfig, newConf *ospfd.OspfHostEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original host config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New host config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfIfEntryConfig(origConf *ospfd.OspfIfEntryConfig, newConf *ospfd.OspfIfEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original interface config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New interface config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfIfMetricEntryConfig(origConf *ospfd.OspfIfMetricEntryConfig, newConf *ospfd.OspfIfMetricEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original interface metric config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New interface metric config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfVirtIfEntryConfig(origConf *ospfd.OspfVirtIfEntryConfig, newConf *ospfd.OspfVirtIfEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original virtual interface config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New virtual interface config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfNbrEntryConfig(origConf *ospfd.OspfNbrEntryConfig, newConf *ospfd.OspfNbrEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original neighbor config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New neighbor config attrs:", newConf))
    return true, nil
}

func (h *OSPFHandler) UpdateOspfAreaAggregateEntryConfig(origConf *ospfd.OspfAreaAggregateEntryConfig, newConf *ospfd.OspfAreaAggregateEntryConfig, attrset []bool) (bool, error) {
    h.logger.Info(fmt.Sprintln("Original Area Aggregate config attrs:", origConf))
    h.logger.Info(fmt.Sprintln("New Area Aggregate config attrs:", newConf))
    return true, nil
}


