// ribdAsicdServer.go
package server

import (
	"asicdInt"
	"fmt"
)

func addAsicdRoute(routeInfoRecord RouteInfoRecord) {
	logger.Info(fmt.Sprintln("addAsicdRoute, weight = ", routeInfoRecord.weight+1))
	asicdclnt.ClientHdl.OnewayCreateIPv4Route([]*asicdInt.IPv4Route{
		&asicdInt.IPv4Route{
			routeInfoRecord.destNetIp.String(),
			routeInfoRecord.networkMask.String(),
			[]*asicdInt.IPv4NextHop{
				&asicdInt.IPv4NextHop{
					NextHopIp: routeInfoRecord.resolvedNextHopIpIntf.NextHopIp,
					Weight:    int32(routeInfoRecord.weight + 1),
				},
			},
		},
	})
}
func delAsicdRoute(routeInfoRecord RouteInfoRecord) {
	logger.Info("delAsicdRoute")
	asicdclnt.ClientHdl.OnewayDeleteIPv4Route([]*asicdInt.IPv4Route{
		&asicdInt.IPv4Route{
			routeInfoRecord.destNetIp.String(),
			routeInfoRecord.networkMask.String(),
			[]*asicdInt.IPv4NextHop{
				&asicdInt.IPv4NextHop{
					NextHopIp: routeInfoRecord.resolvedNextHopIpIntf.NextHopIp,
					Weight:    int32(routeInfoRecord.weight + 1),
					//NextHopIfType: int32(routeInfoRecord.resolvedNextHopIpIntf.NextHopIfType),
				},
			},
		},
	})
}
func (ribdServiceHandler *RIBDServer) StartAsicdServer() {
	logger.Info("Starting the asicdserver loop")
	for {
		select {
		case route := <-ribdServiceHandler.AsicdAddRouteCh:
			logger.Info(" received message on AsicdAddRouteCh")
			addAsicdRoute(route)
		case route := <-ribdServiceHandler.AsicdDelRouteCh:
			logger.Info(" received message on AsicdDelRouteCh")
			delAsicdRoute(route)
		}
	}
}
