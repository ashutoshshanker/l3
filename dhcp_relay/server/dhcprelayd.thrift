namespace go dhcprelayd
typedef i32 int
typedef i16 uint16
struct DhcpRelayGlobalConfig{
	1 : string 	DhcpRelay
	2 : bool 	Enable
}
struct DhcpRelayIntfConfig{
	1 : string 	IpSubnet
	2 : string 	Netmask
	3 : string 	IfIndex
	4 : i32 	AgentSubType
	5 : bool 	Enable
	6 : list<string> 	ServerIp
}
struct DhcpRelayHostDhcpState{
	1 : string 	MacAddr
	2 : string 	ServerIp
	3 : string 	OfferedIp
	4 : string 	GatewayIp
	5 : string 	AcceptedIp
	6 : string 	RequestedIp
	7 : string 	ClientDiscover
	8 : string 	ClientRequest
	9 : i32 	ClientRequests
	10 : i32 	ClientResponses
	11 : string 	ServerOffer
	12 : string 	ServerAck
	13 : i32 	ServerRequests
	14 : i32 	ServerResponses
}
struct DhcpRelayHostDhcpStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayHostDhcpState> DhcpRelayHostDhcpStateList
}
struct DhcpRelayIntfState{
	1 : i32 	IntfId
	2 : i32 	TotalDrops
	3 : i32 	TotalDhcpClientRx
	4 : i32 	TotalDhcpClientTx
	5 : i32 	TotalDhcpServerRx
	6 : i32 	TotalDhcpServerTx
}
struct DhcpRelayIntfStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfState> DhcpRelayIntfStateList
}
struct DhcpRelayIntfServerState{
	1 : i32 	IntfId
	2 : string 	ServerIp
	3 : i32 	Request
	4 : i32 	Responses
}
struct DhcpRelayIntfServerStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfServerState> DhcpRelayIntfServerStateList
}
service DHCPRELAYDServices {
	bool CreateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);
	bool UpdateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig origconfig, 2: DhcpRelayGlobalConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);

	bool CreateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);
	bool UpdateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig origconfig, 2: DhcpRelayIntfConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);

	DhcpRelayHostDhcpStateGetInfo GetBulkDhcpRelayHostDhcpState(1: int fromIndex, 2: int count);
	DhcpRelayIntfStateGetInfo GetBulkDhcpRelayIntfState(1: int fromIndex, 2: int count);
	DhcpRelayIntfServerStateGetInfo GetBulkDhcpRelayIntfServerState(1: int fromIndex, 2: int count);
}