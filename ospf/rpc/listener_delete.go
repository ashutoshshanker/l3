package rpc

import (
    "ospfd"
    "fmt"
//    "l3/ospf/config"
//    "l3/ospf/server"
//    "log/syslog"
//    "net"
)

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


