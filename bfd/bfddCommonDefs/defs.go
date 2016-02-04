package bfddCommonDefs

import ()

const (
	PROTOCOL_BGP = iota + 1
	PROTOCOL_OSPF
	MAX_NUM_PROTOCOLS
)

const (
	PUB_SOCKET_ADDR = "ipc:///tmp/bfdd.ipc"
)

type BfdSessionConfig struct {
	DestIp    string
	Protocol  int
	Operation bool
}
