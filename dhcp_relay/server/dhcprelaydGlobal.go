package relayServer

import (
	"dhcprelayd"
	"github.com/google/gopacket/pcap"
	nanomsg "github.com/op/go-nanomsg"
	"golang.org/x/net/ipv4"
	"log/syslog"
	"net"
	"time"
)

// type is similar to typedef in c
type DhcpOptionCode byte
type OpCode byte
type MessageType byte // Option 53

// Map of DHCP options
type DhcpRelayAgentOptions map[DhcpOptionCode][]byte

// A DHCP packet
type DhcpRelayAgentPacket []byte

/*
 * Global DRA Data Structure:
 *	    IntfConfig Info
 *	    PCAP Handler specific to Interface
 */
type DhcpRelayAgentGlobalInfo struct {
	IntfConfig dhcprelayd.DhcpRelayIntfConfig
	PcapHandle *pcap.Handle
}
type Option struct {
	Code  DhcpOptionCode
	Value []byte
}

type DhcpRelayAgentIntfInfo struct {
	linuxInterface *net.Interface
	logicalId      int
}

/*
 * Global DS local to DHCP RELAY AGENT
 */
type DhcpRelayServiceHandler struct {
}

/*
 * Global Variable
 */
var (
	asicdClient                       AsicdClient
	asicdSubSocket                    *nanomsg.SubSocket
	snapshot_len                      int32         = 1024
	promiscuous                       bool          = false
	timeout                           time.Duration = 30 * time.Second
	dhcprelayLogicalIntfId2LinuxIntId map[int]int   // Linux Intf Id ---> Logical ID
	dhcprelayEnable                   bool
	dhcprelayClientConn               *ipv4.PacketConn
	dhcprelayServerConn               *ipv4.PacketConn
	logger                            *syslog.Writer

	// map key would be if_name
	// When we receive a udp packet... we will get interface id and that can
	// be used to collect the global info...
	dhcprelayGblInfo map[int]DhcpRelayAgentGlobalInfo

	// PadddingToMinimumSize pads a packet so that when sent over UDP,
	// the entire packet, is 300 bytes (which is BOOTP/DHCP min)
	dhcprelayPadder [DHCP_PACKET_MIN_SIZE]byte

	//map for mac_address to interface id for sending unicast packet
	dhcprelayReverseMap map[string]*net.Interface

	// map key would be MACADDR_SERVERIP
	dhcprelayHostServerStateMap   map[string]dhcprelayd.DhcpRelayHostDhcpState
	dhcprelayHostServerStateSlice []string

	// map key is interface id
	dhcprelayIntfStateMap   map[int]dhcprelayd.DhcpRelayIntfState
	dhcprelayIntfStateSlice []int
	// map key is interface id + server
	dhcprelayIntfServerStateMap   map[string]dhcprelayd.DhcpRelayIntfServerState
	dhcprelayIntfServerStateSlice []string
)

// Dhcp OpCodes Types
const (
	Request OpCode = 1 // From Client
	Reply   OpCode = 2 // From Server
)

// DHCP Packet global constants
const DHCP_PACKET_MIN_SIZE = 272
const DHCP_PACKET_HEADER_SIZE = 16
const DHCP_PACKET_MIN_BYTES = 240
const DHCP_SERVER_PORT = 67
const DHCP_CLIENT_PORT = 68
const DHCP_BROADCAST_IP = "255.255.255.255"
const DHCP_NO_IP = "0.0.0.0"

// DHCP Client/Server Message Type 53
const (
	DhcpDiscover MessageType = 1 // From Client - Can I have an IP?
	DhcpOffer    MessageType = 2 // From Server - Here's an IP
	DhcpRequest  MessageType = 3 // From Client - I'll take that IP (Also start for renewals)
	DhcpDecline  MessageType = 4 // From Client - Sorry I can't use that IP
	DhcpACK      MessageType = 5 // From Server, Yes you can have that IP
	DhcpNAK      MessageType = 6 // From Server, No you cannot have that IP
	DhcpRelease  MessageType = 7 // From Client, I don't need that IP anymore
	DhcpInform   MessageType = 8 // From Client, I have this IP and there's nothing you can do about it
)

// DHCP Available Options enum type.... This will cover most of the options type
const (
	End                          DhcpOptionCode = 255
	Pad                          DhcpOptionCode = 0
	OptionSubnetMask             DhcpOptionCode = 1
	OptionTimeOffset             DhcpOptionCode = 2
	OptionRouter                 DhcpOptionCode = 3
	OptionTimeServer             DhcpOptionCode = 4
	OptionNameServer             DhcpOptionCode = 5
	OptionDomainNameServer       DhcpOptionCode = 6
	OptionLogServer              DhcpOptionCode = 7
	OptionCookieServer           DhcpOptionCode = 8
	OptionLPRServer              DhcpOptionCode = 9
	OptionImpressServer          DhcpOptionCode = 10
	OptionResourceLocationServer DhcpOptionCode = 11
	OptionHostName               DhcpOptionCode = 12
	OptionBootFileSize           DhcpOptionCode = 13
	OptionMeritDumpFile          DhcpOptionCode = 14
	OptionDomainName             DhcpOptionCode = 15
	OptionSwapServer             DhcpOptionCode = 16
	OptionRootPath               DhcpOptionCode = 17
	OptionExtensionsPath         DhcpOptionCode = 18

	// IP Layer Parameters per Host
	OptionIPForwardingEnableDisable          DhcpOptionCode = 19
	OptionNonLocalSourceRoutingEnableDisable DhcpOptionCode = 20
	OptionPolicyFilter                       DhcpOptionCode = 21
	OptionMaximumDatagramReassemblySize      DhcpOptionCode = 22
	OptionDefaultIPTimeToLive                DhcpOptionCode = 23
	OptionPathMTUAgingTimeout                DhcpOptionCode = 24
	OptionPathMTUPlateauTable                DhcpOptionCode = 25

	// IP Layer Parameters per Interface
	OptionInterfaceMTU              DhcpOptionCode = 26
	OptionAllSubnetsAreLocal        DhcpOptionCode = 27
	OptionBroadcastAddress          DhcpOptionCode = 28
	OptionPerformMaskDiscovery      DhcpOptionCode = 29
	OptionMaskSupplier              DhcpOptionCode = 30
	OptionPerformRouterDiscovery    DhcpOptionCode = 31
	OptionRouterSolicitationAddress DhcpOptionCode = 32
	OptionStaticRoute               DhcpOptionCode = 33

	// Link Layer Parameters per Interface
	OptionTrailerEncapsulation  DhcpOptionCode = 34
	OptionARPCacheTimeout       DhcpOptionCode = 35
	OptionEthernetEncapsulation DhcpOptionCode = 36

	// TCP Parameters
	OptionTCPDefaultTTL        DhcpOptionCode = 37
	OptionTCPKeepaliveInterval DhcpOptionCode = 38
	OptionTCPKeepaliveGarbage  DhcpOptionCode = 39

	// Application and Service Parameters
	OptionNetworkInformationServiceDomain            DhcpOptionCode = 40
	OptionNetworkInformationServers                  DhcpOptionCode = 41
	OptionNetworkTimeProtocolServers                 DhcpOptionCode = 42
	OptionVendorSpecificInformation                  DhcpOptionCode = 43
	OptionNetBIOSOverTCPIPNameServer                 DhcpOptionCode = 44
	OptionNetBIOSOverTCPIPDatagramDistributionServer DhcpOptionCode = 45
	OptionNetBIOSOverTCPIPNodeType                   DhcpOptionCode = 46
	OptionNetBIOSOverTCPIPScope                      DhcpOptionCode = 47
	OptionXWindowSystemFontServer                    DhcpOptionCode = 48
	OptionXWindowSystemDisplayManager                DhcpOptionCode = 49
	OptionNetworkInformationServicePlusDomain        DhcpOptionCode = 64
	OptionNetworkInformationServicePlusServers       DhcpOptionCode = 65
	OptionMobileIPHomeAgent                          DhcpOptionCode = 68
	OptionSimpleMailTransportProtocol                DhcpOptionCode = 69
	OptionPostOfficeProtocolServer                   DhcpOptionCode = 70
	OptionNetworkNewsTransportProtocol               DhcpOptionCode = 71
	OptionDefaultWorldWideWebServer                  DhcpOptionCode = 72
	OptionDefaultFingerServer                        DhcpOptionCode = 73
	OptionDefaultInternetRelayChatServer             DhcpOptionCode = 74
	OptionStreetTalkServer                           DhcpOptionCode = 75
	OptionStreetTalkDirectoryAssistance              DhcpOptionCode = 76

	OptionRelayAgentInformation DhcpOptionCode = 82

	// DHCP Extensions
	OptionRequestedIPAddress     DhcpOptionCode = 50
	OptionIPAddressLeaseTime     DhcpOptionCode = 51
	OptionOverload               DhcpOptionCode = 52
	OptionDHCPMessageType        DhcpOptionCode = 53
	OptionServerIdentifier       DhcpOptionCode = 54
	OptionParameterRequestList   DhcpOptionCode = 55
	OptionMessage                DhcpOptionCode = 56
	OptionMaximumDHCPMessageSize DhcpOptionCode = 57
	OptionRenewalTimeValue       DhcpOptionCode = 58
	OptionRebindingTimeValue     DhcpOptionCode = 59
	OptionVendorClassIdentifier  DhcpOptionCode = 60
	OptionClientIdentifier       DhcpOptionCode = 61

	OptionTFTPServerName DhcpOptionCode = 66
	OptionBootFileName   DhcpOptionCode = 67

	OptionUserClass DhcpOptionCode = 77

	OptionClientArchitecture DhcpOptionCode = 93

	OptionTZPOSIXString    DhcpOptionCode = 100
	OptionTZDatabaseString DhcpOptionCode = 101

	OptionClasslessRouteFormat DhcpOptionCode = 121
)
