namespace go bgpd
typedef i32 int

struct BGPGlobalConfig {
	1: i32 ASNum,
	2: string RouterId,
}

struct BGPGlobalState {
	1: i32 AS,
	2: string RouterId,
	3: i32 TotalPaths,
	4: i32 TotalPrefixes,
}

enum PeerType {
	PeerTypeInternal = 0,
	PeerTypeExternal
}

struct BgpCounters {
	1: i64 Update,
	2: i64 Notification,
}

struct BGPMessages {
	1: BgpCounters Sent,
	2: BgpCounters Received,
}

struct BGPQueues {
	1: i32 Input
	2: i32 Output
}

struct BGPNeighborConfig {
	1: i32 PeerAS,
	2: i32 LocalAS,
	3: string AuthPassword,
	4: string Description,
	5: string NeighborAddress,
	6: i32 RouteReflectorClusterId,
	7: bool RouteReflectorClient,
	8: bool MultiHopEnable,
	9: byte MultiHopTTL,
	10: i32 ConnectRetryTime,
	11: i32 HoldTime,
	12: i32 KeepaliveTime,
	13: string PeerGroup,
}

struct BGPNeighborState {
	1: i32 PeerAS,
	2: i32 LocalAS,
	3: string AuthPassword,
	4: PeerType PeerType,
	5: string Description,
	6: string NeighborAddress,
	7: i32 SessionState,
	8: BGPMessages Messages,
	9: BGPQueues Queues,
	10: i32 RouteReflectorClusterId,
	11: bool RouteReflectorClient,
	12: bool MultiHopEnable,
	13: byte MultiHopTTL,
	14: i32 ConnectRetryTime,
	15: i32 HoldTime,
	16: i32 KeepaliveTime,
    17: string GroupName,
}

struct BGPNeighborStateBulk {
	1: i64 CurrIndex,
	2: i64 NextIndex,
	3: i64 Count,
	4: bool More,
	5: list<BGPNeighborState> StateList,
}

struct BGPPeerGroup {
	1: i32 PeerAS,
	2: i32 LocalAS,
	3: string AuthPassword,
	4: string Description,
	5: string Name,
	6: i32 RouteReflectorClusterId,
	7: bool RouteReflectorClient,
	8: bool MultiHopEnable,
	9: byte MultiHopTTL,
	10: i32 ConnectRetryTime,
	11: i32 HoldTime,
	12: i32 KeepaliveTime,
}

struct BGPRoute {
	1: string Network
	2: string Mask
	3: string NextHop
	4: i32 Metric
	5: i32 LocalPref
	6: list<i32> Path
	7: string Updated
}

struct BGPRouteBulk {
	1: i64 CurrIndex,
	2: i64 NextIndex,
	3: i64 Count,
	4: bool More,
	5: list<BGPRoute> RouteList,
}

service BGPServer
{
	bool CreateBGPGlobal(1: BGPGlobalConfig bgpConf);
	BGPGlobalState GetBGPGlobal();
	bool UpdateBGPGlobal(1: BGPGlobalConfig origGlobal, 2: BGPGlobalConfig updatedGlobal, 3: list<bool> attrSet);
	//bool DeleteBGPGlobal(1: BGPGlobal bgpConf);

	bool CreateBGPNeighbor(1: BGPNeighborConfig neighbor);
	BGPNeighborState GetBGPNeighbor(1: string ip);
	BGPNeighborStateBulk BulkGetBGPNeighbors(1: i64 index, 2: i64 count);
	bool UpdateBGPNeighbor(1: BGPNeighborConfig origNeighbor, 2: BGPNeighborConfig updatedNeighbor, 3: list<bool> attrSet);
	bool DeleteBGPNeighbor(1: string neighborAddress);

	bool CreateBGPPeerGroup(1: BGPPeerGroup group);
	bool UpdateBGPPeerGroup(1: BGPPeerGroup origGroup, 2: BGPPeerGroup updatedGroup, 3: list<bool> attrSet);
	bool DeleteBGPPeerGroup(1: string groupName);

	BGPRoute GetBGPRoute(1: string ip);
	BGPRouteBulk BulkGetBGPRoutes(1: i64 index, 2: i64 count);
}
