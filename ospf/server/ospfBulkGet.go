package server

import (
        "fmt"
        "l3/ospf/config"
)


func (server *OSPFServer) GetBulkOspfAreaEntryState(idx int, cnt int) (int, int, []config.AreaState) {
        var nextIdx int = 10
        var count int = 10

        //server.AreaStateMutex.RLock()
        server.AreaStateTimer.Stop()
        length := len(server.AreaStateSlice)
        if idx + cnt > length {
                count = length - idx
                nextIdx = 0
        }
        result := make([]config.AreaState, count)

        for i := 0; i < count; i++ {
                key := server.AreaStateSlice[idx+i]
                result[i].AreaId = key.AreaId
                ent, exist := server.AreaStateMap[key]
                if exist {
                        result[i].SpfRuns = ent.SpfRuns
                        result[i].AreaBdrRtrCount = ent.AreaBdrRtrCount
                        result[i].AsBdrRtrCount = ent.AsBdrRtrCount
                        result[i].AreaLsaCount = ent.AreaLsaCount
                        result[i].AreaLsaCksumSum = ent.AreaLsaCksumSum
                        result[i].AreaNssaTranslatorState = ent.AreaNssaTranslatorState
                        result[i].AreaNssaTranslatorEvents = ent.AreaNssaTranslatorEvents
                } else {
                        result[i].SpfRuns = -1
                        result[i].AreaBdrRtrCount = -1
                        result[i].AsBdrRtrCount = -1
                        result[i].AreaLsaCount = -1
                        result[i].AreaLsaCksumSum = -1
                        result[i].AreaNssaTranslatorState = -1
                        result[i].AreaNssaTranslatorEvents = -1
                }

        }

        //server.AreaStateMutex.RUnlock()
        server.AreaStateTimer.Reset(server.RefreshDuration)
        server.logger.Info(fmt.Sprintln("length:", length, "count:", count, "nextIdx:", nextIdx, "result:", result))
        return nextIdx, count, result
}
