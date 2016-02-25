package vrrpServer

import (
	"asicdServices"
	"database/sql"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"golang.org/x/net/ipv4"
	"log/syslog"
	"net"
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

type VrrpServiceHandler struct {
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
	MasterDownInterval int32
	// IfIndex IpAddr which needs to be used if no Virtual Ip is specified
	IpAddr string
	// cached info for IfName is required in future
	IfName string
}

var (
	logger                        *syslog.Writer
	vrrpDbHdl                     *sql.DB
	paramsDir                     string
	asicdClient                   VrrpAsicdClient
	asicdSubSocket                *nanomsg.SubSocket
	vrrpGblInfo                   map[string]VrrpGlobalInfo // IfIndex + VRID
	vrrpIntfStateSlice            []string
	vrrpNetPktConn                net.PacketConn
	vrrpListener                  *ipv4.PacketConn
	vrrpLinuxIfIndex2AsicdIfIndex map[int]int32
	vrrpIfIndexIpAddr             map[int32]string
	vrrpVlanId2Name               map[int]string
)

const (
	// Error Message
	VRRP_USR_CONF_DB                    = "/UsrConfDb.db"
	VRRP_INVALID_VRID                   = "VRID is invalid"
	VRRP_CLIENT_CONNECTION_NOT_REQUIRED = "Connection to Client is not required"
	// Control Message
	vrrpCtrlFlag = ipv4.FlagTTL | ipv4.FlagSrc |
		ipv4.FlagDst | ipv4.FlagInterface
	VRRP_GROUP_IP = "224.0.0.18"
)
