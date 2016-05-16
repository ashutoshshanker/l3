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
package server

import (
	"fmt"
)

func (server *ARPServer) printArpEntries() {
	server.logger.Debug("************")
	for ip, arpEnt := range server.arpCache {
		server.logger.Debug(fmt.Sprintln("IP:", ip, "MAC:", arpEnt.MacAddr, "VlanId:", arpEnt.VlanId, "IfName:", arpEnt.IfName, "IfIndex", arpEnt.L3IfIdx, "Counter:", arpEnt.Counter, "Timestamp:", arpEnt.TimeStamp, "PortNum:", arpEnt.PortNum, "Type:", arpEnt.Type))
	}
	server.logger.Debug("************")
}
