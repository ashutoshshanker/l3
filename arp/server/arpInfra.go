package server

import (
	"asicd/asicdConstDefs"
	"asicdInt"
	"asicdServices"
	"fmt"
	"net"
	"utils/commonDefs"
	//"asicd/pluginManager/pluginCommon"
	"github.com/google/gopacket/pcap"
)

type L3IntfProperty struct {
	Netmask net.IPMask
	IpAddr  string
}

type PortProperty struct {
	IfName   string
	MacAddr  string
	IpAddr   string
	Netmask  net.IPMask
	L3IfIdx  int
	LagIfIdx int
	CtrlCh   chan bool
	PcapHdl  *pcap.Handle
}

type VlanProperty struct {
	UntagPortMap map[int]bool
}

type LagProperty struct {
	PortMap map[int]bool
}

func (server *ARPServer) getL3IntfOnSameSubnet(ip string) int {
	ipAddr := net.ParseIP(ip)
	for l3Idx, l3Ent := range server.l3IntfPropMap {
		if l3Ent.IpAddr == ip {
			return -1
		}

		l3IpAddr := net.ParseIP(l3Ent.IpAddr)
		l3Net := l3IpAddr.Mask(l3Ent.Netmask)
		ipNet := ipAddr.Mask(l3Ent.Netmask)
		if l3Net.Equal(ipNet) {
			return l3Idx
		}
	}
	return -1
}

func (server *ARPServer) processIPv4IntfCreate(msg asicdConstDefs.IPv4IntfNotifyMsg) {
	ip, ipNet, _ := net.ParseCIDR(msg.IpAddr)
	ifType := asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)
	ifIdx := int(msg.IfIndex)

	server.logger.Info(fmt.Sprintln("Received IPv4 Create Notification for IP:", ip, "IfIndex:", msg.IfIndex))

	l3IntfEnt, _ := server.l3IntfPropMap[ifIdx]
	l3IntfEnt.IpAddr = ip.String()
	l3IntfEnt.Netmask = ipNet.Mask
	server.l3IntfPropMap[ifIdx] = l3IntfEnt

	if ifType == commonDefs.IfTypeVlan {
		vlanEnt, _ := server.vlanPropMap[ifIdx]
		server.logger.Info(fmt.Sprintln("Received IPv4 Create Notification for Untag Port List:", vlanEnt.UntagPortMap))
		for port, _ := range vlanEnt.UntagPortMap {
			portEnt := server.portPropMap[port]
			portEnt.IpAddr = ip.String()
			portEnt.Netmask = ipNet.Mask
			portEnt.L3IfIdx = ifIdx
			server.portPropMap[port] = portEnt
			server.logger.Info(fmt.Sprintln("Start Rx on port:", port))
			server.StartArpRxTx(port)
		}
	} else if ifType == commonDefs.IfTypeLag {
		lagEnt, _ := server.lagPropMap[ifIdx]
		server.logger.Info(fmt.Sprintln("Received IPv4 Create Notification for LagId:", ifIdx, "Port List:", lagEnt.PortMap))
		for port, _ := range lagEnt.PortMap {
			portEnt := server.portPropMap[port]
			portEnt.IpAddr = ip.String()
			portEnt.Netmask = ipNet.Mask
			portEnt.L3IfIdx = ifIdx
			server.portPropMap[port] = portEnt
			server.logger.Info(fmt.Sprintln("Start Rx on port:", port))
			server.StartArpRxTx(port)
		}
	} else if ifType == commonDefs.IfTypePort {
		port := ifIdx
		portEnt := server.portPropMap[port]
		portEnt.IpAddr = ip.String()
		portEnt.Netmask = ipNet.Mask
		portEnt.L3IfIdx = ifIdx
		server.portPropMap[port] = portEnt
		server.logger.Info(fmt.Sprintln("Start Rx on port:", port))
		server.StartArpRxTx(port)
	}
}

func (server *ARPServer) processIPv4IntfDelete(msg asicdConstDefs.IPv4IntfNotifyMsg) {
	ifType := asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)
	ifIdx := int(msg.IfIndex)

	if ifType == commonDefs.IfTypeVlan {
		vlanEnt, _ := server.vlanPropMap[ifIdx]
		for port, _ := range vlanEnt.UntagPortMap {
			portEnt := server.portPropMap[port]
			// Stop Rx Thread
			server.logger.Info(fmt.Sprintln("Closing Rx on port:", port))
			portEnt.CtrlCh <- true
			<-portEnt.CtrlCh
			server.logger.Info(fmt.Sprintln("Rx is closed successfully on port:", port))
			//Delete ARP Entry
			server.logger.Info(fmt.Sprintln("Flushing Arp Entry learned on port:", port))
			server.arpEntryDeleteCh <- DeleteArpEntryMsg{
				PortNum: port,
			}
			portEnt.IpAddr = ""
			portEnt.Netmask = nil
			portEnt.L3IfIdx = -1
			server.portPropMap[port] = portEnt
		}
	} else if ifType == commonDefs.IfTypeLag {
		lagEnt, _ := server.lagPropMap[ifIdx]
		for port, _ := range lagEnt.PortMap {
			portEnt := server.portPropMap[port]
			// Stop Rx Thread
			server.logger.Info(fmt.Sprintln("Closing Rx on port:", port))
			portEnt.CtrlCh <- true
			<-portEnt.CtrlCh
			server.logger.Info(fmt.Sprintln("Rx is closed successfully on port:", port))
			//Delete ARP Entry
			server.logger.Info(fmt.Sprintln("Flushing Arp Entry learned on port:", port))
			server.arpEntryDeleteCh <- DeleteArpEntryMsg{
				PortNum: port,
			}
			portEnt.IpAddr = ""
			portEnt.Netmask = nil
			portEnt.L3IfIdx = -1
			server.portPropMap[port] = portEnt
		}
	} else if ifType == commonDefs.IfTypePort {
		port := ifIdx
		portEnt := server.portPropMap[port]
		// Stop Rx Thread
		server.logger.Info(fmt.Sprintln("Closing Rx on port:", port))
		portEnt.CtrlCh <- true
		<-portEnt.CtrlCh
		server.logger.Info(fmt.Sprintln("Rx is closed successfully on port:", port))
		//Delete ARP Entry
		server.logger.Info(fmt.Sprintln("Flushing Arp Entry learned on port:", port))
		server.arpEntryDeleteCh <- DeleteArpEntryMsg{
			PortNum: port,
		}
		portEnt.IpAddr = ""
		portEnt.Netmask = nil
		portEnt.L3IfIdx = -1
		server.portPropMap[port] = portEnt
	}
	delete(server.l3IntfPropMap, ifIdx)
}

func (server *ARPServer) updateIpv4Infra(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE {
		server.processIPv4IntfCreate(msg)
	} else {
		server.processIPv4IntfDelete(msg)
	}
}

func (server *ARPServer) processL3StateChange(msg asicdConstDefs.L3IntfStateNotifyMsg) {
	ifIdx := int(msg.IfIndex)
	ifType := asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)
	if msg.IfState == 0 {
		if ifType == commonDefs.IfTypeVlan {
			vlanEnt := server.vlanPropMap[ifIdx]
			for port, _ := range vlanEnt.UntagPortMap {
				//Delete ARP Entry
				server.logger.Info(fmt.Sprintln("Flushing Arp Entry learned on port:", port))
				server.arpEntryDeleteCh <- DeleteArpEntryMsg{
					PortNum: port,
				}
			}
		} else if ifType == commonDefs.IfTypeLag {
			lagEnt := server.lagPropMap[ifIdx]
			for port, _ := range lagEnt.PortMap {
				//Delete ARP Entry
				server.logger.Info(fmt.Sprintln("Flushing Arp Entry learned on port:", port))
				server.arpEntryDeleteCh <- DeleteArpEntryMsg{
					PortNum: port,
				}
			}
		} else if ifType == commonDefs.IfTypePort {
			port := ifIdx
			//Delete ARP Entry
			server.logger.Info(fmt.Sprintln("Flushing Arp Entry learned on port:", port))
			server.arpEntryDeleteCh <- DeleteArpEntryMsg{
				PortNum: port,
			}
		}
	}
}

func (server *ARPServer) processIPv4NbrMacMove(msg asicdConstDefs.IPv4NbrMacMoveNotifyMsg) {
	server.arpEntryMacMoveCh <- msg
}

/*

func (server *ARPServer)processL2StateChange(msg asicdConstDefs.L2IntfStateNotifyMsg) {
        ifIdx := int(msg.IfIndex)
        ifType := asicdConstDefs.GetIntfTypeFromIfIndex(msg.IfIndex)
        if msg.IfState == 0 {
                if ifType == commonDefs.IfTypeVlan {
                        vlanEnt := server.vlanPropMap[ifIdx]
                        for port, _ := range vlanEnt.UntagPortMap {
                                //Delete ARP Entry
                                server.arpEntryDeleteCh <- DeleteArpEntryMsg {
                                        PortNum: port,
                                }
                        }
                } else if ifType == commonDefs.IfTypeLag {
                        lagEnt := server.lagPropMap[ifIdx]
                        for port, _ := range lagEnt.PortMap {
                                //Delete ARP Entry
                                server.arpEntryDeleteCh <- DeleteArpEntryMsg {
                                        PortNum: port,
                                }
                        }
                } else if ifType == commonDefs.IfTypePort {
                        port := ifIdx
                        //Delete ARP Entry
                        server.arpEntryDeleteCh <- DeleteArpEntryMsg {
                                PortNum: port,
                        }
                }
        }
}
*/

func (server *ARPServer) processArpInfra() {
	for ifIdx, _ := range server.l3IntfPropMap {
		ifType := asicdConstDefs.GetIntfTypeFromIfIndex(int32(ifIdx))
		if ifType == commonDefs.IfTypeVlan {
			vlanEnt := server.vlanPropMap[ifIdx]
			for port, _ := range vlanEnt.UntagPortMap {
				server.logger.Info(fmt.Sprintln("Start Rx on port:", port))
				server.StartArpRxTx(port)
			}
		} else if ifType == commonDefs.IfTypeLag {
			lagEnt := server.lagPropMap[ifIdx]
			for port, _ := range lagEnt.PortMap {
				server.logger.Info(fmt.Sprintln("Start Rx on port:", port))
				server.StartArpRxTx(port)
			}
		} else if ifType == commonDefs.IfTypePort {
			port := ifIdx
			server.logger.Info(fmt.Sprintln("Start Rx on port:", port))
			server.StartArpRxTx(port)
		}
	}
}

func (server *ARPServer) buildArpInfra() {
	server.constructPortInfra()
	server.constructVlanInfra()
	server.constructL3Infra()
	//server.constructLagInfra()
	//server.logger.Info(fmt.Sprintln("Port Property Map:", server.portPropMap))
	//server.logger.Info(fmt.Sprintln("Lag Property Map:", server.portPropMap))
	//server.logger.Info(fmt.Sprintln("Vlan Property Map:", server.portPropMap))
	//server.logger.Info(fmt.Sprintln("L3 Intf Property Map:", server.l3IntfPropMap))
}

func (server *ARPServer) constructL3Infra() {
	curMark := 0
	server.logger.Info("Calling Asicd for getting L3 Interfaces")
	count := 100
	for {
		bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkIPv4IntfState(asicdServices.Int(curMark), asicdServices.Int(count))
		if bulkInfo == nil {
			break
		}
		objCnt := int(bulkInfo.Count)
		more := bool(bulkInfo.More)
		curMark = int(bulkInfo.EndIdx)
		for i := 0; i < objCnt; i++ {
			ip, ipNet, _ := net.ParseCIDR(bulkInfo.IPv4IntfStateList[i].IpAddr)
			ifIdx := int(bulkInfo.IPv4IntfStateList[i].IfIndex)
			ifType := asicdConstDefs.GetIntfTypeFromIfIndex(int32(ifIdx))
			if ifType == commonDefs.IfTypeVlan {
				vlanEnt := server.vlanPropMap[ifIdx]
				for port, _ := range vlanEnt.UntagPortMap {
					portEnt := server.portPropMap[port]
					portEnt.L3IfIdx = ifIdx
					portEnt.IpAddr = ip.String()
					portEnt.Netmask = ipNet.Mask
					server.portPropMap[port] = portEnt
				}
			} else if ifType == commonDefs.IfTypeLag {
				lagEnt := server.lagPropMap[ifIdx]
				for port, _ := range lagEnt.PortMap {
					portEnt := server.portPropMap[port]
					portEnt.L3IfIdx = ifIdx
					portEnt.IpAddr = ip.String()
					portEnt.Netmask = ipNet.Mask
					server.portPropMap[port] = portEnt
				}
			} else if ifType == commonDefs.IfTypePort {
				port := ifIdx
				portEnt := server.portPropMap[port]
				portEnt.L3IfIdx = ifIdx
				portEnt.IpAddr = ip.String()
				portEnt.Netmask = ipNet.Mask
				server.portPropMap[port] = portEnt
			}

			ent := server.l3IntfPropMap[ifIdx]
			ent.Netmask = ipNet.Mask
			ent.IpAddr = ip.String()
			server.l3IntfPropMap[ifIdx] = ent
		}
		if more == false {
			break
		}
	}
}

func (server *ARPServer) constructPortInfra() {
	//server.logger.Info(fmt.Sprintln("Port Property Map:", server.portPropMap))
	server.getBulkPortState()
	server.getBulkPortConfig()
	//server.logger.Info(fmt.Sprintln("Port Property Map:", server.portPropMap))
}

func (server *ARPServer) getBulkPortConfig() {
	curMark := int(asicdConstDefs.MIN_SYS_PORTS)
	server.logger.Info("Calling Asicd for getting Port Property")
	count := 100
	for {
		bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPort(asicdServices.Int(curMark), asicdServices.Int(count))
		if bulkInfo == nil {
			break
		}
		objCnt := int(bulkInfo.Count)
		more := bool(bulkInfo.More)
		curMark = int(bulkInfo.EndIdx)
		for i := 0; i < objCnt; i++ {
			portNum := int(bulkInfo.PortList[i].PortNum)
			ent := server.portPropMap[portNum]
			ent.MacAddr = bulkInfo.PortList[i].MacAddr
			ent.CtrlCh = make(chan bool)
			server.portPropMap[portNum] = ent
		}
		if more == false {
			break
		}
	}
}

func (server *ARPServer) getBulkPortState() {
	curMark := int(asicdConstDefs.MIN_SYS_PORTS)
	server.logger.Info("Calling Asicd for getting Port Property")
	count := 100
	for {
		bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPortState(asicdServices.Int(curMark), asicdServices.Int(count))
		if bulkInfo == nil {
			break
		}
		objCnt := int(bulkInfo.Count)
		more := bool(bulkInfo.More)
		curMark = int(bulkInfo.EndIdx)
		for i := 0; i < objCnt; i++ {
			portNum := int(bulkInfo.PortStateList[i].PortNum)
			ent := server.portPropMap[portNum]
			ent.IfName = bulkInfo.PortStateList[i].Name
			ent.L3IfIdx = -1
			ent.LagIfIdx = -1
			ent.CtrlCh = make(chan bool)
			ent.PcapHdl = nil
			server.portPropMap[portNum] = ent
		}
		if more == false {
			break
		}
	}
}

func (server *ARPServer) constructVlanInfra() {
	curMark := 0
	server.logger.Info("Calling Asicd for getting Vlan Property")
	count := 100
	for {
		bulkVlanInfo, _ := server.asicdClient.ClientHdl.GetBulkVlan(asicdInt.Int(curMark), asicdInt.Int(count))
		if bulkVlanInfo == nil {
			break
		}
		/* Get bulk on vlan state can re-use curMark and count used by get bulk vlan, as there is a 1:1 mapping in terms of cfg/state objs */
		bulkVlanStateInfo, _ := server.asicdClient.ClientHdl.GetBulkVlanState(asicdServices.Int(curMark), asicdServices.Int(count))
		if bulkVlanStateInfo == nil {
			break
		}
		objCnt := int(bulkVlanInfo.Count)
		more := bool(bulkVlanInfo.More)
		curMark = int(bulkVlanInfo.EndIdx)
		for i := 0; i < objCnt; i++ {
			ifIndex := int(bulkVlanStateInfo.VlanStateList[i].IfIndex)
			ent := server.vlanPropMap[ifIndex]
			untaggedIfIndexList := bulkVlanInfo.VlanList[i].UntagIfIndexList
			ent.UntagPortMap = make(map[int]bool)
			for i := 0; i < len(untaggedIfIndexList); i++ {
				ent.UntagPortMap[int(untaggedIfIndexList[i])] = true
			}
			server.vlanPropMap[ifIndex] = ent
		}
		if more == false {
			break
		}
	}
	//server.logger.Info(fmt.Sprintln("Vlan Property Map:", server.vlanPropMap))
}

func (server *ARPServer) updateVlanInfra(msg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	vlanId := int(msg.VlanId)
	ifIdx := int(asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(vlanId, commonDefs.IfTypeVlan))
	portList := msg.UntagPorts
	//server.logger.Info(fmt.Sprintln("Vlan Property Map:", server.vlanPropMap))
	vlanEnt, _ := server.vlanPropMap[ifIdx]
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // VLAN CREATE
		server.logger.Info(fmt.Sprintln("Received Vlan Create or Update Notification Vlan:", vlanId, "PortList:", portList))
		vlanEnt.UntagPortMap = nil
		vlanEnt.UntagPortMap = make(map[int]bool)
		for i := 0; i < len(portList); i++ {
			port := int(portList[i])
			vlanEnt.UntagPortMap[port] = true
		}
		server.vlanPropMap[ifIdx] = vlanEnt
	} else if msgType == asicdConstDefs.NOTIFY_VLAN_UPDATE { //VLAN UPDATE
		newPortMap := make(map[int]bool)
		for i := 0; i < len(portList); i++ {
			port := int(portList[i])
			newPortMap[port] = true
		}
		for oldPort, _ := range vlanEnt.UntagPortMap {
			_, exist := newPortMap[oldPort]
			if !exist { // There in Old but Not in New so flush arp cache
				/*
				   server.arpEntryDeleteCh <- DeleteArpEntryMsg {
				           PortNum: oldPort,
				   }
				*/
			} else { //Intersecting Ports (already there in UntagPortMap)
				delete(newPortMap, oldPort)
			}
		}
		for newPort, _ := range newPortMap { // All new ports need to be added
			vlanEnt.UntagPortMap[newPort] = true
		}
		server.vlanPropMap[ifIdx] = vlanEnt
	} else { // VLAN DELETE
		server.logger.Info(fmt.Sprintln("Received Vlan Delete Notification Vlan:", vlanId, "PortList:", portList))
		/*
		   // Note : To be Discussed
		   for portNum, _ := range vlanEnt.UntagPortMap {
		           server.arpEntryDeleteCh <- DeleteArpEntryMsg {
		                   PortNum: portNum,
		           }
		   }
		*/
		vlanEnt.UntagPortMap = nil
		delete(server.vlanPropMap, ifIdx)
	}
	//server.logger.Info(fmt.Sprintln("Vlan Property Map:", server.vlanPropMap))
}

func (server *ARPServer) updateLagInfra(msg asicdConstDefs.LagNotifyMsg, msgType uint8) {
	ifIdx := int(msg.IfIndex)
	portList := msg.IfIndexList
	//server.logger.Info(fmt.Sprintln("Lag Property Map:", server.lagPropMap))
	lagEnt, _ := server.lagPropMap[ifIdx]
	if msgType == asicdConstDefs.NOTIFY_LAG_CREATE {
		server.logger.Info(fmt.Sprintln("Received Lag Create Notification IfIdx:", ifIdx, "PortList:", portList))
		lagEnt.PortMap = nil
		lagEnt.PortMap = make(map[int]bool)
		for i := 0; i < len(portList); i++ {
			port := int(portList[i])
			portEnt, _ := server.portPropMap[port]
			portEnt.LagIfIdx = ifIdx
			server.portPropMap[port] = portEnt
			lagEnt.PortMap[port] = true
		}
		server.lagPropMap[ifIdx] = lagEnt
	} else if msgType == asicdConstDefs.NOTIFY_LAG_UPDATE {
		newPortMap := make(map[int]bool)
		for i := 0; i < len(portList); i++ {
			port := int(portList[i])
			newPortMap[port] = true
		}
		for oldPort, _ := range lagEnt.PortMap {
			_, exist := newPortMap[oldPort]
			if !exist { // There in Old but Not in New so flush arp cache
				/*
				   server.arpEntryDeleteCh <- DeleteArpEntryMsg {
				           PortNum: oldPort,
				   }
				*/
				portEnt, _ := server.portPropMap[oldPort]
				portEnt.LagIfIdx = -1
				server.portPropMap[oldPort] = portEnt
			} else { //Intersecting Ports (already there in PortMap)
				delete(newPortMap, oldPort)
			}
		}
		for newPort, _ := range newPortMap { // All new ports need to be added
			portEnt, _ := server.portPropMap[newPort]
			portEnt.LagIfIdx = ifIdx
			server.portPropMap[newPort] = portEnt
			lagEnt.PortMap[newPort] = true
		}
		server.lagPropMap[ifIdx] = lagEnt
	} else {
		server.logger.Info(fmt.Sprintln("Received Lag Delete Notification IfIdx:", ifIdx, "PortList:", portList))
		for i := 0; i < len(portList); i++ {
			port := int(portList[i])
			portEnt, _ := server.portPropMap[port]
			portEnt.LagIfIdx = -1
			server.portPropMap[port] = portEnt
		}
		/*
		   // Do we need to flush
		   for portNum, _ := range lagEnt.PortMap {
		           server.arpEntryDeleteCh <- DeleteArpEntryMsg {
		                   PortNum: portNum,
		           }
		   }
		*/
		lagEnt.PortMap = nil
		delete(server.lagPropMap, ifIdx)
	}
	//server.logger.Info(fmt.Sprintln("Lag Property Map:", server.lagPropMap))
}

/*
func (server *ARPServer)constructLagInfra() {
        curMark := 0
        server.logger.Info("Calling Asicd for getting Lag Property")
        count := 100
        for {
                bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkLag(asicdServices.Int(curMark), asicdServices.Int(count))
                if bulkInfo == nil {
                        break
                }
                objCnt := int(bulkInfo.Count)
                more := bool(bulkInfo.More)
                curMark = asicdServices.Int(bulkInfo.EndIdx)
                for i := 0; i < objCnt; i++ {
                        ifIdx := int(bulkInfo.LagList[i].IfIndex)
                        ent := server.lagPropMap[ifIdx]
                        ifIndexList := ParseUsrPortStrToPortList(bulkInfo.LagList[i].IfIndexList)

                        for i := 0; i < len(ifIndexList); i++ {
                                port := ifIndexList[i]
                                portEnt := server.portPropMap[port]
                                portEnt.LagIfIndex = ifIdx
                                server.portPropMap[port] = portEnt
                                ent.PortMap[port] = true
                        }
                        server.lagPropMap[ifIdx] = ent
                }
                if more == false {
                        break
                }
        }
}
*/
