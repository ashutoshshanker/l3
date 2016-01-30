package config

import (
	"l3/bfd/protocol"
)

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
	ReqisteredProtocols   string
	SessionState          protocol.BfdSessionState
	RemoteSessionState    protocol.BfdSessionState
	LocalDicriminator     uint32
	RemoteDiscriminator   uint32
	LocalDiagType         protocol.BfdDiagnostic
	DesiredMinTxInterval  int
	RequiredMinRxInterval int
	RemoteMinRxInterval   int
	DetectionMultiplier   uint32
	DemandMode            bool
	RemoteDemandMode      bool
	AuthType              protocol.AuthenticationType
	AuthSeqKnown          bool
	ReceivedAuthSeq       uint32
	SentAuthSeq           uint32
}
