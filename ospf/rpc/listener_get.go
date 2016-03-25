package rpc

import (
//    "ospfd"
//    "fmt"
//    "l3/ospf/config"
//    "l3/ospf/server"
//    "utils/logging"
//    "net"
)

/*
func (h *OSPFHandler) GetOspfGlobalState() (*ospfd.OspfGlobalState, error) {
    h.logger.Info(fmt.Sprintln("Get global attrs"))
    ospfGlobalResponse := ospfd.NewOspfGlobalState()
    return ospfGlobalResponse, nil
}

func (h *OSPFHandler) GetOspfAreaState(areaId string) (*ospfd.OspfAreaState, error) {
    h.logger.Info(fmt.Sprintln("Get Area attrs"))
    ospfAreaResponse := ospfd.NewOspfAreaState()
    return ospfAreaResponse, nil
}

func (h *OSPFHandler) GetOspfStubAreaState(stubAreaId string, stubTOS int32) (*ospfd.OspfStubAreaState, error) {
    h.logger.Info(fmt.Sprintln("Get Area Stub attrs"))
    ospfStubAreaResponse := ospfd.NewOspfStubAreaState()
    return ospfStubAreaResponse, nil
}

func (h *OSPFHandler) GetOspfLsdbState(lsdbAreaId string, lsdbLsid string, lsdbRouterId string) (*ospfd.OspfLsdbState, error) {
    h.logger.Info(fmt.Sprintln("Get Link State Database attrs"))
    ospfLsdbResponse := ospfd.NewOspfLsdbState()
    return ospfLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfAreaRangeState(rangeAreaId string, areaRangeNet string) (*ospfd.OspfAreaRangeState, error) {
    h.logger.Info(fmt.Sprintln("Get Address range attrs"))
    ospfAreaRangeResponse := ospfd.NewOspfAreaRangeState()
    return ospfAreaRangeResponse, nil
}

func (h *OSPFHandler) GetOspfHostState(hostIpAddress string, hostTOS int32) (*ospfd.OspfHostState, error) {
    h.logger.Info(fmt.Sprintln("Get Host attrs"))
    ospfHostResponse := ospfd.NewOspfHostState()
    return ospfHostResponse, nil
}

func (h *OSPFHandler) GetOspfIfState(ifIpAddress string, addressLessIf int32) (*ospfd.OspfIfState, error) {
    h.logger.Info(fmt.Sprintln("Get Interface attrs"))
    ospfIfResponse := ospfd.NewOspfIfState()
    return ospfIfResponse, nil
}

func (h *OSPFHandler) GetOspfIfMetricState(ifMetricIpAddress string, ifMetricAddressLessIf int32, ifMetricTOS int32) (*ospfd.OspfIfMetricState, error) {
    h.logger.Info(fmt.Sprintln("Get Interface Metric attrs"))
    ospfIfMetricResponse := ospfd.NewOspfIfMetricState()
    return ospfIfMetricResponse, nil
}

func (h *OSPFHandler) GetOspfVirtIfState(virtIfAreaId string, virtIfNeighbor string) (*ospfd.OspfVirtIfState, error) {
    h.logger.Info(fmt.Sprintln("Get Virtual Interface attrs"))
    ospfVirtIfResponse := ospfd.NewOspfVirtIfState()
    return ospfVirtIfResponse, nil
}

func (h *OSPFHandler) GetOspfNbrState(nbrIpAddress string, nbrAddressLessIndex int32) (*ospfd.OspfNbrState, error) {
    h.logger.Info(fmt.Sprintln("Get Neighbor attrs"))
    ospfNbrResponse := ospfd.NewOspfNbrState()
    return ospfNbrResponse, nil
}

func (h *OSPFHandler) GetOspfVirtNbrState(virtNbrArea string, virtNbrRtrId string) (*ospfd.OspfVirtNbrState, error) {
    h.logger.Info(fmt.Sprintln("Get Virtual Neighbor attrs"))
    ospfVirtNbrResponse := ospfd.NewOspfVirtNbrState()
    return ospfVirtNbrResponse, nil
}

func (h *OSPFHandler) GetOspfExtLsdbState(extLsdbType ospfd.LsaType, extLsdbLsid string, extLsdbRouterId string) (*ospfd.OspfExtLsdbState, error) {
    h.logger.Info(fmt.Sprintln("Get External LSA Link State attrs"))
    ospfExtLsdbResponse := ospfd.NewOspfExtLsdbState()
    return ospfExtLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfAreaAggregateState(areaAggregateAreaId string, areaAggregateLsdbType ospfd.LsaType, areaAggregateNet string, areaAggregateMask string) (*ospfd.OspfAreaAggregateState, error) {
    h.logger.Info(fmt.Sprintln("Get Area Aggregate State attrs"))
    ospfAreaAggregateResponse := ospfd.NewOspfAreaAggregateState()
    return ospfAreaAggregateResponse, nil
}

func (h *OSPFHandler) GetOspfLocalLsdbState(localLsdbIpAddress string, localLsdbAddressLessIf int32, localLsdbType ospfd.LsaType, localLsdbLsid string, localLsdbRouterId string) (*ospfd.OspfLocalLsdbState, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for non virtual links attrs"))
    ospfLocalLsdbResponse := ospfd.NewOspfLocalLsdbState()
    return ospfLocalLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfVirtLocalLsdbState(virtLocalLsdbTransitArea string, virtLocalLsdbNeighbor string, virtLocalLsdbType ospfd.LsaType, virtLocalLsdbLsid string, virtLocalLsdbRouterId string) (*ospfd.OspfVirtLocalLsdbState, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for virtual links attrs"))
    ospfVirtLocalLsdbResponse := ospfd.NewOspfVirtLocalLsdbState()
    return ospfVirtLocalLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfAsLsdbState(asLsdbType ospfd.LsaType, asLsdbLsid string, asLsdbRouterId string) (*ospfd.OspfAsLsdbState, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for AS attrs"))
    ospfAsLsdbResponse := ospfd.NewOspfAsLsdbState()
    return ospfAsLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfAreaLsaCountState(areaLsaCountAreaId string, areaLsaCountLsaType ospfd.LsaType) (*ospfd.OspfAreaLsaCountState, error) {
    h.logger.Info(fmt.Sprintln("Get Area LSA Counter"))
    ospfAreaLsaCountResponse := ospfd.NewOspfAreaLsaCountState()
    return ospfAreaLsaCountResponse, nil
}

*/
