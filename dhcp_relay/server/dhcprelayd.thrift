namespace go dhcprelayd
typedef i32 int
typedef i16 uint16
struct DhcpRelayGlobalConfig{
	1 : bool 	Enable
}
struct DhcpRelayGlobalConfigGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayGlobalConfig> DhcpRelayGlobalConfigList
}
struct DhcpRelayIntfConfig{
	1 : string 	IpSubnet
	2 : string 	Netmask
	3 : string 	IfIndex
	4 : i32 	AgentSubType
	5 : bool 	Enable
	6 : set<string> 	ServerIp
}
struct DhcpRelayIntfConfigGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<DhcpRelayIntfConfig> DhcpRelayIntfConfigList
}
service DHCPRELAYDServices {
	bool CreateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);
	bool UpdateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig origconfig, 2: DhcpRelayGlobalConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);

	bool CreateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);
	bool UpdateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig origconfig, 2: DhcpRelayIntfConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);

}