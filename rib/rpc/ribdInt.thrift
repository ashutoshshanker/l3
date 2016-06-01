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
// _______   __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----  \   \/    \/   /  |  |  ---|  |---- |  ,---- |  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

namespace go ribdInt
typedef i32 int
struct NextHopInfo {
    2: string NextHopIp,
    3: int NextHopIfIndex,
	4: int Metric,
	5: string Ipaddr,
	6: string Mask,
	7: bool IsReachable
}
struct Routes {
	1: string Ipaddr,
	2: string Mask,
	3: string NextHopIp,
	5: int IfIndex,
	6: int Metric,
	7: int Prototype,
	8: bool IsValid,
	9: int SliceIdx,
	10: int PolicyHitCounter,
	11: list<string> PolicyList,
//	11: map<string,list<string>> PolicyList,
    12: 	bool IsPolicyBasedStateValid,
	13: string RouteCreated,
	14: string RouteUpdated,
	15: string RoutePrototypeString
	16: string DestNetIp
	17: bool NetworkStatement
	18: string RouteOrigin
	19: int Weight
}
struct RoutesGetInfo {
	1: int StartIdx,
	2: int EndIdx,
	3: int Count,
	4: bool More,
	5: list<Routes> RouteList,
}
struct PolicyAction {
	1 : string Name
	2 : string ActionType
	3 : i32 SetAdminDistanceValue
	4 : bool Accept
	5 : bool Reject
	6 : string RedistributeAction
	7 : string RedistributeTargetProtocol
	8 : string NetworkStatementTargetProtocol
}
struct PolicyPrefix {
	1 : string	IpPrefix,
	2 : string 	MasklengthRange,
}
struct PolicyPrefixSet{
	1 : string 	PrefixSetName,
	2 : list<PolicyPrefix> 	IpPrefixList,
}
struct PolicyPrefixSetGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyPrefixSet> PolicyPrefixSetList
}
struct PolicyDstIpMatchPrefixSetCondition{
	1 : string 	PrefixSet
	2 : PolicyPrefix Prefix
}
struct IPv4Route {
	1 : string DestinationNw
	2 : string NetworkMask
	3 : string NextHopIp
	4 : i32 Cost
	6 : string NextHopIntRef
	7 : string Protocol
	8 : string CreateTime
	9 : i32    Weight
}
struct ConditionInfo {
	1 : string ConditionType
	2 : string Protocol
	3 : string IpPrefix
	4 : string MasklengthRange 
}
struct PatchOpInfo {
    1 : string Op
    2 : string Path
    3 : list<map<string,string>> Value
}

service RIBDINTServices 
{
    NextHopInfo getRouteReachabilityInfo(1: string desIPv4MasktNet);
	string GetNextHopIfTypeStr(1: int nextHopIfType);
	//list<Routes> getConnectedRoutesInfo();
    //void printV4Routes();
	RoutesGetInfo getBulkRoutesForProtocol(1: string srcProtocol, 2: int fromIndex ,3: int rcount)
    void TrackReachabilityStatus(1: string ipAddr, 2: string protocol, 3:string op) //op:"add"/"del"
	//RoutesGetInfo getBulkRoutes(1: int fromIndex, 2: int count);
	Routes getRoute(1: string destNetIp, 2:string networkMask);
	oneway void OnewayCreateBulkIPv4Route(1: list<IPv4Route> config);
	bool CreatePolicyAction(1: PolicyAction config);
	bool UpdatePolicyAction(1: PolicyAction origconfig, 2: PolicyAction newconfig, 3: list<bool> attrset, 4: list<PatchOpInfo> op);
	bool DeletePolicyAction(1: PolicyAction config);
	void ApplyPolicy(1: string source, 2: string policy, 3: string action, 4: list<ConditionInfo>conditions)
	void UpdateApplyPolicy(1: string source, 2: string policy, 3: string action, 4: list<ConditionInfo>conditions)
}
