package bfddCommonDefs

import ()

const (
	PUB_SOCKET_ADDR = "ipc:///tmp/bfdd.ipc"
)

const (
	BGP = iota + 1
	OSPF
	MAX_NUM_PROTOCOLS
)

const (
	CREATE = iota + 1
	DELETE
	ADMINDOWN
)

type BfdSessionConfig struct {
	DestIp    string
	Protocol  int
	Operation int
}
