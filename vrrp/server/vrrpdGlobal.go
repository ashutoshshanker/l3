package vrrpServer

import (
	"asicdServices"
	"database/sql"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	nanomsg "github.com/op/go-nanomsg"
	"net"
	"sync"
	"time"
	"utils/logging"
	"vrrpd"
)

/*
	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                    IPv4 Fields or IPv6 Fields                 |
	...                                                             ...
	|                                                               |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|Version| Type  | Virtual Rtr ID|   Priority    |Count IPvX Addr|
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|(rsvd) |     Max Adver Int     |          Checksum             |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                                                               |
	+                                                               +
	|                       IPvX Address(es)                        |
	+                                                               +
	+                                                               +
	+                                                               +
	+                                                               +
	|                                                               |
	+                                                               +
	|                                                               |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

type VrrpPktHeader struct {
	Version       uint8
	Type          uint8
	VirtualRtrId  uint8
	Priority      uint8
	CountIPv4Addr uint8
	Rsvd          uint8
	MaxAdverInt   uint16
	CheckSum      uint16
	IPv4Addr      []net.IP
}

type VrrpFsm struct {
	key      string
	vrrpHdr  *VrrpPktHeader
	vrrpInFo *VrrpGlobalInfo
	inPkt    gopacket.Packet
}

type VrrpClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type VrrpClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type VrrpAsicdClient struct {
	VrrpClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type VrrpGlobalInfo struct {
	IntfConfig vrrpd.VrrpIntfConfig
	// The initial value is the same as Advertisement_Interval.
	MasterAdverInterval int32
	// (((256 - priority) * Master_Adver_Interval) / 256)
	SkewTime int32
	// (3 * Master_Adver_Interval) + Skew_time
	MasterDownValue int32
	MasterDownTimer *time.Timer
	// Advertisement Timer
	AdverTimer int32
	// IfIndex IpAddr which needs to be used if no Virtual Ip is specified
	IpAddr string
	// cached info for IfName is required in future
	IfName string
	// Pcap Handler for receiving packets
	pHandle *pcap.Handle
	// Pcap Handler lock to write data one routine at a time
	PcapHdlLock *sync.RWMutex
	// State Name
	StateName string
	// Lock to read current state of vrrp object
	StateLock *sync.RWMutex
}

type VrrpPktChannelInfo struct {
	pkt     gopacket.Packet
	key     string
	IfIndex int32
}

type VrrpServer struct {
	logger                        *logging.Writer
	vrrpDbHdl                     *sql.DB
	paramsDir                     string
	asicdClient                   VrrpAsicdClient
	asicdSubSocket                *nanomsg.SubSocket
	vrrpGblInfo                   map[string]VrrpGlobalInfo // IfIndex + VRID
	vrrpIntfStateSlice            []string
	vrrpLinuxIfIndex2AsicdIfIndex map[int32]*net.Interface
	vrrpIfIndexIpAddr             map[int32]string
	vrrpVlanId2Name               map[int]string
	VrrpIntfConfigCh              chan vrrpd.VrrpIntfConfig //VrrpGlobalInfo
	vrrpRxPktCh                   chan VrrpPktChannelInfo
	vrrpTxPktCh                   chan string //VrrpPktChannelInfo
	vrrpFsmCh                     chan VrrpFsm
	vrrpRxChStarted               bool
	vrrpTxChStarted               bool
	vrrpMacConfigAdded            bool
	vrrpSnapshotLen               int32
	vrrpPromiscuous               bool
	vrrpTimeout                   time.Duration
}

const (
	// Error Message
	VRRP_USR_CONF_DB                    = "/UsrConfDb.db"
	VRRP_INVALID_VRID                   = "VRID is invalid"
	VRRP_CLIENT_CONNECTION_NOT_REQUIRED = "Connection to Client is not required"
	VRRP_INCORRECT_VERSION              = "Version is not correct for received VRRP Packet"
	VRRP_INCORRECT_FIELDS               = "Field like type/count ip addr/Advertisement Interval are not valid"
	VRRP_SAME_OWNER                     = "Local Router should not be same as the VRRP Ip Address"
	VRRP_MISSING_VRID_CONFIG            = "VRID is not configured on interface"
	VRRP_CHECKSUM_ERR                   = "VRRP checksum failure"
	VRRP_INVALID_PCAP                   = "Invalid Pcap Handler"

	// VRRP multicast ip address for join
	VRRP_GROUP_IP     = "224.0.0.18"
	VRRP_BPF_FILTER   = "ip host " + VRRP_GROUP_IP
	VRRP_PROTOCOL_MAC = "01:00:5e:00:00:12"
	VRRP_MAC_MASK     = "ff:ff:ff:ff:ff:ff"
	VRRP_PROTO_ID     = 112

	// Default Size
	VRRP_GLOBAL_INFO_DEFAULT_SIZE         = 50
	VRRP_VLAN_MAPPING_DEFAULT_SIZE        = 5
	VRRP_INTF_STATE_SLICE_DEFAULT_SIZE    = 5
	VRRP_LINUX_INTF_MAPPING_DEFAULT_SIZE  = 5
	VRRP_INTF_IPADDR_MAPPING_DEFAULT_SIZE = 5
	VRRP_RX_BUF_CHANNEL_SIZE              = 100
	VRRP_TX_BUF_CHANNEL_SIZE              = 1
	VRRP_FSM_CHANNEL_SIZE                 = 1
	VRRP_INTF_CONFIG_CH_SIZE              = 1

	// ip/vrrp header Check Defines
	VRRP_TTL                        = 255
	VRRP_VERSION2                   = 2
	VRRP_VERSION3                   = 3
	VRRP_PKT_TYPE_ADVERTISEMENT     = 1 // Only one type is supported which is advertisement
	VRRP_RSVD                       = 0
	VRRP_HDR_CREATE_CHECKSUM        = 0
	VRRP_HEADER_SIZE_EXCLUDING_IPVX = 8 // 8 bytes...
	VRRP_IPV4_HEADER_MIN_SIZE       = 20
	VRRP_HEADER_MIN_SIZE            = 20
	VRRP_MASTER_PRIORITY            = 255

	// vrrp default configs
	VRRP_DEFAULT_PRIORITY = 100
	VRRP_IEEE_MAC_ADDR    = "00-00-5E-00-01-"

	// vrrp state names
	VRRP_INITIALIZE_STATE = "Initialize"
	VRRP_BACKUP_STATE     = "Backup"
	VRRP_MASTER_STATE     = "Master"
)
