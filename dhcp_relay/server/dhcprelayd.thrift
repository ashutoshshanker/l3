namespace go dhcprelayd
typedef i32 int
typedef i16 uint16
struct DhcpRelayHostDhcpState {
	1 : string MacAddr
	2 : string ServerAck
	3 : string RequestedIp
	4 : i32 ServerRequests
	5 : string AcceptedIp
	6 : string GatewayIp
	7 : i32 ClientRequests
	8 : string ServerIp
	9 : i32 ClientResponses
	10 : string ClientDiscover
	11 : string OfferedIp
	12 : i32 ServerResponses
	13 : string ServerOffer
	14 : string ClientRequest
}
struct DhcpRelayHostDhcpStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayHostDhcpState> DhcpRelayHostDhcpStateList
}
struct DhcpRelayIntfServerState {
	1 : i32 Request
	2 : i32 Responses
	3 : i32 IntfId
	4 : string ServerIp
}
struct DhcpRelayIntfServerStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfServerState> DhcpRelayIntfServerStateList
}
struct DhcpRelayIntfConfig {
	1 : i32 IfIndex
	2 : bool Enable
	3 : list<string> ServerIp
}
struct DhcpRelayIntfState {
	1 : i32 TotalDhcpServerRx
	2 : i32 TotalDrops
	3 : i32 TotalDhcpServerTx
	4 : i32 TotalDhcpClientRx
	5 : i32 IntfId
	6 : i32 TotalDhcpClientTx
}
struct DhcpRelayIntfStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfState> DhcpRelayIntfStateList
}
struct DhcpRelayGlobalConfig {
	1 : bool Enable
	2 : string DhcpRelay
}
service DHCPRELAYDServices {
	DhcpRelayHostDhcpStateGetInfo GetBulkDhcpRelayHostDhcpState(1: int fromIndex, 2: int count);
	DhcpRelayIntfServerStateGetInfo GetBulkDhcpRelayIntfServerState(1: int fromIndex, 2: int count);
	bool CreateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);
	bool UpdateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig origconfig, 2: DhcpRelayIntfConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);

	DhcpRelayIntfStateGetInfo GetBulkDhcpRelayIntfState(1: int fromIndex, 2: int count);
	bool CreateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);
	bool UpdateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig origconfig, 2: DhcpRelayGlobalConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);

}