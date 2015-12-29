namespace go ospfd
typedef i32 int

struct OSPFGlobalConf {
    1: i32      RouterId,
    2: bool     RFC1583Compatibility
}

struct OSPFAddressRange {
    1: string   IP,
    2: string   Mask,
    3: bool     Status,
}

struct OSPFAreaConf {
    1: i32                      AreaId,
    2: list<OSPFAddressRange>   AddressRange,
    3: bool                     ExternalRoutingCapability,
    4: i32                      StubDefaultCost,
}

service OSPFServer {
    bool CreateOSPFGlobalConf(1: OSPFGlobalConf ospfGlobalConf)
    bool CreateOSPFAreaConf(1: OSPFAreaConf ospfAreaConf)
//    bool UpdateOSPFGlobalConf(1: OSPFGlobalConf ospfConf)
}
