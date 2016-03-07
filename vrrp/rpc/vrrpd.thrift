namespace go vrrpd
typedef i32 int
typedef i16 uint16
struct VrrpIntfConfig {
	1 : string VirtualRouterMACAddress
	2 : bool PreemptMode
	3 : i32 VRID
	4 : i32 Priority
	5 : i32 AdvertisementInterval
	6 : bool AcceptMode
	7 : string VirtualIPv4Addr
	8 : i32 IfIndex
}
struct VrrpIntfState {
	1 : string VirtualRouterMACAddress
	2 : bool PreemptMode
	3 : i32 AdvertisementInterval
	4 : i32 VRID
	5 : i32 Priority
	6 : i32 SkewTime
	7 : string VirtualIPv4Addr
	8 : i32 IfIndex
	9 : i32 MasterDownInterval
	10 : string IntfIpAddr
}
struct VrrpIntfStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VrrpIntfState> VrrpIntfStateList
}
service VRRPDServices {
	bool CreateVrrpIntfConfig(1: VrrpIntfConfig config);
	bool UpdateVrrpIntfConfig(1: VrrpIntfConfig origconfig, 2: VrrpIntfConfig newconfig, 3: list<bool> attrset);
	bool DeleteVrrpIntfConfig(1: VrrpIntfConfig config);

	VrrpIntfStateGetInfo GetBulkVrrpIntfState(1: int fromIndex, 2: int count);
}