package server

import (
	"fmt"
	"l3/ospf/config"
	"net"
	"time"
)

func (server *OSPFServer) GetBulkOspfAreaEntryState(idx int, cnt int) (int, int, []config.AreaState) {
	var nextIdx int
	var count int

	//server.AreaStateMutex.RLock()
	ret := server.AreaStateTimer.Stop()
	if ret == false {
		server.logger.Err("Ospf is busy refreshing the cache")
		return nextIdx, count, nil
	}
	length := len(server.AreaStateSlice)
	if idx+cnt > length {
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

	ret := server.IntfStateTimer.Stop()
	if ret == false {
		server.logger.Err("Ospf is busy refreshing the cache")
		return nextIdx, count, nil
	}
	length := len(server.IntfKeySlice)
	if idx+cnt > length {
		count = length - idx
		nextIdx = 0
	}
	result := make([]config.InterfaceState, count)

	for i := 0; i < count; i++ {
		key := server.IntfKeySlice[idx+i]
		result[i].IfIpAddress = key.IPAddr
		result[i].AddressLessIf = key.IntfIdx
		if server.IntfKeyToSliceIdxMap[key] == true {
			//if exist {
			ent, _ := server.IntfConfMap[key]
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

func (server *OSPFServer) GetOspfGlobalState() *config.GlobalState {
	result := new(config.GlobalState)
	ent := server.ospfGlobalConf

	ip := net.IPv4(ent.RouterId[0], ent.RouterId[1], ent.RouterId[2], ent.RouterId[3])
	result.RouterId = config.RouterId(ip.String())
	result.VersionNumber = int32(ent.Version)
	result.AreaBdrRtrStatus = ent.AreaBdrRtrStatus
	result.ExternLsaCount = ent.ExternLsaCount
	result.ExternLsaChecksum = ent.ExternLsaChecksum
	result.OriginateNewLsas = ent.OriginateNewLsas
	result.RxNewLsas = ent.RxNewLsas
	result.OpaqueLsaSupport = ent.OpaqueLsaSupport
	result.RestartStatus = ent.RestartStatus
	result.RestartAge = ent.RestartAge
	result.RestartExitReason = ent.RestartExitReason
	result.AsLsaCount = ent.AsLsaCount
	result.AsLsaCksumSum = ent.AsLsaCksumSum
	result.StubRouterSupport = ent.StubRouterSupport
	result.DiscontinuityTime = ent.DiscontinuityTime
	server.logger.Info(fmt.Sprintln("Global State:", result))
	return result
}

func (server *OSPFServer) GetBulkOspfNbrEntryState(idx int, cnt int) (int, int, []config.NeighborState) {
	var nextIdx int
	var count int

	server.neighborSliceRefCh.Stop()
	/*	if ret == false {
		server.logger.Err("Ospf is busy refreshing the cache")
		return nextIdx, count, nil
	} */
	length := len(server.neighborBulkSlice)
	if idx+cnt > length {
		count = length - idx
		nextIdx = 0
	}
	result := make([]config.NeighborState, count)

	for i := 0; i < count; i++ {
		key := server.neighborBulkSlice[idx+i]
		server.logger.Info(fmt.Sprintln("Key ", key))
		/* get map entries.
		 */

		if ent, ok := server.NeighborConfigMap[key]; ok {
			result[i].NbrIpAddress = ent.OspfNbrIPAddr
			result[i].NbrAddressLessIndex = int(ent.intfConfKey.IntfIdx)
			result[i].NbrRtrId = string(key)
			result[i].NbrOptions = ent.OspfNbrOptions
			result[i].NbrPriority = uint8(ent.OspfRtrPrio)
			result[i].NbrState = ent.OspfNbrState
			result[i].NbrEvents = 0
			result[i].NbrLsRetransQLen = 0
			result[i].NbmaNbrPermanence = 0
			result[i].NbrHelloSuppressed = false
			result[i].NbrRestartHelperStatus = 0
			result[i].NbrRestartHelperAge = 0
			result[i].NbrRestartHelperExitReason = 0
		}

	}

	server.neighborSliceRefCh = time.NewTicker(time.Minute * 10)
	server.refreshNeighborSlice()
	server.logger.Info(fmt.Sprintln("length:", length, "count:", count, "nextIdx:", nextIdx, "result:", result))
	return nextIdx, count, result
}
