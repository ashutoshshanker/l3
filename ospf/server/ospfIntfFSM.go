package server

import (
    "fmt"
    "time"
    "l3/ospf/config"
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
            server.logger.Info(fmt.Sprintf("Transit to backup seen state", msg))
            ent.IfFSMState = config.OtherDesignatedRouter
            server.IntfConfMap[key] = ent
        case createMsg := <-ent.NeighCreateCh:
            neighborKey := NeighborKey {
                RouterId:       createMsg.RouterId,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if !exist {
                neighborEntry.TwoWayStatus = createMsg.TwoWayStatus
                neighborEntry.RtrPrio = createMsg.RtrPrio
                copy(neighborEntry.DRtr, createMsg.DRtr)
                copy(neighborEntry.BDRtr, createMsg.BDRtr)
                ent.NeighborMap[neighborKey] = neighborEntry
                server.IntfConfMap[key] = ent
            }
        case changeMsg:= <-ent.NeighChangeCh:
            neighborKey := NeighborKey {
                RouterId:       changeMsg.RouterId,
            }
            neighborEntry, exist := ent.NeighborMap[neighborKey]
            if !exist {
                if (neighborEntry.RtrPrio != changeMsg.RtrPrio ||
                    bytesEqual(neighborEntry.DRtr, changeMsg.DRtr) == false ||
                    bytesEqual(neighborEntry.BDRtr, changeMsg.BDRtr) == false) &&
                    changeMsg.TwoWayStatus == true {
                    neighborEntry.TwoWayStatus = changeMsg.TwoWayStatus
                    neighborEntry.RtrPrio = changeMsg.RtrPrio
                    copy(neighborEntry.DRtr, changeMsg.DRtr)
                    copy(neighborEntry.BDRtr, changeMsg.BDRtr)
                    ent.NeighborMap[neighborKey] = neighborEntry
                    server.IntfConfMap[key] = ent
                    server.ElectBDRAndDR(key)
                    // Update Neighbor and Re-elect BDR And DR
                }
            }
        case nbrStateChangeMsg := <-ent.NbrStateChangeCh:
            // Elect BDR and DR
            server.logger.Info(fmt.Sprintf("Recev Neighbor State Change message", nbrStateChangeMsg))
            server.ElectBDRAndDR(key)
        case state := <-ent.PktSendCh:
            if state == false {
                server.StopSendHelloPkt(key)
                ent.PktSendStatusCh<-false
                return
            }
        }
    }
}

func (server *OSPFServer)ElectBDRAndDR(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    if ent.IfFSMState == config.OtherDesignatedRouter ||
        ent.IfFSMState == config.DesignatedRouter ||
        ent.IfFSMState == config.BackupDesignatedRouter {
        server.logger.Info("Election of BDR andDR")
    }
    return
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
