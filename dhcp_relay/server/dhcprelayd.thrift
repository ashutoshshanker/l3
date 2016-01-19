namespace go dhcprelayd
typedef i32 int
typedef i16 uint16
struct DhcpRelayGlobal{
	1 : bool 	Enable
}
struct DhcpRelayGlobalGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayGlobal> DhcpRelayGlobalList
}
struct DhcpRelayConf{
	1 : string 	IpSubnet
	2 : string 	Netmask
	3 : string 	IfIndex
	4 : i32 	AgentSubType
	5 : bool 	Enable
	6 : set<string> 	ServerIp
}
struct DhcpRelayConfGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayConf> DhcpRelayConfList
}
service DHCPRELAYDServices {
	bool CreateDhcpRelayGlobal(1: DhcpRelayGlobal config);
	bool UpdateDhcpRelayGlobal(1: DhcpRelayGlobal origconfig, 2: DhcpRelayGlobal newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayGlobal(1: DhcpRelayGlobal config);

	bool CreateDhcpRelayConf(1: DhcpRelayConf config);
	bool UpdateDhcpRelayConf(1: DhcpRelayConf origconfig, 2: DhcpRelayConf newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayConf(1: DhcpRelayConf config);

}