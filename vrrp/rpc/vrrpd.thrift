namespace go vrrpd
typedef i32 int
typedef i16 uint16
struct VrrpVridState {
	1 : i32 IfIndex
	2 : i32 VRID
	3 : i32 AdverRx
	4 : i32 AdverTx
	5 : string LastAdverRx
	6 : string LastAdverTx
	7 : string MasterIp
	8 : string CurrentState
	9 : string PreviousState
	10 : string TransitionReason
}
struct VrrpVridStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VrrpVridState> VrrpVridStateList
}
struct VrrpIntf {
	1 : i32 IfIndex
	2 : i32 VRID
	3 : i32 Priority
	4 : string VirtualIPv4Addr
	5 : i32 AdvertisementInterval
	6 : bool PreemptMode
	7 : bool AcceptMode
}
struct VrrpIntfState {
	1 : i32 IfIndex
	2 : i32 VRID
	3 : string IntfIpAddr
	4 : i32 Priority
	5 : string VirtualIPv4Addr
	6 : i32 AdvertisementInterval
	7 : bool PreemptMode
	8 : string VirtualRouterMACAddress
	9 : i32 SkewTime
	10 : i32 MasterDownTimer
	11 : string VrrpState
}
struct VrrpIntfStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VrrpIntfState> VrrpIntfStateList
}
service VRRPDServices {
	VrrpVridStateGetInfo GetBulkVrrpVridState(1: int fromIndex, 2: int count);
	VrrpVridState GetVrrpVridState(1: i32 IfIndex, 2: i32 VRID);
	bool CreateVrrpIntf(1: VrrpIntf config);
	bool UpdateVrrpIntf(1: VrrpIntf origconfig, 2: VrrpIntf newconfig, 3: list<bool> attrset);
	bool DeleteVrrpIntf(1: VrrpIntf config);

	VrrpIntfStateGetInfo GetBulkVrrpIntfState(1: int fromIndex, 2: int count);
	VrrpIntfState GetVrrpIntfState(1: i32 IfIndex, 2: i32 VRID);
}