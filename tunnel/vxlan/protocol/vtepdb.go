//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

// vtepdb.go
package vxlan

import (
	"encoding/json"
	"io/ioutil"
	"net"
)

type vtepDbKey struct {
	VtepId  uint32
	VxlanId uint32
}

var vtepDB map[vtepDbKey]*vtepDbEntry

var VxlanVtepSrcIp net.IP
var VxlanVtepSrcNetMac net.HardwareAddr
var VxlanVtepSrcMac [6]uint8

type vtepStatus string

const (
	VtepStatusUp         vtepStatus = "UP"
	VtepStatusDown                  = "DOWN"
	VtepStatusAdminDown             = "ADMIN DOWN"
	VtepStatusIncomplete            = "INCOMPLETE PROV"
)

type vtepDbEntry struct {
	VtepId                uint32
	VxlanId               uint32
	VtepName              string
	SrcIfName             string
	UDP                   uint16
	TTL                   uint16
	TOS                   uint16
	InnerVlanHandlingMode bool
	Learning              bool
	Rsc                   bool
	L2miss                bool
	L3miss                bool
	SrcIp                 net.IP
	DstIp                 net.IP
	VlanId                uint16
	SrcMac                net.HardwareAddr
	DstMac                net.HardwareAddr
	Status                vtepStatus
	VtepIfIndex           int32
}

/* TODO may need to keep a table to map customer macs to vtep
type srcMacVtepMap struct {
	SrcMac      net.HardwareAddr
	VtepIfIndex int32
}
*/

func NewVtepDbEntry(c *VtepConfig) *vtepDbEntry {
	vtep := &vtepDbEntry{
		VtepId:    c.VtepId,
		VxlanId:   c.VxlanId,
		VtepName:  c.VtepName,
		SrcIfName: c.SrcIfName,
		UDP:       c.UDP,
		TTL:       c.TTL,
		TOS:       c.TOS,
		InnerVlanHandlingMode: c.InnerVlanHandlingMode,
		Learning:              c.Learning,
		Rsc:                   c.Rsc,
		L2miss:                c.L2miss,
		L3miss:                c.L3miss,
		DstIp:                 c.TunnelDstIp,
		SrcIp:                 c.TunnelSrcIp,
		SrcMac:                c.TunnelSrcMac,
		DstMac:                c.TunnelDstMac,
		VlanId:                c.VlanId,
	}
	return vtep
}

func (s *VXLANServer) saveVtepConfigData(c *VtepConfig) {
	key := vtepDbKey{
		VtepId:  c.VtepId,
		VxlanId: c.VxlanId,
	}
	if _, ok := vtepDB[key]; !ok {
		vtep := NewVtepDbEntry(c)
		vtepDB[key] = vtep
	}
}

func (s *VXLANServer) SaveVtepSrcMacSrcIp(paramspath string) {
	var cfgFile cfgFileJson
	asicdconffilename := paramspath + "asicd.conf"
	cfgFileData, err := ioutil.ReadFile(asicdconffilename)
	if err != nil {
		s.logger.Info("Error reading config file - asicd.conf")
		return
	}
	err = json.Unmarshal(cfgFileData, &cfgFile)
	if err != nil {
		s.logger.Info("Error parsing config file")
		return
	}

	VxlanVtepSrcNetMac, _ := net.ParseMAC(cfgFile.SwitchMac)
	VxlanVtepSrcMac = [6]uint8{VxlanVtepSrcNetMac[0], VxlanVtepSrcNetMac[1], VxlanVtepSrcNetMac[2], VxlanVtepSrcNetMac[3], VxlanVtepSrcNetMac[4], VxlanVtepSrcNetMac[5]}

}
