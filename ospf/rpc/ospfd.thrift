namespace go ospfd
typedef i32 int
typedef i16 uint16
struct OspfVirtIfEntryConfig {
	1 : string VirtIfNeighbor
	2 : string VirtIfAreaId
	3 : i32 VirtIfTransitDelay
	4 : i32 VirtIfRetransInterval
	5 : i32 VirtIfHelloInterval
	6 : i32 VirtIfRtrDeadInterval
	7 : string VirtIfAuthKey
	8 : i32 VirtIfAuthType
}
struct OspfLsdbEntryState {
	1 : i32 LsdbType
	2 : string LsdbLsid
	3 : string LsdbAreaId
	4 : string LsdbRouterId
	5 : i32 LsdbSequence
	6 : i32 LsdbAge
	7 : i32 LsdbChecksum
	8 : string LsdbAdvertisement
}
struct OspfLsdbEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfLsdbEntryState> OspfLsdbEntryStateList
}
struct OspfExtLsdbEntryState {
	1 : i32 ExtLsdbType
	2 : string ExtLsdbLsid
	3 : string ExtLsdbRouterId
	4 : i32 ExtLsdbSequence
	5 : i32 ExtLsdbAge
	6 : i32 ExtLsdbChecksum
	7 : string ExtLsdbAdvertisement
}
struct OspfExtLsdbEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfExtLsdbEntryState> OspfExtLsdbEntryStateList
}
struct OspfGlobalConfig {
	1 : string RouterId
	2 : i32 AdminStat
	3 : bool ASBdrRtrStatus
	4 : bool TOSSupport
	5 : i32 ExtLsdbLimit
	6 : i32 MulticastExtensions
	7 : i32 ExitOverflowInterval
	8 : bool DemandExtensions
	9 : bool RFC1583Compatibility
	10 : i32 ReferenceBandwidth
	11 : i32 RestartSupport
	12 : i32 RestartInterval
	13 : bool RestartStrictLsaChecking
	14 : i32 StubRouterAdvertisement
}
struct OspfAsLsdbEntryState {
	1 : i32 AsLsdbType
	2 : string AsLsdbRouterId
	3 : string AsLsdbLsid
	4 : i32 AsLsdbSequence
	5 : i32 AsLsdbAge
	6 : i32 AsLsdbChecksum
	7 : string AsLsdbAdvertisement
}
struct OspfAsLsdbEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfAsLsdbEntryState> OspfAsLsdbEntryStateList
}
struct OspfNbrEntryConfig {
	1 : string NbrIpAddr
	2 : i32 NbrAddressLessIndex
	3 : i32 NbrPriority
}
struct OspfLocalLsdbEntryState {
	1 : i32 LocalLsdbAddressLessIf
	2 : i32 LocalLsdbType
	3 : string LocalLsdbIpAddress
	4 : string LocalLsdbRouterId
	5 : string LocalLsdbLsid
	6 : i32 LocalLsdbSequence
	7 : i32 LocalLsdbAge
	8 : i32 LocalLsdbChecksum
	9 : string LocalLsdbAdvertisement
}
struct OspfLocalLsdbEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfLocalLsdbEntryState> OspfLocalLsdbEntryStateList
}
struct OspfVirtNbrEntryState {
	1 : string VirtNbrRtrId
	2 : string VirtNbrArea
	3 : string VirtNbrIpAddr
	4 : i32 VirtNbrOptions
	5 : i32 VirtNbrState
	6 : i32 VirtNbrEvents
	7 : i32 VirtNbrLsRetransQLen
	8 : bool VirtNbrHelloSuppressed
	9 : i32 VirtNbrRestartHelperStatus
	10 : i32 VirtNbrRestartHelperAge
	11 : i32 VirtNbrRestartHelperExitReason
}
struct OspfVirtNbrEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfVirtNbrEntryState> OspfVirtNbrEntryStateList
}
struct OspfIfEntryConfig {
	1 : string IfIpAddress
	2 : i32 AddressLessIf
	3 : string IfAreaId
	4 : i32 IfType
	5 : i32 IfAdminStat
	6 : i32 IfRtrPriority
	7 : i32 IfTransitDelay
	8 : i32 IfRetransInterval
	9 : i32 IfHelloInterval
	10 : i32 IfRtrDeadInterval
	11 : i32 IfPollInterval
	12 : string IfAuthKey
	13 : i32 IfMulticastForwarding
	14 : bool IfDemand
	15 : i32 IfAuthType
}
struct OspfNbrEntryState {
	1 : string NbrIpAddr
	2 : i32 NbrAddressLessIndex
	3 : string NbrRtrId
	4 : i32 NbrOptions
	5 : i32 NbrState
	6 : i32 NbrEvents
	7 : i32 NbrLsRetransQLen
	8 : i32 NbmaNbrPermanence
	9 : bool NbrHelloSuppressed
	10 : i32 NbrRestartHelperStatus
	11 : i32 NbrRestartHelperAge
	12 : i32 NbrRestartHelperExitReason
}
struct OspfNbrEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfNbrEntryState> OspfNbrEntryStateList
}
struct OspfAreaEntryState {
	1 : string AreaId
	2 : i32 SpfRuns
	3 : i32 AreaBdrRtrCount
	4 : i32 AsBdrRtrCount
	5 : i32 AreaLsaCount
	6 : i32 AreaLsaCksumSum
	7 : i32 AreaNssaTranslatorState
	8 : i32 AreaNssaTranslatorEvents
}
struct OspfAreaEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfAreaEntryState> OspfAreaEntryStateList
}
struct OspfAreaLsaCountEntryState {
	1 : string AreaLsaCountAreaId
	2 : i32 AreaLsaCountLsaType
	3 : i32 AreaLsaCountNumber
}
struct OspfAreaLsaCountEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfAreaLsaCountEntryState> OspfAreaLsaCountEntryStateList
}
struct OspfAreaAggregateEntryConfig {
	1 : i32 AreaAggregateLsdbType
	2 : string AreaAggregateAreaID
	3 : string AreaAggregateNet
	4 : string AreaAggregateMask
	5 : i32 AreaAggregateEffect
	6 : i32 AreaAggregateExtRouteTag
}
struct OspfHostEntryConfig {
	1 : i32 HostTOS
	2 : string HostIpAddress
	3 : i32 HostMetric
	4 : string HostCfgAreaID
}
struct OspfHostEntryState {
	1 : i32 HostTOS
	2 : string HostIpAddress
	3 : string HostAreaID
}
struct OspfHostEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfHostEntryState> OspfHostEntryStateList
}
struct OspfStubAreaEntryConfig {
	1 : i32 StubTOS
	2 : string StubAreaId
	3 : i32 StubMetric
	4 : i32 StubMetricType
}
struct OspfGlobalState {
	1 : string RouterId
	2 : i32 VersionNumber
	3 : bool AreaBdrRtrStatus
	4 : i32 ExternLsaCount
	5 : i32 ExternLsaCksumSum
	6 : i32 OriginateNewLsas
	7 : i32 RxNewLsas
	8 : bool OpaqueLsaSupport
	9 : i32 RestartStatus
	10 : i32 RestartAge
	11 : i32 RestartExitReason
	12 : i32 AsLsaCount
	13 : i32 AsLsaCksumSum
	14 : bool StubRouterSupport
	15 : i32 DiscontinuityTime
}
struct OspfGlobalStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfGlobalState> OspfGlobalStateList
}
struct OspfAreaEntryConfig {
	1 : string AreaId
	2 : i32 AuthType
	3 : i32 ImportAsExtern
	4 : i32 AreaSummary
	5 : i32 AreaNssaTranslatorRole
	6 : i32 AreaNssaTranslatorStabilityInterval
}
struct OspfIfMetricEntryConfig {
	1 : i32 IfMetricAddressLessIf
	2 : i32 IfMetricTOS
	3 : string IfMetricIpAddress
	4 : i32 IfMetricValue
}
struct OspfVirtLocalLsdbEntryState {
	1 : i32 VirtLocalLsdbType
	2 : string VirtLocalLsdbNeighbor
	3 : string VirtLocalLsdbLsid
	4 : string VirtLocalLsdbTransitArea
	5 : string VirtLocalLsdbRouterId
	6 : i32 VirtLocalLsdbSequence
	7 : i32 VirtLocalLsdbAge
	8 : i32 VirtLocalLsdbChecksum
	9 : string VirtLocalLsdbAdvertisement
}
struct OspfVirtLocalLsdbEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfVirtLocalLsdbEntryState> OspfVirtLocalLsdbEntryStateList
}
struct OspfAreaRangeEntryConfig {
	1 : string AreaRangeAreaId
	2 : string AreaRangeNet
	3 : string AreaRangeMask
	4 : i32 AreaRangeEffect
}
struct OspfIfEntryState {
	1 : string IfIpAddress
	2 : i32 AddressLessIf
	3 : i32 IfState
	4 : string IfDesignatedRouter
	5 : string IfBackupDesignatedRouter
	6 : i32 IfEvents
	7 : i32 IfLsaCount
	8 : i32 IfLsaCksumSum
	9 : string IfDesignatedRouterId
	10 : string IfBackupDesignatedRouterId
}
struct OspfIfEntryStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<OspfIfEntryState> OspfIfEntryStateList
}
service OSPFDServices {
	bool CreateOspfVirtIfEntryConfig(1: OspfVirtIfEntryConfig config);
	bool UpdateOspfVirtIfEntryConfig(1: OspfVirtIfEntryConfig origconfig, 2: OspfVirtIfEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfVirtIfEntryConfig(1: OspfVirtIfEntryConfig config);

	OspfLsdbEntryStateGetInfo GetBulkOspfLsdbEntryState(1: int fromIndex, 2: int count);
	OspfLsdbEntryState GetOspfLsdbEntryState(1: i32 LsdbType, 2: string LsdbLsid, 3: string LsdbAreaId, 4: string LsdbRouterId);
	OspfExtLsdbEntryStateGetInfo GetBulkOspfExtLsdbEntryState(1: int fromIndex, 2: int count);
	OspfExtLsdbEntryState GetOspfExtLsdbEntryState(1: i32 ExtLsdbType, 2: string ExtLsdbLsid, 3: string ExtLsdbRouterId);
	bool CreateOspfGlobalConfig(1: OspfGlobalConfig config);
	bool UpdateOspfGlobalConfig(1: OspfGlobalConfig origconfig, 2: OspfGlobalConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfGlobalConfig(1: OspfGlobalConfig config);

	OspfAsLsdbEntryStateGetInfo GetBulkOspfAsLsdbEntryState(1: int fromIndex, 2: int count);
	OspfAsLsdbEntryState GetOspfAsLsdbEntryState(1: i32 AsLsdbType, 2: string AsLsdbRouterId, 3: string AsLsdbLsid);
	bool CreateOspfNbrEntryConfig(1: OspfNbrEntryConfig config);
	bool UpdateOspfNbrEntryConfig(1: OspfNbrEntryConfig origconfig, 2: OspfNbrEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfNbrEntryConfig(1: OspfNbrEntryConfig config);

	OspfLocalLsdbEntryStateGetInfo GetBulkOspfLocalLsdbEntryState(1: int fromIndex, 2: int count);
	OspfLocalLsdbEntryState GetOspfLocalLsdbEntryState(1: i32 LocalLsdbAddressLessIf, 2: i32 LocalLsdbType, 3: string LocalLsdbIpAddress, 4: string LocalLsdbRouterId, 5: string LocalLsdbLsid);
	OspfVirtNbrEntryStateGetInfo GetBulkOspfVirtNbrEntryState(1: int fromIndex, 2: int count);
	OspfVirtNbrEntryState GetOspfVirtNbrEntryState(1: string VirtNbrRtrId, 2: string VirtNbrArea);
	bool CreateOspfIfEntryConfig(1: OspfIfEntryConfig config);
	bool UpdateOspfIfEntryConfig(1: OspfIfEntryConfig origconfig, 2: OspfIfEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfIfEntryConfig(1: OspfIfEntryConfig config);

	OspfNbrEntryStateGetInfo GetBulkOspfNbrEntryState(1: int fromIndex, 2: int count);
	OspfNbrEntryState GetOspfNbrEntryState(1: string NbrIpAddr, 2: i32 NbrAddressLessIndex);
	OspfAreaEntryStateGetInfo GetBulkOspfAreaEntryState(1: int fromIndex, 2: int count);
	OspfAreaEntryState GetOspfAreaEntryState(1: string AreaId);
	OspfAreaLsaCountEntryStateGetInfo GetBulkOspfAreaLsaCountEntryState(1: int fromIndex, 2: int count);
	OspfAreaLsaCountEntryState GetOspfAreaLsaCountEntryState(1: string AreaLsaCountAreaId, 2: i32 AreaLsaCountLsaType);
	bool CreateOspfAreaAggregateEntryConfig(1: OspfAreaAggregateEntryConfig config);
	bool UpdateOspfAreaAggregateEntryConfig(1: OspfAreaAggregateEntryConfig origconfig, 2: OspfAreaAggregateEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfAreaAggregateEntryConfig(1: OspfAreaAggregateEntryConfig config);

	bool CreateOspfHostEntryConfig(1: OspfHostEntryConfig config);
	bool UpdateOspfHostEntryConfig(1: OspfHostEntryConfig origconfig, 2: OspfHostEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfHostEntryConfig(1: OspfHostEntryConfig config);

	OspfHostEntryStateGetInfo GetBulkOspfHostEntryState(1: int fromIndex, 2: int count);
	OspfHostEntryState GetOspfHostEntryState(1: i32 HostTOS, 2: string HostIpAddress);
	bool CreateOspfStubAreaEntryConfig(1: OspfStubAreaEntryConfig config);
	bool UpdateOspfStubAreaEntryConfig(1: OspfStubAreaEntryConfig origconfig, 2: OspfStubAreaEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfStubAreaEntryConfig(1: OspfStubAreaEntryConfig config);

	OspfGlobalStateGetInfo GetBulkOspfGlobalState(1: int fromIndex, 2: int count);
	OspfGlobalState GetOspfGlobalState(1: string RouterId);
	bool CreateOspfAreaEntryConfig(1: OspfAreaEntryConfig config);
	bool UpdateOspfAreaEntryConfig(1: OspfAreaEntryConfig origconfig, 2: OspfAreaEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfAreaEntryConfig(1: OspfAreaEntryConfig config);

	bool CreateOspfIfMetricEntryConfig(1: OspfIfMetricEntryConfig config);
	bool UpdateOspfIfMetricEntryConfig(1: OspfIfMetricEntryConfig origconfig, 2: OspfIfMetricEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfIfMetricEntryConfig(1: OspfIfMetricEntryConfig config);

	OspfVirtLocalLsdbEntryStateGetInfo GetBulkOspfVirtLocalLsdbEntryState(1: int fromIndex, 2: int count);
	OspfVirtLocalLsdbEntryState GetOspfVirtLocalLsdbEntryState(1: i32 VirtLocalLsdbType, 2: string VirtLocalLsdbNeighbor, 3: string VirtLocalLsdbLsid, 4: string VirtLocalLsdbTransitArea, 5: string VirtLocalLsdbRouterId);
	bool CreateOspfAreaRangeEntryConfig(1: OspfAreaRangeEntryConfig config);
	bool UpdateOspfAreaRangeEntryConfig(1: OspfAreaRangeEntryConfig origconfig, 2: OspfAreaRangeEntryConfig newconfig, 3: list<bool> attrset);
	bool DeleteOspfAreaRangeEntryConfig(1: OspfAreaRangeEntryConfig config);

	OspfIfEntryStateGetInfo GetBulkOspfIfEntryState(1: int fromIndex, 2: int count);
	OspfIfEntryState GetOspfIfEntryState(1: string IfIpAddress, 2: i32 AddressLessIf);
}