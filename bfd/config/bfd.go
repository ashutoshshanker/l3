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
	RemoteSessionState    int
	LocalDicriminator     uint32
	RemoteDiscriminator   uint32
	LocalDiagType         int
	DesiredMinTxInterval  int
	RequiredMinRxInterval int
	RemoteMinRxInterval   int
	DetectionMultiplier   uint32
	DemandMode            bool
	RemoteDemandMode      bool
	AuthSeqKnown          bool
	AuthType              uint32
	ReceivedAuthSeq       uint32
	SentAuthSeq           uint32
}

func createBfdInterface() error {
	return nil
}

func createBfdSession() error {
	return nil
}
