package rpc

import (
	"fmt"
	"ospfd"
	//    "l3/ospf/config"
	//    "l3/ospf/server"
	//    "utils/logging"
	//    "net"
)

func (h *OSPFHandler) UpdateOspfGlobal(origConf *ospfd.OspfGlobal, newConf *ospfd.OspfGlobal, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original global config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New global config attrs:", newConf))
	return true, nil
}

func (h *OSPFHandler) UpdateOspfAreaEntry(origConf *ospfd.OspfAreaEntry, newConf *ospfd.OspfAreaEntry, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original area config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New area config attrs:", newConf))
	return true, nil
}

func (h *OSPFHandler) UpdateOspfStubAreaEntry(origConf *ospfd.OspfStubAreaEntry, newConf *ospfd.OspfStubAreaEntry, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original stub area config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New stub area config attrs:", newConf))
	return true, nil
}

func (h *OSPFHandler) UpdateOspfIfEntry(origConf *ospfd.OspfIfEntry, newConf *ospfd.OspfIfEntry, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original interface config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New interface config attrs:", newConf))
	return true, nil
}

func (h *OSPFHandler) UpdateOspfIfMetricEntry(origConf *ospfd.OspfIfMetricEntry, newConf *ospfd.OspfIfMetricEntry, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original interface metric config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New interface metric config attrs:", newConf))
	return true, nil
}

func (h *OSPFHandler) UpdateOspfVirtIfEntry(origConf *ospfd.OspfVirtIfEntry, newConf *ospfd.OspfVirtIfEntry, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original virtual interface config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New virtual interface config attrs:", newConf))
	return true, nil
}

