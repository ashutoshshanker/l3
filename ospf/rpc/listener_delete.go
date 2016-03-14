package rpc

import (
	"fmt"
	"ospfd"
	//    "l3/ospf/config"
	//    "l3/ospf/server"
	//    "utils/logging"
	//    "net"
)

func (h *OSPFHandler) DeleteOspfGlobalConfig(ospfGlobalConf *ospfd.OspfGlobalConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", ospfGlobalConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfAreaEntryConfig(ospfAreaConf *ospfd.OspfAreaEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Area Config attrs:", ospfAreaConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfStubAreaEntryConfig(ospfStubAreaConf *ospfd.OspfStubAreaEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Stub Area Config attrs:", ospfStubAreaConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfAreaRangeEntryConfig(ospfAreaRangeConf *ospfd.OspfAreaRangeEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete address range config attrs:", ospfAreaRangeConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfHostEntryConfig(ospfHostConf *ospfd.OspfHostEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete host config attrs:", ospfHostConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfIfEntryConfig(ospfIfConf *ospfd.OspfIfEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete interface config attrs:", ospfIfConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfIfMetricEntryConfig(ospfIfMetricConf *ospfd.OspfIfMetricEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete interface metric config attrs:", ospfIfMetricConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfVirtIfEntryConfig(ospfVirtIfConf *ospfd.OspfVirtIfEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete virtual interface config attrs:", ospfVirtIfConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfNbrEntryConfig(ospfNbrConf *ospfd.OspfNbrEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Neighbor config attrs:", ospfNbrConf))
	return true, nil
}

func (h *OSPFHandler) DeleteOspfAreaAggregateEntryConfig(ospfAreaAggregateConf *ospfd.OspfAreaAggregateEntryConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete Area Agggregate config attrs:", ospfAreaAggregateConf))
	return true, nil
}
