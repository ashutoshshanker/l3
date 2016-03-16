package vrrpServer

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	_ "net"
	"strconv"
	"strings"
)

type VrrpFsmIntf interface {
	VrrpFsmStart(fsmObj VrrpFsm)
	VrrpCreateObject(gblInfo VrrpGlobalInfo) (fsmObj VrrpFsm)
	VrrpInitState(gblInfo *VrrpGlobalInfo, key string)
	VrrpBackupState(inPkt gopacket.Packet, vrrpHdr *VrrpPktHeader,
		gblInfo *VrrpGlobalInfo, key string)
	VrrpMasterState(gblInfo *VrrpGlobalInfo)
	VrrpUpdateSecIp(gblInfo *VrrpGlobalInfo, configure bool)
	VrrpStopTimers(IfIndex int32)
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
		Type:          VRRP_PKT_TYPE,
		VirtualRtrId:  uint8(gblInfo.IntfConfig.VRID),
		Priority:      uint8(gblInfo.IntfConfig.Priority),
		CountIPv4Addr: 1, // FIXME for more than 1 vip
		Rsvd:          VRRP_RSVD,
		MaxAdverInt:   uint16(gblInfo.IntfConfig.AdvertisementInterval),
		CheckSum:      VRRP_HDR_CREATE_CHECKSUM,
	}

	return VrrpFsm{
		vrrpHdr:  &vrrpHeader,
		vrrpInFo: &gblInfo,
	}
}

func (svr *VrrpServer) VrrpUpdateSecIp(gblInfo *VrrpGlobalInfo, configure bool) {
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

func (svr *VrrpServer) VrrpInitState(gblInfo *VrrpGlobalInfo, key string) {
	if gblInfo.IntfConfig.Priority == VRRP_MASTER_PRIORITY {
		// (110) + Send an ADVERTISEMENT
		svr.vrrpTxPktCh <- key
		svr.VrrpUpdateSecIp(gblInfo, true /*configure or set*/)
		// (140) + Set the Adver_Timer to Advertisement_Interval
		gblInfo.AdverTimer = gblInfo.IntfConfig.AdvertisementInterval
		// (145) + Transition to the {Master} state
		gblInfo.StateLock.Lock()
		gblInfo.StateName = VRRP_MASTER_STATE
		gblInfo.StateLock.Unlock()
	} else {
		svr.VrrpUpdateSecIp(gblInfo, false /*configure or set*/)
		//(155) + Set Master_Adver_Interval to Advertisement_Interval
		gblInfo.MasterAdverInterval = gblInfo.IntfConfig.AdvertisementInterval
		//(160) + Set the Master_Down_Timer to Master_Down_Interval
		if gblInfo.IntfConfig.Priority != 0 && gblInfo.MasterAdverInterval != 0 {
			gblInfo.SkewTime = ((256 - gblInfo.IntfConfig.Priority) *
				gblInfo.MasterAdverInterval) / 256
		}
		gblInfo.MasterDownValue = (3 * gblInfo.MasterAdverInterval) + gblInfo.SkewTime
		//(165) + Transition to the {Backup} state
		gblInfo.StateLock.Lock()
		gblInfo.StateName = VRRP_BACKUP_STATE
		gblInfo.StateLock.Unlock()
		// Transition to backup state first
	}
	svr.vrrpGblInfo[key] = *gblInfo
}

func (svr *VrrpServer) VrrpBackupState(inPkt gopacket.Packet, vrrpHdr *VrrpPktHeader,
	gblInfo *VrrpGlobalInfo, key string) {
	// @TODO: Handle arp drop...

	// Check dmac address from the inPacket and if it is same discard the packet
	ethLayer := inPkt.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		svr.logger.Err("Not an eth packet?")
		return
	}
	eth := ethLayer.(*layers.Ethernet)
	if (eth.DstMAC).String() == gblInfo.IntfConfig.VirtualRouterMACAddress {
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

}

func (svr *VrrpServer) VrrpFsmStart(fsmObj VrrpFsm) {
	gblInfo := fsmObj.vrrpInFo
	key := fsmObj.key
	pktInfo := fsmObj.inPkt
	pktHdr := fsmObj.vrrpHdr

	gblInfo.StateLock.Lock()
	currentState := gblInfo.StateName
	gblInfo.StateLock.Unlock()

	switch currentState {
	case VRRP_INITIALIZE_STATE:
		svr.VrrpInitState(gblInfo, key)
	case VRRP_BACKUP_STATE:
		svr.VrrpBackupState(pktInfo, pktHdr, gblInfo, key)
	case VRRP_MASTER_STATE:
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
		}
		svr.logger.Info("Stopping Master Down Timer for Ifindex:" +
			splitString[0] + " VRID:" + splitString[1])
		gblInfo.MasterDownTimer.Stop()
		// Transition to Init State
		gblInfo.StateLock.Lock()
		gblInfo.StateName = VRRP_INITIALIZE_STATE
		gblInfo.StateLock.Unlock()
		svr.vrrpGblInfo[key] = gblInfo
	}
}
