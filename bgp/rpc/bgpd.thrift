namespace go bgpd
typedef i32 int

struct BgpGlobal {
    1: i64 AS,
    2: string RouterId,
}

struct BgpNeighbor {
    1: i64 PeerAS,
    2: i64 LocalAS,
    3: string Description,
    4: string NeighborAddress,
}

service BgpServer
{
    bool CreateBgpGlobal(1: BgpGlobal bgpConf);
    bool UpdateBgpGlobal(1: BgpGlobal bgpConf);
    bool DeleteBgpGlobal(1: BgpGlobal bgpConf);

    bool CreateBgpNeighbor(1: BgpNeighbor neighbor); 
    bool UpdateBgpNeighbor(1: BgpNeighbor neighbor); 
    bool DeleteBgpNeighbor(1: BgpNeighbor neighbor); 
}
