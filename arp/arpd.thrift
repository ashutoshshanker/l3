namespace go arpd
typedef i32 int
typedef i16 uint16
struct ArpConfig{
	1 : string 	ArpConfigKey
	2 : i32 	Timeout
}
struct ArpEntry{
	1 : string 	IpAddr
	2 : string 	MacAddr
	3 : int         Vlan,
	4 : string 	Intf
	5 : string 	ExpiryTimeLeft
}
struct ArpEntryGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<ArpEntry> ArpEntryList
}

struct ArpEntryBulk {
        1: int StartIdx
        2: int EndIdx
        3: int Count
        4: bool More
        5: list<ArpEntry> ArpList
}
service ARPDServices {
	
	//ArpEntryGetInfo GetBulkArpEntry(1: int fromIndex, 2: int count);
    
    int ResolveArpIPV4(1:string destNetIp,2:int iftype, 3:int vlanid);
    int SetArpConfig(1:int Timeout);
    int UpdateUntaggedPortToVlanMap(1:int vlanid, 2:string untaggedMembers);
    ArpEntryBulk GetBulkArpEntry(1:int currMarker, 2:int count)
    int ArpProbeV4Intf(1:string ipAddr, 2:int intf, 3:int ifType);
}
