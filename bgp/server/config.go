// config.go
package server

import (
    "net"
)

type GlobalConfig struct {
    AS uint16
}

type PeerConfig struct {
    IP net.IP
    AS uint16
    SessionState uint32
}

type PeerCommand struct {
    IP net.IP
    Command int
}

type Peers struct {
    PeerList []PeerConfig
}

type Bgp struct {
	GlobalConfig GlobalConfig
	Peers Peers
}

type ConnDir int

const (
    ConnDirOut ConnDir = iota
    ConnDirIn
    ConnDirMax
)

type PeerType int

const (
    PeerTypeInternal PeerType = iota
    PeerTypeExternal
    PeerTypeMax
)
