package server

import (
        "fmt"
        "l3/ospf/config"
        "net"
)


func (server *OSPFServer) GetBulkOspfAreaEntryState(idx int, cnt int) (int, int, []config.AreaState) {
        var nextIdx int
        var count int

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

func (server *OSPFServer) GetBulkOspfIfEntryState(idx int, cnt int) (int, int, []config.InterfaceState) {
        var nextIdx int
        var count int

        server.IntfStateTimer.Stop()
        length := len(server.IntfKeySlice)
        if idx + cnt > length {
                count = length - idx
                nextIdx = 0
        }
        result := make([]config.InterfaceState, count)

        for i := 0; i < count; i++ {
                key := server.IntfKeySlice[idx+i]
                result[i].IfIpAddress = key.IPAddr
                result[i].AddressLessIf = key.IntfIdx
                if server.IntfKeyToSliceIdxMap[key] == true {
                        server.logger.Info("Hello3")
                //if exist {
                        ent, _:= server.IntfConfMap[key]
                        result[i].IfState = ent.IfFSMState
                        ip := net.IPv4(ent.IfDRIp[0], ent.IfDRIp[1], ent.IfDRIp[2], ent.IfDRIp[3])
                        result[i].IfDesignatedRouter = config.IpAddress(ip.String())
                        ip = net.IPv4(ent.IfBDRIp[0], ent.IfBDRIp[1], ent.IfBDRIp[2], ent.IfBDRIp[3])
                        result[i].IfBackupDesignatedRouter = config.IpAddress(ip.String())
                        result[i].IfEvents = ent.IfEvents
                        result[i].IfLsaCount = ent.IfLsaCount
                        result[i].IfLsaCksumSum = ent.IfLsaCksumSum
                        result[i].IfDesignatedRouterId = config.RouterId(convertUint32ToIPv4(ent.IfDRtrId))
                        result[i].IfBackupDesignatedRouterId = config.RouterId(convertUint32ToIPv4(ent.IfBDRtrId))
                } else {
                        result[i].IfState = 0
                        result[i].IfDesignatedRouter = "0.0.0.0"
                        result[i].IfBackupDesignatedRouter = "0.0.0.0"
                        result[i].IfEvents = 0
                        result[i].IfLsaCount = 0
                        result[i].IfLsaCksumSum = 0
                        result[i].IfDesignatedRouterId = "0.0.0.0"
                        result[i].IfBackupDesignatedRouterId = "0.0.0.0"
                }
        }

        server.IntfStateTimer.Reset(server.RefreshDuration)
        server.logger.Info(fmt.Sprintln("length:", length, "count:", count, "nextIdx:", nextIdx, "result:", result))
        return nextIdx, count, result
}
