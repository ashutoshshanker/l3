// Dhcp Relay Agent Protocol Handling for Packet Send/Receive
package relayServer

import (
	_ "asicd/asicdConstDefs"
	_ "asicdServices"
	_ "dhcprelayd"
	_ "encoding/json"
	_ "flag"
	_ "fmt"
	_ "git.apache.org/thrift.git/lib/go/thrift"
	"github.com/google/gopacket/pcap"
	_ "io/ioutil"
	_ "log/syslog"
	_ "os"
	_ "os/signal"
	_ "strconv"
	_ "syscall"
	_ "utils/ipcutils"
)

type DhcpRelayPcapHandle struct {
	pcap_handle *pcap.Handle
	ifName      string
}

func DhcpRelayAgentHandleProtocol(ifName string) {

}

/*
*
type DhcpRelayAgentGlobalInfo struct {
	IntfConfig dhcprelayd.DhcpRelayIntfConfig
	PcapHandle DhcpRelayPcapHandle
	StateInfo  DhcpRelayAgentStateInfo
}
	dhcprelayGblInfo map[string]DhcpRelayAgentGlobalInfo
*/

func DhcpRelayAgentInitGblHandling(ifName string, ifNum int) {
	//logger.Info("DRA: Initializaing PCAP Handling")
	//dhcprelayGblInfo = make(map[int]DhcpRelayAgentGlobalInfo)
	//for ifNum, portInfo := range portInfoMap {
	//ifName := portInfo.Name
	// Created a global Entry for Interface
	gblEntry := dhcprelayGblInfo[ifNum]
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IpSubnet = ""
	gblEntry.IntfConfig.Netmask = ""
	gblEntry.IntfConfig.IfIndex = ifName
	gblEntry.IntfConfig.AgentSubType = 0
	gblEntry.IntfConfig.Enable = false

	//gblEntry.enable = make(chan bool)
	// Mark Channel as disabled...only when enabled spawn a pcap
	// handler
	//gblEntry.enable <- false

	// Stats information
	gblEntry.StateDebugInfo.initDone = "init done"

	dhcprelayGblInfo[ifNum] = gblEntry

	//}
	//logger.Info("DRA: PCAP Handling Initialized successfully")
}
