package server

import (
	"asicd/asicdConstDefs"
	"fmt"
	"time"
	"utils/commonDefs"
)

type UpdateArpEntryMsg struct {
	PortNum int
	IpAddr  string
	MacAddr string
	Type    bool // True: RIB False: Rx
}

/*
type CreateArpEntryMsg struct {
        PortNum         int
        IpAddr          string
        MacAddr         string
}
*/

type DeleteArpEntryMsg struct {
	PortNum int
}

func (server *ARPServer) updateArpCache() {
	for {
		select {
		/*
		   case msg := <-server.arpEntryCreateCh:
		           server.processArpEntryCreateMsg(msg)
		*/
		case msg := <-server.arpEntryUpdateCh:
			server.processArpEntryUpdateMsg(msg)
		case msg := <-server.arpEntryDeleteCh:
			server.processArpEntryDeleteMsg(msg)
		case <-server.arpSliceRefreshStartCh:
			server.processArpSliceRefreshMsg()
		case <-server.arpCounterUpdateCh:
			server.processArpCounterUpdateMsg()
		case cnt := <-server.arpEntryCntUpdateCh:
			server.processArpEntryCntUpdateMsg(cnt)
		case msg := <-server.arpEntryMacMoveCh:
			server.processArpEntryMacMoveMsg(msg)
		}
	}
}

func (server *ARPServer) processArpEntryCntUpdateMsg(cnt int) {
	for key, ent := range server.arpCache {
		if ent.Counter > cnt {
			ent.Counter = cnt
			server.arpCache[key] = ent
		}
	}
}

func (server *ARPServer) processArpEntryMacMoveMsg(msg asicdConstDefs.IPv4NbrMacMoveNotifyMsg) {
	if entry, ok := server.arpCache[msg.IpAddr]; ok {
		entry.PortNum = int(msg.IfIndex)
		server.arpCache[msg.IpAddr] = entry
	} else {
		server.logger.Info(fmt.Sprintf("Mac move message received. Neighbor IP does not exist in arp cache - %x", msg.IpAddr))
	}
}

/*
func (server *ARPServer)processArpEntryCreateMsg(msg CreateArpEntryMsg) {

}
*/

func (server *ARPServer) processArpEntryDeleteMsg(msg DeleteArpEntryMsg) {
	for key, ent := range server.arpCache {
		if msg.PortNum == ent.PortNum {
			server.logger.Info(fmt.Sprintln("1 Calling Asicd Delete Ip:", key))
			rv, err := server.asicdClient.ClientHdl.DeleteIPv4Neighbor(key,
				"00:00:00:00:00:00", 0, 0)
			if rv < 0 || err != nil {
				server.logger.Err(fmt.Sprintln("Asicd was unable to delete neigbhor entry for", key, "err:", err, "rv:", rv))
				return
			}
			delete(server.arpCache, key)
			server.deleteArpEntryInDB(key)
		}
	}

}

func (server *ARPServer) processArpEntryUpdateMsg(msg UpdateArpEntryMsg) {
	portEnt, _ := server.portPropMap[msg.PortNum]
	l3IfIdx := portEnt.L3IfIdx
	ifType := asicdConstDefs.GetIntfTypeFromIfIndex(int32(l3IfIdx))
	ifId := asicdConstDefs.GetIntfIdFromIfIndex(int32(l3IfIdx))
	var vlanId int
	if l3IfIdx == -1 {
		vlanId = asicdConstDefs.SYS_RSVD_VLAN
	} else {
		_, exist := server.l3IntfPropMap[l3IfIdx]
		if !exist {
			server.logger.Info(fmt.Sprintln("Port", msg.PortNum, "doesnot belong to L3 Interface"))
			return
		}
		if ifType == commonDefs.IfTypeVlan {
			vlanId = ifId
		} else {
			vlanId = asicdConstDefs.SYS_RSVD_VLAN
		}
	}
	arpEnt, exist := server.arpCache[msg.IpAddr]
	if exist {
		if arpEnt.MacAddr == msg.MacAddr &&
			arpEnt.PortNum == msg.PortNum &&
			arpEnt.VlanId == vlanId &&
			arpEnt.L3IfIdx == portEnt.L3IfIdx {
			arpEnt.Counter = server.timeoutCounter
			if arpEnt.MacAddr != "incomplete" {
				arpEnt.TimeStamp = time.Now()
			}
			server.arpCache[msg.IpAddr] = arpEnt
			return
		}

		if arpEnt.MacAddr != "incomplete" &&
			msg.MacAddr == "incomplete" {
			server.logger.Err(fmt.Sprintln("Neighbor", msg.IpAddr, "is already resolved at port:", arpEnt.IfName, "with MacAddr:", arpEnt.MacAddr, "vlanId:", arpEnt.VlanId))
			return
		}

		var ifIdx int32
		if portEnt.LagIfIdx == -1 {
			ifIdx = int32(msg.PortNum)
		} else {
			ifIdx = int32(portEnt.LagIfIdx)
		}
		if arpEnt.MacAddr != "incomplete" &&
			msg.MacAddr != "incomplete" {
			server.logger.Info(fmt.Sprintln("2 Calling Asicd Update Ip:", msg.IpAddr, "mac:", msg.MacAddr, "vlanId:", vlanId, "IfIndex:", ifIdx))
			rv, err := server.asicdClient.ClientHdl.UpdateIPv4Neighbor(msg.IpAddr,
				msg.MacAddr, int32(vlanId), ifIdx)
			if rv < 0 || err != nil {
				server.logger.Err(fmt.Sprintln("Asicd Update IPv4 Neighbor failed for IpAddr:", msg.IpAddr, "MacAddr:", msg.MacAddr, "VlanId:", vlanId, "IfIdx:", ifIdx, "err:", err, "rv:", rv))
				return
			}
		} else if arpEnt.MacAddr == "incomplete" &&
			msg.MacAddr != "incomplete" {
			if arpEnt.Type == false {
				server.logger.Info(fmt.Sprintln("3 Calling Asicd Create Ip:", msg.IpAddr, "mac:", msg.MacAddr, "vlanId:", vlanId, "IfIndex:", ifIdx))
				rv, err := server.asicdClient.ClientHdl.CreateIPv4Neighbor(msg.IpAddr,
					msg.MacAddr, int32(vlanId), ifIdx)
				if rv < 0 || err != nil {
					server.logger.Err(fmt.Sprintln("Asicd Create IPv4 Neighbor failed for IpAddr:", msg.IpAddr, "VlanId:", vlanId, "IfIdx:", ifIdx, "err:", err, "rv:", rv))
					return
				}
			} else if arpEnt.Type == true {
				// Since RIB would already created the neighbor entry
				server.logger.Info(fmt.Sprintln("2.1 Calling Asicd Update Ip:", msg.IpAddr, "mac:", msg.MacAddr, "vlanId:", vlanId, "IfIndex:", ifIdx))
				rv, err := server.asicdClient.ClientHdl.UpdateIPv4Neighbor(msg.IpAddr,
					msg.MacAddr, int32(vlanId), ifIdx)
				if rv < 0 || err != nil {
					server.logger.Err(fmt.Sprintln("Asicd Update IPv4 Neighbor failed for IpAddr:", msg.IpAddr, "MacAddr:", msg.MacAddr, "VlanId:", vlanId, "IfIdx:", ifIdx, "err:", err, "rv:", rv))
					return
				}
			}
		}
		/*
		   else if arpEnt.MacAddr != "incomplete" &&

		           msg.MacAddr == "incomplete" {
		           server.logger.Info(fmt.Sprintln("=============Hello21==========Calling Asicd Delete Ip:", msg.IpAddr))
		           rv, err := server.asicdClient.ClientHdl.DeleteIPv4Neighbor(msg.IpAddr,
		                   "00:00:00:00:00:00", 0, 0)
		           if rv < 0 || err != nil {
		                   server.logger.Err(fmt.Sprintln("Asicd was unable to delete neigbhor entry for", msg.IpAddr, "err:", err, "rv:", rv))
		                   return
		           }
		           server.deleteArpDBEntry(msg.IpAddr)
		           delete(server.arpCache, msg.IpAddr)
		   }
		*/
	} else {
		var ifIdx int32
		if portEnt.LagIfIdx == -1 {
			ifIdx = int32(msg.PortNum)
		} else {
			ifIdx = int32(portEnt.LagIfIdx)
		}
		if msg.MacAddr != "incomplete" {
			server.logger.Info(fmt.Sprintln("4 Calling Asicd Create Ip:", msg.IpAddr, "mac:", msg.MacAddr, "vlanId:", vlanId, "IfIndex:", ifIdx))
			rv, err := server.asicdClient.ClientHdl.CreateIPv4Neighbor(msg.IpAddr,
				msg.MacAddr, int32(vlanId), ifIdx)
			if rv < 0 || err != nil {
				server.logger.Err(fmt.Sprintln("Asicd Create IPv4 Neighbor failed for IpAddr:", msg.IpAddr, "VlanId:", vlanId, "IfIdx:", ifIdx, "err:", err, "rv:", rv))
				return
			}
		}
		// Store In DB to handle Restart
		server.storeArpEntryInDB(msg.IpAddr, msg.PortNum)
	}
	arpEnt.MacAddr = msg.MacAddr
	arpEnt.PortNum = msg.PortNum
	arpEnt.VlanId = vlanId
	arpEnt.IfName = portEnt.IfName
	arpEnt.L3IfIdx = portEnt.L3IfIdx
	arpEnt.Counter = server.timeoutCounter
	if exist &&
		arpEnt.Type == true {
		arpEnt.Type = true
	} else {
		arpEnt.Type = msg.Type
	}
	if arpEnt.MacAddr != "incomplete" {
		arpEnt.TimeStamp = time.Now()
	}
	server.arpCache[msg.IpAddr] = arpEnt
	for i := 0; i < len(server.arpSlice); i++ {
		if server.arpSlice[i] == msg.IpAddr {
			return
		}
	}
	server.arpSlice = append(server.arpSlice, msg.IpAddr)
}

func (server *ARPServer) processArpCounterUpdateMsg() {
	oneMinCnt := (60 / server.timerGranularity)
	thirtySecCnt := (30 / server.timerGranularity)
	for ip, arpEnt := range server.arpCache {
		if arpEnt.Counter <= server.minCnt {
			server.deleteArpEntryInDB(ip)
			delete(server.arpCache, ip)
			server.logger.Info(fmt.Sprintln("5 Calling Asicd Delete Ip:", ip))
			rv, err := server.asicdClient.ClientHdl.DeleteIPv4Neighbor(ip,
				"00:00:00:00:00:00", 0, 0)
			if rv < 0 || err != nil {
				server.logger.Err(fmt.Sprintln("Asicd was unable to delete neigbhor entry for", ip, "err:", err, "rv:", rv))
				return
			}
			server.printArpEntries()
		} else {
			arpEnt.Counter--
			server.arpCache[ip] = arpEnt
			if arpEnt.Counter <= (server.minCnt+server.retryCnt+1) ||
				arpEnt.Counter == (server.timeoutCounter/2) ||
				arpEnt.Counter == (server.timeoutCounter/4) ||
				arpEnt.Counter == oneMinCnt ||
				arpEnt.Counter == thirtySecCnt {
				server.refreshArpEntry(ip, arpEnt.PortNum)
			} else if arpEnt.Counter <= server.timeoutCounter &&
				arpEnt.Counter > (server.timeoutCounter-server.retryCnt) &&
				arpEnt.MacAddr == "incomplete" {
				server.retryForArpEntry(ip, arpEnt.PortNum)
			} else if arpEnt.Counter > (server.minCnt+server.retryCnt+1) &&
				arpEnt.MacAddr != "incomplete" {
				continue
			} else {
				server.deleteArpEntryInDB(ip)
				delete(server.arpCache, ip)
				server.printArpEntries()
			}
		}
	}
}

func (server *ARPServer) refreshArpEntry(ipAddr string, port int) {
	// TimeoutCounter set to retryCnt
	server.logger.Info(fmt.Sprintln("Refreshing Arp entry for IP:", ipAddr, "on port:", port))
	server.sendArpReq(ipAddr, port)
}

func (server *ARPServer) retryForArpEntry(ipAddr string, port int) {
	server.logger.Info(fmt.Sprintln("Retry Arp entry for IP:", ipAddr, "on port:", port))
	server.sendArpReq(ipAddr, port)
}

func (server *ARPServer) processArpSliceRefreshMsg() {
	server.logger.Info("Refresh Arp Slice used for Getbulk")
	server.arpSlice = server.arpSlice[:0]
	server.arpSlice = nil
	server.arpSlice = make([]string, 0)
	for ip, _ := range server.arpCache {
		server.arpSlice = append(server.arpSlice, ip)
	}
	server.arpSliceRefreshDoneCh <- true
}

func (server *ARPServer) refreshArpSlice() {
	refreshArpSlicefunc := func() {
		server.arpSliceRefreshStartCh <- true
		msg := <-server.arpSliceRefreshDoneCh
		if msg == true {
			server.logger.Info("ARP Entry refresh done")
		} else {
			server.logger.Err("ARP Entry refresh not done")
		}

		server.arpSliceRefreshTimer.Reset(server.arpSliceRefreshDuration)
	}

	server.arpSliceRefreshTimer = time.AfterFunc(server.arpSliceRefreshDuration, refreshArpSlicefunc)
}

func (server *ARPServer) arpCacheTimeout() {
	var count int
	for {
		time.Sleep(server.timeout)
		count++
		if server.dumpArpTable == true &&
			(count%60) == 0 {
			server.logger.Info("===============Message from ARP Timeout Thread==============")
			server.printArpEntries()
			server.logger.Info("========================================================")
			server.logger.Info(fmt.Sprintln("Arp Slice: ", server.arpSlice))
		}
		server.arpCounterUpdateCh <- true
	}
}
