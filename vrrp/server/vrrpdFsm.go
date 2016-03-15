package vrrpServer

import (
	"net"
)

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

func (svr *VrrpServer) VrrpCreateFsmObject(gblInfo VrrpGlobalInfo) (fsmObj VrrpFsm) {
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

func (svr *VrrpServer) VrrpFsmInitState(gblInfo *VrrpGlobalInfo, key string) {
	if gblInfo.IntfConfig.Priority == VRRP_MASTER_PRIORITY {
		// Go directly into master state..
		// (110) + Send an ADVERTISEMENT
		svr.vrrpTxPktCh <- key // Sending vrrp packet with the information

		// (115) + If the protected IPvX address is an IPv4 address, then:
		ip, _, _ := net.ParseCIDR(gblInfo.IpAddr)
		if ip.To4() != nil { // If not nill then its ipv4
			/*
			   (120) * Broadcast a gratuitous ARP request containing the
			   virtual router MAC address for each IP address associated
			   with the virtual router.
			*/
			svr.VrrpSendGratuitousArp(gblInfo)
		} else { // @TODO: ipv6 implementation
			// (125) + else // IPv6
			/*
			   (130) * For each IPv6 address associated with the virtual
			   router, send an unsolicited ND Neighbor Advertisement with
			   the Router Flag (R) set, the Solicited Flag (S) unset, the
			   Override flag (O) set, the target address set to the IPv6
			   address of the virtual router, and the target link-layer
			   address set to the virtual router MAC address.
			*/
		}
	} else {
		/*
			(150) - else // rtr does not own virt addr

			(155) + Set Master_Adver_Interval to Advertisement_Interval

			(160) + Set the Master_Down_Timer to Master_Down_Interval

			(165) + Transition to the {Backup} state

		*/
		gblInfo.MasterAdverInterval = gblInfo.IntfConfig.AdvertisementInterval
		if gblInfo.IntfConfig.Priority != 0 && gblInfo.MasterAdverInterval != 0 {
			gblInfo.SkewTime = ((256 - gblInfo.IntfConfig.Priority) *
				gblInfo.MasterAdverInterval) / 256
		}
		gblInfo.MasterDownInterval = (3 * gblInfo.MasterAdverInterval) + gblInfo.SkewTime
		gblInfo.StateLock.Lock()
		gblInfo.StateName = VRRP_BACKUP_STATE
		gblInfo.StateLock.Unlock()
		// Transition to backup state first
	}
	svr.vrrpGblInfo[key] = *gblInfo
}

func (svr *VrrpServer) VrrpFsmStart(fsmObj VrrpFsm) {
	gblInfo := fsmObj.vrrpInFo
	key := fsmObj.key

	gblInfo.StateLock.Lock()
	currentState := gblInfo.StateName
	gblInfo.StateLock.Unlock()

	switch currentState {
	case VRRP_INITIALIZE_STATE:
		svr.VrrpFsmInitState(gblInfo, key)
	case VRRP_BACKUP_STATE:

	case VRRP_MASTER_STATE:
	}
}
