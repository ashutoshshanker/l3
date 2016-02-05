package server

import (
        "fmt"
        "l3/ospf/config"
)

type LsdbUpdateMsg struct {
        MsgType         uint8
        AreaId          uint32
        Data            []byte
}

type LSAChangeMsg struct {
        areaId          uint32
}

const (
        LsdbAdd         uint8 = 0
        LsdbDel         uint8 = 1
        LsdbUpdate      uint8 = 2
)

const (
        P2PLink         uint8 = 1
        TransitLink     uint8 = 2
        StubLink        uint8 = 3
        VirtualLink     uint8 = 4
)


func (server *OSPFServer)initLSDatabase(areaId uint32) {
        lsdbKey := LsdbKey {
                AreaId:         areaId,
        }
        lsDbEnt, exist := server.AreaLsdb[lsdbKey]
        if !exist {
                lsDbEnt.RouterLsaMap = make(map[LsaKey]RouterLsa)
                lsDbEnt.NetworkLsaMap = make(map[LsaKey]NetworkLsa)
                lsDbEnt.Summary3LsaMap = make(map[LsaKey]Summary3Lsa)
                lsDbEnt.Summary4LsaMap = make(map[LsaKey]Summary4Lsa)
                lsDbEnt.ASExternalLsaMap = make(map[LsaKey]ASExternalLsa)
                server.AreaLsdb[lsdbKey] = lsDbEnt
        }
        selfOrigLsaEnt, exist := server.AreaSelfOrigLsa[lsdbKey]
        if !exist {
                selfOrigLsaEnt = make(map[LsaKey]bool)
                server.AreaSelfOrigLsa[lsdbKey] = selfOrigLsaEnt
        }
}

func (server *OSPFServer)StartLSDatabase() {
        server.logger.Info("Initializing LSA Database")
        for key, _ := range server.AreaConfMap {
                areaId := convertAreaOrRouterIdUint32(string(key.AreaId))
                server.initLSDatabase(areaId)
        }

        go server.processLSDatabaseUpdates()
        return
}


func (server *OSPFServer)StopLSDatabase() {

}


func (server *OSPFServer)generateRouterLSA(areaId uint32) {
        var linkDetails []LinkDetail = nil
        for _, ent := range server.IntfConfMap {
                AreaId := convertIPv4ToUint32(ent.IfAreaId)
                if areaId != AreaId {
                        continue
                }
                if ent.IfFSMState <= config.Waiting {
                        continue
                }
                var linkDetail LinkDetail
                if ent.IfType == config.Broadcast {
                        if len(ent.NeighborMap) == 0 { // Stub Network
                                server.logger.Info("Stub Network")
                                ipAddr := convertAreaOrRouterIdUint32(ent.IfIpAddr.String())
                                netmask := convertIPv4ToUint32(ent.IfNetmask)
                                linkDetail.LinkId = ipAddr & netmask
                                /* For links to stub networks, this field specifies the stub
                                network’s IP address mask. */
                                linkDetail.LinkData = netmask
                                linkDetail.LinkType = StubLink
                                /* Todo: Need to handle IfMetricConf */
                                linkDetail.NumOfTOS = 0
                                linkDetail.LinkMetric = 10
                        } else { // Transit Network
                                server.logger.Info("Transit Network")
                                linkDetail.LinkId = convertIPv4ToUint32(ent.IfDRIp)
                                /* For links to transit networks, numbered point-to-point links
                                and virtual links, this field specifies the IP interface
                                address of the associated router interface*/
                                linkDetail.LinkData = convertAreaOrRouterIdUint32(ent.IfIpAddr.String())
                                linkDetail.LinkType = TransitLink
                                /* Todo: Need to handle IfMetricConf */
                                linkDetail.NumOfTOS = 0
                                linkDetail.LinkMetric = 10
                        }
                } else if ent.IfType == config.PointToPoint {
                       // linkDetial.LinkId = NBRs Router ID
                }
                linkDetails = append(linkDetails, linkDetail)
        }

        numOfLinks := len(linkDetails)

        LSType := RouterLSA
        LSId := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
        Options := uint8(2) // Need to be revisited 
        LSAge := 0
        AdvRouter := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
        BitE := false //not an AS boundary router (Todo)
        BitB := false //not an Area Border Router (Todo)
        lsaKey :=  LsaKey {
                LSType: LSType,
                LSId:   LSId,
                AdvRouter: AdvRouter,
        }

        lsdbKey := LsdbKey {
                AreaId:         areaId,
        }
        lsDbEnt, _ := server.AreaLsdb[lsdbKey]
        selfOrigLsaEnt, _ := server.AreaSelfOrigLsa[lsdbKey]

        if numOfLinks == 0 {
                delete(lsDbEnt.RouterLsaMap, lsaKey)
                delete(selfOrigLsaEnt, lsaKey)
                server.AreaSelfOrigLsa[lsdbKey] = selfOrigLsaEnt
                server.AreaLsdb[lsdbKey] = lsDbEnt
                return
        }
        ent, exist := lsDbEnt.RouterLsaMap[lsaKey]
        ent.LsaMd.LSAge = 0
        ent.LsaMd.Options = Options
        if !exist {
                ent.LsaMd.LSSequenceNum = InitialSequenceNumber
        } else {
                ent.LsaMd.LSSequenceNum = ent.LsaMd.LSSequenceNum + 1
        }
        ent.LsaMd.LSChecksum = 0
        // Length of Per Link Details = 12 bytes
        // Length of Router LSA Metadata (BitE, BitB, NumofLinks)  = 4 bytes
        ent.LsaMd.LSLen = uint16(OSPF_LSA_HEADER_LEN + 4 + (12 * numOfLinks))
        ent.BitE = BitE
        ent.BitB = BitB
        ent.NumOfLinks = uint16(numOfLinks)
        ent.LinkDetails = make([]LinkDetail, numOfLinks)
        copy(ent.LinkDetails, linkDetails[0:])
        server.logger.Info(fmt.Sprintln("Hello... LinkDetails:", ent.LinkDetails))
        selfOrigLsaEnt[lsaKey] = true
        server.AreaSelfOrigLsa[lsdbKey] = selfOrigLsaEnt
        server.logger.Info(fmt.Sprintln("Self Originated Router LSA Key:", server.AreaSelfOrigLsa[lsdbKey]))
        LsaEnc := encodeRouterLsa(ent, lsaKey)
        ent.LsaMd.LSChecksum = computeFletcherChecksum(LsaEnc)
        ent.LsaMd.LSAge = uint16(LSAge)
        lsDbEnt.RouterLsaMap[lsaKey] = ent
        server.AreaLsdb[lsdbKey] = lsDbEnt
        return
}

func (server *OSPFServer)processNewRecvdRouterLsa(data []byte, areaId uint32) bool {
        lsakey := NewLsaKey()
        routerLsa := NewRouterLsa()
        decodeRouterLsa(data, routerLsa, lsakey)
        //Check Checksum
        //If there is already existing entry Verify the seq num
        //Handle LsaAge
        //Add entry in LSADatabase 
        return true
}

func (server *OSPFServer)processNewRecvdNetworkLsa(data []byte, areaId uint32) bool {

        return true
}

func (server *OSPFServer)processNewRecvdSummary3Lsa(data []byte, areaId uint32) bool {

        return true
}

func (server *OSPFServer)processNewRecvdSummary4Lsa(data []byte, areaId uint32) bool {

        return true
}

func (server *OSPFServer)processNewRecvdASExternalLsa(data []byte, areaId uint32) bool {

        return true
}

func (server *OSPFServer)processNewRecvdLsa(data []byte, areaId uint32) bool {
        LSType := uint8(data[3])
        if LSType == RouterLSA {
                return server.processNewRecvdRouterLsa(data, areaId)
        } else if LSType == NetworkLSA {
                return server.processNewRecvdNetworkLsa(data, areaId)
        } else if LSType == Summary3LSA {
                return server.processNewRecvdSummary3Lsa(data, areaId)
        } else if LSType == Summary4LSA {
                return server.processNewRecvdSummary4Lsa(data, areaId)
        } else if LSType == ASExternalLSA {
                return server.processNewRecvdASExternalLsa(data, areaId)
        } else {
                return false
        }
}

func (server *OSPFServer)processLSDatabaseUpdates() {
        for {
                select {
                case msg := <-server.LsdbUpdateCh:
                        if msg.MsgType == LsdbAdd {
                                server.logger.Info("Adding LS in the Lsdb")
                                server.logger.Info("Received New LSA")
                                ret := server.processNewRecvdLsa(msg.Data, msg.AreaId)
                                server.LsaUpdateRetCodeCh <- ret
                        } else if msg.MsgType == LsdbDel {
                                server.logger.Info("Deleting LS in the Lsdb")
                        } else if msg.MsgType == LsdbUpdate {
                                server.logger.Info("Deleting LS in the Lsdb")
                        }
                case msg := <-server.IntfStateChangeCh:
                        server.logger.Info(fmt.Sprintf("Interface State change msg", msg))
                        server.generateRouterLSA(msg.areaId)
                        server.logger.Info(fmt.Sprintln("LS Database", server.AreaLsdb))
                case msg := <-server.NetworkDRChangeCh:
                        server.logger.Info(fmt.Sprintf("Network DR change msg", msg))
                        // Create a new router LSA
                        server.generateRouterLSA(msg.areaId)
                        server.logger.Info(fmt.Sprintln("LS Database", server.AreaLsdb))
                case msg := <-server.CreateNetworkLSACh:
                        server.logger.Info(fmt.Sprintf("Create Network LSA msg", msg))
                        // Flush the old Network LSA
                        // Check if link is broadcast or not
                        // If link is broadcast
                        // Create Network LSA
                case msg := <-server.FlushNetworkLSACh:
                        server.logger.Info(fmt.Sprintf("Flush Network LSA msg", msg))
                        // Flush the old Network LSA
                }
        }
}
