package config

import (
//"net"
)

type GlobalConf struct {
	Enable bool
}

type GlobalState struct {
	Enable        bool
	NumInterfaces uint32
	NumSessions   uint32
}

type InterfaceConf struct {
	InterfaceId   int32
	Mode          string
	TxMinInterval uint32
	RxMinInterval uint32
	Multiplier    uint32
	EchoFunction  bool
}

type InterfaceState struct {
}

type SessionState struct {
}
