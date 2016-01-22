// Dhcp Relay Agent Interface Handling
package relayServer

import (
	"asicd/asicdConstDefs"
	_ "asicd/asicdConstDefs"
	"asicdServices"
	_ "dhcprelayd"
	_ "encoding/json"
	_ "flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	_ "io/ioutil"
	_ "log/syslog"
	_ "os"
	_ "os/signal"
	_ "strconv"
	_ "syscall"
	_ "utils/ipcutils"
)

type portInfo struct {
	Name string // Port Name used for configuration
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type DHCPRELAYClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	DHCPRELAYClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

/*
 * Global Variable
 */
var (
	portInfoMap map[int]portInfo // PORT NAME
	asicdClient AsicdClient
)

/*
 * DhcpRelayInitPortParams:
 *	    API to handle initialization of port parameter
 */
func DhcpRelayInitPortParams() error {
	logger.Info("DRA initializing Port Parameters & Global Init")
	// constructing port configs...
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	if !asicdClient.IsConnected {
		logger.Info("DRA is not connected to asicd.... is it bad?")
	}
	logger.Info("DRA calling asicd for port config")
	count := 10
	// for optimization initializing 25 interfaces map...
	dhcprelayGblInfo = make(map[string]DhcpRelayAgentGlobalInfo, 25)
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkPortConfig(
			int64(currMarker), int64(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA getting bulk port config"+
				" from asicd failed with reason", err))
			//return err <--- DRA doesn't start as no bulk port
			//		  config
			return nil
		}
		objCount := int(bulkInfo.ObjCount)
		more := bool(bulkInfo.More)
		currMarker = int64(bulkInfo.NextMarker)
		for i := 0; i < objCount; i++ {
			portNum := int(bulkInfo.PortConfigList[i].IfIndex)
			entry := portInfoMap[portNum]
			entry.Name = bulkInfo.PortConfigList[i].Name
			portInfoMap[portNum] = entry
			// Init DRA Global Handling for all interfaces....
			DhcpRelayAgentInitGblHandling(entry.Name, portNum)
		}
		if more == false {
			return nil
		}
	}
	logger.Info("DRA initialized Port Parameters & Global Info successfully")
	return nil
}

func dhcpRelayAgentProcessAsicdNotification(rxBuf []byte) {

}
