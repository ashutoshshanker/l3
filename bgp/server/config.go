// config.go
package server

import (
    "net"
)

type GlobalConfig struct {
    AS int
}

type PeerConfig struct {
    IP net.IP
    AS int
    SessionState uint32
}

type Peers struct {
    PeerLisr []PeerConfig
}

type Bgp struct {
	GlobalConfig GlobalConfig
	Peers Peers
}