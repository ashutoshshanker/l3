namespace go vxland
typedef i32 int
typedef i16 uint16
struct VxlanVtepInstances {
	1 : i32 VtepId
	2 : i32 VxlanId
	3 : string VtepName
	4 : i32 SrcIfIndex
	5 : i16 UDP
	6 : i16 TTL
	7 : i16 TOS
	8 : i32 InnerVlanHandlingMode
	9 : bool Learning
	10 : bool Rsc
	11 : bool L2miss
	12 : bool L3miss
	13 : string DstIp
	14 : string SrcIp
	15 : string SrcMac
	16 : string DstMac
	17 : i16 VlanId
}
struct VxlanInstance {
	1 : i32 VxlanId
	2 : string McDestIp
	3 : i16 VlanId
	4 : i32 Mtu
}
service VXLANDServices {
	bool CreateVxlanVtepInstances(1: VxlanVtepInstances config);
	bool UpdateVxlanVtepInstances(1: VxlanVtepInstances origconfig, 2: VxlanVtepInstances newconfig, 3: list<bool> attrset);
	bool DeleteVxlanVtepInstances(1: VxlanVtepInstances config);

	bool CreateVxlanInstance(1: VxlanInstance config);
	bool UpdateVxlanInstance(1: VxlanInstance origconfig, 2: VxlanInstance newconfig, 3: list<bool> attrset);
	bool DeleteVxlanInstance(1: VxlanInstance config);

}