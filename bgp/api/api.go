package api

import (
	"l3/bgp/config"
	"sync"
)

type ApiLayer struct {
	bfdCh   chan config.BfdInfo
	intfCh  chan config.IntfStateInfo
	routeCh chan *config.RouteCh
	//routeCh chan []*config.RouteInfo
}

var bgpapi *ApiLayer = nil
var once sync.Once

/*  Singleton instance should be accesible only within api
 */
func getInstance() *ApiLayer {
	once.Do(func() {
		bgpapi = &ApiLayer{}
	})
	return bgpapi
}

/*  Initialize bgp api layer with the channels that will be used for communicating
 *  with the server
 */
func Init(bfdCh chan config.BfdInfo, intfCh chan config.IntfStateInfo, rCh chan *config.RouteCh) {
	bgpapi = getInstance()
	bgpapi.bfdCh = bfdCh
	bgpapi.intfCh = intfCh
	bgpapi.routeCh = rCh
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

/*  Send interface state notification to server
 */
func SendIntfNotification(ifIndex int32, ipAddr string, state config.Operation) {
	bgpapi.intfCh <- config.IntfStateInfo{
		Idx:    ifIndex,
		Ipaddr: ipAddr,
		State:  state,
	}
}

/*  Send Routes information to server
 */
func SendRouteNotification(add []*config.RouteInfo, remove []*config.RouteInfo) {
	bgpapi.routeCh <- &config.RouteCh{
		Add:    add,
		Remove: remove,
	}
}
