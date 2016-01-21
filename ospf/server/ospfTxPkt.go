package server

import (
    "fmt"
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
    server.logger.Info(fmt.Sprintln("Started Send Hello Pkt Thread", ent.IfName))
    handle := ent.SendPcapHdl
    ospfHelloPkt := server.BuildHelloPkt(ent)
    if handle == nil {
        server.logger.Err("Invalid pcap handle")
        return
    }
    if err := handle.WritePacketData(ospfHelloPkt); err != nil {
        server.logger.Err("Unable to send the hello pkt")
    }
    return
}

