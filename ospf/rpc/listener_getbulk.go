package rpc

import (
    "ospfd"
    "fmt"
    "l3/ospf/config"
//    "l3/ospf/server"
//    "log/syslog"
//    "net"
)

func (h *OSPFHandler) convertAreaEntryStateToThrift(ent config.AreaState) *ospfd.OspfAreaEntryState {
        areaEntry := ospfd.NewOspfAreaEntryState()
        areaEntry.AreaIdKey = string(ent.AreaId)
        areaEntry.SpfRuns = ent.SpfRuns
        areaEntry.AreaBdrRtrCount = ent.AreaBdrRtrCount
        areaEntry.AsBdrRtrCount = ent.AsBdrRtrCount
        areaEntry.AreaLsaCount = ent.AreaLsaCount
        areaEntry.AreaLsaCksumSum = ent.AreaLsaCksumSum
        areaEntry.AreaNssaTranslatorState = int32(ent.AreaNssaTranslatorState)
        areaEntry.AreaNssaTranslatorEvents = ent.AreaNssaTranslatorEvents

        return areaEntry

}


func (h *OSPFHandler) GetBulkOspfAreaEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfAreaEntryStateGetInfo, error) {
        h.logger.Info(fmt.Sprintln("Get Area attrs"))

        nextIdx, currCount, ospfAreaEntryStates := h.server.GetBulkOspfAreaEntryState(int(fromIdx), int(count))
        ospfAreaEntryStateResponse := make([]*ospfd.OspfAreaEntryState, len(ospfAreaEntryStates))
        for idx, item := range ospfAreaEntryStates {
                ospfAreaEntryStateResponse[idx] = h.convertAreaEntryStateToThrift(item)
        }
        ospfAreaEntryStateGetInfo := ospfd.NewOspfAreaEntryStateGetInfo()
        ospfAreaEntryStateGetInfo.Count = ospfd.Int(currCount)
        ospfAreaEntryStateGetInfo.StartIdx = ospfd.Int(fromIdx)
        ospfAreaEntryStateGetInfo.EndIdx = ospfd.Int(nextIdx)
        ospfAreaEntryStateGetInfo.More = (nextIdx != 0)
        ospfAreaEntryStateGetInfo.OspfAreaEntryStateList = ospfAreaEntryStateResponse
        return ospfAreaEntryStateGetInfo, nil
}

func (h *OSPFHandler) GetBulkOspfLsdbEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfLsdbEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Link State Database attrs"))
    ospfLsdbResponse := ospfd.NewOspfLsdbEntryStateGetInfo()
    return ospfLsdbResponse, nil
}

func (h *OSPFHandler) GetBulkOspfIfEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfIfEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Interface attrs"))
    ospfIfResponse := ospfd.NewOspfIfEntryStateGetInfo()
    return ospfIfResponse, nil
}

func (h *OSPFHandler) GetBulkOspfNbrEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfNbrEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Neighbor attrs"))
    ospfNbrResponse := ospfd.NewOspfNbrEntryStateGetInfo()
    return ospfNbrResponse, nil
}

func (h *OSPFHandler) GetBulkOspfVirtNbrEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfVirtNbrEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Virtual Neighbor attrs"))
    ospfVirtNbrResponse := ospfd.NewOspfVirtNbrEntryStateGetInfo()
    return ospfVirtNbrResponse, nil
}

func (h *OSPFHandler) GetBulkOspfExtLsdbEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfExtLsdbEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get External LSA Link State attrs"))
    ospfExtLsdbResponse := ospfd.NewOspfExtLsdbEntryStateGetInfo()
    return ospfExtLsdbResponse, nil
}

/*
func (h *OSPFHandler) GetOspfAreaAggregateEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfAreaAggregateEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Area Aggregate State attrs"))
    ospfAreaAggregateResponse := ospfd.NewOspfAreaAggregateEntryStateGetInfo()
    return ospfAreaAggregateResponse, nil
}
*/

func (h *OSPFHandler) GetBulkOspfLocalLsdbEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfLocalLsdbEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for non virtual links attrs"))
    ospfLocalLsdbResponse := ospfd.NewOspfLocalLsdbEntryStateGetInfo()
    return ospfLocalLsdbResponse, nil
}

func (h *OSPFHandler) GetBulkOspfVirtLocalLsdbEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfVirtLocalLsdbEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for virtual links attrs"))
    ospfVirtLocalLsdbResponse := ospfd.NewOspfVirtLocalLsdbEntryStateGetInfo()
    return ospfVirtLocalLsdbResponse, nil
}

func (h *OSPFHandler) GetBulkOspfAsLsdbEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfAsLsdbEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Local Link State for AS attrs"))
    ospfAsLsdbResponse := ospfd.NewOspfAsLsdbEntryStateGetInfo()
    return ospfAsLsdbResponse, nil
}

func (h *OSPFHandler) GetBulkOspfAreaLsaCountEntryState(fromIdx ospfd.Int, count ospfd.Int) (*ospfd.OspfAreaLsaCountEntryStateGetInfo, error) {
    h.logger.Info(fmt.Sprintln("Get Area LSA Counter"))
    ospfAreaLsaCountResponse := ospfd.NewOspfAreaLsaCountEntryStateGetInfo()
    return ospfAreaLsaCountResponse, nil
}


