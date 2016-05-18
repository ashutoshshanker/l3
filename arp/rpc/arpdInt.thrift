namespace go arpdInt
typedef i32 int
service ARPDINTServices {
        oneway void ResolveArpIPV4(1:string destNetIp, 2:int vlanid);
	oneway void DeleteResolveArpIPv4(1:string NbrIP);
}
