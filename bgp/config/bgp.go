// bgp.go
package config

import (
	"net"
)

type GlobalConfig struct {
	AS       uint32
	RouterId net.IP
}

type GlobalState struct {
	AS            uint32
	RouterId      net.IP
	TotalPaths    uint32
	TotalPrefixes uint32
}

type Global struct {
	Config GlobalConfig
	State  GlobalState
}

type PeerType int

const (
	PeerTypeInternal PeerType = iota
	PeerTypeExternal
)

type BgpCounters struct {
	Update       uint64
	Notification uint64
}

type Messages struct {
	Sent     BgpCounters
	Received BgpCounters
}

type Queues struct {
	Input  uint32
	Output uint32
}

type NeighborConfig struct {
	PeerAS                  uint32
	LocalAS                 uint32
	AuthPassword            string
	Description             string
	NeighborAddress         net.IP
	RouteReflectorClusterId uint32
	RouteReflectorClient    bool
	MultiHopEnable          bool
	MultiHopTTL             uint8
	ConnectRetryTime        uint32
	HoldTime                uint32
	KeepaliveTime           uint32
}

type NeighborState struct {
	PeerAS                  uint32
	LocalAS                 uint32
	PeerType                PeerType
	AuthPassword            string
	Description             string
	NeighborAddress         net.IP
	SessionState            uint32
	Messages                Messages
	Queues                  Queues
	RouteReflectorClusterId uint32
	RouteReflectorClient    bool
	MultiHopEnable          bool
	MultiHopTTL             uint8
	ConnectRetryTime        uint32
	HoldTime                uint32
	KeepaliveTime           uint32
}

type TransportConfig struct {
	TcpMss       uint16
	MTUDiscovery bool
	PassiveMode  bool
	LocalAddress net.IP
}

type TransportState struct {
	TcpMss        uint16
	MTUDiscovery  bool
	PassiveMode   bool
	LocalAddress  net.IP
	LocalPort     uint16
	RemoteAddress net.IP
	RemotePort    net.IP
}

type Transport struct {
	Config TransportConfig
	State  TransportState
}

type PeerCommand struct {
	IP      net.IP
	Command int
}

type Neighbor struct {
	NeighborAddress net.IP
	Config          NeighborConfig
	State           NeighborState
	Transport       Transport
}

type Neighbors struct {
	Neighbors []Neighbor
}

type Bgp struct {
	Global    Global
	Neighbors Neighbors
}
