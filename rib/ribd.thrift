namespace go ribd
typedef i32 int
struct NextHopInfo {
    1: string NextHopIp,
    2: int NextHopIfIndex,
	3: int Metric,
}
struct Routes {
	1: string Ipaddr,
	2: string Mask,
	3: int IfIndex,
}
//typedef RouteList  list<Routes>
service RouteService 
{
    int createV4Route (1:string destNetIp, 2:string networkMask, 3:int metric, 4:string nextHopIp, 5:int nextHopIfIndex, 6:int routeType);
    void updateV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType, 4:string nextHopIp, 5:int nextHopIfIndex, 6:int metric);
    int deleteV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType);
    NextHopInfo getRouteReachabilityInfo(1: string desIPv4MasktNet)
	list<Routes> getConnectedRoutesInfo()
    void printV4Routes();
}
