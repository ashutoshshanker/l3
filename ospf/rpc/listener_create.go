package rpc

import (
    "ospfd"
    "fmt"
//    "l3/ospf/config"
//    "l3/ospf/server"
//    "log/syslog"
//    "net"
)

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

