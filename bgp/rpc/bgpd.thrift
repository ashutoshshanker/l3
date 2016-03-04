namespace go bgpd
typedef i32 int

struct BGPPolicyPrefix {
	1 : string	IpPrefix,
	2 : string 	MasklengthRange,
}
struct BGPPolicyPrefixSet{
	1 : string 	PrefixSetName,
	2 : list<BGPPolicyPrefix> 	IpPrefixList,
}
struct BGPPolicyPrefixSetGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyPrefixSet> PolicyPrefixSetList
}

struct PolicyDstIpMatchPrefixSetCondition{
	1 : string 	PrefixSet
	2 : BGPPolicyPrefix Prefix
}

struct BGPPolicyConditionConfig{
	1 : string 	Name
	2 : string 	ConditionType
    3: optional PolicyDstIpMatchPrefixSetCondition MatchDstIpPrefixConditionInfo        
}
struct BGPPolicyConditionState{
	1 : string 	Name
	2 : string 	ConditionInfo
	3 : list<string> 	PolicyStmtList
}
struct BGPPolicyConditionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyConditionState> PolicyConditionStateList
}
struct BGPPolicyAggregateAction {
	1: bool GenerateASSet 
	2: bool SendSummaryOnly  
}

struct BGPPolicyActionConfig{
	1 : string 	Name
	2 : string 	ActionType
	3: optional BGPPolicyAggregateAction AggregateActionInfo      
}
struct BGPPolicyActionState{
	1 : string 	Name
	2 : string 	ActionInfo
	3 : list<string> 	PolicyStmtList
}
struct BGPPolicyActionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyActionState> PolicyActionStateList
}
struct BGPPolicyStmtConfig{
	1 : string 	Name
	2 : string 	MatchConditions
	3 : list<string> 	Conditions
	4 : list<string> 	Actions
}
struct BGPPolicyStmtState{
	1 : string 	Name
	2 : string 	MatchConditions
	3 : list<string> 	Conditions
	4 : list<string> 	Actions
}
struct BGPPolicyStmtStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyStmtState> PolicyStmtStateList
}

struct PolicyDefinitionStmtPrecedence  {
	1: int Precedence
	2: string Statement
}
struct BGPPolicyDefinitionConfig{
	1 : string 	Name
	2: int Precedence
	3 : string 	MatchType
	4: list<PolicyDefinitionStmtPrecedence> PolicyDefinitionStatements
	5 : bool 	Export
	6 : bool 	Import
	7 : bool 	Global
}
struct BGPPolicyDefinitionState{
	2 : string 	Name
	3 : int      HitCounter
	4 : list<string> 	IpPrefixList
}
struct BGPPolicyDefinitionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BGPPolicyDefinitionState> PolicyDefinitionStateList
}

struct BGPGlobalConfig {
	1: i32 ASNum,
	2: string RouterId,
	3: bool UseMultiplePaths
	4: i32 EBGPMaxPaths
	5: bool EBGPAllowMultipleAS
	6: i32 IBGPMaxPaths
}

struct BGPGlobalState {
	1: i32 AS,
	2: string RouterId,
	3: bool UseMultiplePaths
	4: i32 EBGPMaxPaths
	5: bool EBGPAllowMultipleAS
	6: i32 IBGPMaxPaths
	7: i32 TotalPaths,
	8: i32 TotalPrefixes,
}

enum PeerType {
	PeerTypeInternal = 0,
	PeerTypeExternal
}

struct BgpCounters {
	1: i64 Update,
	2: i64 Notification,
}

struct BGPMessages {
	1: BgpCounters Sent,
	2: BgpCounters Received,
}

struct BGPQueues {
	1: i32 Input
	2: i32 Output
}

struct BGPNeighborConfig {
	1: i32 PeerAS,
	2: i32 LocalAS,
	3: string AuthPassword,
	4: string Description,
	5: string NeighborAddress,
	6: i32 RouteReflectorClusterId,
	7: bool RouteReflectorClient,
	8: bool MultiHopEnable,
	9: byte MultiHopTTL,
	10: i32 ConnectRetryTime,
	11: i32 HoldTime,
	12: i32 KeepaliveTime,
	13: bool AddPathsRx,
	14: byte AddPathsMaxTx,
	15: bool BfdEnable
	16: string PeerGroup,
}

struct BGPNeighborState {
	1: i32 PeerAS,
	2: i32 LocalAS,
	3: string AuthPassword,
	4: PeerType PeerType,
	5: string Description,
	6: string NeighborAddress,
	7: i32 SessionState,
	8: BGPMessages Messages,
	9: BGPQueues Queues,
	10: i32 RouteReflectorClusterId,
	11: bool RouteReflectorClient,
	12: bool MultiHopEnable,
	13: byte MultiHopTTL,
	14: i32 ConnectRetryTime,
	15: i32 HoldTime,
	16: i32 KeepaliveTime,
	17: bool AddPathsRx,
	18: byte AddPathsMaxTx,
	19: string BfdNeighborState
	20: string GroupName,
}

struct BGPNeighborStateBulk {
	1: i64 CurrIndex,
	2: i64 NextIndex,
	3: i64 Count,
	4: bool More,
	5: list<BGPNeighborState> StateList,
}

struct BGPPeerGroup {
	1: i32 PeerAS,
	2: i32 LocalAS,
	3: string AuthPassword,
	4: string Description,
	5: string Name,
	6: i32 RouteReflectorClusterId,
	7: bool RouteReflectorClient,
	8: bool MultiHopEnable,
	9: byte MultiHopTTL,
	10: i32 ConnectRetryTime,
	11: i32 HoldTime,
	12: i32 KeepaliveTime,
	13: bool AddPathsRx,
	14: byte AddPathsMaxTx,
}

struct BGPRoute {
	1: string Network,
	2: i16 CIDRLen,
	3: string NextHop,
	4: i32 Metric,
	5: i32 LocalPref,
	6: list<i32> Path,
	7: string Updated,
	8: list<string> PolicyList,
	9: int PolicyHitCounter,
	10: i32 PathId,
}

struct BGPRouteBulk {
	1: i64 CurrIndex,
	2: i64 NextIndex,
	3: i64 Count,
	4: bool More,
	5: list<BGPRoute> RouteList,
}

struct BGPAggregate {
	1: string IPPrefix
	2: bool GenerateASSet
	3: bool SendSummaryOnly
}

service BGPServer
{
	bool CreateBGPPolicyConditionConfig(1: BGPPolicyConditionConfig config);
//	bool UpdateBGPPolicyConditionConfig(1: BGPPolicyConditionConfig origconfig, 2: BGPPolicyConditionConfig newconfig, 3: list<bool> attrset);
//	bool DeleteBGPPolicyConditionConfig(1: BGPPolicyConditionConfig config);

	BGPPolicyConditionStateGetInfo GetBulkBGPPolicyConditionState(1: int fromIndex, 2: int count);
	bool CreateBGPPolicyActionConfig(1: BGPPolicyActionConfig config);
//	bool UpdateBGPPolicyActionConfig(1: BGPPolicyActionConfig origconfig, 2: BGPPolicyActionConfig newconfig, 3: list<bool> attrset);
//	bool DeleteBGPPolicyActionConfig(1: BGPPolicyActionConfig config);

	BGPPolicyActionStateGetInfo GetBulkBGPPolicyActionState(1: int fromIndex, 2: int count);
	bool CreateBGPPolicyStmtConfig(1: BGPPolicyStmtConfig config);
//	bool UpdateBGPPolicyStmtConfig(1: BGPPolicyStmtConfig origconfig, 2: BGPPolicyStmtConfig newconfig, 3: list<bool> attrset);
	bool DeleteBGPPolicyStmtConfig(1: string name);

	BGPPolicyStmtStateGetInfo GetBulkBGPPolicyStmtState(1: int fromIndex, 2: int count);
	bool CreateBGPPolicyDefinitionConfig(1: BGPPolicyDefinitionConfig config);
//	bool UpdateBGPPolicyDefinitionConfig(1: BGPPolicyDefinitionConfig origconfig, 2: BGPPolicyDefinitionConfig newconfig, 3: list<bool> attrset);
	bool DeleteBGPPolicyDefinitionConfig(1: string name);

	BGPPolicyDefinitionStateGetInfo GetBulkBGPPolicyDefinitionState(1: int fromIndex, 2: int count);
	bool CreateBGPGlobal(1: BGPGlobalConfig bgpConf);
	BGPGlobalState GetBGPGlobal();
	bool UpdateBGPGlobal(1: BGPGlobalConfig origGlobal, 2: BGPGlobalConfig updatedGlobal, 3: list<bool> attrSet);
	//bool DeleteBGPGlobal(1: BGPGlobal bgpConf);

	bool CreateBGPNeighbor(1: BGPNeighborConfig neighbor);
	BGPNeighborState GetBGPNeighbor(1: string ip);
	BGPNeighborStateBulk BulkGetBGPNeighbors(1: i64 index, 2: i64 count);
	bool UpdateBGPNeighbor(1: BGPNeighborConfig origNeighbor, 2: BGPNeighborConfig updatedNeighbor, 3: list<bool> attrSet);
	bool DeleteBGPNeighbor(1: string neighborAddress);

	bool CreateBGPPeerGroup(1: BGPPeerGroup group);
	bool UpdateBGPPeerGroup(1: BGPPeerGroup origGroup, 2: BGPPeerGroup updatedGroup, 3: list<bool> attrSet);
	bool DeleteBGPPeerGroup(1: string groupName);

	list<BGPRoute> GetBGPRoute(1: string ip);
	BGPRouteBulk BulkGetBGPRoutes(1: i64 index, 2: i64 count);

	bool CreateBGPAggregate(1: BGPAggregate agg);
	bool UpdateBGPAggregate(1: BGPAggregate origAgg, 2: BGPAggregate updatedAgg, 3: list<bool> attrSet);
	bool DeleteBGPAggregate(1: string ipPrefix);
}
