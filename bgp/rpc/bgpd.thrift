include "bgpdInt.thrift"
namespace go bgpd
typedef i32 int
typedef i16 uint16
struct BGPPeerGroup {
	1 : i32 PeerAS
	2 : i32 LocalAS
	3 : string AuthPassword
	4 : string Description
	5 : string Name
	6 : i32 RouteReflectorClusterId
	7 : bool RouteReflectorClient
	8 : bool MultiHopEnable
	9 : byte MultiHopTTL
	10 : i32 ConnectRetryTime
	11 : i32 HoldTime
	12 : i32 KeepaliveTime
	13 : bool AddPathsRx
	14 : byte AddPathsMaxTx
}
struct BGPPolicyStmtState {
	1 : string Name
	2 : string MatchConditions
	3 : list<string> Conditions
	4 : list<string> Actions
}
struct BGPPolicyStmtStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyStmtState> BGPPolicyStmtStateList
}
struct BGPPolicyActionState {
	1 : string Name
	2 : string ActionInfo
	3 : list<string> PolicyStmtList
}
struct BGPPolicyActionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyActionState> BGPPolicyActionStateList
}
struct BGPGlobalState {
	1 : i32 AS
	2 : string RouterId
	3 : bool UseMultiplePaths
	4 : i32 EBGPMaxPaths
	5 : bool EBGPAllowMultipleAS
	6 : i32 IBGPMaxPaths
	7 : i32 TotalPaths
	8 : i32 TotalPrefixes
}
struct BGPGlobalStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPGlobalState> BGPGlobalStateList
}
struct BGPPolicyDefinitionStmtPrecedence {
	1 : i32 Precedence
	2 : string Statement
}
struct BGPPolicyDefinitionState {
	1 : string Name
	2 : i32 HitCounter
	3 : list<string> IpPrefixList
}
struct BGPPolicyDefinitionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyDefinitionState> BGPPolicyDefinitionStateList
}
struct BGPQueues {
	1 : i32 Input
	2 : i32 Output
}
struct BGPGlobal {
	1 : i32 ASNum
	2 : string RouterId
	3 : bool UseMultiplePaths
	4 : i32 EBGPMaxPaths
	5 : bool EBGPAllowMultipleAS
	6 : i32 IBGPMaxPaths
}
struct BGPNeighborState {
	1 : i32 PeerAS
	2 : i32 LocalAS
	3 : byte PeerType
	4 : string AuthPassword
	5 : string Description
	6 : string NeighborAddress
	7 : i32 IfIndex
	8 : i32 SessionState
	9 : BGPMessages Messages
	10 : BGPQueues Queues
	11 : i32 RouteReflectorClusterId
	12 : bool RouteReflectorClient
	13 : bool MultiHopEnable
	14 : byte MultiHopTTL
	15 : i32 ConnectRetryTime
	16 : i32 HoldTime
	17 : i32 KeepaliveTime
	18 : string PeerGroup
	19 : string BfdNeighborState
	20 : bool AddPathsRx
	21 : byte AddPathsMaxTx
}
struct BGPNeighborStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPNeighborState> BGPNeighborStateList
}
struct BGPPolicyConditionState {
	1 : string Name
	2 : string ConditionInfo
	3 : list<string> PolicyStmtList
}
struct BGPPolicyConditionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyConditionState> BGPPolicyConditionStateList
}
struct BGPNeighbor {
	1 : i32 PeerAS
	2 : i32 LocalAS
	3 : string AuthPassword
	4 : string Description
	5 : string NeighborAddress
	6 : i32 IfIndex
	7 : i32 RouteReflectorClusterId
	8 : bool RouteReflectorClient
	9 : bool MultiHopEnable
	10 : byte MultiHopTTL
	11 : i32 ConnectRetryTime
	12 : i32 HoldTime
	13 : i32 KeepaliveTime
	14 : bool AddPathsRx
	15 : byte AddPathsMaxTx
	16 : string PeerGroup
	17 : bool BfdEnable
}
struct BGPRoute {
	1 : string Network
	2 : i16 CIDRLen
	3 : string NextHop
	4 : i32 Metric
	5 : i32 LocalPref
	6 : list<string> Path
	7 : string Updated
	8 : i32 PathId
}
struct BGPRouteGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPRoute> BGPRouteList
}
struct BGPPolicyDefinition {
	1 : string Name
	2 : i32 Precedence
	3 : string MatchType
	4 : list<BGPPolicyDefinitionStmtPrecedence> StatementList
}
struct BGPMessages {
	1 : BGPCounters Sent
	2 : BGPCounters Received
}
struct BGPPolicyAction {
	1 : string Name
	2 : string ActionType
	3 : bool GenerateASSet
	4 : bool SendSummaryOnly
}
struct BGPPolicyStmt {
	1 : string Name
	2 : string MatchConditions
	3 : list<string> Conditions
	4 : list<string> Actions
}
struct BGPPolicyCondition {
	1 : string Name
	2 : string ConditionType
	3 : string IpPrefix
	4 : string MaskLengthRange
}
struct BGPCounters {
	1 : i64 Update
	2 : i64 Notification
}
service BGPDServices extends bgpdInt.BGPDINTServices {
	bool CreateBGPPeerGroup(1: BGPPeerGroup config);
	bool UpdateBGPPeerGroup(1: BGPPeerGroup origconfig, 2: BGPPeerGroup newconfig, 3: list<bool> attrset);
	bool DeleteBGPPeerGroup(1: BGPPeerGroup config);

	BGPPolicyStmtStateGetInfo GetBulkBGPPolicyStmtState(1: int fromIndex, 2: int count);
	BGPPolicyStmtState GetBGPPolicyStmtState(1: string Name);
	BGPPolicyActionStateGetInfo GetBulkBGPPolicyActionState(1: int fromIndex, 2: int count);
	BGPPolicyActionState GetBGPPolicyActionState(1: string Name);
	BGPGlobalStateGetInfo GetBulkBGPGlobalState(1: int fromIndex, 2: int count);
	BGPGlobalState GetBGPGlobalState(1: string RouterId);
	BGPPolicyDefinitionStateGetInfo GetBulkBGPPolicyDefinitionState(1: int fromIndex, 2: int count);
	BGPPolicyDefinitionState GetBGPPolicyDefinitionState(1: string Name);
	bool CreateBGPGlobal(1: BGPGlobal config);
	bool UpdateBGPGlobal(1: BGPGlobal origconfig, 2: BGPGlobal newconfig, 3: list<bool> attrset);
	bool DeleteBGPGlobal(1: BGPGlobal config);

	BGPNeighborStateGetInfo GetBulkBGPNeighborState(1: int fromIndex, 2: int count);
	BGPNeighborState GetBGPNeighborState(1: string NeighborAddress, 2: i32 IfIndex);
	BGPPolicyConditionStateGetInfo GetBulkBGPPolicyConditionState(1: int fromIndex, 2: int count);
	BGPPolicyConditionState GetBGPPolicyConditionState(1: string Name);
	bool CreateBGPNeighbor(1: BGPNeighbor config);
	bool UpdateBGPNeighbor(1: BGPNeighbor origconfig, 2: BGPNeighbor newconfig, 3: list<bool> attrset);
	bool DeleteBGPNeighbor(1: BGPNeighbor config);

	BGPRouteGetInfo GetBulkBGPRoute(1: int fromIndex, 2: int count);
	BGPRoute GetBGPRoute(1: string Network, 2: i16 CIDRLen, 3: string NextHop);
	bool CreateBGPPolicyDefinition(1: BGPPolicyDefinition config);
	bool UpdateBGPPolicyDefinition(1: BGPPolicyDefinition origconfig, 2: BGPPolicyDefinition newconfig, 3: list<bool> attrset);
	bool DeleteBGPPolicyDefinition(1: BGPPolicyDefinition config);

	bool CreateBGPPolicyAction(1: BGPPolicyAction config);
	bool UpdateBGPPolicyAction(1: BGPPolicyAction origconfig, 2: BGPPolicyAction newconfig, 3: list<bool> attrset);
	bool DeleteBGPPolicyAction(1: BGPPolicyAction config);

	bool CreateBGPPolicyStmt(1: BGPPolicyStmt config);
	bool UpdateBGPPolicyStmt(1: BGPPolicyStmt origconfig, 2: BGPPolicyStmt newconfig, 3: list<bool> attrset);
	bool DeleteBGPPolicyStmt(1: BGPPolicyStmt config);

	bool CreateBGPPolicyCondition(1: BGPPolicyCondition config);
	bool UpdateBGPPolicyCondition(1: BGPPolicyCondition origconfig, 2: BGPPolicyCondition newconfig, 3: list<bool> attrset);
	bool DeleteBGPPolicyCondition(1: BGPPolicyCondition config);

}