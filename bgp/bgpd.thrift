namespace go bgpd
typedef i32 int

struct BgpGlobal {
    1: i32 AS,
}

struct BgpPeer {
    1: i32 AS,
    2: string ip,
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
