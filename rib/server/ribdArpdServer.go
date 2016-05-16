Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
// ribdArpdServer.go
package server

import (
	"arpdInt"
	"fmt"
)

func arpdResolveRoute(routeInfoRecord RouteInfoRecord) {
	logger.Info(fmt.Sprintln(" arpdResolveRoute: Sending ARP Resolve for ", routeInfoRecord.nextHopIp.String(), routeInfoRecord.nextHopIfType))
	arpdclnt.ClientHdl.ResolveArpIPV4(routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, arpdInt.Int(routeInfoRecord.nextHopIfType), arpdInt.Int(routeInfoRecord.nextHopIfIndex))
	logger.Info(fmt.Sprintln("ARP resolve for ", routeInfoRecord.resolvedNextHopIpIntf.NextHopIp, arpdInt.Int(routeInfoRecord.nextHopIfType), arpdInt.Int(routeInfoRecord.nextHopIfIndex), " returned "))
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
