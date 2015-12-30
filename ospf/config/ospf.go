package config

import (
    "net"
)

type GlobalConfig struct {
    RouterId               uint32
    RFC1583Compatibility   bool
}

type AddressRange struct {
    IP      net.IP
    Mask    net.IP
    Status  bool
}

type AreaConfig struct {
    AreaId                      uint32
    AddressRanges               []AddressRange
    ExternalRoutingCapability   bool
    StubDefaultCost             uint32
}

type Ospf struct {
    globalConfig    GlobalConfig
    areaConfig      AreaConfig
}
