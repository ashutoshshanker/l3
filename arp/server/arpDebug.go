package server

import (
        "fmt"
)

func (server *ARPServer) printArpEntries() {
        server.logger.Info("************")
        for ip, arpEnt := range server.arpCache {
                server.logger.Info(fmt.Sprintln("IP:", ip, "MAC:", arpEnt.MacAddr, "VlanId:", arpEnt.VlanId, "IfName:", arpEnt.IfName, "IfIndex", arpEnt.L3IfIdx, "Counter:", arpEnt.Counter, "Timestamp:", arpEnt.TimeStamp, "PortNum:", arpEnt.PortNum))
        }
        server.logger.Info("************")
}

