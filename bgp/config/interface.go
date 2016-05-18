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

package config

/*  Port/Interface state change manager.
 */
type IntfStateMgrIntf interface {
	Start()
	PortStateChange()
	GetIPv4Intfs() []*IntfStateInfo
	GetIPv4Information(ifIndex int32) (string, error)
	GetIfIndex(int, int) int32
}

/*  Adding routes to rib/switch/linux interface
 */
type RouteMgrIntf interface {
	Start()
	GetNextHopInfo(ipAddr string) (*NextHopInfo, error)
	CreateRoute(*RouteConfig)
	DeleteRoute(*RouteConfig)
	UpdateRoute(cfg *RouteConfig , op string)
	ApplyPolicy(protocol string, policy string, action string, conditions []*ConditionInfo)
	GetRoutes() ([]*RouteInfo, []*RouteInfo)
}

/*  Interface for handling policy related operations
 */
type PolicyMgrIntf interface {
	Start()
}

/*  Interface for handling bfd state notifications
 */
type BfdMgrIntf interface {
	Start()
	CreateBfdSession(ipAddr string, sessionParam string) (bool, error)
	DeleteBfdSession(ipAddr string) (bool, error)
}
