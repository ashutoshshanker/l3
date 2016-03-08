namespace go ribd
typedef i32 int
struct NextHopInfo {
	1: int NextHopIfType,
    2: string NextHopIp,
    3: int NextHopIfIndex,
	4: int Metric,
	5: string Ipaddr,
	6: string Mask
}
struct Routes {
	1: string Ipaddr,
	2: string Mask,
	3: string NextHopIp,
	4: int NextHopIfType
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
}
struct RoutesGetInfo {
	1: int StartIdx,
	2: int EndIdx,
	3: int Count,
	4: bool More,
	5: list<Routes> RouteList,
}
struct IPV4EventState{
	1 : string 	TimeStamp
	2 : string   EventInfo
}
struct IPV4EventStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<IPV4EventState> IPV4EventStateList
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
//Neighbor 
//NeighborSet 
struct PolicyMatchNeighborSetCondition{
	1 : string 	NeighborSet
	2 : i32 	MatchSetOptions
}
//Tag 
//TagSet
struct PolicyMatchTagSetCondition{
	1 : string 	TagSet
	2 : i32 	MatchSetOptions
}

struct PolicyStmtConfig{
	1:  string  Name
	2 : string 	AdminState
	3 : string 	MatchConditions
	4 : list<string> 	Conditions
	5 : list<string> 	Actions
	//6 : bool     Export
	//7 : bool     Import
}
struct PolicyStmtState{
	1 : string  Name
	2 : string 	AdminState
	3 : string 	OperState
	4 : string 	MatchConditions
	5 : list<string> 	Conditions
	6 : list<string> 	Actions
	7 : list<string> 	PolicyList
}
struct PolicyStmtStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyStmtState> PolicyStmtStateList
}
struct PolicyConditionConfig {
	1: string Name
	2: string ConditionType
	3: string MatchProtocolConditionInfo           
    4: optional PolicyDstIpMatchPrefixSetCondition MatchDstIpPrefixConditionInfo        
}
struct PolicyConditionState{
	1 : string 	Name
	2 : string 	ConditionInfo
	3 : list<string> 	PolicyStmtList
}
struct PolicyConditionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyConditionState> PolicyConditionStateList
}
struct PolicyActionConfig {
	1: string Name
	2: string ActionType
	3: int SetAdminDistanceValue
	4: bool Accept 
	5: bool Reject 
	6: string RedistributeAction
	7: string RedistributeTargetProtocol   
	8: string NetworkStatementTargetProtocol 
}

struct PolicyActionState{
	1 : string 	Name
	2 : string 	ActionInfo
	3 : list<string> 	PolicyStmtList
}
struct PolicyActionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyActionState> PolicyActionStateList
}

struct PolicyDefinitionStmtPrecedence  {
	1: int Precedence
	2: string Statement
}
struct PolicyDefinitionConfig{
	1: string Name
	2: int Precedence
	3: string MatchType
	4: list<PolicyDefinitionStmtPrecedence> PolicyDefinitionStatements
}

struct PolicyDefinitionState{
	1 : string  Name
	2 : int      HitCounter
	3: list<string> IpPrefixList
}
struct PolicyDefinitionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionState> PolicyDefinitionStateList
}

struct RouteDistanceState{
	1 : string 	Protocol
	2 : int     Distance
}
struct RouteDistanceStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<RouteDistanceState> RouteDistanceStateList
}
//typedef RouteList  list<Routes>
service RouteService 
{
    int createV4Route (1:string destNetIp, 2:string networkMask, 3:int metric, 4:string nextHopIp, 5: int nextHopIfType, 6:int nextHopIfIndex, 7:string routeType);
    //void updateV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType, 4:string nextHopIp, 5:int nextHopIfIndex, 6:int metric);
	bool UpdateIPV4Route(1: Routes origconfig, 2: Routes newconfig, 3: list<bool> attrset);
    int deleteV4Route (1:string destNetIp, 2:string networkMask, 3:string routeType, 4:string nextHopIp);
    NextHopInfo getRouteReachabilityInfo(1: string desIPv4MasktNet);
	//list<Routes> getConnectedRoutesInfo();
    void printV4Routes();
	RoutesGetInfo getBulkRoutes(1: int fromIndex, 2: int count);
	RoutesGetInfo getBulkRoutesForProtocol(1: string srcProtocol, 2: int fromIndex ,3: int rcount)
	IPV4EventStateGetInfo GetBulkIPV4EventState(1: int fromIndex, 2: int count);
	Routes getRoute(1: string destNetIp, 2:string networkMask);
	void linkDown(1: int ifType, 2:int ifIndex);
	void linkUp(1: int ifType, 2:int ifIndex);
	void intfUp(1:string ipAddr);
	void intfDown(1:string ipAddr);

	RouteDistanceStateGetInfo GetBulkRouteDistanceState(1: int fromIndex, 2: int count);

	bool CreatePolicyPrefixSet(1: PolicyPrefixSet config);
//	bool UpdatePolicyPrefixSet(1: PolicyPrefixSet origconfig, 2: PolicyPrefixSet newconfig, 3: list<bool> attrset);
//	bool DeletePolicyPrefixSet(1: PolicyPrefixSet config);

	bool CreatePolicyCondition(1: PolicyConditionConfig config);
//	bool UpdatePolicyCondition(1: PolicyConditionConfig origconfig, 2: PolicyConditionConfig newconfig, 3: list<bool> attrset);
//	bool DeletePolicyCondition(1: PolicyConditionConfig config);

	bool CreatePolicyAction(1: PolicyActionConfig config);
//	bool UpdatePolicyAction(1: PolicyActionConfig origconfig, 2: PolicyActionConfig newconfig, 3: list<bool> attrset);
//	bool DeletePolicyAction(1: PolicyActionConfig config);

	bool CreatePolicyStatement(1: PolicyStmtConfig config);
//	bool UpdatePolicyStatement(1: PolicyStmtConfig origconfig, 2: PolicyStmtConfig newconfig, 3: list<bool> attrset);
	bool DeletePolicyStatement(1: PolicyStmtConfig config);

	PolicyStmtStateGetInfo GetBulkPolicyStmtState(1: int fromIndex, 2: int count);
	PolicyConditionStateGetInfo GetBulkPolicyConditionState(1: int fromIndex, 2: int count);
	PolicyActionStateGetInfo GetBulkPolicyActionState(1: int fromIndex, 2: int count);

	bool CreatePolicyDefinition(1: PolicyDefinitionConfig config);
//	bool UpdatePolicyDefinition(1: PolicyDefinitionConfig origconfig, 2: PolicyDefinitionConfig newconfig, 3: list<bool> attrset);
	bool DeletePolicyDefinition(1: PolicyDefinitionConfig config);
	PolicyDefinitionStateGetInfo GetBulkPolicyDefinitionState(1: int fromIndex, 2: int count);
}
