namespace go bfdd
typedef i32 int
typedef i16 uint16
struct BfdSessionState {
	1 : string IpAddr
	2 : i32 SessionId
	3 : string LocalIpAddr
	4 : i32 IfIndex
	5 : bool PerLinkSession
	6 : string LocalMacAddr
	7 : string RemoteMacAddr
	8 : string RegisteredProtocols
	9 : string SessionState
	10 : string RemoteSessionState
	11 : i32 LocalDiscriminator
	12 : i32 RemoteDiscriminator
	13 : string LocalDiagType
	14 : string DesiredMinTxInterval
	15 : string RequiredMinRxInterval
	16 : string RemoteMinRxInterval
	17 : i32 DetectionMultiplier
	18 : bool DemandMode
	19 : bool RemoteDemandMode
	20 : bool AuthSeqKnown
	21 : string AuthType
	22 : i32 ReceivedAuthSeq
	23 : i32 SentAuthSeq
	24 : i32 NumTxPackets
	25 : i32 NumRxPackets
}
struct BfdSessionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BfdSessionState> BfdSessionStateList
}
struct BfdInterfaceState {
	1 : i32 IfIndex
	2 : bool Enabled
	3 : i32 NumSessions
	4 : i32 LocalMultiplier
	5 : string DesiredMinTxInterval
	6 : string RequiredMinRxInterval
	7 : string RequiredMinEchoRxInterval
	8 : bool DemandEnabled
	9 : bool AuthenticationEnabled
	10 : string AuthenticationType
	11 : i32 AuthenticationKeyId
	12 : string AuthenticationData
}
struct BfdInterfaceStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BfdInterfaceState> BfdInterfaceStateList
}
struct BfdSession {
	1 : string IpAddr
	2 : bool PerLink
	3 : string Owner
}
struct BfdGlobal {
	1 : string Bfd
	2 : bool Enable
}
struct BfdGlobalState {
	1 : string Bfd
	2 : bool Enable
	3 : i32 NumInterfaces
	4 : i32 NumTotalSessions
	5 : i32 NumUpSessions
	6 : i32 NumDownSessions
	7 : i32 NumAdminDownSessions
}
struct BfdGlobalStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BfdGlobalState> BfdGlobalStateList
}
struct BfdInterface {
	1 : i32 IfIndex
	2 : i32 LocalMultiplier
	3 : i32 DesiredMinTxInterval
	4 : i32 RequiredMinRxInterval
	5 : i32 RequiredMinEchoRxInterval
	6 : bool DemandEnabled
	7 : bool AuthenticationEnabled
	8 : string AuthType
	9 : i32 AuthKeyId
	10 : string AuthData
}
service BFDDServices {
	BfdSessionStateGetInfo GetBulkBfdSessionState(1: int fromIndex, 2: int count);
	BfdSessionState GetBfdSessionState(1: string IpAddr);
	BfdInterfaceStateGetInfo GetBulkBfdInterfaceState(1: int fromIndex, 2: int count);
	BfdInterfaceState GetBfdInterfaceState(1: i32 IfIndex);
	bool CreateBfdSession(1: BfdSession config);
	bool UpdateBfdSession(1: BfdSession origconfig, 2: BfdSession newconfig, 3: list<bool> attrset);
	bool DeleteBfdSession(1: BfdSession config);

	bool CreateBfdGlobal(1: BfdGlobal config);
	bool UpdateBfdGlobal(1: BfdGlobal origconfig, 2: BfdGlobal newconfig, 3: list<bool> attrset);
	bool DeleteBfdGlobal(1: BfdGlobal config);

	BfdGlobalStateGetInfo GetBulkBfdGlobalState(1: int fromIndex, 2: int count);
	BfdGlobalState GetBfdGlobalState(1: string Bfd);
	bool CreateBfdInterface(1: BfdInterface config);
	bool UpdateBfdInterface(1: BfdInterface origconfig, 2: BfdInterface newconfig, 3: list<bool> attrset);
	bool DeleteBfdInterface(1: BfdInterface config);

}