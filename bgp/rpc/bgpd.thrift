namespace go bgpd
typedef i32 int

struct BGPGlobal {
    1: i32 AS,
    2: string RouterId,
}

struct BGPNeighbor {
    1: i32 PeerAS,
    2: i32 LocalAS,
    3: string Description,
    4: string NeighborAddress,
}

service BGPServer
{
    bool CreateBGPGlobal(1: BGPGlobal bgpConf);
    bool UpdateBGPGlobal(1: BGPGlobal bgpConf);
    bool DeleteBGPGlobal(1: BGPGlobal bgpConf);

    bool CreateBGPNeighbor(1: BGPNeighbor neighbor); 
    bool UpdateBGPNeighbor(1: BGPNeighbor neighbor); 
    bool DeleteBGPNeighbor(1: BGPNeighbor neighbor); 
}
