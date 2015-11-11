namespace go ribd
typedef i32 int
service RouteService 
{
    int createV4Route (1:int destNet, 2:int prefixLen, 3:int metric, 4:int nextHop, 5:int nextHopIfIndex, 6:int routeType);
    void updateV4Route (1:int destNet, 2:int prefixLen, 3:int routeType, 4:int nextHop, 5:int nextHopIfIndex, 6:int metric);
    int deleteV4Route (1:int destNet, 2:int prefixLen, 3:int routeType);
    void printV4Routes();
}
