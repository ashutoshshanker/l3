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

func DhcpRelayAgentInitPcapHandling() error {
	logger.Info("DRA: Initializaing PCAP Handling")
	logger.Info("DRA: PCAP Handling Initialized successfully")
	return nil
}
