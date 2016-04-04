include "arpdInt.thrift"
namespace go arpd
typedef i32 int
typedef i16 uint16
struct ArpConfig {
	1 : string ArpConfigKey
	2 : i32 Timeout
}
struct ArpEntry {
	1 : string IpAddr
	2 : string MacAddr
	3 : i32 Vlan
	4 : string Intf
	5 : string ExpiryTimeLeft
}
struct ArpEntryGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<ArpEntry> ArpEntryList
}
service ARPDServices extends arpdInt.ARPDINTServices {
	bool CreateArpConfig(1: ArpConfig config);
	bool UpdateArpConfig(1: ArpConfig origconfig, 2: ArpConfig newconfig, 3: list<bool> attrset);
	bool DeleteArpConfig(1: ArpConfig config);

	ArpEntryGetInfo GetBulkArpEntry(1: int fromIndex, 2: int count);
	ArpEntry GetArpEntry(1: string IpAddr);
}