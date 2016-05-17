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

// vxlandb.go
package vxlan

import (
	"net"
)

var vxlanDB map[uint32]*vxlanDbEntry

type vxlanDbEntry struct {
	VNI         uint32
	VlanId      uint16 // used to tag inner ethernet frame when egressing
	Group       net.IP // multicast group IP
	MTU         uint32 // MTU size for each VTEP
	VtepMembers []uint32
}

func NewVxlanDbEntry(c *VxlanConfig) *vxlanDbEntry {
	return &vxlanDbEntry{
		VNI:         c.VNI,
		VlanId:      c.VlanId,
		Group:       c.Group,
		MTU:         c.MTU,
		VtepMembers: make([]uint32, 0),
	}
}

func (s *VXLANServer) saveVxLanConfigData(c *VxlanConfig) {
	if _, ok := vxlanDB[c.VNI]; !ok {
		vxlan := NewVxlanDbEntry(c)
		vxlanDB[c.VNI] = vxlan
	}
}
