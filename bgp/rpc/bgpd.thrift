namespace go bgpd
typedef i32 int

struct BgpGlobal {
    1: i64 AS,
    2: string RouterId,
}

struct BgpPeer {
    1: i64 PeerAS,
    2: i64 LocalAS,
    3: string Description,
    4: string NeighborAddress,
}

service BgpServer
{
    bool CreateBgp(1: BgpGlobal bgpConf);
    bool UpdateBgp(1: BgpGlobal bgpConf);
    bool DeleteBgp(1: BgpGlobal bgpConf);

    bool CreatePeer(1: BgpPeer peer); 
    bool UpdatePeer(1: BgpPeer peer); 
    bool DeletePeer(1: BgpPeer peer); 
}
