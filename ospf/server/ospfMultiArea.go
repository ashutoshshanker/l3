package server

import (
	"fmt"
	"l3/ospf/config"
)

type SummaryLsaMap map[LsaKey]SummaryLsa

func (server *OSPFServer) HandleSummaryLsa(areaId uint32) {
	server.logger.Info(fmt.Sprintln("Inside Handling Summary LSA for area:", areaId))
	server.HandleSummaryType3Lsa(areaId)
	server.HandleSummaryType4Lsa(areaId)
	server.HandleASExternalLsa(areaId)
}

func (server *OSPFServer) HandleSummaryType3Lsa(areaId uint32) {
	server.logger.Info(fmt.Sprintln("Inside Handling Summary Type 3 LSA for area:", areaId))
	server.CalcInterAreaRoutes(areaId)
}

func (server *OSPFServer) HandleTransitAreaSummaryLsa() {

}

// Handling Summary LSA in case of FS is internal router
func (server *OSPFServer) CalcInterAreaRoutes(areaId uint32) {
	lsdbKey := LsdbKey{
		AreaId: areaId,
	}
	lsDbEnt, exist := server.AreaLsdb[lsdbKey]
	if !exist {
		server.logger.Err(fmt.Sprintln("Unable to find Area Lsdb entry"))
		return
	}

	for lsaKey, lsaEnt := range lsDbEnt.Summary3LsaMap {
		server.logger.Info(fmt.Sprintln("Summary LSAKey:", lsaKey, "lsaENt:", lsaEnt))
		if lsaEnt.Metric == LSInfinity ||
			lsaEnt.LsaMd.LSAge == config.MaxAge {
			server.logger.Info("Ignoring Summary LSA...")
			continue
		}
		rtrId := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
		if lsaKey.AdvRouter == rtrId {
			server.logger.Info("Self originated summary 3 LSA, so no need to process for routing table calc")
			continue
		}

		areaIdKey := AreaIdKey{
			AreaId: areaId,
		}
		// TODO: Handle Area Range Section 16.2 Point 3
		//Network := lsaKey.LSId & lsaEnt.Netmask
		//Mask := lsaEnt.Netmask
		rKey := RoutingTblEntryKey{
			DestId:   lsaKey.AdvRouter,
			AddrMask: 0,
			DestType: AreaBdrRouter,
		}

		tempAreaRoutingTbl := server.TempAreaRoutingTbl[areaIdKey]
		rEnt, exist := tempAreaRoutingTbl.RoutingTblMap[rKey]
		if !exist {
			server.logger.Info("Area Router routing table entry doesnot exists for Summary Lsa Advertising Router")
			rKey = RoutingTblEntryKey{
				DestId:   lsaKey.AdvRouter,
				AddrMask: 0,
				DestType: ASAreaBdrRouter,
			}
			rEnt, exist = tempAreaRoutingTbl.RoutingTblMap[rKey]
			if !exist {
				server.logger.Info("AS Area Border Router routing table entry doesnot exists for Summary Lsa Advertising Router")
				continue
			}
		}
		if rEnt.NumOfPaths == 0 {
			continue
		}
		cost := rEnt.Cost + uint16(lsaEnt.Metric)
		nextHopMap := rEnt.NextHops
		numOfNextHops := rEnt.NumOfPaths
		rKey = RoutingTblEntryKey{
			DestId:   lsaKey.LSId & lsaEnt.Netmask,
			AddrMask: lsaEnt.Netmask,
			DestType: Network,
		}

		tempAreaRoutingTbl = server.TempAreaRoutingTbl[areaIdKey]
		rEnt, exist = tempAreaRoutingTbl.RoutingTblMap[rKey]
		if exist {
			if rEnt.PathType == IntraArea {
				continue
			}

			if rEnt.Cost < cost {
				server.logger.Info("Route already exists with lesser cost")
				continue
			} else if rEnt.Cost > cost {
				rEnt.OptCapabilities = 0 //TODO
				//rEnt.PathType = InterArea
				rEnt.Cost = cost
				//rEnt.Type2Cost = 0
				//rEnt.LSOrigin = lsaKey
				rEnt.NumOfPaths = numOfNextHops
				rEnt.NextHops = make(map[NextHop]bool)
				for key, _ := range nextHopMap {
					key.AdvRtr = lsaKey.AdvRouter
					rEnt.NextHops[key] = true
				}
			} else {
				cnt := 0
				for key, _ := range nextHopMap {
					_, exist = rEnt.NextHops[key]
					if !exist {
						key.AdvRtr = lsaKey.AdvRouter
						rEnt.NextHops[key] = true
						cnt++
					}
				}
				rEnt.NumOfPaths = numOfNextHops + cnt
			}
		} else {
			rEnt.OptCapabilities = 0 //TODO
			rEnt.PathType = InterArea
			rEnt.Cost = cost
			rEnt.Type2Cost = 0
			//rEnt.LSOrigin = lsaKey
			rEnt.NumOfPaths = numOfNextHops
			rEnt.NextHops = make(map[NextHop]bool)
			for key, _ := range nextHopMap {
				key.AdvRtr = lsaKey.AdvRouter
				rEnt.NextHops[key] = true
			}
		}
		tempAreaRoutingTbl.RoutingTblMap[rKey] = rEnt
		server.TempAreaRoutingTbl[areaIdKey] = tempAreaRoutingTbl
	}
}

func (server *OSPFServer) GenerateSummaryLsa() {
	server.logger.Info("Generating Summary LSA")
	server.SummaryLsDb = make(map[LsdbKey]SummaryLsaMap)
	for aKey, aEnt := range server.AreaConfMap {
		if len(aEnt.IntfListMap) == 0 {
			continue
		}

		areaId := convertAreaOrRouterIdUint32(string(aKey.AreaId))
		lsDbKey := LsdbKey{
			AreaId: areaId,
		}
		sEnt, _ := server.SummaryLsDb[lsDbKey]
		sEnt = make(map[LsaKey]SummaryLsa)
		for rKey, rEnt := range server.GlobalRoutingTbl {
			if rKey.DestType == AreaBdrRouter ||
				rEnt.RoutingTblEnt.PathType == Type1Ext ||
				rEnt.RoutingTblEnt.PathType == Type2Ext ||
				rEnt.AreaId == areaId ||
				rKey.DestType == InternalRouter ||
				uint32(rEnt.RoutingTblEnt.Cost) >= LSInfinity {
				// 1. If Dest Type is Area Border Router
				// 2. If Path Type is Type 1 Ext
				// 3. If Path Type is Type 2 Ext
				// 4. If Area associated with set of paths is Area itself
				// 5. If Cost >= LSInfinity
				// 6. TODO: Distance Vector Split Horizon Problem
				continue
			}

			// TODO: AS External Routes
			// If DestType == ASBdrRouter
			// If Routing Table Entry Describes the preferred path to
			// AS Boundary Router
			if rKey.DestType == ASAreaBdrRouter ||
				rKey.DestType == ASBdrRouter {
				lsaKey, summaryLsa := server.GenerateType4SummaryLSA(rKey, rEnt)
				sEnt[lsaKey] = summaryLsa
			}

			// Dest Type Network, Inter Area Routes
			if rKey.DestType == Network &&
				rEnt.RoutingTblEnt.PathType == InterArea {
				// Generate Type 3 Summary LSA for the desitnation
				// LSId = networks's address
				// Metric = Routing Table cost
				lsaKey, summaryLsa := server.GenerateType3SummaryLSA(rKey, rEnt)
				sEnt[lsaKey] = summaryLsa
			} else if rKey.DestType == Network &&
				rEnt.RoutingTblEnt.PathType == IntraArea {
				//TODO: Address Range
				// By default LSId = network's address
				// Metric = Routing Table cost
				lsaKey, summaryLsa := server.GenerateType3SummaryLSA(rKey, rEnt)
				sEnt[lsaKey] = summaryLsa
			}
		}
		server.SummaryLsDb[lsDbKey] = sEnt
	}

}

func (server *OSPFServer) GenerateType3SummaryLSA(rKey RoutingTblEntryKey, rEnt GlobalRoutingTblEntry) (LsaKey, SummaryLsa) {
	var summaryLsa SummaryLsa

	AdvRouter := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
	lsaKey := LsaKey{
		LSType:    Summary3LSA,
		LSId:      rKey.DestId,
		AdvRouter: AdvRouter,
	}

	summaryLsa.LsaMd.Options = uint8(2) //Need to be re-visited
	summaryLsa.LsaMd.LSAge = 0
	summaryLsa.LsaMd.LSSequenceNum = InitialSequenceNumber
	summaryLsa.LsaMd.LSLen = uint16(OSPF_LSA_HEADER_SIZE + 8)
	summaryLsa.Netmask = rKey.AddrMask
	summaryLsa.Metric = uint32(rEnt.RoutingTblEnt.Cost)

	return lsaKey, summaryLsa
}
