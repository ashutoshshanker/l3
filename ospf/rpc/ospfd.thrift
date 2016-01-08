namespace go ospfd
typedef i32 int
//typedef areaId string
//typedef routerId string
//typedef metric int // 0x0 to 0xffff
//typedef bigMetric int // 0x0 to 0xffffff
//typedef positiveInteger int // 0x0 to 0x7fffffff
//typedef helloRange int // 0x1 to 0xffff
//typedef upToMaxAge int // 0x0 to 3600
//typedef designatedRouterPriority int // 0x0 to 0xff
//typedef tosType int // 0x0 to 30
//typedef ipAddress string
//typedef interfaceIndexOrZero int

enum status {
    enabled = 1,
    disabled = 2
}

enum authType {
    none = 0,
    simplePassword = 1,
    md5 = 2,
    reserved
}

enum restartSupport {
    none = 1,
    plannedOnly = 2,
    plannedAndUnplanned = 3
}

enum advertiseAction {
    doNotAdvertise = 1,
    advertise
}

enum importAsExtern {
    importExternal = 1,
    importNoExternal = 2,
    importNssa = 3
}

enum areaSummary {
    noAreaSummary = 1,
    sendAreaSummary = 2
}

enum nssaTranslatorRole {
    always = 1,
    cadidate = 2
}

enum metricType {
    ospfMetric = 1,
    comparableCost = 2,
    nonComparable = 3
}

enum areaRangeEffect {
    advertiseMatching = 1,
    doNotAdvertiseMatching
}

enum areaAggregateLsdbType {
    summaryLink = 3,
    nssaExternalLink = 7
}

enum ifType {
    broadcast = 1,
    nbma = 2,
    pointToPoint = 3,
    pointToMultipoint = 5
}

enum multicastForwarding {
    blocked = 1,
    multicast = 2,
    unicast = 3
}

enum restartStatus {
    notRestarting = 1,
    plannedRestart = 2,
    unplannedRestart = 3
}

enum restartExitReason {
    noneAttempt = 1,
    inProgress = 2,
    completed = 3,
    timedOut = 4,
    topologyChanged = 5
}

enum nssaTranslatorState {
    enabled = 1,
    elected = 2,
    disabled = 3,
}

enum lsdbType {
    routerLink = 1,
    networkLink = 2,
    summaryLink = 3,
    asSummaryLink = 4,
    asExternalLink = 5,
    multicastLink = 6,
    nssaExternalLink = 7,
    areaOpaqueLink = 10
}

enum ifState {
    down = 1,
    loopback = 2,
    waiting = 3,
    pointToPoint = 4,
    designatedRouter = 5,
    backupDesignatedRouter = 6,
    otherDesignatedRouter = 7
}

// Global Configuration Objects
// General Variables
struct OspfGlobalConf {
    1: string               RouterId,
    2: status               AdminStat,
    3: bool                 ASBdrRtrStatus,
    4: bool                 TOSSupport,
    5: i32                  ExtLsdbLimit,
    6: i32                  MulticastExtensions,
    7: i32                  ExitOverflowInterval,
    8: bool                 DemandExtensions,
    9: bool                 RFC1583Compatibility,
    10: i32                 ReferenceBandwidth,
    11: restartSupport      RestartSupport,
    12: i32                 RestartInterval,
    13: bool                RestartStrictLsaChecking,
    14: advertiseAction     StubRouterAdvertisement,
}

struct OspfGlobalState {
    1: string               RouterId,
    2: status               AdminStat,
    3: i32                  VersionNumber,
    4: bool                 AreaBdrRtrStatus,
    5: bool                 ASBdrRtrStatus,
    6: i32                  ExternLsaCount,
    7: i32                  ExternLsaCksumSum,
    8: bool                 TOSSupport,
    9: i32                  OriginateNewLsas,
    10: i32                 RxNewLsas,
    11: i32                 ExtLsdbLimit,
    12: i32                 MulticastExtensions,
    13: i32                 ExitOverflowInterval,
    14: bool                DemandExtensions,
    15: bool                RFC1583Compatibility,
    16: bool                OpaqueLsaSupport,
    17: i32                 ReferenceBandwidth,
    18: restartSupport      RestartSupport,
    19: i32                 RestartInterval,
    20: bool                RestartStrictLsaChecking,
    21: restartStatus       RestartStatus,
    22: i32                 RestartAge,
    23: restartExitReason   RestartExitReason,
    24: i32                 AsLsaCount,
    25: i32                 AsLsaCksumSum,
    26: bool                StubRouterSupport,
    27: advertiseAction     StubRouterAdvertisement,
    28: string              DiscontinuityTime,
}

// Configuration Parameter for Router's Attached Area
// Area Data Structure
// Indexed By AreaId
struct OspfAreaConf {
    1: string               AreaId,
    2: authType             AuthType,
    3: importAsExtern       ImportAsExtern,
    4: areaSummary          AreaSummary,
    5: nssaTranslatorRole   AreaNssaTranslatorRole,
    6: i32                  AreaNssaTranslatorStabilityInterval,
}

struct OspfAreaState {
    1: string               AreaId,
    2: authType             AuthType,
    3: importAsExtern       ImportAsExtern,
    4: i32                  SpfRuns,
    5: i32                  AreaBdrRtrCount,
    6: i32                  AsBdrRtrCount,
    7: i32                  AreaLsaCount,
    8: i32                  AreaLsaCksumSum,
    9: areaSummary          AreaSummary,
    10: nssaTranslatorRole  AreaNssaTranslatorRole,
    11: nssaTranslatorState AreaNssaTranslatorState,
    12: i32                 AreaNssaTranslatorStabilityInterval,
    13: i32                 AreaNssaTranslatorEvents,
}

// Area Stub Metric Table
// The Metric for a given TOS that will be advertised by default
// Area Border Router into a stub area
// Indexed by StubAreaId and StubTOS

struct OspfStubAreaConf {
    1: string               StubAreaId,
    2: i32                  StubTOS,
    3: i32                  StubMetric,
    4: metricType           StubMetricType,
}

struct OspfStubAreaState {
    1: string               StubAreaId,
    2: i32                  StubTOS,
    3: i32                  StubMetric,
    4: metricType           StubMetricType,
}

// Link State Advertisement Database
// Indexed by LsdbAreaId, LsdbType, LsdbLsid, LsdbRouterId
struct OspfLsdbState {
    1: string               LsdbAreaId,
    2: lsdbType             LsdbType,
    3: string               LsdbLsid,
    4: string               LsdbRouterId,
    5: i32                  LsdbSequence,
    6: i32                  LsdbAge,
    7: i32                  LsdbChecksum,
    8: string               LsdbAdvertisement,
}

// Address Range Table
// A single area address range
// Indexed By RangeAreaId, RangeNet

struct OspfAreaRangeConf {
    1: string                   RangeAreaId,
    2: string                   AreaRangeNet,
    3: string                   AreaRangeMask,
    4: areaRangeEffect          AreaRangeEffect,
}

struct OspfAreaRangeState {
    1: string                   RangeAreaId,
    2: string                   AreaRangeNet,
    3: string                   AreaRangeMask,
    4: areaRangeEffect          AreaRangeEffect,
}

// Host Table
// Metric to be advertised for a given TOS when
// given host is reachable
// Indexed By HostIpAddress, HostTOS

struct OspfHostConf {
    1: string       HostIpAddress,
    2: i32          HostTOS,
    3: i32          HostMetric,
    4: string       HostCfgAreaID,
}

struct OspfHostState {
    1: string       HostIpAddress,
    2: i32          HostTOS,
    3: i32          HostMetric,
    4: string       HostCfgAreaID,
}

// Interface Table
// Ospf Interface Entry describes one interface
// from the viewpoint of OSPF
// Indexed By IfIpAddress, AddressLessIf

struct OspfIfConf {
    1: string                   IfIpAddress,
    2: i32                      AddressLessIf,
    3: string                   IfAreaId,
    4: ifType                   IfType,
    5: status                   IfAdminStat,
    6: i32                      IfRtrPriority,
    7: i32                      IfTransitDelay,
    8: i32                      IfRetransInterval,
    9: i32                      IfHelloInterval,
    10: i32                     IfRtrDeadInterval,
    11: i32                     IfPollInterval,
    12: string                  IfAuthKey,
    14: multicastForwarding     IfMulticastForwarding,
    15: bool                    IfDemand,
    16: authType                IfAuthType,
}

struct OspfIfState {
    1: string                   IfIpAddress,
    2: i32                      AddressLessIf,
    3: string                   IfAreaId,
    4: ifType                   IfType,
    5: status                   IfAdminStat,
    6: i32                      IfRtrPriority,
    7: i32                      IfTransitDelay,
    8: i32                      IfRetransInterval,
    9: i32                      IfHelloInterval,
    10: i32                     IfRtrDeadInterval,
    11: i32                     IfPollInterval,
    12: ifState                 IfState,
    13: string                  IfDesignatedRouter,
    14: string                  IfBackupDesignatedRouter,
    15: i32                     IfEvents,
    16: string                  IfAuthKey,
    17: multicastForwarding     IfMulticastForwarding,
    18: bool                    IfDemand,
    19: authType                IfAuthType,
    20: i32                     IfLsaCount,
    21: i32                     IfLsaCksumSum,
    22: string                  IfDesignatedRouterId,
    23: string                  IfBackupDesignatedRouterId,
}

// Interface Metric Table
// Particular TOS Metric for a non virtual interface
// Indexed By  IfMetricIpAddress, IfMetricAddressLessIf, IfMetricTOS

struct OspfIfMetricConf {
    1: string                   IfMetricIpAddress,
    2: i32                      IfMetricAddressLessIf,
    3: i32                      IfMetricTOS,
    4: i32                      IfMetricValue,
}

struct OspfIfMetricState {
    1: string                   IfMetricIpAddress,
    2: i32                      IfMetricAddressLessIf,
    3: i32                      IfMetricTOS,
    4: i32                      IfMetricValue,
}

// Virtual Interface Table
// OSPF Virtual Interface Entry
// Indexed By VirtIfAreaId, VirtIfNeighbor
struct OspfVirtIfConf {
    1: string                   VirtIfAreaId,
    2: string                   VirtIfNeighbor,
    3: i32                      VirtIfTransitDelay,
    4: i32                      VirtIfRetransInterval,
    5: i32                      VirtIfHelloInterval,
    6: i32                      VirtIfRtrDeadInterval,
    7: string                   VirtIfAuthKey,
    8: authType                 VirtIfAuthType,
}

struct OspfVirtIfState {
    1: string                   VirtIfAreaId,
    2: string                   VirtIfNeighbor,
    3: i32                      VirtIfTransitDelay,
    4: i32                      VirtIfRetransInterval,
    5: i32                      VirtIfHelloInterval,
    6: i32                      VirtIfRtrDeadInterval,
    7: ifState                  VirtIfState,
    8: i32                      VirtIfEvents,
    9: string                   VirtIfAuthKey,
    10: authType                VirtIfAuthType,
    11: i32                     VirtIfLsaCount,
    12: i32                     VirtIfLsaCksumSum,
}

// Neighbor Table
// OSPF Neighbor Entry
// Indexed by NbrIpAddress, NbrAddressLessIndex
struct OspfNbrConf {
    1: string                   NbrIpAddress,
    2: i32                      NbrAddressLessIndex,
    3: i32                      NbrPriority,
}

// Todo: Virtual Neighbor Table (Read Only)
// OSPF Virtual Neighbor Entry
// Indexed By VirtNbrArea, VirtNbrRtrId

// OSPF Area Aggregate Table
// Replaces OSPF Area Summary Table
// Indexed By AreaAggregateAreaId, AreaAggregateLsdbType,
// AreaAggregateNet, AreaAggregateMask
struct OspfAreaAggregateConf {
    1: string                   AreaAggregateAreaId,
    2: areaAggregateLsdbType    AreaAggregateLsdbType,
    3: string                   AreaAggregateNet,
    4: string                   AreaAggregateMask,
    5: areaRangeEffect          AreaAggregateEffect,
    6: i32                      AreaAggregateExtRouteTag,
}

/*
struct OSPFAreaConf {
    1: i32                      AreaId,
    2: list<OSPFAddressRange>   AddressRange,
    3: bool                     ExternalRoutingCapability,
    4: i32                      StubDefaultCost,
}
*/


service OSPFServer {
    bool CreateOspfGlobalConf(1: OspfGlobalConf ospfGlobalConf)
    bool CreateOspfAreaConf(1: OspfAreaConf ospfAreaConf)
    bool CreateOspfStubAreaConf(1: OspfStubAreaConf ospfStubAreaConf)
    bool CreateOspfAreaRangeConf(1: OspfAreaRangeConf ospfAreaRangeConf)
    bool CreateOspfHostConf(1: OspfHostConf ospfHostConf)
    bool CreateOspfIfConf(1: OspfIfConf ospfIfConf)
    bool CreateOspfIfMetricConf(1: OspfIfMetricConf ospfIfMetricConf)
    bool CreateOspfVirtIfConf(1: OspfVirtIfConf ospfVirtIfConf)
    bool CreateOspfNbrConf(1: OspfNbrConf ospfNbrConf)
    bool CreateOspfAreaAggregateConf(1: OspfAreaAggregateConf ospfAreaAggregateConf)

    bool DeleteOspfGlobalConf(1: OspfGlobalConf ospfGlobalConf)
    bool DeleteOspfAreaConf(1: OspfAreaConf ospfAreaConf)
    bool DeleteOspfStubAreaConf(1: OspfStubAreaConf ospfStubAreaConf)
    bool DeleteOspfAreaRangeConf(1: OspfAreaRangeConf ospfAreaRangeConf)
    bool DeleteOspfHostConf(1: OspfHostConf ospfHostConf)
    bool DeleteOspfIfConf(1: OspfIfConf ospfIfConf)
    bool DeleteOspfIfMetricConf(1: OspfIfMetricConf ospfIfMetricConf)
    bool DeleteOspfVirtIfConf(1: OspfVirtIfConf ospfVirtIfConf)
    bool DeleteOspfNbrConf(1: OspfNbrConf ospfNbrConf)
    bool DeleteOspfAreaAggregateConf(1: OspfAreaAggregateConf ospfAreaAggregateConf)

//    bool CreateOSPFAreaConf(1: OSPFAreaConf ospfAreaConf)
//    bool UpdateOSPFGlobalConf(1: OSPFGlobalConf ospfConf)
}
