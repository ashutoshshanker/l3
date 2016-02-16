namespace go bfdd
typedef i32 int
typedef i16 uint16
struct BfdGlobalConfig{
	1 : string 	Bfd
	2 : bool 	Enable
}
struct BfdGlobalState{
	1 : bool 	Enable
	2 : i32 	NumInterfaces
	3 : i32 	NumTotalSessions
	4 : i32 	NumUpSessions
	5 : i32 	NumDownSessions
	6 : i32 	NumAdminDownSessions
}
struct BfdGlobalStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BfdGlobalState> BfdGlobalStateList
}
struct BfdIntfConfig{
	1 : i32 	Interface
	2 : i32 	LocalMultiplier
	3 : i32 	DesiredMinTxInterval
	4 : i32 	RequiredMinRxInterval
	5 : i32 	RequiredMinEchoRxInterval
	6 : bool 	DemandEnabled
	7 : bool 	AuthenticationEnabled
	8 : i32 	AuthType
	9 : i32 	AuthKeyId
	10 : string 	AuthData
}
struct BfdIntfState{
	1 : i32 	InterfaceId
	2 : bool 	Enabled
	3 : i32 	NumSessions
	4 : i32 	LocalMultiplier
	5 : i32 	DesiredMinTxInterval
	6 : i32 	RequiredMinRxInterval
	7 : i32 	RequiredMinEchoRxInterval
	8 : bool 	DemandEnabled
	9 : bool 	AuthenticationEnabled
	10 : i32 	AuthenticationType
	11 : i32 	AuthenticationKeyId
	12 : string 	AuthenticationData
}
struct BfdIntfStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BfdIntfState> BfdIntfStateList
}
struct BfdSessionConfig{
	1 : string 	IpAddr
	2 : i32 	Owner
	3 : i32 	Operation
}
struct BfdSessionState{
	1 : i32 	SessionId
	2 : string 	LocalIpAddr
	3 : string 	RemoteIpAddr
	4 : i32 	InterfaceId
	5 : string 	RegisteredProtocols
	6 : i32 	SessionState
	7 : i32 	RemoteSessionState
	8 : i32 	LocalDiscriminator
	9 : i32 	RemoteDiscriminator
	10 : i32 	LocalDiagType
	11 : i32 	DesiredMinTxInterval
	12 : i32 	RequiredMinRxInterval
	13 : i32 	RemoteMinRxInterval
	14 : i32 	DetectionMultiplier
	15 : bool 	DemandMode
	16 : bool 	RemoteDemandMode
	17 : bool 	AuthSeqKnown
	18 : i32 	AuthType
	19 : i32 	ReceivedAuthSeq
	20 : i32 	SentAuthSeq
}
struct BfdSessionStateGetInfo {
	1: int StartIdx
	2: int EndIdx
	3: int Count
	4: bool More
	5: list<BfdSessionState> BfdSessionStateList
}
service BFDDServices {
	bool CreateBfdGlobalConfig(1: BfdGlobalConfig config);
	bool UpdateBfdGlobalConfig(1: BfdGlobalConfig origconfig, 2: BfdGlobalConfig newconfig, 3: list<bool> attrset);
	bool DeleteBfdGlobalConfig(1: BfdGlobalConfig config);

	BfdGlobalStateGetInfo GetBulkBfdGlobalState(1: int fromIndex, 2: int count);
	bool CreateBfdIntfConfig(1: BfdIntfConfig config);
	bool UpdateBfdIntfConfig(1: BfdIntfConfig origconfig, 2: BfdIntfConfig newconfig, 3: list<bool> attrset);
	bool DeleteBfdIntfConfig(1: BfdIntfConfig config);

	BfdIntfStateGetInfo GetBulkBfdIntfState(1: int fromIndex, 2: int count);
	bool CreateBfdSessionConfig(1: BfdSessionConfig config);
	bool UpdateBfdSessionConfig(1: BfdSessionConfig origconfig, 2: BfdSessionConfig newconfig, 3: list<bool> attrset);
	bool DeleteBfdSessionConfig(1: BfdSessionConfig config);

	BfdSessionStateGetInfo GetBulkBfdSessionState(1: int fromIndex, 2: int count);
}