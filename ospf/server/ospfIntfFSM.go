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
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if !exist {
                neighborEntry.TwoWayStatus = createMsg.TwoWayStatus
                neighborEntry.RtrPrio = createMsg.RtrPrio
                neighborEntry.DRtr = createMsg.DRtr
                neighborEntry.BDRtr = createMsg.BDRtr
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
                server.logger.Info(fmt.Sprintln("1 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap))
                if createMsg.TwoWayStatus == true &&
                    ent.IfFSMState != config.Waiting {
                    server.ElectBDRAndDR(key)
                }
            }
        case changeMsg:= <-ent.NeighChangeCh:
            neighborKey := NeighborKey {
                RouterId:       changeMsg.RouterId,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if exist {
                server.logger.Info(fmt.Sprintln("Change msg: ", changeMsg, "neighbor entry:", neighborEntry))
                rtrId := changeMsg.RouterId
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
                neighborEntry.TwoWayStatus = changeMsg.TwoWayStatus
                neighborEntry.RtrPrio = changeMsg.RtrPrio
                neighborEntry.DRtr = changeMsg.DRtr
                neighborEntry.BDRtr = changeMsg.BDRtr
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
                server.logger.Info(fmt.Sprintln("2 IntfConf neighbor entry", server.IntfConfMap[key].NeighborMap))
                if ent.IfFSMState > config.Waiting {
                    // RFC2328 Section 9.2 (Neighbor Change Event)
                    if (oldDRtr == rtrId && newDRtr != rtrId && oldTwoWayStatus == true) ||
                        (oldDRtr != rtrId && newDRtr == rtrId && oldTwoWayStatus == true) ||
                        (oldBDRtr == rtrId && newBDRtr != rtrId && oldTwoWayStatus == true) ||
                        (oldBDRtr != rtrId && newBDRtr == rtrId && oldTwoWayStatus == true) ||
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
            server.logger.Info("Hello4")
            server.logger.Info(fmt.Sprintf("Recev Neighbor State Change message", nbrStateChangeMsg))
            nbrKey := NeighborKey {
                RouterId:   nbrStateChangeMsg.RouterId,
            }
            neighborEntry, exist := ent.NeighborMap[nbrKey]
            if exist {
                oldTwoWayStatus := neighborEntry.TwoWayStatus
                delete(ent.NeighborMap, nbrKey)
                server.IntfConfMap[key] = ent
                if ent.IfFSMState > config.Waiting {
                    // RFC2328 Section 9.2 (Neighbor Change Event)
                    if oldTwoWayStatus == true {
                        server.ElectBDRAndDR(key)
                    }
                }
            }
        case state := <-ent.PktSendCh:
            if state == false {
                server.StopSendHelloPkt(key)
                ent.PktSendStatusCh<-false
                return
            }
        }
    }
}

func (server *OSPFServer)ElectBDR(key IntfConfKey) ([]byte) {
    ent, _ := server.IntfConfMap[key]
    electedBDR := []byte {0, 0, 0, 0}
    var RtrPrio uint8
    var MaxRtrPrio uint8
    var RtrWithMaxPrio uint32

    for nbrkey, nbrEntry := range ent.NeighborMap {
        if nbrEntry.TwoWayStatus == true &&
            nbrEntry.RtrPrio > 0 &&
            nbrkey.RouterId != 0 {
            tempDR := binary.BigEndian.Uint32(nbrEntry.DRtr)
            if tempDR == nbrkey.RouterId {
                continue
            }
            tempBDR := binary.BigEndian.Uint32(nbrEntry.BDRtr)
            if tempBDR == nbrkey.RouterId {
                if nbrEntry.RtrPrio > RtrPrio {
                    RtrPrio = nbrEntry.RtrPrio
                    electedBDR = nbrEntry.BDRtr
                } else if nbrEntry.RtrPrio == RtrPrio {
                    tempBDRtr := convertIPv4ToUint32(electedBDR)
                    curBDRtr := convertIPv4ToUint32(nbrEntry.BDRtr)
                    if tempBDRtr < curBDRtr {
                        electedBDR = nbrEntry.BDRtr
                    }
                }
            }
            if MaxRtrPrio < nbrEntry.RtrPrio {
                MaxRtrPrio = nbrEntry.RtrPrio
                RtrWithMaxPrio = nbrkey.RouterId
            } else if MaxRtrPrio == nbrEntry.RtrPrio {
                if RtrWithMaxPrio < nbrkey.RouterId {
                    RtrWithMaxPrio = nbrkey.RouterId
                }
            }
        }
    }

    if ent.IfRtrPriority != 0 &&
        bytesEqual(server.ospfGlobalConf.RouterId, []byte {0, 0, 0, 0}) == false {
        if bytesEqual(server.ospfGlobalConf.RouterId, ent.IfDR) == false {
            if bytesEqual(server.ospfGlobalConf.RouterId, ent.IfBDR) == true {
                if ent.IfRtrPriority > RtrPrio {
                    RtrPrio = ent.IfRtrPriority
                    electedBDR = server.ospfGlobalConf.RouterId
                } else if ent.IfRtrPriority == RtrPrio {
                    tempBDRtr := convertIPv4ToUint32(electedBDR)
                    curBDRtr := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
                    if tempBDRtr < curBDRtr {
                        electedBDR = server.ospfGlobalConf.RouterId
                    }
                }
            }

            tempRtrId := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
            if MaxRtrPrio < ent.IfRtrPriority {
                RtrWithMaxPrio = tempRtrId
            } else if MaxRtrPrio == ent.IfRtrPriority {
                if RtrWithMaxPrio < tempRtrId {
                    RtrWithMaxPrio = tempRtrId
                }
            }

        }
    }
    if bytesEqual(electedBDR, []byte{0, 0, 0, 0}) == true {
        binary.BigEndian.PutUint32(electedBDR, RtrWithMaxPrio)
    }

    return electedBDR
}

func (server *OSPFServer)ElectDR(key IntfConfKey, electedBDR []byte) ([]byte) {
    ent, _ := server.IntfConfMap[key]
    electedDR := []byte {0, 0, 0, 0}
    var RtrPrio uint8

    for key, nbrEntry := range ent.NeighborMap {
        if nbrEntry.TwoWayStatus == true &&
            nbrEntry.RtrPrio > 0  &&
            key.RouterId != 0 {
            tempDR := binary.BigEndian.Uint32(nbrEntry.DRtr)
            if tempDR == key.RouterId {
                if nbrEntry.RtrPrio > RtrPrio {
                    RtrPrio = nbrEntry.RtrPrio
                    electedDR = nbrEntry.DRtr
                } else if nbrEntry.RtrPrio == RtrPrio {
                    tempDRtr := convertIPv4ToUint32(electedDR)
                    curDRtr := convertIPv4ToUint32(nbrEntry.DRtr)
                    if tempDRtr < curDRtr {
                        electedDR = nbrEntry.DRtr
                    }
                }
            }
        }
    }

    if ent.IfRtrPriority > 0 &&
        bytesEqual(server.ospfGlobalConf.RouterId, []byte {0, 0, 0, 0}) == false {
        if bytesEqual(server.ospfGlobalConf.RouterId, ent.IfDR) == true {
            if ent.IfRtrPriority > RtrPrio {
                RtrPrio = ent.IfRtrPriority
                electedDR = server.ospfGlobalConf.RouterId
            } else if ent.IfRtrPriority == RtrPrio {
                tempDRtr := convertIPv4ToUint32(electedDR)
                curDRtr := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
                if tempDRtr < curDRtr {
                    electedDR = server.ospfGlobalConf.RouterId
                }
            }
        }
    }

    if bytesEqual(electedDR, []byte{0, 0, 0, 0}) == true {
        electedDR = electedBDR
    }
    return electedDR
}

func (server *OSPFServer)ElectBDRAndDR(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    server.logger.Info(fmt.Sprintln("Election of BDR andDR", ent.IfFSMState))

    //oldDR := ent.IfDR
    //oldBDR := ent.IfBDR
    oldState := ent.IfFSMState
    var newState config.IfState

    electedBDR := server.ElectBDR(key)
    ent.IfBDR = electedBDR
    electedDR := server.ElectDR(key, electedBDR)
    ent.IfDR = electedDR
    if bytesEqual(ent.IfDR, server.ospfGlobalConf.RouterId) == true {
        newState = config.DesignatedRouter
    } else if bytesEqual(ent.IfBDR, server.ospfGlobalConf.RouterId) == true {
        newState = config.BackupDesignatedRouter
    } else {
        newState = config.OtherDesignatedRouter
    }

    server.logger.Info(fmt.Sprintln("1. Election of BDR:", ent.IfBDR, " and DR:", ent.IfDR, "new State:", newState))
    server.IntfConfMap[key] = ent

    if (newState != oldState &&
        !(newState == config.OtherDesignatedRouter &&
            oldState < config.OtherDesignatedRouter)) {
        ent, _ = server.IntfConfMap[key]
        electedBDR = server.ElectBDR(key)
        ent.IfBDR = electedBDR
        electedDR = server.ElectDR(key, electedBDR)
        ent.IfDR = electedDR
        if bytesEqual(ent.IfDR, server.ospfGlobalConf.RouterId) == true {
            newState = config.DesignatedRouter
        } else if bytesEqual(ent.IfBDR, server.ospfGlobalConf.RouterId) == true {
            newState = config.BackupDesignatedRouter
        } else {
            newState = config.OtherDesignatedRouter
        }
        server.logger.Info(fmt.Sprintln("2. Election of BDR:", ent.IfBDR, " and DR:", ent.IfDR, "new State:", newState))
        server.IntfConfMap[key] = ent
    }

    ent, _ = server.IntfConfMap[key]
    ent.IfFSMState = newState
    server.logger.Info(fmt.Sprintln("Final Election of BDR:", ent.IfBDR, " and DR:", ent.IfDR, "new State:", newState))
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
