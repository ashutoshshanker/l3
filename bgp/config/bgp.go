// bgp.go
package config

import (
	"net"
)

type GlobalConfig struct {
	AS                  uint32
	RouterId            net.IP
	UseMultiplePaths    bool
	EBGPMaxPaths        uint32
	EBGPAllowMultipleAS bool
	IBGPMaxPaths        uint32
}

type GlobalState struct {
	AS                  uint32
	RouterId            net.IP
	UseMultiplePaths    bool
	EBGPMaxPaths        uint32
	EBGPAllowMultipleAS bool
	IBGPMaxPaths        uint32
	TotalPaths          uint32
	TotalPrefixes       uint32
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

type BaseConfig struct {
	PeerAS                  uint32
	LocalAS                 uint32
	AuthPassword            string
	Description             string
	RouteReflectorClusterId uint32
	RouteReflectorClient    bool
	MultiHopEnable          bool
	MultiHopTTL             uint8
	ConnectRetryTime        uint32
	HoldTime                uint32
	KeepaliveTime           uint32
	BfdEnable               bool
	BfdSessionParam         string
	AddPathsRx              bool
	AddPathsMaxTx           uint8
	MaxPrefixes             uint32
	MaxPrefixesThresholdPct uint8
	MaxPrefixesDisconnect   bool
	MaxPrefixesRestartTimer uint8
}

type NeighborConfig struct {
	BaseConfig
	NeighborAddress net.IP
	IfIndex         int32
	PeerGroup       string
}

type NeighborState struct {
	PeerAS                  uint32
	LocalAS                 uint32
	PeerType                PeerType
	AuthPassword            string
	Description             string
	NeighborAddress         net.IP
	IfIndex                 int32
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
	BfdNeighborState        string
	PeerGroup               string
	AddPathsRx              bool
	AddPathsMaxTx           uint8
	MaxPrefixes             uint32
	MaxPrefixesThresholdPct uint8
	MaxPrefixesDisconnect   bool
	MaxPrefixesRestartTimer uint8
	TotalPrefixes           uint32
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

type PrefixLimit struct {
	MaxPrefixes          uint32
	ShutdownThresholdPct uint8
	RestartTimer         float64
}

type IPUnicast struct {
	PrefixLimit      PrefixLimit
	SendDefaultRoute bool
}

type IPLabelledUnicast struct {
	PrefixLimit PrefixLimit
}

type L2L3VPN struct {
	PrefixLimit PrefixLimit
}

type UseMultiplePaths struct {
	Enabled             bool
	EBGPAllowMultipleAS bool
	EBGPMaximumPaths    uint32
	IBGPMaximumPaths    uint32
}

type AfiSafiConfig struct {
	AfiSafiName         string
	AfiSafiEnabled      bool
	IPv4Unicast         IPUnicast
	IPv6Unicast         IPUnicast
	IPv4LabelledUnicast IPLabelledUnicast
	IPv6LabelledUnicast IPLabelledUnicast
	L3VPNIPv4Unicast    L2L3VPN
	L3VPNIPv6Unicast    L2L3VPN
	L3VPNIPv4Multicast  L2L3VPN
	L3VPNIPv6Multicast  L2L3VPN
	L2VPNVPLS           L2L3VPN
	L2VPNEVPN           L2L3VPN
	UseMultiplePaths    UseMultiplePaths
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
	AfiSafis        []AfiSafiConfig
}

type PeerGroupConfig struct {
	BaseConfig
	Name string
}

type PeerGroup struct {
	Config   PeerGroupConfig
	State    PeerGroupConfig
	AfiSafis []AfiSafiConfig
}

type BGPAggregate struct {
	IPPrefix
	GenerateASSet   bool
	SendSummaryOnly bool
}

type Bgp struct {
	Global     Global
	PeerGroups map[string]*PeerGroup
	Neighbors  []Neighbor
	BgpAggs    map[string]*BGPAggregate
}

type RouteConfig struct {
	Cost              int32
	IntfType          int32
	Protocol          string
	NextHopIp         string
	NetworkMask       string
	DestinationNw     string
	OutgoingInterface string
}
