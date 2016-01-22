// Dhcp Relay Agent Protocol Handling for Packet Send/Receive
package relayServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"time"
)

type DhcpRelayPcapHandle struct {
	pcapHandle *pcap.Handle
	ifName     string
}

var (
	snapLen     int32         = 65549 // packet capture length
	promisc     bool          = false // mode
	pcapTimeOut time.Duration = 1 * time.Second
)

func DhcpRelayAgentReceiveDhcpPkt(info DhcpRelayAgentGlobalInfo) {
	for {
		//dhcprelayConfigMutex := &sync.Mutex{}
		dhcprelayConfigMutex.RLock()
		if !info.IntfConfig.Enable {
			logger.Info("DRA: relay agent disabled deleting pcap" +
				"handler if any")
			// delete pcap handler and exit out of the go routine
			// @TODO: jgheewala memory leak???
			info.PcapHandler.pcapHandle = nil
			return
		}
		recvHandler := info.PcapHandler.pcapHandle
		dhcprelayConfigMutex.RUnlock()
		src := gopacket.NewPacketSource(recvHandler,
			layers.LayerTypeEthernet)
		in := src.Packets()
		packet, ok := <-in
		if ok {
			logger.Info(fmt.Sprintln("packet is", packet))
		}
	}
}

func DhcpRelayAgentPcapCreate(info DhcpRelayAgentGlobalInfo) {
	logger.Info("DRA: Creating Pcap Handler for intf:" +
		info.IntfConfig.IfIndex)
	info.StateDebugInfo.pcapHandler = "Creating Pcap Handler"
	var filter string = "udp port 67 or udp port 68"
	pcapLocalHandle, err := pcap.OpenLive(info.IntfConfig.IfIndex,
		snapLen, promisc, pcapTimeOut)
	if pcapLocalHandle == nil {
		logger.Err(fmt.Sprintln("DRA: server no device found: ",
			info.IntfConfig.IfIndex, err))
	} else {
		info.StateDebugInfo.pcapHandler = "Setting filter for Pcap " +
			"Handler"
		logger.Info("DRA: setting filter for intf: " +
			info.IntfConfig.IfIndex + " filter: " + filter)
		err = pcapLocalHandle.SetBPFFilter(filter)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Unable to set filter on:",
				info.IntfConfig.IfIndex, err))
		}
		//dhcprelayConfigMutex := &sync.Mutex{}
		dhcprelayConfigMutex.RLock()
		// will the localHandler get destroyed?
		info.PcapHandler.pcapHandle = pcapLocalHandle
		info.PcapHandler.ifName = info.IntfConfig.IfIndex
		info.StateDebugInfo.pcapHandler = "Pcap Handler successfully " +
			"created for intf " + info.IntfConfig.IfIndex
		dhcprelayConfigMutex.RUnlock()
		logger.Info("DRA: Pcap Handler created successfully for intf " +
			info.IntfConfig.IfIndex)
		go DhcpRelayAgentReceiveDhcpPkt(info)
	}
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
	gblEntry := dhcprelayGblInfo[ifName]
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
	gblEntry.PcapHandler = nil

	dhcprelayGblInfo[ifName] = gblEntry

	//}
	//logger.Info("DRA: PCAP Handling Initialized successfully")
}
