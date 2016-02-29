package server

import (
        "fmt"
        //"l3/rib/ribdCommonDefs"
        "ribd"
        //"utils/commonDefs"
        //"errors"
)

type DestType bool
const (
        Network DestType = true
        Router  DestType = false
)

type PathType int
const (
        /* Decreasing order of Precedence */
        IntraArea       PathType = 4
        InterArea       PathType = 3
        Type1Ext        PathType = 2
        Type2Ext        PathType = 1
)

type IfData struct {
        IfIpAddr      uint32
        IfIdx         uint32
}

type NbrIP uint32

type NextHop struct {
        IfIPAddr      uint32
        IfIdx         uint32
        NextHopIP     uint32
}

type RoutingTblKey struct {
        DestId          uint32          // IP address(Network Type) RouterID(Router Type)
        AddrMask        uint32          // Only For Network Type
        DestType        DestType        // true: Network, false: Router
}

type RoutingTblEntry struct {
        OptCapabilities uint8           // Optional Capabilities
        Area            uint32          // Area
        PathType        PathType        // Path Type
        Cost            uint16
        Type2Cost       uint16
        LSOrigin        LsaKey
        NumOfPaths      int
        NextHops        map[NextHop]bool        // Next Hop
        AdvRtr          uint32                  // Nbr Router Id
}

func (server *OSPFServer)dumpRoutingTbl() {
        server.logger.Info("=============Routing Table============")
        server.logger.Info("DestId      AddrMask        DestType        OprCapabilities Area    PathType        Cost    Type2Cost       LSOrigin        NumOfPaths      NextHops        AdvRtr")
        for key, ent := range server.RoutingTbl {
                DestId := convertUint32ToIPv4(key.DestId)
                AddrMask := convertUint32ToIPv4(key.AddrMask)
                var DestType string
                if key.DestType == Network {
                        DestType = "Network"
                } else {
                        DestType = "Router"
                }
                Area := convertUint32ToIPv4(ent.Area)
                var PathType string
                if ent.PathType == IntraArea {
                        PathType = "IntraArea"
                } else if ent.PathType == InterArea {
                        PathType = "InterArea"
                } else if ent.PathType == Type1Ext {
                        PathType = "Type1Ext"
                } else {
                        PathType = "Type2Ext"
                }
                var LsaType string
                if ent.LSOrigin.LSType == RouterLSA {
                        LsaType = "RouterLSA"
                } else if ent.LSOrigin.LSType == NetworkLSA {
                        LsaType = "NetworkLSA"
                } else if ent.LSOrigin.LSType == Summary3LSA {
                        LsaType = "Summary3LSA"
                } else if ent.LSOrigin.LSType == Summary4LSA {
                        LsaType = "Summary4LSA"
                } else {
                        LsaType = "ASExternalLSA"
                }
                LsaLSId := convertUint32ToIPv4(ent.LSOrigin.LSId)
                LsaAdvRouter := convertUint32ToIPv4(ent.LSOrigin.AdvRouter)
                AdvRtr := convertUint32ToIPv4(ent.AdvRtr)
                var NextHops string = "["
                for nxtHopKey, _ := range ent.NextHops {
                        NextHops = NextHops + "{"
                        IfIPAddr := convertUint32ToIPv4(nxtHopKey.IfIPAddr)
                        NextHopIP := convertUint32ToIPv4(nxtHopKey.NextHopIP)
                        nextHops := fmt.Sprint("IfIpAddr:", IfIPAddr, "IfIdx:", nxtHopKey.IfIdx, "NextHopIP:", NextHopIP)
                        NextHops = NextHops + nextHops
                        NextHops = NextHops + "}"
                }
                NextHops = NextHops + "]"
                server.logger.Info(fmt.Sprintln(DestId, AddrMask, DestType, ent.OptCapabilities, Area, PathType, ent.Cost, ent.Type2Cost, "[", LsaType, LsaLSId, LsaAdvRouter, "]", ent.NumOfPaths, NextHops, AdvRtr))
        }
        server.logger.Info("==============End of Routing Table================")
}

func (server *OSPFServer)UpdateRoutingTblForRouter(vKey VertexKey, tVertex TreeVertex, rootVKey VertexKey) {
        server.logger.Info(fmt.Sprintln("Updating Routing Table for Router Vertex", vKey, tVertex))

        gEnt, exist := server.AreaGraph[vKey]
        if !exist {
                server.logger.Err(fmt.Sprintln("Entry doesn't exist in Area Graph for:", vKey))
                return
        }
        rKey := RoutingTblKey {
                DestType: Router,
                AddrMask: 0, //TODO
                DestId: vKey.ID,
        }

        rEnt, exist := server.TempRoutingTbl[rKey]
        if exist {
                server.logger.Info(fmt.Sprintln("Routing Tbl entry already exist for:", rKey))
                return
        }

        rEnt.OptCapabilities = 0 //TODO
        rEnt.Area = gEnt.AreaId
        rEnt.PathType = IntraArea
        rEnt.Cost = tVertex.Distance
        rEnt.Type2Cost = 0 //TODO
        rEnt.LSOrigin = gEnt.LsaKey
        rEnt.NumOfPaths = tVertex.NumOfPaths
        rEnt.NextHops = make(map[NextHop]bool, tVertex.NumOfPaths)
        for i := 0; i < tVertex.NumOfPaths; i++ {
                pathlen := len(tVertex.Paths[i])
                if tVertex.Paths[i][0] != rootVKey {
                        server.logger.Info("Starting vertex is not our router, hence ignoring this path")
                        continue
                }
                if pathlen < 2 {
                        server.logger.Info("Connected Route so no next hops")
                        continue
                }
                vFirst := tVertex.Paths[i][0]
                vSecond := tVertex.Paths[i][1]
                var vThird VertexKey
                if pathlen == 2 {
                        vThird = vKey
                } else {
                        vThird = tVertex.Paths[i][2]
                }
                gFirst, exist := server.AreaGraph[vFirst]
                if !exist {
                        server.logger.Info(fmt.Sprintln("1. Entry does not exist for:", vFirst, "in Area Graph"))
                        continue
                }
                gThird, exist := server.AreaGraph[vThird]
                if !exist {
                        server.logger.Info(fmt.Sprintln("3. Entry does not exist for:", vThird, "in Area Graph"))
                        continue
                }
                ifIPAddr := gFirst.LinkData[vSecond]
                nextHopIP := gThird.LinkData[vSecond]
                nextHop := NextHop {
                        IfIPAddr: ifIPAddr,
                        IfIdx: 0, //TODO
                        NextHopIP: nextHopIP,
                }
                rEnt.NextHops[nextHop] = true
        }
        rEnt.AdvRtr = vKey.AdvRtr
        server.TempRoutingTbl[rKey] = rEnt
}

func (server *OSPFServer)UpdateRoutingTblForSNetwork(vKey VertexKey, tVertex TreeVertex, rootVKey VertexKey) {
        server.logger.Info(fmt.Sprintln("Updating Routing Table for Stub Network Vertex", vKey, tVertex))

        sEnt, exist := server.AreaStubs[vKey]
        if !exist {
                server.logger.Err(fmt.Sprintln("Entry doesn't exist in Area Stubs for:", vKey))
                return
        }
        rKey := RoutingTblKey {
                DestType: Network,
                AddrMask: sEnt.LinkData, //TODO
                DestId: vKey.ID,
        }

        rEnt, exist := server.TempRoutingTbl[rKey]
        if exist {
                server.logger.Info(fmt.Sprintln("Routing Tbl entry already exist for:", rKey))
                return
        }

        rEnt.OptCapabilities = 0 //TODO
        rEnt.Area = sEnt.AreaId
        rEnt.PathType = IntraArea //TODO
        rEnt.Cost = tVertex.Distance
        rEnt.Type2Cost = 0 //TODO
        rEnt.LSOrigin = sEnt.LsaKey
        rEnt.NumOfPaths = tVertex.NumOfPaths
        rEnt.NextHops = make(map[NextHop]bool, tVertex.NumOfPaths)
        for i := 0; i < tVertex.NumOfPaths; i++ {
                pathlen := len(tVertex.Paths[i])
                if tVertex.Paths[i][0] != rootVKey {
                        server.logger.Info("Starting vertex is not our router, hence ignoring this path")
                        continue
                }
                if pathlen < 3 { //Path Example {R1}, {R1, N1, R2} -- TODO
                        server.logger.Info("Connected Route so no next hops")
                        continue
                }
                vFirst := tVertex.Paths[i][0]
                vSecond := tVertex.Paths[i][1]
                vThird := tVertex.Paths[i][2]
/*
                var vThird VertexKey
                if pathlen == 2 {
                        vThird = vKey
                } else {
                        vThird = tVertex.Paths[i][2]
                }
*/
                gFirst, exist := server.AreaGraph[vFirst]
                if !exist {
                        server.logger.Info(fmt.Sprintln("1. Entry does not exist for:", vFirst, "in Area Graph"))
                        continue
                }
                gThird, exist := server.AreaGraph[vThird]
                if !exist {
                        server.logger.Info(fmt.Sprintln("3. Entry does not exist for:", vThird, "in Area Graph"))
                        continue
                }
                ifIPAddr := gFirst.LinkData[vSecond]
                nextHopIP := gThird.LinkData[vSecond]
                nextHop := NextHop {
                        IfIPAddr: ifIPAddr,
                        IfIdx: 0, //TODO
                        NextHopIP: nextHopIP,
                }
                rEnt.NextHops[nextHop] = true
        }
        rEnt.AdvRtr = vKey.AdvRtr
        server.TempRoutingTbl[rKey] = rEnt
}

func (server *OSPFServer)UpdateRoutingTblForTNetwork(vKey VertexKey, tVertex TreeVertex, rootVKey VertexKey) {
        server.logger.Info(fmt.Sprintln("Updating Routing Table for Transit Network Vertex", vKey, tVertex))

        gEnt, exist := server.AreaGraph[vKey]
        if !exist {
                server.logger.Err(fmt.Sprintln("Entry doesn't exist in Area Graph for:", vKey))
                return
        }

        //Need to add check for len of gEnt.NbrVertexKey
        if len(gEnt.NbrVertexKey) < 1 {
                server.logger.Info(fmt.Sprintln("Vertex", vKey, "is listed as Transit but doesn't have any Neighboring routers"))
                return
        }
        addrMask, exist := gEnt.LinkData[gEnt.NbrVertexKey[0]]
        if !exist {
                server.logger.Err(fmt.Sprintln("Vertex", vKey, "has neighboring router but no corresponding linkdata"))
        }
        rKey := RoutingTblKey {
                DestType: Network,
                AddrMask: addrMask, //TODO
                DestId: vKey.ID & addrMask,
        }

        rEnt, exist := server.TempRoutingTbl[rKey]
        if exist {
                server.logger.Info(fmt.Sprintln("Routing Tbl entry already exist for:", rKey))
                return
        }

        rEnt.OptCapabilities = 0 //TODO
        rEnt.Area = gEnt.AreaId
        rEnt.PathType = IntraArea //TODO
        rEnt.Cost = tVertex.Distance
        rEnt.Type2Cost = 0 //TODO
        rEnt.LSOrigin = gEnt.LsaKey
        rEnt.NumOfPaths = tVertex.NumOfPaths
        rEnt.NextHops = make(map[NextHop]bool, tVertex.NumOfPaths)
        for i := 0; i < tVertex.NumOfPaths; i++ {
                pathlen := len(tVertex.Paths[i])
                if tVertex.Paths[i][0] != rootVKey {
                        server.logger.Info("Starting vertex is not our router, hence ignoring this path")
                        continue
                }
                if pathlen < 3 { //Path Example {R1}, {R1, N1, R2} -- TODO
                        server.logger.Info("Connected Route so no next hops")
                        continue
                }
                vFirst := tVertex.Paths[i][0]
                vSecond := tVertex.Paths[i][1]
                vThird := tVertex.Paths[i][2]
/*
                var vThird VertexKey
                if pathlen == 2 {
                        vThird = vKey
                } else {
                        vThird = tVertex.Paths[i][2]
                }
*/
                gFirst, exist := server.AreaGraph[vFirst]
                if !exist {
                        server.logger.Info(fmt.Sprintln("1. Entry does not exist for:", vFirst, "in Area Graph"))
                        continue
                }
                gThird, exist := server.AreaGraph[vThird]
                if !exist {
                        server.logger.Info(fmt.Sprintln("3. Entry does not exist for:", vThird, "in Area Graph"))
                        continue
                }
                ifIPAddr := gFirst.LinkData[vSecond]
                nextHopIP := gThird.LinkData[vSecond]
                nextHop := NextHop {
                        IfIPAddr: ifIPAddr,
                        IfIdx: 0, //TODO
                        NextHopIP: nextHopIP,
                }
                rEnt.NextHops[nextHop] = true
        }
        rEnt.AdvRtr = vKey.AdvRtr
        server.TempRoutingTbl[rKey] = rEnt
}

// Compare Old and New Route
func (server *OSPFServer)CompareRoutes(rKey RoutingTblKey) bool {
        oldEnt, exist := server.OldRoutingTbl[rKey]
        if !exist {
                server.logger.Err(fmt.Sprintln("No Route with", rKey, "was there in Old Routing Table"))
                return true
        }
        newEnt, exist := server.TempRoutingTbl[rKey]
        if !exist {
                server.logger.Err(fmt.Sprintln("No Route with", rKey, "is there in New Routing Table"))
                return true
        }
        if oldEnt.Cost != newEnt.Cost {
                return false
        }
        if len(oldEnt.NextHops) != len(newEnt.NextHops) {
                return false
        }

        for key, _ := range oldEnt.NextHops {
                _, exist := newEnt.NextHops[key]
                if !exist {
                        return false
                }
        }
        return true
}

func (server *OSPFServer)DeleteRoute(rKey RoutingTblKey) {
        server.logger.Info(fmt.Sprintln("Deleting route for rKey:", rKey))
        oldEnt, exist := server.OldRoutingTbl[rKey]
        if !exist {
                server.logger.Info(fmt.Sprintln("No route installed for rKey:", rKey, "hence, not deleting it"))
                return
        }
        destNetIp := convertUint32ToIPv4(rKey.DestId) //String :1
        networkMask := convertUint32ToIPv4(rKey.AddrMask) //String : 2
        routeType := "OSPF" //3 : String
        for key, _ := range oldEnt.NextHops {
                nextHopIp := convertUint32ToIPv4(key.NextHopIP) //String : 4
                server.logger.Info(fmt.Sprintln("Deleting Route: destNetIp:", destNetIp, "networkMask:", networkMask, "nextHopIp:", nextHopIp, "routeType:", routeType))
                ret, err := server.ribdClient.ClientHdl.DeleteV4Route(destNetIp, networkMask, routeType, nextHopIp)
                if err != nil {
                        server.logger.Err(fmt.Sprintln("Error Installing Route:", err))
                }
                server.logger.Info(fmt.Sprintln("Return Value for RIB DeleteV4Route call: ", ret))
        }
}

func (server *OSPFServer)UpdateRoute(rKey RoutingTblKey) {
        server.logger.Info(fmt.Sprintln("Updating route for rKey:", rKey))
}

func (server *OSPFServer)InstallRoute(rKey RoutingTblKey) {
        server.logger.Info(fmt.Sprintln("Installing new route for rKey", rKey))
        newEnt, exist := server.TempRoutingTbl[rKey]
        if !exist {
                server.logger.Info(fmt.Sprintln("No new routing table entry exist for rkey:", rKey, "hence not installing it"))
                return
        }
        destNetIp := convertUint32ToIPv4(rKey.DestId) //String :1
        networkMask := convertUint32ToIPv4(rKey.AddrMask) //String : 2
        metric := ribd.Int(newEnt.Cost) //int : 3
        routeType := "OSPF" // 7 : String
        for key, _ := range newEnt.NextHops {
                nextHopIp := convertUint32ToIPv4(key.NextHopIP) //String : 4
                ipProp, exist := server.ipPropertyMap[key.IfIPAddr]
                if !exist {
                        server.logger.Err(fmt.Sprintln("Unable to find entry for ip:", key.IfIPAddr, "in ipPropertyMap"))
                        continue
                }
                nextHopIfType := ribd.Int(ipProp.IfType)// ifType int : 5
                nextHopIfIndex := ribd.Int(ipProp.IfId) // Vlan Id int : 6
                server.logger.Info(fmt.Sprintln("Installing Route: destNetIp:", destNetIp, "networkMask:", networkMask, "metric:", metric, "nextHopIp:", nextHopIp, "nextHopIfType:", nextHopIfType, "nextHopIfIndex:", nextHopIfIndex, "routeType:", routeType))
                ret, err := server.ribdClient.ClientHdl.CreateV4Route(destNetIp, networkMask, metric, nextHopIp, nextHopIfType, nextHopIfIndex, routeType)
                if err != nil {
                        server.logger.Err(fmt.Sprintln("Error Installing Route:", err))
                }
                server.logger.Info(fmt.Sprintln("Return Value for RIB CreateV4Route call: ", ret))
        }
}

func (server *OSPFServer)InstallRoutingTbl() {
        server.logger.Info("Installing Routing Table")
        OldRoutingTblKeys := make(map[RoutingTblKey]bool)
        NewRoutingTblKeys := make(map[RoutingTblKey]bool)

        for rKey, rEnt := range server.OldRoutingTbl {
                if rKey.DestType != Network {
                        continue
                }
                if len(rEnt.NextHops) > 0 {
                        OldRoutingTblKeys[rKey] = false
                }
        }
        for rKey, rEnt := range server.TempRoutingTbl {
                if rKey.DestType != Network {
                        continue
                }
                if len(rEnt.NextHops) > 0 {
                        NewRoutingTblKeys[rKey] = false
                }
        }
        for rKey, _ := range NewRoutingTblKeys {
                _, exist := OldRoutingTblKeys[rKey]
                if exist {
                        ret := server.CompareRoutes(rKey)
                        if ret == false { // Old Routes and New Routes are not same
                                server.UpdateRoute(rKey)
                                OldRoutingTblKeys[rKey] = true
                                NewRoutingTblKeys[rKey] = true
                        } else { // Old Routes and New Routes are same
                                OldRoutingTblKeys[rKey] = true
                                NewRoutingTblKeys[rKey] = true
                        }
                }
        }

        for rKey, ent := range OldRoutingTblKeys {
                if ent == false {
                        server.DeleteRoute(rKey)
                }
                OldRoutingTblKeys[rKey]= true
        }

        for rKey, ent := range NewRoutingTblKeys {
                if ent == false {
                        server.InstallRoute(rKey)
                }
                NewRoutingTblKeys[rKey]= true
        }
}
