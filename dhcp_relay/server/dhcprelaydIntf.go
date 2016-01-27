// Dhcp Relay Agent Interface Handling
package relayServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
)

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
	portInfoMap map[string]int
	asicdClient AsicdClient
)

/*
 * DhcpRelayInitPortParams:
 *	    API to handle initialization of port parameter
 */
func DhcpRelayInitPortParams() error {
	logger.Info("DRA: initializing Port Parameters & Global Init")
	// constructing port configs...
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	hack := false // dra hack for running the code on localhost
	more := false
	objCount := 0
	portNum := 0
	if !asicdClient.IsConnected {
		logger.Info("DRA: is not connected to asicd.... is it bad?")
	}
	logger.Info("DRA calling asicd for port config")
	count := 10
	// for optimization initializing 25 interfaces map...
	//dhcprelayGblInfo = make(map[string]DhcpRelayAgentGlobalInfo, 25)
	dhcprelayGblInfo = make(map[int]DhcpRelayAgentGlobalInfo, 25)
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkPortConfig(
			int64(currMarker), int64(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk port config"+
				" from asicd failed with reason", err))
			//return err <--- DRA doesn't start as no bulk port
			//
			logger.Info("DRA: HACK For interface is invoked")
			hack = true
			//return nil
		}
		if hack == true {
			objCount = 1
			portNum = 1
		} else {
			objCount = int(bulkInfo.ObjCount)
			more = bool(bulkInfo.More)
			currMarker = int64(bulkInfo.NextMarker)
		}
		for i := 0; i < objCount; i++ {
			//var entry portInfo
			var ifName string
			if hack == true {
				portNum = 1
				ifName = "wlp2s0" //"enp1s0f0"
			} else {
				portNum = int(bulkInfo.PortConfigList[i].IfIndex)
				ifName = bulkInfo.PortConfigList[i].Name
			}
			portInfoMap[ifName] = portNum
			// Init DRA Global Handling for all interfaces....
			DhcpRelayAgentInitGblHandling(ifName, portNum)
		}
		if hack {
			logger.Info("DRA: HACK and hence creating clien/server right away")
			DhcpRelayAgentCreateClientServerConn()
		}
		if more == false {
			break
		}
	}
	logger.Info("DRA: initialized Port Parameters & Global Info successfully")
	return nil
}
