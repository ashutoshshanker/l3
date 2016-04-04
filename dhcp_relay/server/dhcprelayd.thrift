namespace go dhcprelayd
typedef i32 int
typedef i16 uint16
struct DhcpRelayHostDhcpState {
	1 : string MacAddr
	2 : string ServerIp
	3 : string OfferedIp
	4 : string GatewayIp
	5 : string AcceptedIp
	6 : string RequestedIp
	7 : string ClientDiscover
	8 : string ClientRequest
	9 : i32 ClientRequests
	10 : i32 ClientResponses
	11 : string ServerOffer
	12 : string ServerAck
	13 : i32 ServerRequests
	14 : i32 ServerResponses
}
struct DhcpRelayHostDhcpStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayHostDhcpState> DhcpRelayHostDhcpStateList
}
struct DhcpRelayIntfServerState {
	1 : i32 IntfId
	2 : string ServerIp
	3 : i32 Request
	4 : i32 Responses
}
struct DhcpRelayIntfServerStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfServerState> DhcpRelayIntfServerStateList
}
struct DhcpRelayIntfState {
	1 : i32 IntfId
	2 : i32 TotalDrops
	3 : i32 TotalDhcpClientRx
	4 : i32 TotalDhcpClientTx
	5 : i32 TotalDhcpServerRx
	6 : i32 TotalDhcpServerTx
}
struct DhcpRelayIntfStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfState> DhcpRelayIntfStateList
}
struct DhcpRelayGlobal {
	1 : string DhcpRelay
	2 : bool Enable
}
struct DhcpRelayIntf {
	1 : i32 IfIndex
	2 : bool Enable
	3 : list<string> ServerIp
}
service DHCPRELAYDServices {
	DhcpRelayHostDhcpStateGetInfo GetBulkDhcpRelayHostDhcpState(1: int fromIndex, 2: int count);
	DhcpRelayHostDhcpState GetDhcpRelayHostDhcpState(1: string MacAddr);
	DhcpRelayIntfServerStateGetInfo GetBulkDhcpRelayIntfServerState(1: int fromIndex, 2: int count);
	DhcpRelayIntfServerState GetDhcpRelayIntfServerState(1: i32 IntfId);
	DhcpRelayIntfStateGetInfo GetBulkDhcpRelayIntfState(1: int fromIndex, 2: int count);
	DhcpRelayIntfState GetDhcpRelayIntfState(1: i32 IntfId);
	bool CreateDhcpRelayGlobal(1: DhcpRelayGlobal config);
	bool UpdateDhcpRelayGlobal(1: DhcpRelayGlobal origconfig, 2: DhcpRelayGlobal newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayGlobal(1: DhcpRelayGlobal config);

	bool CreateDhcpRelayIntf(1: DhcpRelayIntf config);
	bool UpdateDhcpRelayIntf(1: DhcpRelayIntf origconfig, 2: DhcpRelayIntf newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayIntf(1: DhcpRelayIntf config);

}