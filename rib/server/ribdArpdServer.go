// ribdArpdServer.go
package server

import (
	"arpdInt"
	"fmt"
)

func arpdResolveRoute(routeInfoRecord RouteInfoRecord) {
	logger.Info(fmt.Sprintln(" arpdResolveRoute: Sending ARP Resolve for ", routeInfoRecord.nextHopIp.String(), routeInfoRecord.nextHopIfType))
	arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, arpdInt.Int(routeInfoRecord.resolvedNextHopIpIntf.NextHopIfType), arpdInt.Int(routeInfoRecord.resolvedNextHopIpIntf.NextHopIfIndex))
	logger.Info(fmt.Sprintln("ARP resolve for ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, arpdInt.Int(routeInfoRecord.resolvedNextHopIpIntf.NextHopIfType), arpdInt.Int(routeInfoRecord.resolvedNextHopIpIntf.NextHopIfIndex), " returned "))
}
func arpdRemoveRoute(routeInfoRecord RouteInfoRecord) {
	logger.Info(fmt.Sprintln("arpdRemoveRoute: for ", routeInfoRecord.nextHopIp.String()))
	arpdclnt.ClientHdl.DeleteResolveArpIPv4(routeInfoRecord.resolvedNextHopIpIntf.NextHopIp)
	logger.Info(fmt.Sprintln("ARP remove for ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, " returned "))
}
func (ribdServiceHandler *RIBDServer) StartArpdServer() {
	logger.Info("Starting the arpdserver loop")
	for {
		select {
		case route := <-ribdServiceHandler.ArpdResolveRouteCh:
			logger.Info(" received message on ArpdResolveRouteCh")
			arpdResolveRoute(route)
		case route := <-ribdServiceHandler.ArpdRemoveRouteCh:
			logger.Info(" received message on ArpdRemoveRouteCh")
			arpdRemoveRoute(route)
		}
	}
}
