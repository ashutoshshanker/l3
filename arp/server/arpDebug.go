package server

import (
	"fmt"
)

func (server *ARPServer) printArpEntries() {
	server.logger.Debug("************")
	for ip, arpEnt := range server.arpCache {
		server.logger.Debug(fmt.Sprintln("IP:", ip, "MAC:", arpEnt.MacAddr, "VlanId:", arpEnt.VlanId, "IfName:", arpEnt.IfName, "IfIndex", arpEnt.L3IfIdx, "Counter:", arpEnt.Counter, "Timestamp:", arpEnt.TimeStamp, "PortNum:", arpEnt.PortNum))
	}
	server.logger.Debug("************")
}
