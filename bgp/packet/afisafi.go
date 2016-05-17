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

// bgp.go
package packet

import (
	"l3/bgp/config"
)

type AFI uint16
type SAFI uint8

const (
	AfiIP AFI = iota + 1
	AfiIP6
)

const (
	SafiUnicast SAFI = iota + 1
	SafiMulticast
)

var ProtocolFamilyMap = map[string]uint32{
	"ipv4-unicast":   GetProtocolFamily(AfiIP, SafiUnicast),
	"ipv6-unicast":   GetProtocolFamily(AfiIP6, SafiUnicast),
	"ipv4-multicast": GetProtocolFamily(AfiIP, SafiMulticast),
	"ipv6-multicast": GetProtocolFamily(AfiIP6, SafiMulticast),
}

func GetProtocolFromConfig(afiSafis *[]config.AfiSafiConfig) (map[uint32]bool, bool) {
	afiSafiMap := make(map[uint32]bool)
	rv := true
	for _, afiSafi := range *afiSafis {
		if afiSafiVal, ok := ProtocolFamilyMap[afiSafi.AfiSafiName]; ok {
			afiSafiMap[afiSafiVal] = true
		} else {
			rv = false
			break
		}
	}

	if len(afiSafiMap) == 0 {
		afiSafiMap[ProtocolFamilyMap["ipv4-unicast"]] = true
	}
	return afiSafiMap, rv
}

func GetProtocolFamily(afi AFI, safi SAFI) uint32 {
	return uint32(afi<<8) | uint32(safi)
}

func GetAfiSafi(protocolFamily uint32) (AFI, SAFI) {
	return AFI(protocolFamily >> 8), SAFI(protocolFamily & 0xFF)
}

func GetProtocolFromOpenMsg(openMsg *BGPOpen) map[uint32]bool {
	afiSafiMap := make(map[uint32]bool)
	for _, optParam := range openMsg.OptParams {
		if capabilities, ok := optParam.(*BGPOptParamCapability); ok {
			for _, capability := range capabilities.Value {
				if val, ok := capability.(*BGPCapMPExt); ok {
					afiSafiMap[GetProtocolFamily(val.AFI, val.SAFI)] = true
				}
			}
		}
	}

	return afiSafiMap
}
