package server

import (
    //"fmt"
    "time"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
)

func (server *OSPFServer)StartOspfTransPkts(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
/*
    waitTimerCb := func() {
        server.logger.Info("Wait timer expired")
        ent.WaitTimerExpired <- true
    }
*/
    for {
        select {
        case <-ent.HelloIntervalTicker.C:
            server.StartSendHelloPkt(key)
        case <-ent.WaitTimer.C:
            server.logger.Info("Wait timer expired")
        case state := <-ent.PktSendCh:
            if state == false {
                server.StopSendHelloPkt(key)
                ent.PktSendStatusCh<-false
                return
            }
        }
    }
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
