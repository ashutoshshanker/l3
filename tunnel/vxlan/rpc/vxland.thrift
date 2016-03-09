namespace go vxland
typedef i32 int
typedef i16 uint16
struct VxlanStateVxlanInstanceVxlanEvpnVpnTargets {
	1 : i32 AccessTypeVlan
	2 : string InterfaceName
	3 : bool AccessTypeL3interface
	4 : i16 VlanId
	5 : string RouteDistinguisher
	6 : string Mac
	7 : i32 VxlanId
	8 : i32 RtType
	9 : bool AccessTypeMac
	10 : bool AccessTypeL2interface
	11 : string RtValue
}
struct VxlanStateVxlanInstanceVxlanEvpnVpnTargetsGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VxlanStateVxlanInstanceVxlanEvpnVpnTargets> VxlanStateVxlanInstanceVxlanEvpnVpnTargetsList
}
struct VxlanStateStaticVxlanTunnelAddressFamilyBindVxlanId {
	1 : i32 VxlanId
	2 : i32 TunnelSourceIp
	3 : i32 Af
	4 : i32 VxlanTunnelId
	5 : i32 TunnelDestinationIp
	6 : string VxlanTunnelName
}
struct VxlanStateStaticVxlanTunnelAddressFamilyBindVxlanIdGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VxlanStateStaticVxlanTunnelAddressFamilyBindVxlanId> VxlanStateStaticVxlanTunnelAddressFamilyBindVxlanIdList
}
struct VxlanVxlanInstanceAccessTypeL3interfaceL3interface {
	1 : i32 VxlanId
	2 : string InterfaceName
}
struct VxlanStateVxlanInstanceAccessVlan {
	1 : i32 AccessTypeVlan
	2 : string InterfaceName
	3 : i32 VxlanId
	4 : i16 VlanId
	5 : bool AccessTypeL3interface
	6 : string Mac
	7 : bool AccessTypeMac
	8 : bool AccessTypeL2interface
}
struct VxlanStateVxlanInstanceAccessVlanGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VxlanStateVxlanInstanceAccessVlan> VxlanStateVxlanInstanceAccessVlanList
}
struct VxlanVxlanInstanceVxlanEvpnVpnTargets {
	1 : string RouteDistinguisher
	2 : string RtValue
	3 : i32 VxlanId
	4 : i32 RtType
}
struct VxlanVxlanInstanceAccessTypeMac {
	1 : bool L2interface
	2 : string Mac
	3 : i32 VxlanId
	4 : string InterfaceName
	5 : i16 VlanId
}
struct VxlanStateVxlanInstanceMapL3interface {
	1 : i32 AccessTypeVlan
	2 : string InterfaceName
	3 : i32 VxlanId
	4 : i16 VlanId
	5 : bool AccessTypeL3interface
	6 : string Mac
	7 : bool AccessTypeMac
	8 : bool AccessTypeL2interface
}
struct VxlanStateVxlanInstanceMapL3interfaceGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VxlanStateVxlanInstanceMapL3interface> VxlanStateVxlanInstanceMapL3interfaceList
}
struct VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId {
	1 : i32 TunnelSourceIp
	2 : string Name
	3 : i32 Af
	4 : i32 VxlanTunnelId
	5 : i32 VxlanId
	6 : i32 TunnelDestinationIp
	7 : string VxlanTunnelName
}
struct VxlanInterfacesInterfaceVtepInstancesBindVxlanId {
	1 : i32 InnerVlanHandlingMode
	2 : string Name
	3 : i32 VxlanId
	4 : string VtepName
	5 : list<string> MulticastIp
	6 : i32 VtepId
	7 : string SourceInterface
}
struct VxlanStateVtepInstanceBindVxlanId {
	1 : i32 InnerVlanHandlingMode
	2 : i32 VxlanId
	3 : string VtepName
	4 : list<string> MulticastIp
	5 : i32 VtepId
	6 : string SourceInterface
}
struct VxlanStateVtepInstanceBindVxlanIdGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<VxlanStateVtepInstanceBindVxlanId> VxlanStateVtepInstanceBindVxlanIdList
}
struct VxlanVxlanInstanceAccessTypeVlanVlanList {
	1 : i32 VxlanId
	2 : i16 VlanId
}
service VXLANDServices {
	VxlanStateVxlanInstanceVxlanEvpnVpnTargetsGetInfo GetBulkVxlanStateVxlanInstanceVxlanEvpnVpnTargets(1: int fromIndex, 2: int count);
	VxlanStateStaticVxlanTunnelAddressFamilyBindVxlanIdGetInfo GetBulkVxlanStateStaticVxlanTunnelAddressFamilyBindVxlanId(1: int fromIndex, 2: int count);
	bool CreateVxlanVxlanInstanceAccessTypeL3interfaceL3interface(1: VxlanVxlanInstanceAccessTypeL3interfaceL3interface config);
	bool UpdateVxlanVxlanInstanceAccessTypeL3interfaceL3interface(1: VxlanVxlanInstanceAccessTypeL3interfaceL3interface origconfig, 2: VxlanVxlanInstanceAccessTypeL3interfaceL3interface newconfig, 3: list<bool> attrset);
	bool DeleteVxlanVxlanInstanceAccessTypeL3interfaceL3interface(1: VxlanVxlanInstanceAccessTypeL3interfaceL3interface config);

	VxlanStateVxlanInstanceAccessVlanGetInfo GetBulkVxlanStateVxlanInstanceAccessVlan(1: int fromIndex, 2: int count);
	bool CreateVxlanVxlanInstanceVxlanEvpnVpnTargets(1: VxlanVxlanInstanceVxlanEvpnVpnTargets config);
	bool UpdateVxlanVxlanInstanceVxlanEvpnVpnTargets(1: VxlanVxlanInstanceVxlanEvpnVpnTargets origconfig, 2: VxlanVxlanInstanceVxlanEvpnVpnTargets newconfig, 3: list<bool> attrset);
	bool DeleteVxlanVxlanInstanceVxlanEvpnVpnTargets(1: VxlanVxlanInstanceVxlanEvpnVpnTargets config);

	bool CreateVxlanVxlanInstanceAccessTypeMac(1: VxlanVxlanInstanceAccessTypeMac config);
	bool UpdateVxlanVxlanInstanceAccessTypeMac(1: VxlanVxlanInstanceAccessTypeMac origconfig, 2: VxlanVxlanInstanceAccessTypeMac newconfig, 3: list<bool> attrset);
	bool DeleteVxlanVxlanInstanceAccessTypeMac(1: VxlanVxlanInstanceAccessTypeMac config);

	VxlanStateVxlanInstanceMapL3interfaceGetInfo GetBulkVxlanStateVxlanInstanceMapL3interface(1: int fromIndex, 2: int count);
	bool CreateVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId(1: VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId config);
	bool UpdateVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId(1: VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId origconfig, 2: VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId newconfig, 3: list<bool> attrset);
	bool DeleteVxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId(1: VxlanInterfacesInterfaceStaticVxlanTunnelAddressFamilyBindVxlanId config);

	bool CreateVxlanInterfacesInterfaceVtepInstancesBindVxlanId(1: VxlanInterfacesInterfaceVtepInstancesBindVxlanId config);
	bool UpdateVxlanInterfacesInterfaceVtepInstancesBindVxlanId(1: VxlanInterfacesInterfaceVtepInstancesBindVxlanId origconfig, 2: VxlanInterfacesInterfaceVtepInstancesBindVxlanId newconfig, 3: list<bool> attrset);
	bool DeleteVxlanInterfacesInterfaceVtepInstancesBindVxlanId(1: VxlanInterfacesInterfaceVtepInstancesBindVxlanId config);

	VxlanStateVtepInstanceBindVxlanIdGetInfo GetBulkVxlanStateVtepInstanceBindVxlanId(1: int fromIndex, 2: int count);
	bool CreateVxlanVxlanInstanceAccessTypeVlanVlanList(1: VxlanVxlanInstanceAccessTypeVlanVlanList config);
	bool UpdateVxlanVxlanInstanceAccessTypeVlanVlanList(1: VxlanVxlanInstanceAccessTypeVlanVlanList origconfig, 2: VxlanVxlanInstanceAccessTypeVlanVlanList newconfig, 3: list<bool> attrset);
	bool DeleteVxlanVxlanInstanceAccessTypeVlanVlanList(1: VxlanVxlanInstanceAccessTypeVlanVlanList config);

}