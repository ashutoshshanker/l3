package server

import (
        //"fmt"
        "errors"
)

func (server *OSPFServer)StopSendHelloPkt(key IntfConfKey) {
        ent, _ := server.IntfConfMap[key]
        if ent.HelloIntervalTicker == nil {
                server.logger.Err("No thread is there to stop.")
                return
        }
        ent.HelloIntervalTicker.Stop()
        server.logger.Info("Successfully stopped sending Hello Pkt")
        ent.HelloIntervalTicker = nil
        server.IntfConfMap[key] = ent
        return
}

func (server *OSPFServer)StartSendHelloPkt(key IntfConfKey) {
        ent, _ := server.IntfConfMap[key]
        //server.logger.Info(fmt.Sprintln("Started Send Hello Pkt Thread", ent.IfName))
        ospfHelloPkt := server.BuildHelloPkt(ent)
        err := server.SendOspfPkt(key, ospfHelloPkt)
        if err != nil {
                server.logger.Err("Unable to send the ospf Hello pkt")
        }
        return
}


func (server *OSPFServer)SendOspfPkt(key IntfConfKey, ospfPkt []byte) error {
        entry, _ := server.IntfTxMap[key]
        handle := entry.SendPcapHdl
        if handle == nil {
                server.logger.Err("Invalid pcap handle")
                err := errors.New("Invalid pcap handle")
                return err
        }
        entry.SendMutex.Lock()
        err := handle.WritePacketData(ospfPkt)
        entry.SendMutex.Unlock()
        return err
}
