namespace go ribd
typedef i32 int
service RouteService 
{
    int createV4Route (1:int destNet, 2:int prefixLen, 3:int nextHop, 4:int nextHopIfIndex, 5:int metric);
    int deleteV4Route (1:int destNet, 2:int prefixLen, 3:int nextHop, 4:int nextHopIfIndex);
}
