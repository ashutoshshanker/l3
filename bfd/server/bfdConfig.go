package server

import (
	"l3/bfd/bfddCommonDefs"
	"net"
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
	AuthenticationType        AuthenticationType
	AuthenticationKeyId       int32
	AuthenticationData        string
}

type IntfState struct {
	InterfaceId               int32
	Enabled                   bool
	NumSessions               int32
	LocalMultiplier           int32
	DesiredMinTxInterval      int32
	RequiredMinRxInterval     int32
	RequiredMinEchoRxInterval int32
	DemandEnabled             bool
	AuthenticationEnabled     bool
	AuthenticationType        AuthenticationType
	AuthenticationKeyId       int32
	AuthenticationData        string
}

type SessionConfig struct {
	DestIp    string
	ParamName string
	Interface string
	PerLink   bool
	Protocol  bfddCommonDefs.BfdSessionOwner
	Operation bfddCommonDefs.BfdSessionOperation
}

type SessionState struct {
	IpAddr                string
	SessionId             int32
	LocalIpAddr           string
	InterfaceName         string
	InterfaceId           int32
	ParamName             string
	PerLinkSession        bool
	LocalMacAddr          net.HardwareAddr
	RemoteMacAddr         net.HardwareAddr
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
	NumTxPackets          uint32
	NumRxPackets          uint32
}

type SessionParamConfig struct {
	Name                      string
	LocalMultiplier           int32
	DesiredMinTxInterval      int32
	RequiredMinRxInterval     int32
	RequiredMinEchoRxInterval int32
	DemandEnabled             bool
	AuthenticationEnabled     bool
	AuthenticationType        AuthenticationType
	AuthenticationKeyId       int32
	AuthenticationData        string
}

type SessionParamState struct {
	Name                      string
	NumSessions               int32
	LocalMultiplier           int32
	DesiredMinTxInterval      int32
	RequiredMinRxInterval     int32
	RequiredMinEchoRxInterval int32
	DemandEnabled             bool
	AuthenticationEnabled     bool
	AuthenticationType        AuthenticationType
	AuthenticationKeyId       int32
	AuthenticationData        string
}
