namespace go arpd
typedef i32 int

service ARPService
{
    int ResolveArpIPV4(1:string destNetIp,2:int iftype, 3:int vlanid);
    int ArpTimeout(1:int Timeout);
    int UpdateUntaggedPortToVlanMap(1:int vlanid, 2:string untaggedMembers);
}
