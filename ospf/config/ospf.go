package config

import (
    //"net"
)

type areaId string
type routerId string
type metric int // 0x0 to 0xffff
type bigMetric int // 0x0 to 0xffffff
type positiveInteger int // 0x0 to 0x7fffffff
type helloRange int // 0x1 to 0xffff
type upToMaxAge int // 0x0 to 3600
type designatedRouterPriority int // 0x0 to 0xff
type tosType int // 0x0 to 30
type ipAddress string
type interfaceIndexOrZero int

type status int
const (
    enabled status = 1
    disabled status = 2
)

type authType int
const (
    noAuth authType = 0
    simplePassword authType = 1
    md5 authType = 2
    reserved authType = 3
)

type restartSupport int
const (
    none restartSupport = 1
    plannedOnly restartSupport = 2
    plannedAndUnplanned restartSupport = 3
)

type advertiseAction int
const (
    doNotAdvertise advertiseAction = 1
    advertise advertiseAction = 2
)

type importAsExtern int
const (
    importExternal importAsExtern = 1
    importNoExternal importAsExtern = 2
    importNssa importAsExtern = 3
)

type areaSummary int
const (
    noAreaSummary areaSummary = 1
    sendAreaSummary areaSummary = 2
)

type nssaTranslatorRole int
const (
    always nssaTranslatorRole = 1
    cadidate nssaTranslatorRole = 2
)


type metricType int
const (
    ospfMetric metricType = 1
    comparableCost metricType = 2
    nonComparable metricType = 3
)

type areaRangeEffect int
const (
    advertiseMatching areaRangeEffect = 1
    doNotAdvertiseMatching areaRangeEffect = 2
)

type areaAggregateLsdbType int
const (
    summaryLink areaAggregateLsdbType = 3
    nssaExternalLink areaAggregateLsdbType = 7
)

type ifType int
const (
    broadcast = 1
    nbma = 2
    pointToPoint = 3
    pointToMultipoint = 4
)

type multicastForwarding int
const (
    blocked = 1
    multicast = 2
    unicast = 3
)

type GlobalConf struct {
    RouterId                    routerId
    AdminStat                   status
    ASBdrRtrStatus              bool
    TOSSupport                  bool
    ExtLsdbLimit                int
    MulticastExtension          int
    ExitOverflowInterval        positiveInteger
    DemandExtensions            bool
    RFC1583Compatibility        bool
    ReferenceBandwidth          int
    RestartSupport              restartSupport
    RestartInterval             int
    RestartStrictLsaChecking    bool
    StubRouterAdvertisement     advertiseAction
}

type AreaConf struct {
    AreaId                                  areaId
    AuthType                                authType
    ImportAsExtern                          importAsExtern
    AreaSummary                             areaSummary
    AreaNssaTranslatorRole                  nssaTranslatorRole
    AreaNssaTranslatorStabilityInterval     positiveInteger
}

type StubAreaConf struct {
    StubAreaId              areaId
    StubTOS                 tosType
    StubMetric              bigMetric
    StubMetricType          metricType
}

type AreaRangeConf struct {
    RangeAreaId             areaId
    AreaRangeNet            ipAddress
    ArearangeMask           ipAddress
    AreaRangeEffect         areaRangeEffect
}

type HostConf struct {
    HostIpAddress           ipAddress
    HostTOS                 tosType
    HostMetric              metric
    HostCfgAreaID           areaId
}

type IfConf struct {
    IfIpAddress             ipAddress
    AddressLessIf           interfaceIndexOrZero
    IfAreaId                areaId
    IfType                  ifType
    IfAdminStat             status
    IfRtrPriority           designatedRouterPriority
    IfTransitDelay          upToMaxAge
    IfRetransInterval       upToMaxAge
    IfHelloInterval         helloRange
    IfRtrDeadInterval       positiveInteger
    IfPollInterval          positiveInteger
    IfAuthKey               string
    IfMulticastForwarding   multicastForwarding
    IfDemand                bool
    IfAuthType              authType
}

type IfMetricConf struct {
    IfMetricIpAddress       ipAddress
    IfMetricAddressLessIf   interfaceIndexOrZero
    IfMetricTOS             tosType
    IfMetricValue           metric
}

type VirtIfConf struct {
    VirtIfAreaId            areaId
    VirtIfNeighbor          routerId
    VirtIfTransitDelay      upToMaxAge
    VirtIfRetransInterval   upToMaxAge
    VirtIfHelloInterval     helloRange
    VirtIfRtrDeadInterval   positiveInteger
    VirtIfAuthKey           string
    VirtIfAuthType          authType
}

type NbrConf struct {
    NbrIpAddress                ipAddress
    NbrAddressLessIndex         interfaceIndexOrZero
    NbrPriority                 designatedRouterPriority
}

type AreaAggregateConf struct {
    AreaAggregateAreaId         areaId
    AreaAggregateLsdbType       areaAggregateLsdbType
    AreaAggregateNet            ipAddress
    AreaAggregateMask           ipAddress
    AreaAggregateEffect         areaRangeEffect
    AreaAggregateExtRouteTag    int
}

/*
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
*/
