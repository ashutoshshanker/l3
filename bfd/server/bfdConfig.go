package server

import ()

type GlobalConfig struct {
	Enable bool
}

type GlobalState struct {
	Enable               bool
	NumInterfaces        uint32
	NumSessions          uint32
	NumUpSessions        uint32
	NumDownSessions      uint32
	NumAdminDownSessions uint32
}

type IntfConfig struct {
	InterfaceId               int32
	LocalMultiplier           int32
	DesiredMinTxInterval      int32
	RequiredMinRxInterval     int32
	RequiredMinEchoRxInterval int32
	DemandEnabled             bool
	AuthenticationEnabled     bool
	AuthenticationType        int32
	AuthenticationKeyId       int32
	SequenceNumber            int32
	AuthenticationData        string
}

type IntfState struct {
}

type SessionState struct {
	SessionId             int32
	LocalIpAddr           string
	RemoteIpAddr          string
	InterfaceId           int32
	RegisteredProtocols   []bool
	SessionState          BfdSessionState
	RemoteSessionState    BfdSessionState
	LocalDiscriminator    uint32
	RemoteDiscriminator   uint32
	LocalDiagType         BfdDiagnostic
	DesiredMinTxInterval  int32
	RequiredMinRxInterval int32
	RemoteMinRxInterval   int32
	DetectionMultiplier   int32
	DemandMode            bool
	RemoteDemandMode      bool
	AuthType              AuthenticationType
	AuthSeqKnown          bool
	ReceivedAuthSeq       uint32
	SentAuthSeq           uint32
}
