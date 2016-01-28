package server

import (
    "fmt"
    "time"
    "l3/ospf/config"
    "encoding/binary"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
)


func (server *OSPFServer)StartOspfTransPkts(key IntfConfKey) {
/*
    waitTimerCb := func() {
        server.logger.Info("Wait timer expired")
        ent.WaitTimerExpired <- true
    }
*/
    for {
        ent, _ := server.IntfConfMap[key]
        select {
        case <-ent.HelloIntervalTicker.C:
            server.StartSendHelloPkt(key)
        case <-ent.WaitTimer.C:
            server.logger.Info("Wait timer expired")
            //server.IntfConfMap[key] = ent
            // Elect BDR And DR
            server.ElectBDRAndDR(key)
        case msg := <-ent.BackupSeenCh:
            server.logger.Info(fmt.Sprintf("Transit to action state because of backup seen", msg))
/*
            ent.IfFSMState = config.OtherDesignatedRouter
*/
            server.ElectBDRAndDR(key)
        case createMsg := <-ent.NeighCreateCh:
            neighborKey := NeighborKey {
                RouterId:       createMsg.RouterId,
                //NbrIP:          createMsg.NbrIP,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if !exist {
                neighborEntry.NbrIP = createMsg.NbrIP
                neighborEntry.TwoWayStatus = createMsg.TwoWayStatus
                neighborEntry.RtrPrio = createMsg.RtrPrio
                neighborEntry.DRtr = createMsg.DRtr
                neighborEntry.BDRtr = createMsg.BDRtr
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
                server.logger.Info(fmt.Sprintln("1 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap, "neighborKey:", neighborKey))
                if createMsg.TwoWayStatus == true &&
                    ent.IfFSMState > config.Waiting {
                    server.ElectBDRAndDR(key)
                }
            }
        case changeMsg:= <-ent.NeighChangeCh:
            neighborKey := NeighborKey {
                RouterId:       changeMsg.RouterId,
                //NbrIP:          changeMsg.NbrIP,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if exist {
                server.logger.Info(fmt.Sprintln("Change msg: ", changeMsg, "neighbor entry:", neighborEntry, "neighbor key:", neighborKey))
                //rtrId := changeMsg.RouterId
                NbrIP := changeMsg.NbrIP
                oldRtrPrio := neighborEntry.RtrPrio
                oldDRtr := binary.BigEndian.Uint32(neighborEntry.DRtr)
                oldBDRtr := binary.BigEndian.Uint32(neighborEntry.BDRtr)
                newDRtr := binary.BigEndian.Uint32(changeMsg.DRtr)
                newBDRtr := binary.BigEndian.Uint32(changeMsg.BDRtr)
                oldTwoWayStatus := neighborEntry.TwoWayStatus
/*
                if (neighborEntry.RtrPrio != changeMsg.RtrPrio ||
                    bytesEqual(neighborEntry.DRtr, changeMsg.DRtr) == false ||
                    bytesEqual(neighborEntry.BDRtr, changeMsg.BDRtr) == false) &&
                    changeMsg.TwoWayStatus == true {
*/
                neighborEntry.NbrIP = changeMsg.NbrIP
                neighborEntry.TwoWayStatus = changeMsg.TwoWayStatus
                neighborEntry.RtrPrio = changeMsg.RtrPrio
                neighborEntry.DRtr = changeMsg.DRtr
                neighborEntry.BDRtr = changeMsg.BDRtr
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
                server.logger.Info(fmt.Sprintln("2 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap))
                if ent.IfFSMState > config.Waiting {
                    // RFC2328 Section 9.2 (Neighbor Change Event)
                    if (oldDRtr == NbrIP && newDRtr != NbrIP && oldTwoWayStatus == true) ||
                        (oldDRtr != NbrIP && newDRtr == NbrIP && oldTwoWayStatus == true) ||
                        (oldBDRtr == NbrIP && newBDRtr != NbrIP && oldTwoWayStatus == true) ||
                        (oldBDRtr != NbrIP && newBDRtr == NbrIP && oldTwoWayStatus == true) ||
                        (oldTwoWayStatus != changeMsg.TwoWayStatus) ||
                        (oldRtrPrio != changeMsg.RtrPrio && oldTwoWayStatus == true) {

                        // Update Neighbor and Re-elect BDR And DR
                        server.ElectBDRAndDR(key)
                    }
                }
/*
                }
*/
            }
        case nbrStateChangeMsg := <-ent.NbrStateChangeCh:
            // Only when Neighbor Went Down from TwoWayStatus
            // Todo: Handle NbrIP: Ashutosh
            server.logger.Info(fmt.Sprintf("Recev Neighbor State Change message", nbrStateChangeMsg))
            nbrKey := NeighborKey {
                RouterId:   nbrStateChangeMsg.RouterId,
            }
            neighborEntry, exist := ent.NeighborMap[nbrKey]
            if exist {
                oldTwoWayStatus := neighborEntry.TwoWayStatus
                delete(ent.NeighborMap, nbrKey)
                server.logger.Info(fmt.Sprintln("Deleting", nbrKey))
                server.IntfConfMap[key] = ent
                if ent.IfFSMState > config.Waiting {
                    // RFC2328 Section 9.2 (Neighbor Change Event)
                    if oldTwoWayStatus == true {
                        server.ElectBDRAndDR(key)
                    }
                }
            }
            server.logger.Info(fmt.Sprintln("Hello", server.IntfConfMap))
        case state := <-ent.PktSendCh:
            if state == false {
                server.StopSendHelloPkt(key)
                ent.PktSendStatusCh<-false
                return
            }
        }
    }
}

func (server *OSPFServer)ElectBDR(key IntfConfKey) ([]byte, uint32) {
    ent, _ := server.IntfConfMap[key]
    electedBDR := []byte {0, 0, 0, 0}
    var electedRtrPrio uint8
    var electedRtrId uint32
    var MaxRtrPrio uint8
    var RtrIdWithMaxPrio uint32
    var NbrIPWithMaxPrio uint32

    for nbrkey, nbrEntry := range ent.NeighborMap {
        if nbrEntry.TwoWayStatus == true &&
            nbrEntry.RtrPrio > 0 &&
            nbrEntry.NbrIP != 0 {
            tempDR := binary.BigEndian.Uint32(nbrEntry.DRtr)
            if tempDR == nbrEntry.NbrIP {
                continue
            }
            tempBDR := binary.BigEndian.Uint32(nbrEntry.BDRtr)
            if tempBDR == nbrEntry.NbrIP {
                if nbrEntry.RtrPrio > electedRtrPrio {
                    electedRtrPrio = nbrEntry.RtrPrio
                    electedRtrId = nbrkey.RouterId
                    electedBDR = nbrEntry.BDRtr
                } else if nbrEntry.RtrPrio == electedRtrPrio {
                    if electedRtrId < nbrkey.RouterId {
                        electedRtrPrio = nbrEntry.RtrPrio
                        electedRtrId = nbrkey.RouterId
                        electedBDR = nbrEntry.BDRtr
                    }
                }
            }
            if MaxRtrPrio < nbrEntry.RtrPrio {
                MaxRtrPrio = nbrEntry.RtrPrio
                RtrIdWithMaxPrio = nbrkey.RouterId
                NbrIPWithMaxPrio = nbrEntry.NbrIP
            } else if MaxRtrPrio == nbrEntry.RtrPrio {
                if RtrIdWithMaxPrio < nbrkey.RouterId {
                    MaxRtrPrio = nbrEntry.RtrPrio
                    RtrIdWithMaxPrio = nbrkey.RouterId
                    NbrIPWithMaxPrio = nbrEntry.NbrIP
                }
            }
        }
    }

    if ent.IfRtrPriority != 0 &&
        bytesEqual(ent.IfIpAddr.To4(), []byte {0, 0, 0, 0}) == false {
        if bytesEqual(ent.IfIpAddr.To4(), ent.IfDRIp) == false {
            if bytesEqual(ent.IfIpAddr.To4(), ent.IfBDRIp) == true {
                rtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
                if ent.IfRtrPriority > electedRtrPrio {
                    electedRtrPrio = ent.IfRtrPriority
                    electedRtrId = rtrId
                    electedBDR = ent.IfIpAddr.To4()
                } else if ent.IfRtrPriority == electedRtrPrio {
                    if electedRtrId < rtrId {
                        electedRtrPrio = ent.IfRtrPriority
                        electedRtrId = rtrId
                        electedBDR = ent.IfIpAddr.To4()
                    }
                }
            }

            tempRtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
            if MaxRtrPrio < ent.IfRtrPriority {
                MaxRtrPrio = ent.IfRtrPriority
                NbrIPWithMaxPrio = binary.BigEndian.Uint32(ent.IfIpAddr.To4())
                RtrIdWithMaxPrio = tempRtrId
            } else if MaxRtrPrio == ent.IfRtrPriority {
                if RtrIdWithMaxPrio < tempRtrId {
                    MaxRtrPrio = ent.IfRtrPriority
                    NbrIPWithMaxPrio = binary.BigEndian.Uint32(ent.IfIpAddr.To4())
                    RtrIdWithMaxPrio = tempRtrId
                }
            }

        }
    }
    if bytesEqual(electedBDR, []byte{0, 0, 0, 0}) == true {
        binary.BigEndian.PutUint32(electedBDR, NbrIPWithMaxPrio)
        electedRtrId = RtrIdWithMaxPrio
    }

    return electedBDR, electedRtrId
}

func (server *OSPFServer)ElectDR(key IntfConfKey, electedBDR []byte, electedBDRtrId  uint32) ([]byte, uint32) {
    ent, _ := server.IntfConfMap[key]
    electedDR := []byte {0, 0, 0, 0}
    var electedRtrPrio uint8
    var electedDRtrId uint32

    for key, nbrEntry := range ent.NeighborMap {
        if nbrEntry.TwoWayStatus == true &&
            nbrEntry.RtrPrio > 0  &&
            nbrEntry.NbrIP != 0 {
            tempDR := binary.BigEndian.Uint32(nbrEntry.DRtr)
            if tempDR == nbrEntry.NbrIP {
                if nbrEntry.RtrPrio > electedRtrPrio {
                    electedRtrPrio = nbrEntry.RtrPrio
                    electedDRtrId = key.RouterId
                    electedDR = nbrEntry.DRtr
                } else if nbrEntry.RtrPrio == electedRtrPrio {
                    if electedDRtrId < key.RouterId {
                        electedRtrPrio = nbrEntry.RtrPrio
                        electedDRtrId = key.RouterId
                        electedDR = nbrEntry.DRtr
                    }
                }
            }
        }
    }

    if ent.IfRtrPriority > 0 &&
        bytesEqual(ent.IfIpAddr.To4(), []byte {0, 0, 0, 0}) == false {
        if bytesEqual(ent.IfIpAddr.To4(), ent.IfDRIp) == true {
            rtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
            if ent.IfRtrPriority > electedRtrPrio {
                electedRtrPrio = ent.IfRtrPriority
                electedDRtrId = rtrId
                electedDR = ent.IfIpAddr.To4()
            } else if ent.IfRtrPriority == electedRtrPrio {
                if electedDRtrId < rtrId {
                    electedRtrPrio = ent.IfRtrPriority
                    electedDRtrId = rtrId
                    electedDR = ent.IfIpAddr.To4()
                }
            }
        }
    }

    if bytesEqual(electedDR, []byte{0, 0, 0, 0}) == true {
        electedDR = electedBDR
        electedDRtrId = electedBDRtrId
    }
    return electedDR, electedDRtrId
}

func (server *OSPFServer)ElectBDRAndDR(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    server.logger.Info(fmt.Sprintln("Election of BDR andDR", ent.IfFSMState))

    //oldDR := ent.IfDRIp
    //oldBDR := ent.IfBDRIp
    oldState := ent.IfFSMState
    var newState config.IfState

    electedBDR, electedBDRtrId := server.ElectBDR(key)
    ent.IfBDRIp = electedBDR
    ent.IfBDRtrId = electedBDRtrId
    electedDR, electedDRtrId := server.ElectDR(key, electedBDR, electedBDRtrId)
    ent.IfDRIp = electedDR
    ent.IfDRtrId = electedDRtrId
    if bytesEqual(ent.IfDRIp, ent.IfIpAddr.To4()) == true {
        newState = config.DesignatedRouter
    } else if bytesEqual(ent.IfBDRIp, ent.IfIpAddr.To4()) == true {
        newState = config.BackupDesignatedRouter
    } else {
        newState = config.OtherDesignatedRouter
    }

    server.logger.Info(fmt.Sprintln("1. Election of BDR:", ent.IfBDRIp, " and DR:", ent.IfDRIp, "new State:", newState, "DR Id:", ent.IfDRtrId, "BDR Id:", ent.IfBDRtrId))
    server.IntfConfMap[key] = ent

    if (newState != oldState &&
        !(newState == config.OtherDesignatedRouter &&
            oldState < config.OtherDesignatedRouter)) {
        ent, _ = server.IntfConfMap[key]
        electedBDR, electedBDRtrId = server.ElectBDR(key)
        ent.IfBDRIp = electedBDR
        ent.IfBDRtrId = electedBDRtrId
        electedDR, electedDRtrId = server.ElectDR(key, electedBDR, electedBDRtrId)
        ent.IfDRIp = electedDR
        ent.IfDRtrId = electedDRtrId
        if bytesEqual(ent.IfDRIp, ent.IfIpAddr.To4()) == true {
            newState = config.DesignatedRouter
        } else if bytesEqual(ent.IfBDRIp, ent.IfIpAddr.To4()) == true {
            newState = config.BackupDesignatedRouter
        } else {
            newState = config.OtherDesignatedRouter
        }
        server.logger.Info(fmt.Sprintln("2. Election of BDR:", ent.IfBDRIp, " and DR:", ent.IfDRIp, "new State:", newState, "DR Id:", ent.IfDRtrId, "BDR Id:", ent.IfBDRtrId))
        server.IntfConfMap[key] = ent
    }

    ent, _ = server.IntfConfMap[key]
    ent.IfFSMState = newState
    // Need to Check: do we need to add events even when we
    // come back to same state after DR or BDR Election
    ent.IfEvents = ent.IfEvents + 1
    server.logger.Info(fmt.Sprintln("Final Election of BDR:", ent.IfBDRIp, " and DR:", ent.IfDRIp, "new State:", newState))
    server.IntfConfMap[key] = ent
}

func (server *OSPFServer)StopOspfTransPkts(key IntfConfKey) {
   // server.StopSendHelloPkt(key)
    ent, _ := server.IntfConfMap[key]
    ent.PktSendCh<-false
    cnt := 0
    for {
        select {
        case status := <-ent.PktSendStatusCh:
            if status == false { // False Means Trans Pkt Thread Stopped
                server.logger.Info("Stopped Sending Hello Pkt")
                return
            }
        default:
            time.Sleep(time.Duration(10) * time.Millisecond)
            cnt = cnt + 1
            if cnt == 100 {
                server.logger.Err("Unable to stop the Tx thread")
                return
            }
        }
    }
}

func (server *OSPFServer)StopOspfRecvPkts(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    ent.PktRecvCh<-false
    cnt := 0
    for {
        select {
        case status := <-ent.PktRecvStatusCh:
            if status == false { // False Means Recv Pkt Thread Stopped
                server.logger.Info("Stopped Recv Pkt thread")
                return
            }
        default:
            time.Sleep(time.Duration(10) * time.Millisecond)
            cnt = cnt + 1
            if cnt == 100 {
                server.logger.Err("Unable to stop the Rx thread")
                return
            }
        }
    }
}

func (server *OSPFServer)StartOspfRecvPkts(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    handle := ent.RecvPcapHdl
    recv := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
    in := recv.Packets()
    for {
        select {
        case packet, ok := <-in:
            if ok {
                server.logger.Info("Got Some Ospf Packet on the Recv Thread")
                go server.ProcessOspfRecvPkt(key, packet)
            }
        case state := <-ent.PktRecvCh:
            if state == false {
                server.logger.Info("Stopping the Recv Ospf packet thread")
                ent.PktRecvStatusCh<-false
                return
            }
        }
    }
}
