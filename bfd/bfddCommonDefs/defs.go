package bfddCommonDefs

import ()

const (
	PUB_SOCKET_ADDR = "ipc:///tmp/bfdd.ipc"
)

// Owner
const (
	USER = iota + 1
	BGP
	OSPF
	MAX_NUM_PROTOCOLS
)

// Operation
const (
	CREATE = iota + 1
	DELETE
	ADMINUP
	ADMINDOWN
)

type BfdSessionConfig struct {
	DestIp    string
	Protocol  int
	Operation int
}
