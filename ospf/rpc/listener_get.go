package rpc

import (
        "ospfd"
        "fmt"
)

func (h *OSPFHandler) GetOspfGlobalState(ospfGlobal *ospfd.OspfGlobalState) (*ospfd.OspfGlobalState, error) {
    h.logger.Info(fmt.Sprintln("Get global attrs"))
    ospfGlobalResponse := ospfd.NewOspfGlobalState()
    return ospfGlobalResponse, nil
}

func (h *OSPFHandler) GetOspfAreaEntryState(ospfArea *ospfd.OspfAreaEntryState) (*ospfd.OspfAreaEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Area attrs"))
    ospfAreaResponse := ospfd.NewOspfAreaEntryState()
    return ospfAreaResponse, nil
}

/*
func (h *OSPFHandler) GetOspfStubAreaEntryState(stubAreaId string, stubTOS int32) (*ospfd.OspfStubAreaState, error) {
    h.logger.Info(fmt.Sprintln("Get Area Stub attrs"))
    ospfStubAreaResponse := ospfd.NewOspfStubAreaState()
    return ospfStubAreaResponse, nil
}
*/

func (h *OSPFHandler) GetOspfLsdbEntryState(ospfLsdb *ospfd.OspfLsdbEntryState) (*ospfd.OspfLsdbEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Link State Database attrs"))
    ospfLsdbResponse := ospfd.NewOspfLsdbEntryState()
    return ospfLsdbResponse, nil
}

/*
func (h *OSPFHandler) GetOspfAreaRangeEntryState(rangeAreaId string, areaRangeNet string) (*ospfd.OspfAreaRangeState, error) {
    h.logger.Info(fmt.Sprintln("Get Address range attrs"))
    ospfAreaRangeResponse := ospfd.NewOspfAreaRangeState()
    return ospfAreaRangeResponse, nil
}
*/

func (h *OSPFHandler) GetOspfHostEntryState(ospfHost *ospfd.OspfHostEntryState) (*ospfd.OspfHostEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Host attrs"))
    ospfHostResponse := ospfd.NewOspfHostEntryState()
    return ospfHostResponse, nil
}

func (h *OSPFHandler) GetOspfIfEntryState(ospfIfEntry *ospfd.OspfIfEntryState) (*ospfd.OspfIfEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Interface attrs"))
    ospfIfResponse := ospfd.NewOspfIfEntryState()
    return ospfIfResponse, nil
}

/*
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
*/

func (h *OSPFHandler) GetOspfNbrEntryState(ospfNbrEntry *ospfd.OspfNbrEntryState) (*ospfd.OspfNbrEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Neighbor attrs"))
    ospfNbrResponse := ospfd.NewOspfNbrEntryState()
    return ospfNbrResponse, nil
}

func (h *OSPFHandler) GetOspfVirtNbrEntryState(ospfVirtNbr *ospfd.OspfVirtNbrEntryState) (*ospfd.OspfVirtNbrEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Virtual Neighbor attrs"))
    ospfVirtNbrResponse := ospfd.NewOspfVirtNbrEntryState()
    return ospfVirtNbrResponse, nil
}

func (h *OSPFHandler) GetOspfExtLsdbEntryState(ospfExtLsdb *ospfd.OspfExtLsdbEntryState) (*ospfd.OspfExtLsdbEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get External LSA Link State attrs"))
    ospfExtLsdbResponse := ospfd.NewOspfExtLsdbEntryState()
    return ospfExtLsdbResponse, nil
}

/*
func (h *OSPFHandler) GetOspfAreaAggregateState(areaAggregateAreaId string, areaAggregateLsdbType ospfd.LsaType, areaAggregateNet string, areaAggregateMask string) (*ospfd.OspfAreaAggregateState, error) {
    h.logger.Info(fmt.Sprintln("Get Area Aggregate State attrs"))
    ospfAreaAggregateResponse := ospfd.NewOspfAreaAggregateState()
    return ospfAreaAggregateResponse, nil
}
*/

func (h *OSPFHandler) GetOspfLocalLsdbEntryState(ospfLocalLsdb *ospfd.OspfLocalLsdbEntryState) (*ospfd.OspfLocalLsdbEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for non virtual links attrs"))
    ospfLocalLsdbResponse := ospfd.NewOspfLocalLsdbEntryState()
    return ospfLocalLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfVirtLocalLsdbEntryState(ospfVirtLocalLsdb *ospfd.OspfVirtLocalLsdbEntryState) (*ospfd.OspfVirtLocalLsdbEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for virtual links attrs"))
    ospfVirtLocalLsdbResponse := ospfd.NewOspfVirtLocalLsdbEntryState()
    return ospfVirtLocalLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfAsLsdbEntryState(ospfAsLsdb *ospfd.OspfAsLsdbEntryState) (*ospfd.OspfAsLsdbEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for AS attrs"))
    ospfAsLsdbResponse := ospfd.NewOspfAsLsdbEntryState()
    return ospfAsLsdbResponse, nil
}

func (h *OSPFHandler) GetOspfAreaLsaCountEntryState(ospfAreaLsaCount *ospfd.OspfAreaLsaCountEntryState) (*ospfd.OspfAreaLsaCountEntryState, error) {
    h.logger.Info(fmt.Sprintln("Get Area LSA Counter"))
    ospfAreaLsaCountResponse := ospfd.NewOspfAreaLsaCountEntryState()
    return ospfAreaLsaCountResponse, nil
}
