namespace go ribdInt
typedef i32 int
struct NextHopInfo {
	1: int NextHopIfType,
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
	5 : string OutgoingIntfType
	6 : string OutgoingInterface
	7 : string Protocol
	8 : string CreateTime
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
	
}
