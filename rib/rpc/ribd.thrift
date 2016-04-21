include "ribdInt.thrift"
namespace go ribd
typedef i32 int
typedef i16 uint16
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
struct PolicyDefinition {
	1 : string Name
	2 : i32 Precedence
	3 : string MatchType
	4 : list<PolicyDefinitionStmtPrecedence> StatementList
}
struct IPv4RouteState {
	1 : string DestinationNw
	2 : string NextHopIp
	3 : string OutgoingIntfType
	4 : string OutgoingInterface
	5 : string Protocol
	6 : list<string> PolicyList
	7 : bool IsNetworkReachable
	8 : string RouteCreatedTime
	9 : string RouteUpdatedTime
}
struct IPv4RouteStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<IPv4RouteState> IPv4RouteStateList
}
struct PolicyConditionState {
	1 : string Name
	2 : string ConditionInfo
	3 : list<string> PolicyStmtList
}
struct PolicyConditionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyConditionState> PolicyConditionStateList
}
struct PolicyDefinitionState {
	1 : string Name
	2 : i32 HitCounter
	3 : list<string> IpPrefixList
}
struct PolicyDefinitionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyDefinitionState> PolicyDefinitionStateList
}
struct IPv4Route {
	1 : string DestinationNw
	2 : string NetworkMask
	3 : string NextHopIp
	4 : i32 Cost
	5 : string OutgoingIntfType
	6 : string OutgoingInterface
	7 : string Protocol
}
struct PolicyStmtState {
	1 : string Name
	2 : string MatchConditions
	3 : list<string> Conditions
	4 : list<string> Actions
	5 : list<string> PolicyList
}
struct PolicyStmtStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyStmtState> PolicyStmtStateList
}
struct PolicyActionState {
	1 : string Name
	2 : string ActionInfo
	3 : list<string> PolicyStmtList
}
struct PolicyActionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<PolicyActionState> PolicyActionStateList
}
struct RouteDistanceState {
	1 : string Protocol
	2 : i32 Distance
}
struct RouteDistanceStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<RouteDistanceState> RouteDistanceStateList
}
struct IPv4EventState {
	1 : i32 Index
	2 : string TimeStamp
	3 : string EventInfo
}
struct IPv4EventStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<IPv4EventState> IPv4EventStateList
}
struct PolicyDefinitionStmtPrecedence {
	1 : i32 Precedence
	2 : string Statement
}
struct PolicyCondition {
	1 : string Name
	2 : string ConditionType
	3 : string MatchProtocol
	4 : string IpPrefix
	5 : string MaskLengthRange
}
struct PolicyStmt {
	1 : string Name
	2 : string MatchConditions
	3 : list<string> Conditions
	4 : list<string> Actions
}
service RIBDServices extends ribdInt.RIBDINTServices {
	bool CreatePolicyAction(1: PolicyAction config);
	bool UpdatePolicyAction(1: PolicyAction origconfig, 2: PolicyAction newconfig, 3: list<bool> attrset);
	bool DeletePolicyAction(1: PolicyAction config);

	bool CreatePolicyDefinition(1: PolicyDefinition config);
	bool UpdatePolicyDefinition(1: PolicyDefinition origconfig, 2: PolicyDefinition newconfig, 3: list<bool> attrset);
	bool DeletePolicyDefinition(1: PolicyDefinition config);

	IPv4RouteStateGetInfo GetBulkIPv4RouteState(1: int fromIndex, 2: int count);
	IPv4RouteState GetIPv4RouteState(1: string DestinationNw, 2: string NextHopIp);
	PolicyConditionStateGetInfo GetBulkPolicyConditionState(1: int fromIndex, 2: int count);
	PolicyConditionState GetPolicyConditionState(1: string Name);
	PolicyDefinitionStateGetInfo GetBulkPolicyDefinitionState(1: int fromIndex, 2: int count);
	PolicyDefinitionState GetPolicyDefinitionState(1: string Name);
	bool CreateIPv4Route(1: IPv4Route config);
	bool UpdateIPv4Route(1: IPv4Route origconfig, 2: IPv4Route newconfig, 3: list<bool> attrset);
	bool DeleteIPv4Route(1: IPv4Route config);

	oneway void OnewayCreateIPv4Route(1: IPv4Route config);
	oneway void OnewayUpdateIPv4Route(1: IPv4Route origconfig, 2: IPv4Route newconfig, 3: list<bool> attrset);
	oneway void OnewayDeleteIPv4Route(1: IPv4Route config);

	PolicyStmtStateGetInfo GetBulkPolicyStmtState(1: int fromIndex, 2: int count);
	PolicyStmtState GetPolicyStmtState(1: string Name);
	PolicyActionStateGetInfo GetBulkPolicyActionState(1: int fromIndex, 2: int count);
	PolicyActionState GetPolicyActionState(1: string Name);
	RouteDistanceStateGetInfo GetBulkRouteDistanceState(1: int fromIndex, 2: int count);
	RouteDistanceState GetRouteDistanceState(1: string Protocol);
	IPv4EventStateGetInfo GetBulkIPv4EventState(1: int fromIndex, 2: int count);
	IPv4EventState GetIPv4EventState(1: i32 Index);
	bool CreatePolicyCondition(1: PolicyCondition config);
	bool UpdatePolicyCondition(1: PolicyCondition origconfig, 2: PolicyCondition newconfig, 3: list<bool> attrset);
	bool DeletePolicyCondition(1: PolicyCondition config);

	bool CreatePolicyStmt(1: PolicyStmt config);
	bool UpdatePolicyStmt(1: PolicyStmt origconfig, 2: PolicyStmt newconfig, 3: list<bool> attrset);
	bool DeletePolicyStmt(1: PolicyStmt config);

}