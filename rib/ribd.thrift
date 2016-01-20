namespace go ribd
typedef i32 int
struct NextHopInfo {
	1: int NextHopIfType,
    2: string NextHopIp,
    3: int NextHopIfIndex,
	4: int Metric,
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
}
struct RoutesGetInfo {
	1: int StartIdx,
	2: int EndIdx,
	3: int Count,
	4: bool More,
	5: list<Routes> RouteList,
}
struct PolicyDefinitionSetsPrefix {
	1 : string	IpPrefix,
	2 : string 	MasklengthRange,
}
struct PolicyDefinitionSetsPrefixSet{
	1 : string 	PrefixSetName,
	2 : list<PolicyDefinitionSetsPrefix> 	IpPrefixList,
}
struct PolicyDefinitionSetsPrefixSetGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionSetsPrefixSet> PolicyDefinitionSetsPrefixSetList
}
struct PolicyDefinitionStatementMatchPrefixSet{
	1 : string 	PrefixSet
	2 : i32 	MatchSetOptions
}
struct PolicyDefinitionStatementMatchPrefixSetGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionStatementMatchPrefixSet> PolicyDefinitionStatementMatchPrefixSetList
}
//Neighbor 
//NeighborSet 
struct PolicyDefinitionStatementMatchNeighborSet{
	1 : string 	NeighborSet
	2 : i32 	MatchSetOptions
}
struct PolicyDefinitionStatementMatchNeighborSetGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionStatementMatchNeighborSet> PolicyDefinitionStatementMatchNeighborSetList
}
//Tag 
//TagSet
struct PolicyDefinitionStatementMatchTagSet{
	1 : string 	TagSet
	2 : i32 	MatchSetOptions
}
struct PolicyDefinitionStatementMatchTagSetGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionStatementMatchTagSet> PolicyDefinitionStatementMatchTagSetList
}
struct PolicyDefinitionStatementIgpActions{
	1 : set<i32> 	SetTag
}
struct PolicyDefinitionStatementIgpActionsGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionStatementIgpActions> PolicyDefinitionStatementIgpActionsList
}
struct PolicyDefinitionStatement{
    1 : string   Name
	2 : PolicyDefinitionStatementMatchPrefixSet MatchPrefixSetInfo
	3 : int 	InstallProtocolEq
	4 : string   RouteDisposition
	//5 : 
}
struct PolicyDefinitionStatementGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionStatement> PolicyDefinitionStatementList
}
struct PolicyDefinition{
	1: string Name
	2: list<string> PolicyDefinitionStatements
}
struct PolicyDefinitionGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinition> PolicyDefinitionList
}

//typedef RouteList  list<Routes>
service RouteService 
{
    int createV4Route (1:string destNetIp, 2:string networkMask, 3:int metric, 4:string nextHopIp, 5: int nextHopIfType, 6:int nextHopIfIndex, 7:int routeType);
    void updateV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType, 4:string nextHopIp, 5:int nextHopIfIndex, 6:int metric);
    int deleteV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType);
    NextHopInfo getRouteReachabilityInfo(1: string desIPv4MasktNet);
	list<Routes> getConnectedRoutesInfo();
    void printV4Routes();
	RoutesGetInfo getBulkRoutes(1: int fromIndex, 2: int count);
	Routes getRoute(1: string destNetIp, 2:string networkMask);
	void linkDown(1: int ifType, 2:int ifIndex);
	void linkUp(1: int ifType, 2:int ifIndex);

	bool CreatePolicyDefinitionSetsPrefixSet(1: PolicyDefinitionSetsPrefixSet config);
//	bool UpdatePolicyDefinitionSetsPrefixSet(1: PolicyDefinitionSetsPrefixSet origconfig, 2: PolicyDefinitionSetsPrefixSet newconfig, 3: list<bool> attrset);
//	bool DeletePolicyDefinitionSetsPrefixSet(1: PolicyDefinitionSetsPrefixSet config);

	bool CreatePolicyDefinitionStatementMatchPrefixSet(1: PolicyDefinitionStatementMatchPrefixSet config);
//	bool UpdatePolicyDefinitionStatementMatchPrefixSet(1: PolicyDefinitionStatementMatchPrefixSet origconfig, 2: PolicyDefinitionStatementMatchPrefixSet newconfig, 3: list<bool> attrset);
//	bool DeletePolicyDefinitionStatementMatchPrefixSet(1: PolicyDefinitionStatementMatchPrefixSet config);

	//bool CreatePolicyDefinitionStatementMatchNeighborSet(1: PolicyDefinitionStatementMatchNeighborSet config);
	//bool UpdatePolicyDefinitionStatementMatchNeighborSet(1: PolicyDefinitionStatementMatchNeighborSet origconfig, 2: PolicyDefinitionStatementMatchNeighborSet newconfig, 3: list<bool> attrset);
	//bool DeletePolicyDefinitionStatementMatchNeighborSet(1: PolicyDefinitionStatementMatchNeighborSet config);

	//bool CreatePolicyDefinitionStatementMatchTagSet(1: PolicyDefinitionStatementMatchTagSet config);
	//bool UpdatePolicyDefinitionStatementMatchTagSet(1: PolicyDefinitionStatementMatchTagSet origconfig, 2: PolicyDefinitionStatementMatchTagSet newconfig, 3: list<bool> attrset);
	//bool DeletePolicyDefinitionStatementMatchTagSet(1: PolicyDefinitionStatementMatchTagSet config);

	//bool CreatePolicyDefinitionStatementIgpActions(1: PolicyDefinitionStatementIgpActions config);
	//bool UpdatePolicyDefinitionStatementIgpActions(1: PolicyDefinitionStatementIgpActions origconfig, 2: PolicyDefinitionStatementIgpActions newconfig, 3: list<bool> attrset);
//	bool DeletePolicyDefinitionStatementIgpActions(1: PolicyDefinitionStatementIgpActions config);

	bool CreatePolicyDefinitionStatement(1: PolicyDefinitionStatement config);
//	bool UpdatePolicyDefinitionStatement(1: PolicyDefinitionStatement origconfig, 2: PolicyDefinitionStatement newconfig, 3: list<bool> attrset);
//	bool DeletePolicyDefinitionStatement(1: PolicyDefinitionStatement config);
    PolicyDefinitionStatementGetInfo getBulkPolicyStmts(1: int fromIndex, 2: int count);

	bool CreatePolicyDefinition(1: PolicyDefinition config);
//	bool UpdatePolicyDefinition(1: PolicyDefinition origconfig, 2: PolicyDefinition newconfig, 3: list<bool> attrset);
//	bool DeletePolicyDefinition(1: PolicyDefinition config);
}
