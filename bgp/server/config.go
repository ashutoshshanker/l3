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

type CONN_DIR int

const (
    CONN_DIR_OUT CONN_DIR = iota
    CONN_DIR_IN
    CONN_DIR_MAX
)
