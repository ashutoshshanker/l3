package vrrpServer

import (
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"strconv"
	"strings"
	"time"
)

type VrrpFsmIntf interface {
	VrrpFsmStart(fsmObj VrrpFsm)
	VrrpCreateObject(gblInfo VrrpGlobalInfo) (fsmObj VrrpFsm)
	VrrpInitState(key string)
	VrrpBackupState(inPkt gopacket.Packet, vrrpHdr *VrrpPktHeader, key string)
	VrrpMasterState(inPkt gopacket.Packet, vrrpHdr *VrrpPktHeader, key string)
	VrrpTransitionToMaster(key string)
	VrrpTransitionToBackup(key string, AdvertisementInterval int32)
	VrrpHandleIntfUpEvent(IfIndex int32)
	VrrpHandleIntfShutdownEvent(IfIndex int32)
}

/*
			   +---------------+
		+--------->|               |<-------------+
		|          |  Initialize   |              |
		|   +------|               |----------+   |
		|   |      +---------------+          |   |
		|   |                                 |   |
		|   V                                 V   |
	+---------------+                       +---------------+
	|               |---------------------->|               |
	|    Master     |                       |    Backup     |
	|               |<----------------------|               |
	+---------------+                       +---------------+

*/

func (svr *VrrpServer) VrrpCreateObject(gblInfo VrrpGlobalInfo) (fsmObj VrrpFsm) {
	vrrpHeader := VrrpPktHeader{
		Version:       VRRP_VERSION2,
		Type:          VRRP_PKT_TYPE_ADVERTISEMENT,
		VirtualRtrId:  uint8(gblInfo.IntfConfig.VRID),
		Priority:      uint8(gblInfo.IntfConfig.Priority),
		CountIPv4Addr: 1, // FIXME for more than 1 vip
		Rsvd:          VRRP_RSVD,
		MaxAdverInt:   uint16(gblInfo.IntfConfig.AdvertisementInterval),
		CheckSum:      VRRP_HDR_CREATE_CHECKSUM,
	}

	return VrrpFsm{
		vrrpHdr: &vrrpHeader,
	}
}

func (svr *VrrpServer) VrrpUpdateSecIp(gblInfo VrrpGlobalInfo, configure bool) {
	// @TODO: this api will send create secondary ip address... By doing so
	// we are in-directly blocking ping to the host

	// JGHEEWALA: commented out the below code as arp will be handled by creating sec ip
	// (115) + If the protected IPvX address is an IPv4 address, then:
	//ip, _, _ := net.ParseCIDR(gblInfo.IpAddr)
	//if ip.To4() != nil { // If not nill then its ipv4
	/*
	   (120) * Broadcast a gratuitous ARP request containing the
	   virtual router MAC address for each IP address associated
	   with the virtual router.
	*/
	//svr.VrrpSendGratuitousArp(gblInfo)
	//} else { // @TODO: ipv6 implementation
	// (125) + else // IPv6
	/*
	   (130) * For each IPv6 address associated with the virtual
	   router, send an unsolicited ND Neighbor Advertisement with
	   the Router Flag (R) set, the Solicited Flag (S) unset, the
	   Override flag (O) set, the target address set to the IPv6
	   address of the virtual router, and the target link-layer
	   address set to the virtual router MAC address.
	*/
	//}
	return
}

func (svr *VrrpServer) VrrpHandleMasterAdverTimer(key string) {
	var timerCheck_func func()
	timerCheck_func = func() {
		svr.logger.Info(fmt.Sprintln("time to send advertisement to backup"))
		// Send advertisment every time interval expiration
		svr.vrrpTxPktCh <- VrrpTxChannelInfo{
			key:      key,
			priority: VRRP_IGNORE_PRIORITY,
		}
		gblInfo, exists := svr.vrrpGblInfo[key]
		if !exists {
			svr.logger.Err("Gbl Config for " + key + " doesn't exists")
			return
		}
		svr.logger.Info("resetting advertisement timer")
		gblInfo.AdverTimer.Reset(
			time.Duration(gblInfo.IntfConfig.AdvertisementInterval) * time.Second)
		svr.vrrpGblInfo[key] = gblInfo
	}
	gblInfo, exists := svr.vrrpGblInfo[key]
	if exists {
		svr.logger.Info(fmt.Sprintln("setting adver timer to",
			gblInfo.IntfConfig.AdvertisementInterval))
		// Set Timer expire func...
		gblInfo.AdverTimer = time.AfterFunc(
			time.Duration(gblInfo.IntfConfig.AdvertisementInterval)*time.Second,
			timerCheck_func)
		// (145) + Transition to the {Master} state
		gblInfo.StateLock.Lock()
		gblInfo.StateName = VRRP_MASTER_STATE
		gblInfo.StateLock.Unlock()
		svr.vrrpGblInfo[key] = gblInfo
		svr.vrrpGblInfo[key] = gblInfo
	}
}

func (svr *VrrpServer) VrrpTransitionToMaster(key string) {
	// (110) + Send an ADVERTISEMENT
	svr.vrrpTxPktCh <- VrrpTxChannelInfo{
		key:      key,
		priority: VRRP_IGNORE_PRIORITY,
	}
	gblInfo, exists := svr.vrrpGblInfo[key]
	if !exists {
		svr.logger.Err("No entry found ending fsm")
		return
	}
	svr.logger.Info(fmt.Sprintln("adver sent for vrid", gblInfo.IntfConfig.VRID))
	// Configure secondary interface with VMAC and VIP
	svr.VrrpUpdateSecIp(gblInfo, true /*configure or set*/)
	// (140) + Set the Adver_Timer to Advertisement_Interval
	// Start Advertisement Timer
	svr.VrrpHandleMasterAdverTimer(key)
}

func (svr *VrrpServer) VrrpHandleMasterDownTimer(key string) {
	var timerCheck_func func()
	// On Timer expiration we will transition to master
	timerCheck_func = func() {
		svr.logger.Info(fmt.Sprintln("master down timer expired..transition to Master"))
		// do timer expiry handling here
		svr.VrrpTransitionToMaster(key)
	}
	svr.logger.Info("initiating master down timer")
	gblInfo, exists := svr.vrrpGblInfo[key]
	if exists {
		svr.logger.Info(fmt.Sprintln("setting down timer to", gblInfo.MasterDownValue))
		// Set Timer expire func...
		gblInfo.MasterDownTimer = time.AfterFunc(
			time.Duration(gblInfo.MasterDownValue)*time.Second,
			timerCheck_func)
		//(165) + Transition to the {Backup} state
		gblInfo.StateLock.Lock()
		gblInfo.StateName = VRRP_BACKUP_STATE
		gblInfo.StateLock.Unlock()
		svr.vrrpGblInfo[key] = gblInfo
	}
}

func (svr *VrrpServer) VrrpTransitionToBackup(key string, AdvertisementInterval int32) {
	svr.logger.Info(fmt.Sprintln("advertisement timer to be used in backup state for",
		"calculating master down timer is ", AdvertisementInterval))
	gblInfo, exists := svr.vrrpGblInfo[key]
	if !exists {
		svr.logger.Err("No entry found ending fsm")
		return
	}
	//(155) + Set Master_Adver_Interval to Advertisement_Interval
	gblInfo.MasterAdverInterval = AdvertisementInterval
	//(160) + Set the Master_Down_Timer to Master_Down_Interval
	if gblInfo.IntfConfig.Priority != 0 && gblInfo.MasterAdverInterval != 0 {
		gblInfo.SkewTime = ((256 - gblInfo.IntfConfig.Priority) *
			gblInfo.MasterAdverInterval) / 256
	}
	gblInfo.MasterDownValue = (3 * gblInfo.MasterAdverInterval) + gblInfo.SkewTime
	svr.vrrpGblInfo[key] = gblInfo
	// Start with handling MasterDownTimer
	svr.VrrpHandleMasterDownTimer(key)

}

func (svr *VrrpServer) VrrpInitState(key string) {
	svr.logger.Info("in init state decide next state")
	gblInfo, found := svr.vrrpGblInfo[key]
	if !found {
		svr.logger.Err("running info not found, bailing fsm")
		return
	}
	if gblInfo.IntfConfig.Priority == VRRP_MASTER_PRIORITY {
		svr.logger.Info("Transitioning to Master State")
		svr.VrrpTransitionToMaster(key)
	} else {
		svr.logger.Info("Transitioning to Backup State")
		svr.VrrpUpdateSecIp(gblInfo, false /*configure or set*/)
		// Transition to backup state first
		svr.VrrpTransitionToBackup(key,
			gblInfo.IntfConfig.AdvertisementInterval)
	}
}

func (svr *VrrpServer) VrrpBackupState(inPkt gopacket.Packet, vrrpHdr *VrrpPktHeader,
	key string) {
	// @TODO: Handle arp drop...
	// Check dmac address from the inPacket and if it is same discard the packet
	ethLayer := inPkt.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		svr.logger.Err("Not an eth packet?")
		return
	}
	eth := ethLayer.(*layers.Ethernet)
	gblInfo, exists := svr.vrrpGblInfo[key]
	if !exists {
		svr.logger.Err("No entry found ending fsm")
		return
	}
	if (eth.DstMAC).String() == gblInfo.VirtualRouterMACAddress {
		svr.logger.Err("Dmac is equal to VMac and hence fsm is aborted")
		return
	}
	// MUST NOT accept packets addressed to the IPvX address(es)
	// associated with the virtual router. @TODO: check with Hari
	ipLayer := inPkt.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		svr.logger.Err("Not an ip packet?")
		return
	}
	ipHdr := ipLayer.(*layers.IPv4)
	if (ipHdr.DstIP).String() == gblInfo.IpAddr {
		svr.logger.Err("dst ip is equal to interface ip, drop the packet")
		return
	}

	if vrrpHdr.Type == VRRP_PKT_TYPE_ADVERTISEMENT {
		svr.logger.Info(fmt.Sprintln("Advertisement pkt for VRID",
			gblInfo.IntfConfig.VRID, "in backup state"))
		if vrrpHdr.Priority == 0 {
			// Change down Value to Skew time
			gblInfo.MasterDownValue = gblInfo.SkewTime
			svr.vrrpGblInfo[key] = gblInfo
		} else {
			if gblInfo.IntfConfig.PreemptMode == false ||
				vrrpHdr.Priority >= uint8(gblInfo.IntfConfig.Priority) {
				gblInfo.MasterAdverInterval = int32(vrrpHdr.MaxAdverInt)
				gblInfo.SkewTime = ((256 - gblInfo.IntfConfig.Priority) *
					gblInfo.MasterAdverInterval) / 256
				gblInfo.MasterDownValue = (3 * gblInfo.MasterAdverInterval) + gblInfo.SkewTime
				gblInfo.MasterDownTimer.Reset(time.Duration(gblInfo.MasterDownValue) * time.Second)
				svr.vrrpGblInfo[key] = gblInfo
			} else {
				svr.logger.Info("Discarding advertisment")
				return
			} // endif preempt test
		} // endif was priority zero
	} // endif was advertisement received

	// end BACKUP STATE
}

func (svr *VrrpServer) VrrpMasterState(inPkt gopacket.Packet, vrrpHdr *VrrpPktHeader,
	key string) {
	/* // @TODO:
	   (645) - MUST forward packets with a destination link-layer MAC
	   address equal to the virtual router MAC address.

	   (650) - MUST accept packets addressed to the IPvX address(es)
	   associated with the virtual router if it is the IPvX address owner
	   or if Accept_Mode is True.  Otherwise, MUST NOT accept these
	   packets.
	*/
	if vrrpHdr.Priority == VRRP_MASTER_DOWN_PRIORITY {
		svr.vrrpTxPktCh <- VrrpTxChannelInfo{
			key:      key,
			priority: VRRP_IGNORE_PRIORITY,
		}
		svr.VrrpHandleMasterAdverTimer(key)
	} else {
		ipLayer := inPkt.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			svr.logger.Err("Not an ip packet?")
			return
		}
		ipHdr := ipLayer.(*layers.IPv4)
		gblInfo, exists := svr.vrrpGblInfo[key]
		if !exists {
			svr.logger.Err("No entry found ending fsm")
			return
		}
		if int32(vrrpHdr.Priority) > gblInfo.IntfConfig.Priority ||
			(int32(vrrpHdr.Priority) == gblInfo.IntfConfig.Priority &&
				bytes.Compare(ipHdr.SrcIP,
					net.ParseIP(gblInfo.IpAddr)) > 0) {
			svr.logger.Info(fmt.Sprintln("Remote Priority is higher or ",
				"(priority are equal && remote ip is higher then local ip)"))
			svr.logger.Info("because of the above reason stopping adver timer" +
				" and transitioning to Backup State")
			gblInfo.AdverTimer.Stop()
			svr.vrrpGblInfo[key] = gblInfo
			svr.VrrpTransitionToBackup(key, int32(vrrpHdr.MaxAdverInt))
		} else { // new Master logic
			// Discard Advertisement
			return
		} // endif new Master Detected
	} // end if was priority zero
	// end for Advertisemtn received over the channel
	// end MASTER STATE
}

func (svr *VrrpServer) VrrpFsmStart(fsmObj VrrpFsm) {
	key := fsmObj.key
	pktInfo := fsmObj.inPkt
	pktHdr := fsmObj.vrrpHdr
	gblInfo, exists := svr.vrrpGblInfo[key]
	if !exists {
		svr.logger.Err("No entry found ending fsm")
		return
	}
	svr.logger.Info(fmt.Sprintln("Received fsm request for vrid",
		gblInfo.IntfConfig.VRID))
	gblInfo.StateLock.Lock()
	currentState := gblInfo.StateName
	gblInfo.StateLock.Unlock()

	svr.logger.Info("FSM state is " + currentState)
	switch currentState {
	case VRRP_INITIALIZE_STATE:
		svr.VrrpInitState(key)
	case VRRP_BACKUP_STATE:
		svr.VrrpBackupState(pktInfo, pktHdr, key)
	case VRRP_MASTER_STATE:
		svr.VrrpMasterState(pktInfo, pktHdr, key)
	default: // VRRP_UNINTIALIZE_STATE
		svr.logger.Info("No Ip address and hence no need for fsm")
	}
}

/*
 * During a shutdown event stop timers will be called and we will cancel master
 * down timer and transition to initialize state
 */
func (svr *VrrpServer) VrrpStopTimers(IfIndex int32) {
	for _, key := range svr.vrrpIntfStateSlice {
		splitString := strings.Split(key, "_")
		// splitString = { IfIndex, VRID }
		ifindex, _ := strconv.Atoi(splitString[0])
		if int32(ifindex) != IfIndex {
			// Key doesn't match
			continue
		}
		// If IfIndex matches then use that key and stop the timer for
		// that VRID
		gblInfo, found := svr.vrrpGblInfo[key]
		if !found {
			svr.logger.Err("No entry found for Ifindex:" +
				splitString[0] + " VRID:" + splitString[1])
			return
		}
		svr.logger.Info("Stopping Master Down Timer for Ifindex:" +
			splitString[0] + " VRID:" + splitString[1])
		if gblInfo.MasterDownTimer != nil {
			gblInfo.MasterDownTimer.Stop()
		}
		svr.logger.Info("Stopping Master Advertisemen Timer for Ifindex:" +
			splitString[0] + " VRID:" + splitString[1])
		if gblInfo.AdverTimer != nil {
			gblInfo.AdverTimer.Stop()
		}
		// If state is Master then we need to send an advertisement with
		// priority as 0
		gblInfo.StateLock.Lock()
		if gblInfo.StateName == VRRP_MASTER_STATE {
			svr.vrrpTxPktCh <- VrrpTxChannelInfo{
				key:      key,
				priority: VRRP_MASTER_DOWN_PRIORITY,
			}
		}
		// Transition to Init State
		gblInfo.StateName = VRRP_INITIALIZE_STATE
		gblInfo.StateLock.Unlock()
		svr.vrrpGblInfo[key] = gblInfo
		svr.logger.Info(fmt.Sprintln("VRID:", gblInfo.IntfConfig.VRID,
			" transitioned to INIT State"))
	}
}

func (svr *VrrpServer) VrrpHandleIntfShutdownEvent(IfIndex int32) {
	svr.VrrpStopTimers(IfIndex)
}

func (svr *VrrpServer) VrrpHandleIntfUpEvent(IfIndex int32) {
	for _, key := range svr.vrrpIntfStateSlice {
		splitString := strings.Split(key, "_")
		// splitString = { IfIndex, VRID }
		ifindex, _ := strconv.Atoi(splitString[0])
		if int32(ifindex) != IfIndex {
			// Key doesn't match
			continue
		}
		// If IfIndex matches then use that key and stop the timer for
		// that VRID
		gblInfo, found := svr.vrrpGblInfo[key]
		if !found {
			svr.logger.Err("No entry found for Ifindex:" +
				splitString[0] + " VRID:" + splitString[1])
			return
		}

		svr.logger.Info(fmt.Sprintln("Intf State Up Notification",
			" restarting the fsm event for VRID:", gblInfo.IntfConfig.VRID))
		svr.vrrpFsmCh <- VrrpFsm{
			key: key,
		}
	}
}
