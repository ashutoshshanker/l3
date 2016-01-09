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

type restartStatus int
const (
    notRestarting restartStatus = 1
    plannedRestart restartStatus = 2
    unplannedRestart restartStatus = 3
)

type restartExitReason int
const (
    noAttempt restartExitReason = 1
    inProgress restartExitReason = 2
    completed restartExitReaason = 3
    timeedOut restartExitReason = 4
    topologyChanged restartExitReason = 5
)

type nssaTranslatorState int
const (
    enabled nssaTranslatorState = 1
    elected nssaTranslatorState = 2
    disabled nssaTranslatorState = 3
)

type lsaType int
const (
    routerLink lsaType = 1
    networkLink lsaType = 2
    summaryLink lsaType = 3
    asSummaryLink lsaType = 4
    asExternalLink lsaType = 5
    multicastLink lsaType = 6
    nssaExternalLink lsaType = 7
    localOpaqueLink lsaType = 9
    areaOpaqueLink lsaType = 10
    asOpaqueLink lsaType = 11
)

type ifState int
const (
    down ifState = 1
    loopback ifState = 2
    waiting ifState = 3
    piontToPoint ifState = 4
    designatedRouter ifState = 5
    backupDesignatedRouter ifState = 6
    otherDesignatedRouter ifState = 7
)

type nbrState int
const (
    down nbrState = 1
    attempt nbrState = 2
    init nbrState = 3
    twoWay nbrState = 4
    exchangeStart nbrState = 5
    exchange nbrState = 6
    loading nbrState = 7
    full nbrState = 8
)

type nbmaNbrPermanence int
const (
    dynamicNbr nbmaNbrPermanence = 1
    permanentNbr nbmaNbrPermanence = 2
)

type nbrRestartHelperStatus int
const (
    notHelping nbrRestartHelperStatus = 1
    helping nbrRestartHelperStatus = 2
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

type GlobalState struct {
    RouterId                    routerId
    AdminStat                   status
    VersionNumber               int
    AreaBdrRtrStatus            bool
    ASBdrRtrStatus              bool
    ExternLsaCount              int
    ExternLsaChecksum           int
    TOSSupport                  bool
    OriginateNewLsas            int
    RxNewLsas                   int
    ExtLsdbLimit                int
    MulticastExtension          int
    ExitOverflowInterval        positiveInteger
    DemandExtensions            bool
    RFC1583Compatibility        bool
    OpaqueLsaSupport            bool
    ReferenceBandwidth          int
    RestartSupport              restartSupport
    RestartInterval             int
    RestartStrictLsaChecking    bool
    RestartStatus               restartStatus
    RestartAge                  int
    RestartExitReason           restartExitReason
    AsLsaCount                  int
    AsLsaCksumSum               int
    StubRouterSupport           bool
    StubRouterAdvertisement     advertiseAction
    DiscontinuityTime           string
}

// Indexed By AreaId
type AreaConf struct {
    AreaId                                  areaId
    AuthType                                authType
    ImportAsExtern                          importAsExtern
    AreaSummary                             areaSummary
    AreaNssaTranslatorRole                  nssaTranslatorRole
    AreaNssaTranslatorStabilityInterval     positiveInteger
}

type AreaState struct {
    AreaId                                  areaId
    AuthType                                authType
    ImportAsExtern                          importAsExtern
    SpfRuns                                 int
    AreaBdrRtrCount                         int
    AsBdrRtrCount                           int
    AreaLsaCount                            int
    AreaLsaCksumSum                         int
    AreaSummary                             areaSummary
    AreaNssaTranslatorRole                  nssaTranslatorRole
    AreaNssaTranslatorState                 nssaTranslatorState
    AreaNssaTranslatorStabilityInterval     positiveInteger
    AreaNssaTranslatorEvents                int
}

// Indexed by StubAreaId and StubTOS
type StubAreaConf struct {
    StubAreaId              areaId
    StubTOS                 tosType
    StubMetric              bigMetric
    StubMetricType          metricType
}

type StubAreaState struct {
    StubAreaId              areaId
    StubTOS                 tosType
    StubMetric              bigMetric
    StubMetricType          metricType
}

// Indexed by LsdbAreaId, LsdbType, LsdbLsid, LsdbRouterId
type LsdbState struct {
    LsdbAreaId              areaId
    LsdbType                lsaType
    LsdbLsid                ipAddress
    LsdbRouterId            routerId
    LsdbSequence            int
    LsdbAge                 int
    LsdbCheckSum            int
    LsdbAdvertisement       string
}

// Indexed By RangeAreaId, RangeNet
type AreaRangeConf struct {
    RangeAreaId             areaId
    AreaRangeNet            ipAddress
    ArearangeMask           ipAddress
    AreaRangeEffect         areaRangeEffect
}

type AreaRangeState struct {
    RangeAreaId             areaId
    AreaRangeNet            ipAddress
    ArearangeMask           ipAddress
    AreaRangeEffect         areaRangeEffect
}

// Indexed By HostIpAddress, HostTOS
type HostConf struct {
    HostIpAddress           ipAddress
    HostTOS                 tosType
    HostMetric              metric
    HostCfgAreaID           areaId
}

type HostState struct {
    HostIpAddress           ipAddress
    HostTOS                 tosType
    HostMetric              metric
    HostCfgAreaID           areaId
}

// Indexed By IfIpAddress, AddressLessIf
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

type IfState struct {
    IfIpAddress                 ipAddress
    AddressLessIf               interfaceIndexOrZero
    IfAreaId                    areaId
    IfType                      ifType
    IfAdminStat                 status
    IfRtrPriority               designatedRouterPriority
    IfTransitDelay              upToMaxAge
    IfRetransInterval           upToMaxAge
    IfHelloInterval             helloRange
    IfRtrDeadInterval           positiveInteger
    IfPollInterval              positiveInteger
    IfState                     ifState
    IfDesignatedRouter          ipAddress
    IfBackupDesignatedRouter    ipAddress
    IfEvents                    int
    IfAuthKey                   string
    IfMulticastForwarding       multicastForwarding
    IfDemand                    bool
    IfAuthType                  authType
    IfLsaCount                  int
    IfLsaCksumSum               uint32
    IfDesignatedRouterId        routerId
    IfBackupDesignatedRouterId  routerId
}

// Indexed By  IfMetricIpAddress, IfMetricAddressLessIf, IfMetricTOS
type IfMetricConf struct {
    IfMetricIpAddress       ipAddress
    IfMetricAddressLessIf   interfaceIndexOrZero
    IfMetricTOS             tosType
    IfMetricValue           metric
}

type IfMetricState struct {
    IfMetricIpAddress       ipAddress
    IfMetricAddressLessIf   interfaceIndexOrZero
    IfMetricTOS             tosType
    IfMetricValue           metric
}

// Indexed By VirtIfAreaId, VirtIfNeighbor
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

type VirtIfState struct {
    VirtIfAreaId            areaId
    VirtIfNeighbor          routerId
    VirtIfTransitDelay      upToMaxAge
    VirtIfRetransInterval   upToMaxAge
    VirtIfHelloInterval     helloRange
    VirtIfRtrDeadInterval   positiveInteger
    VirtIfState             ifState
    VirtIfEvents            int
    VirtIfAuthKey           string
    VirtIfAuthType          authType
    VirtIfLsaCount          int
    VirtIfLsaCksumSum       int
}

// Indexed by NbrIpAddress, NbrAddressLessIndex
type NbrConf struct {
    NbrIpAddress                ipAddress
    NbrAddressLessIndex         interfaceIndexOrZero
    NbrPriority                 designatedRouterPriority
}

type NbrState struct {
    NbrIpAddress                ipAddress
    NbrAddressLessIndex         interfaceIndexOrZero
    NbrRtrId                    routerId
    NbrOptions                  int
    NbrPriority                 designatedRouterPriority
    NbrState                    nbrState
    NbrEvents                   int
    NbrLsRetransQLen            int
    NbmaNbrPermanence           nbmaNbrPermanence
    NbrHelloSuppressed          bool
    NbrRestartHelperStatus      nbrRestartHelperStatus
    NbrRestartHelperAge         uint32
    NbrRestartHelperExitReason  restartExitReason
}

// Virtual Neighbor Table (Read Only)
// OSPF Virtual Neighbor Entry
// Indexed By VirtNbrArea, VirtNbrRtrId
type VirtNbrState struct {
    VirtNbrArea                     areaId
    VirtNbrRtrId                    routerId
    VirtNbrIpAddress                ipAddress
    VirtNbrOptions                  int
    VirtNbrState                    nbrState
    VirtNbrEvents                   int
    VirtNbrLsRetransQLen            int
    VirtNbrHelloSuppressed          bool
    VirtNbrRestartHelperStatus      nbrRestartHelperStatus
    VirtNbrRestartHelperAge         uint32
    VirtNbrRestartHelperExitReason  restartExitReason
}

// External LSA link State - Deprecated
// Indexed by ExtLsdbType, ExtLsdbLsid, ExtLsdbRouterId
type ExtLsdbState struct {
    ExtLsdbType                     lsaType
    ExtLsdbLsid                     ipAddress
    ExtLsdbRouterId                 routerId
    ExtLsdbSequence                 int
    ExtLsdbAge                      int
    ExtLsdbChecksum                 int
    ExtLsdbAdvertisement            string
}

// OSPF Area Aggregate Table
// Replaces OSPF Area Summary Table
// Indexed By AreaAggregateAreaId, AreaAggregateLsdbType,
// AreaAggregateNet, AreaAggregateMask
type AreaAggregateConf struct {
    AreaAggregateAreaId         areaId
    AreaAggregateLsdbType       areaAggregateLsdbType
    AreaAggregateNet            ipAddress
    AreaAggregateMask           ipAddress
    AreaAggregateEffect         areaRangeEffect
    AreaAggregateExtRouteTag    int
}

type AreaAggregateState struct {
    AreaAggregateAreaId         areaId
    AreaAggregateLsdbType       areaAggregateLsdbType
    AreaAggregateNet            ipAddress
    AreaAggregateMask           ipAddress
    AreaAggregateEffect         areaRangeEffect
    AreaAggregateExtRouteTag    uint32
}

// Link local link state database for non-virtual links
// Indexed by LocalLsdbIpAddress, LocalLsdbAddressLessIf,
// LocalLsdbType, LocalLsdbLsid, LocalLsdbRouterId
type LocalLsdbState struct {
    LocalLsdbIpAddress          ipAddress
    LocalLsdbAddressLessIf      interfaceIndexOrZero
    LocalLsdbType               lsaType
    LocalLsdbLsid               ipAddress
    LocalLsdbRouterId           routerId
    LocalLsdbSequence           int
    LocalLsdbAge                int
    LocalLsdbChecksum           int
    LocalLsdbAdvertisement      string
}

// Link State Database, link-local for Virtual Links
// Indexed By VirtLocalLsdbTransitArea, VirtLocalLsdbTransitArea,
// VirtLocalLsdbType, VirtLocalLsdbLsid, VirtLocalLsdbRouterId
type VirtLocalLsdbState struct {
    VirtLocalLsdbTransitArea    areaId
    VirtLocalLsdbNeighbor       routerId
    VirtLocalLsdbType           lsaType
    VirtLocalLsdbLsid           ipAddress
    VirtLocalLsdbRouterId       routerId
    VirtLocalLsdbSequence       int
    VirtLocalLsdbAge            int
    VirtLocalLsdbChecksum       int
    VirtLocalLsdbAdvertisement  string
}

// Link State Database, AS - scope
// Indexed AsLsdbType, AsLsdbLsid, AsLsdbRouterId
type AsLsdbState struct {
    AsLsdbType                  lsaType
    AsLsdbLsid                  ipAddress
    AsLsdbRouterId              routerId
    AsLsdbSequence              int
    AsLsdbAge                   int
    AsLsdbChecksum              int
    AsLsdbAdvertisement         string
}

// Area LSA Counter Table
// Indexed By AreaLsaCountAreaId, AreaLsaCountLsaType
struct OspfAreaLsaCountState {
    AreaLsaCountAreaId          areaId
    AreaLsaCountLsaType         lsaType
    AreaLsaCountNumber          int
}
