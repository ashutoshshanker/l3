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
	3 : string 	IfIndex // this is if_name....:)
	4 : i32 	AgentSubType
	5 : bool 	Enable
	6 : string 	ServerIp
}
service DHCPRELAYDServices {
	bool CreateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);
	bool UpdateDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig origconfig, 2: DhcpRelayGlobalConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayGlobalConfig(1: DhcpRelayGlobalConfig config);

	bool CreateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);
	bool UpdateDhcpRelayIntfConfig(1: DhcpRelayIntfConfig origconfig, 2: DhcpRelayIntfConfig newconfig, 3: list<bool> attrset);
	bool DeleteDhcpRelayIntfConfig(1: DhcpRelayIntfConfig config);

}
