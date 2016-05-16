package server

import (
	"asicd/asicdCommonDefs"
	"fmt"
	//"github.com/google/gopacket/pcap"
	"utils/commonDefs"
)

func (server *DHCPServer) StopAllDhcpServer() {
	for _, dEnt := range server.DhcpIntfConfMap {
		if dEnt.enable == true {
			server.StopDhcpServer(dEnt.l3IfIdx)
		}
	}
}

func (server *DHCPServer) StopDhcpServer(ifIdx int32) {
	ifType := asicdCommonDefs.GetIntfTypeFromIfIndex(ifIdx)
	if ifType == commonDefs.IfTypeVlan {
		vlanEnt, _ := server.vlanPropertyMap[ifIdx]
		for port, _ := range vlanEnt.UntagPortMap {
			portEnt := server.portPropertyMap[port]
			if portEnt.PcapHdl != nil {
				portEnt.CtrlCh <- true
				<-portEnt.CtrlCh
			}
		}
	} else if ifType == commonDefs.IfTypeLag {
		lagEnt, _ := server.lagPropertyMap[ifIdx]
		for port, _ := range lagEnt.PortMap {
			portEnt := server.portPropertyMap[port]
			if portEnt.PcapHdl != nil {
				portEnt.CtrlCh <- true
				<-portEnt.CtrlCh
			}
		}
	} else {
		port := ifIdx
		portEnt := server.portPropertyMap[port]
		if portEnt.PcapHdl != nil {
			portEnt.CtrlCh <- true
			<-portEnt.CtrlCh
		}
	}
}

func (server *DHCPServer) StartAllDhcpServer() {
	for _, dEnt := range server.DhcpIntfConfMap {
		if dEnt.enable == true {
			server.StartDhcpServer(dEnt.l3IfIdx)
		}
	}
}

func (server *DHCPServer) StartDhcpServer(ifIdx int32) {
	server.logger.Debug(fmt.Sprintln("Starting Dhcp Server on l3 IfIndex:", ifIdx))
	ifType := asicdCommonDefs.GetIntfTypeFromIfIndex(ifIdx)
	if ifType == commonDefs.IfTypeVlan {
		vlanEnt, _ := server.vlanPropertyMap[ifIdx]
		for port, _ := range vlanEnt.UntagPortMap {
			server.StartRxDhcpPkt(port)
		}
	} else if ifType == commonDefs.IfTypeLag {
		lagEnt, _ := server.lagPropertyMap[ifIdx]
		for port, _ := range lagEnt.PortMap {
			server.StartRxDhcpPkt(port)
		}
	} else {
		port := ifIdx
		server.StartRxDhcpPkt(port)
	}
	server.constructDhcpMsg(ifIdx)
}
