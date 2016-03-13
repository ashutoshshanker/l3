namespace go arpdInt
typedef i32 int
service ARPDINTServices {
        int ResolveArpIPV4(1:string destNetIp, 2:int iftype, 3:int vlanid);
}
