namespace go ospfd
typedef i32 int
typedef i16 uint16

struct OspfAreaEntryConfig{
    2 : string  AreaIdKey
    3 : i32     AuthType
    4 : i32     ImportAsExtern
    5 : i32     AreaSummary
    6 : i32     AreaNssaTranslatorRole
    7 : i32     AreaNssaTranslatorStabilityInterval
}
struct OspfAreaEntryState{
    2 : string  AreaIdKey
    3 : i32     SpfRuns
    4 : i32     AreaBdrRtrCount
    5 : i32     AsBdrRtrCount
    6 : i32     AreaLsaCount
    7 : i32     AreaLsaCksumSum
    8 : i32     AreaNssaTranslatorState
    9 : i32     AreaNssaTranslatorEvents
}
struct OspfAreaEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfAreaEntryState> OspfAreaEntryStateList
}
struct OspfStubAreaEntryConfig{
    1 : i32     StubTOSKey
    2 : string  StubAreaIdKey
    4 : i32     StubTOS
    5 : i32     StubMetric
    6 : i32     StubMetricType
}
struct OspfLsdbEntryState{
    1 : i32     LsdbTypeKey
    2 : string  LsdbLsidKey
    3 : string  LsdbAreaIdKey
    4 : string  LsdbRouterIdKey
    9 : i32     LsdbSequence
    10 : i32    LsdbAge
    11 : i32    LsdbChecksum
    12 : string     LsdbAdvertisement
}
struct OspfLsdbEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfLsdbEntryState> OspfLsdbEntryStateList
}
struct OspfAreaRangeEntryConfig{
    1 : string  AreaRangeAreaIdKey
    2 : string  AreaRangeNetKey
    5 : string  AreaRangeMask
    6 : i32     AreaRangeEffect
}
struct OspfHostEntryConfig{
    1 : i32     HostTOSKey
    2 : string  HostIpAddressKey
    5 : i32     HostMetric
    6 : string  HostCfgAreaID
}
struct OspfIfEntryConfig{
    1 : string  IfIpAddressKey
    2 : i32     AddressLessIfKey
    5 : string  IfAreaId
    6 : i32     IfType
    7 : i32     IfAdminStat
    8 : i32     IfRtrPriority
    9 : i32     IfTransitDelay
    10 : i32    IfRetransInterval
    11 : i32    IfHelloInterval
    12 : i32    IfRtrDeadInterval
    13 : i32    IfPollInterval
    14 : string     IfAuthKey
    15 : i32    IfMulticastForwarding
    16 : bool   IfDemand
    17 : i32    IfAuthType
}
struct OspfIfEntryState{
    1 : string  IfIpAddressKey
    2 : i32     AddressLessIfKey
    5 : i32     IfState
    6 : string  IfDesignatedRouter
    7 : string  IfBackupDesignatedRouter
    8 : i32     IfEvents
    9 : i32     IfLsaCount
    10 : i32    IfLsaCksumSum
    11 : string     IfDesignatedRouterId
    12 : string     IfBackupDesignatedRouterId
}
struct OspfIfEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfIfEntryState> OspfIfEntryStateList
}
struct OspfIfMetricEntryConfig{
    1 : i32     IfMetricAddressLessIfKey
    2 : i32     IfMetricTOSKey
    3 : string  IfMetricIpAddressKey
    6 : i32     IfMetricTOS
    7 : i32     IfMetricValue
}
struct OspfVirtIfEntryConfig{
    1 : string  VirtIfNeighborKey
    2 : string  VirtIfAreaIdKey
    5 : i32     VirtIfTransitDelay
    6 : i32     VirtIfRetransInterval
    7 : i32     VirtIfHelloInterval
    8 : i32     VirtIfRtrDeadInterval
    9 : string  VirtIfAuthKey
    10 : i32    VirtIfAuthType
}
struct OspfNbrEntryConfig{
    1 : string  NbrIpAddrKey
    2 : i32     NbrAddressLessIndexKey
    5 : i32     NbrPriority
}
struct OspfNbrEntryState{
    1 : string  NbrIpAddrKey
    2 : i32     NbrAddressLessIndexKey
    5 : string  NbrRtrId
    6 : i32     NbrOptions
    7 : i32     NbrState
    8 : i32     NbrEvents
    9 : i32     NbrLsRetransQLen
    10 : i32    NbmaNbrPermanence
    11 : bool   NbrHelloSuppressed
    12 : i32    NbrRestartHelperStatus
    13 : i32    NbrRestartHelperAge
    14 : i32    NbrRestartHelperExitReason
}
struct OspfNbrEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfNbrEntryState> OspfNbrEntryStateList
}
struct OspfVirtNbrEntryState{
    1 : string  VirtNbrRtrIdKey
    2 : string  VirtNbrAreaKey
    4 : string  VirtNbrRtrId
    5 : string  VirtNbrIpAddr
    6 : i32     VirtNbrOptions
    7 : i32     VirtNbrState
    8 : i32     VirtNbrEvents
    9 : i32     VirtNbrLsRetransQLen
    10 : bool   VirtNbrHelloSuppressed
    11 : i32    VirtNbrRestartHelperStatus
    12 : i32    VirtNbrRestartHelperAge
    13 : i32    VirtNbrRestartHelperExitReason
}
struct OspfVirtNbrEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfVirtNbrEntryState> OspfVirtNbrEntryStateList
}
struct OspfExtLsdbEntryState{
    1 : i32     ExtLsdbTypeKey
    2 : string  ExtLsdbLsidKey
    3 : string  ExtLsdbRouterIdKey
    5 : string  ExtLsdbLsid
    6 : string  ExtLsdbRouterId
    7 : i32     ExtLsdbSequence
    8 : i32     ExtLsdbAge
    9 : i32     ExtLsdbChecksum
    10 : string     ExtLsdbAdvertisement
}
struct OspfExtLsdbEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfExtLsdbEntryState> OspfExtLsdbEntryStateList
}
struct OspfAreaAggregateEntryConfig{
    1 : i32     AreaAggregateLsdbTypeKey
    2 : string  AreaAggregateAreaIDKey
    3 : string  AreaAggregateNetKey
    4 : string  AreaAggregateMaskKey
    7 : string  AreaAggregateNet
    8 : string  AreaAggregateMask
    9 : i32     AreaAggregateEffect
    10 : i32    AreaAggregateExtRouteTag
}
struct OspfLocalLsdbEntryState{
    1 : i32     LocalLsdbAddressLessIfKey
    2 : i32     LocalLsdbTypeKey
    3 : string  LocalLsdbIpAddressKey
    4 : string  LocalLsdbRouterIdKey
    5 : string  LocalLsdbLsidKey
    9 : string  LocalLsdbLsid
    10 : string     LocalLsdbRouterId
    11 : i32    LocalLsdbSequence
    12 : i32    LocalLsdbAge
    13 : i32    LocalLsdbChecksum
    14 : string     LocalLsdbAdvertisement
}
struct OspfLocalLsdbEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfLocalLsdbEntryState> OspfLocalLsdbEntryStateList
}
struct OspfVirtLocalLsdbEntryState{
    1 : i32     VirtLocalLsdbTypeKey
    2 : string  VirtLocalLsdbNeighborKey
    3 : string  VirtLocalLsdbLsidKey
    4 : string  VirtLocalLsdbTransitAreaKey
    5 : string  VirtLocalLsdbRouterIdKey
    7 : string  VirtLocalLsdbNeighbor
    11 : i32    VirtLocalLsdbSequence
    12 : i32    VirtLocalLsdbAge
    13 : i32    VirtLocalLsdbChecksum
    14 : string     VirtLocalLsdbAdvertisement
}
struct OspfVirtLocalLsdbEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfVirtLocalLsdbEntryState> OspfVirtLocalLsdbEntryStateList
}
struct OspfAsLsdbEntryState{
    1 : i32     AsLsdbTypeKey
    2 : string  AsLsdbRouterIdKey
    3 : string  AsLsdbLsidKey
    7 : i32     AsLsdbSequence
    8 : i32     AsLsdbAge
    9 : i32     AsLsdbChecksum
    10 : string     AsLsdbAdvertisement
}
struct OspfAsLsdbEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfAsLsdbEntryState> OspfAsLsdbEntryStateList
}
struct OspfAreaLsaCountEntryState{
    1 : string  AreaLsaCountAreaIdKey
    2 : i32     AreaLsaCountLsaTypeKey
    5 : i32     AreaLsaCountNumber
}
struct OspfAreaLsaCountEntryStateGetInfo {
    1: int StartIdx
    2: int EndIdx
    3: int Count
    4: bool More
    5: list<OspfAreaLsaCountEntryState> OspfAreaLsaCountEntryStateList
}
struct OspfGlobalConfig{
    1 : string  RouterIdKey
    3 : i32     AdminStat
    4 : i32     VersionNumber
    5 : bool    AreaBdrRtrStatus
    6 : bool    ASBdrRtrStatus
    7 : i32     ExternLsaCount
    8 : i32     ExternLsaCksumSum
    9 : bool    TOSSupport
    10 : i32    OriginateNewLsas
    11 : i32    RxNewLsas
    12 : i32    ExtLsdbLimit
    13 : i32    MulticastExtensions
    14 : i32    ExitOverflowInterval
    15 : bool   DemandExtensions
    16 : bool   RFC1583Compatibility
    17 : bool   OpaqueLsaSupport
    18 : i32    ReferenceBandwidth
    19 : i32    RestartSupport
    20 : i32    RestartInterval
    21 : bool   RestartStrictLsaChecking
    22 : i32    RestartStatus
    23 : i32    RestartAge
    24 : i32    RestartExitReason
    25 : i32    AsLsaCount
    26 : i32    AsLsaCksumSum
    27 : bool   StubRouterSupport
    28 : i32    StubRouterAdvertisement
    29 : i32    DiscontinuityTime
}

service OSPFServer {
    bool CreateOspfGlobalConfig(1: OspfGlobalConfig config);
    bool CreateOspfAreaEntryConfig(1: OspfAreaEntryConfig config);
    bool CreateOspfStubAreaEntryConfig(1: OspfStubAreaEntryConfig config);
    bool CreateOspfAreaRangeEntryConfig(1: OspfAreaRangeEntryConfig config);
    bool CreateOspfHostEntryConfig(1: OspfHostEntryConfig config);
    bool CreateOspfIfEntryConfig(1: OspfIfEntryConfig config);
    bool CreateOspfIfMetricEntryConfig(1: OspfIfMetricEntryConfig config);
    bool CreateOspfVirtIfEntryConfig(1: OspfVirtIfEntryConfig config);
    bool CreateOspfNbrEntryConfig(1: OspfNbrEntryConfig config);
    bool CreateOspfAreaAggregateEntryConfig(1: OspfAreaAggregateEntryConfig config);

    bool UpdateOspfGlobalConfig(1: OspfGlobalConfig origconfig, 2: OspfGlobalConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfAreaEntryConfig(1: OspfAreaEntryConfig origconfig, 2: OspfAreaEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfStubAreaEntryConfig(1: OspfStubAreaEntryConfig origconfig, 2: OspfStubAreaEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfAreaRangeEntryConfig(1: OspfAreaRangeEntryConfig origconfig, 2: OspfAreaRangeEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfHostEntryConfig(1: OspfHostEntryConfig origconfig, 2: OspfHostEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfIfEntryConfig(1: OspfIfEntryConfig origconfig, 2: OspfIfEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfIfMetricEntryConfig(1: OspfIfMetricEntryConfig origconfig, 2: OspfIfMetricEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfVirtIfEntryConfig(1: OspfVirtIfEntryConfig origconfig, 2: OspfVirtIfEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfNbrEntryConfig(1: OspfNbrEntryConfig origconfig, 2: OspfNbrEntryConfig newconfig, 3: list<bool> attrset);
    bool UpdateOspfAreaAggregateEntryConfig(1: OspfAreaAggregateEntryConfig origconfig, 2: OspfAreaAggregateEntryConfig newconfig, 3: list<bool> attrset);


    bool DeleteOspfGlobalConfig(1: OspfGlobalConfig config);
    bool DeleteOspfAreaEntryConfig(1: OspfAreaEntryConfig config);
    bool DeleteOspfStubAreaEntryConfig(1: OspfStubAreaEntryConfig config);
    bool DeleteOspfAreaRangeEntryConfig(1: OspfAreaRangeEntryConfig config);
    bool DeleteOspfHostEntryConfig(1: OspfHostEntryConfig config);
    bool DeleteOspfIfEntryConfig(1: OspfIfEntryConfig config);
    bool DeleteOspfIfMetricEntryConfig(1: OspfIfMetricEntryConfig config);
    bool DeleteOspfVirtIfEntryConfig(1: OspfVirtIfEntryConfig config);
    bool DeleteOspfNbrEntryConfig(1: OspfNbrEntryConfig config);
    bool DeleteOspfAreaAggregateEntryConfig(1: OspfAreaAggregateEntryConfig config);


    OspfAreaEntryStateGetInfo GetBulkOspfAreaEntryState(1: int fromIndex, 2: int count);
    OspfLsdbEntryStateGetInfo GetBulkOspfLsdbEntryState(1: int fromIndex, 2: int count);
    OspfIfEntryStateGetInfo GetBulkOspfIfEntryState(1: int fromIndex, 2: int count);
    OspfNbrEntryStateGetInfo GetBulkOspfNbrEntryState(1: int fromIndex, 2: int count);
    OspfVirtNbrEntryStateGetInfo GetBulkOspfVirtNbrEntryState(1: int fromIndex, 2: int count);
    OspfExtLsdbEntryStateGetInfo GetBulkOspfExtLsdbEntryState(1: int fromIndex, 2: int count);
    OspfLocalLsdbEntryStateGetInfo GetBulkOspfLocalLsdbEntryState(1: int fromIndex, 2: int count);
    OspfVirtLocalLsdbEntryStateGetInfo GetBulkOspfVirtLocalLsdbEntryState(1: int fromIndex, 2: int count);
    OspfAsLsdbEntryStateGetInfo GetBulkOspfAsLsdbEntryState(1: int fromIndex, 2: int count);
    OspfAreaLsaCountEntryStateGetInfo GetBulkOspfAreaLsaCountEntryState(1: int fromIndex, 2: int count);

/*
    OspfGlobalState GetOspfGlobalState()
    OspfAreaState GetOspfAreaState(1: string areaId)
    OspfStubAreaState GetOspfStubAreaState(1: string stubAreaId, 2: i32 stubTOS)
    OspfLsdbState GetOspfLsdbState(1: string lsdbAreaId, 2: string lsdbLsid, 3: string lsdbRouterId)
    OspfAreaRangeState GetOspfAreaRangeState(1: string rangeAreaId, 2: string areaRangeNet)
    OspfHostState GetOspfHostState(1: string hostIpAddress, 2: i32 hostTOS)
    OspfIfState GetOspfIfState(1: string ifIpAddress, 2: i32 addressLessIf)
    OspfIfMetricState GetOspfIfMetricState(1: string ifMetricIpAddress, 2: i32 ifMetricAddressLessIf, 3: i32 ifMetricTOS)
    OspfVirtIfState GetOspfVirtIfState(1: string virtIfAreaId, 2: string virtIfNeighbor)
    OspfNbrState GetOspfNbrState(1: string nbrIpAddress, 2: i32 nbrAddressLessIndex)
    OspfVirtNbrState GetOspfVirtNbrState(1: string virtNbrArea, 2: string virtNbrRtrId)
    OspfExtLsdbState GetOspfExtLsdbState(1: lsaType extLsdbType, 2: string extLsdbLsid, 3: string extLsdbRouterId)
    OspfAreaAggregateState GetOspfAreaAggregateState(1: string areaAggregateAreaId, 2: lsaType areaAggregateLsdbType, 3: string areaAggregateNet, 4: string areaAggregateMask)
    OspfLocalLsdbState GetOspfLocalLsdbState(1: string localLsdbIpAddress, 2: i32 localLsdbAddressLessIf, 3: lsaType localLsdbType, 4: string localLsdbLsid, 5: string localLsdbRouterId)
    OspfVirtLocalLsdbState GetOspfVirtLocalLsdbState(1: string virtLocalLsdbTransitArea, 2: string virtLocalLsdbNeighbor, 3: lsaType virtLocalLsdbType, 4: string virtLocalLsdbLsid, 5: string virtLocalLsdbRouterId)
    OspfAsLsdbState GetOspfAsLsdbState(1: lsaType asLsdbType, 2: string asLsdbLsid, 3: string asLsdbRouterId)
    OspfAreaLsaCountState GetOspfAreaLsaCountState(1: string areaLsaCountAreaId, 2: lsaType areaLsaCountLsaType)
*/
}
