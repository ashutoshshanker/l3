package server

import (
        "fmt"
        "l3/ospf/config"
)

type LsaKey struct {
        LSType          uint8 /* LS Type */
        LSId            uint32 /* Link State Id */
        AdvRouter       uint32 /* Avertising Router */
}

type LinkDetail struct {
        LinkId          uint32 /* Link ID */
        LinkData        uint32 /* Link Data */
        LinkType        uint8 /* Link Type */
        TOSMetric       uint8 /* # TOS Metrics */
        LinkMetric      uint16 /* Metric */
}

/* LS Type 1 */
type RouterLsa struct {
        LSAge           uint16 /* LS Age */
        Options         uint8 /* Options */
        BitE            bool /* Bit E */
        BitB            bool /* Bit B */
        NumofLinks      uint16 /* NumOfLinks */
        LinkDetails     []LinkDetail /* List of LinkDetails */
}

/* LS Type 2 */
type NetworkLsa struct {
        /* LS Age */
        /* Options */
        /* Network Mask */
        /* List of attached Routers */
}

/* LS Type 3 */
type Summary3Lsa struct {

}

/* LS Type 4 */
type Summary4Lsa struct {

}

/* LS Type 5 */
type ASExternalLsa struct {

}

type LSDatabase struct {
        RouterLsaMap            map[LsaKey]RouterLsa
        NetworkLsaMap           map[LsaKey]NetworkLsa
        Summary3LsaMap          map[LsaKey]Summary3Lsa
        Summary4LsaMap          map[LsaKey]Summary4Lsa
        ASExternalLsaMap        map[LsaKey]ASExternalLsa
}

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

type LsdbUpdateMsg struct {
        MsgType         uint8
        AreaId          uint32
        LsaKey          LsaKey
        Msg             []byte
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

const (
        RouterLSA               uint8 = 1
        NetworkLSA              uint8 = 2
        Summary3LSA             uint8 = 3
        Summary4LSA             uint8 = 4
        ASExternalLSA           uint8 = 5
)

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
                                linkDetail.TOSMetric = 0
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
                                linkDetail.TOSMetric = 0
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

        if numOfLinks == 0 {
                delete(lsDbEnt.RouterLsaMap, lsaKey)
                server.AreaLsdb[lsdbKey] = lsDbEnt
                return
        }
        ent, _ := lsDbEnt.RouterLsaMap[lsaKey]
        ent.LSAge = uint16(LSAge)
        ent.Options = Options
        ent.BitE = BitE
        ent.BitB = BitB
        ent.NumofLinks = uint16(numOfLinks)
        ent.LinkDetails = make([]LinkDetail, numOfLinks)
        copy(ent.LinkDetails, linkDetails[0:])
        server.logger.Info(fmt.Sprintln("Hello... LinkDetails:", ent.LinkDetails))
        lsDbEnt.RouterLsaMap[lsaKey] = ent
        server.AreaLsdb[lsdbKey] = lsDbEnt
        return
}

func (server *OSPFServer)processLSDatabaseUpdates() {
        for {
                select {
                case msg := <-server.LsdbUpdateCh:
                        if msg.MsgType == LsdbAdd {
                                server.logger.Info("Adding LS in the Lsdb")
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
