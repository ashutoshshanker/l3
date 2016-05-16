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
package rpc

import (
	"arpd"
	"arpdInt"
	"fmt"
	"l3/arp/server"
)

func (h *ARPHandler) SendResolveArpIPv4(targetIp string, ifType arpdInt.Int, ifId arpdInt.Int) {
	rConf := server.ResolveIPv4{
		TargetIP: targetIp,
		IfType:   int(ifType),
		IfId:     int(ifId),
	}
	h.server.ResolveIPv4Ch <- rConf
	return
}

func (h *ARPHandler) SendSetArpConfig(refTimeout int) bool {
	arpConf := server.ArpConf{
		RefTimeout: refTimeout,
	}
	h.server.ArpConfCh <- arpConf
	return true
}

func (h *ARPHandler) ResolveArpIPV4(targetIp string, ifType arpdInt.Int, ifId arpdInt.Int) error {
	h.logger.Info(fmt.Sprintln("Received ResolveArpIPV4 call with targetIp:", targetIp, "ifType:", ifType, "ifId:", ifId))
	h.SendResolveArpIPv4(targetIp, ifType, ifId)
	return nil
}

func (h *ARPHandler) CreateArpConfig(conf *arpd.ArpConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received CreateArpConfig call with Timeout:", conf.Timeout))
	return h.SendSetArpConfig(int(conf.Timeout)), nil
}
