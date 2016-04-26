package api

import (
	"l3/bgp/config"
)

type ApiLayer struct {
	bfdCh  chan config.BfdInfo
	intfCh chan config.IntfStateInfo
}

var bgpapi *ApiLayer = nil

/*  Initialize bgp api layer with the channels that will be used for communicating
 *  with the server
 */
func Init(bfdCh chan config.BfdInfo, intfCh chan config.IntfStateInfo) {
	bgpapi = new(ApiLayer)
	bgpapi.bfdCh = bfdCh
	bgpapi.intfCh = intfCh
}

/*  Send bfd state information from bfd manager to server
 */
func SendBfdNotification(DestIp string, State bool, Oper config.Operation) {
	bgpapi.bfdCh <- config.BfdInfo{
		DestIp: DestIp,
		State:  State,
		Oper:   Oper,
	}
}
