// Dhcp Relay Agent Protocol Handling for Packet Send/Receive
package relayServer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"sync"
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
	logger.Info("DRA: Creating Pcap Handler for intf: " +
		info.IntfConfig.IfIndex)
	DhcpRelayAgentUpdateStats("Creating Pcap Handler", info)
	var filter string = "udp port 67 or udp port 68"
	pcapLocalHandle, err := pcap.OpenLive(info.IntfConfig.IfIndex,
		snapLen, promisc, pcapTimeOut)
	if pcapLocalHandle == nil {
		logger.Err(fmt.Sprintln("DRA: server no device found: ",
			info.IntfConfig.IfIndex, err))
		return
	}
	DhcpRelayAgentUpdateStats("Setting filter for Pcap Handler",
		info)
	logger.Info("DRA: setting filter for intf: " +
		info.IntfConfig.IfIndex + " filter: " + filter)
	err = pcapLocalHandle.SetBPFFilter(filter)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Unable to set filter on:",
			info.IntfConfig.IfIndex, err))
	}
	info.dhcprelayConfigMutex.RLock()
	info.PcapHandler.pcapHandle = pcapLocalHandle
	info.PcapHandler.ifName = info.IntfConfig.IfIndex
	dhcprelayGblInfo[info.IntfConfig.IfIndex] = info
	info.dhcprelayConfigMutex.RUnlock()

	logger.Info("DRA: Pcap Handler successfully updated for intf " +
		info.IntfConfig.IfIndex)

	DhcpRelayAgentUpdateStats("Pcap Handler Successfully Created", info)

	info.dhcprelayConfigMutex.RLock()
	if !info.IntfConfig.Enable || info.PcapHandler.pcapHandle == nil {
		logger.Info("DRA: relay agent disabled deleting pcap" +
			"handler if any")
		// delete pcap handler and exit out of the go routine
		// @TODO: jgheewala memory leak???
		info.PcapHandler.pcapHandle = nil
		dhcprelayGblInfo[info.IntfConfig.IfIndex] = info
		info.dhcprelayConfigMutex.RUnlock()
		return
	}
	recvHandler := info.PcapHandler.pcapHandle
	logger.Info("DRA: opening new packet source for ifName " +
		info.IntfConfig.IfIndex)
	src := gopacket.NewPacketSource(recvHandler,
		layers.LayerTypeEthernet)
	info.inputPacket = src.Packets()
	dhcprelayGblInfo[info.IntfConfig.IfIndex] = info
	info.dhcprelayConfigMutex.RUnlock()

	// Receive packets infintely or unless channel is closed
	for {
		packet, ok := <-info.inputPacket
		if ok {
			logger.Info(fmt.Sprintln("DRA: packet is", packet))
		}
	}
}

func DhcpRelayAgentInitGblHandling(ifName string, ifNum int) {
	logger.Info("DRA: Initializaing Global Info for " + ifName + " " +
		string(ifNum))
	// Created a global Entry for Interface
	gblEntry := dhcprelayGblInfo[ifName]
	// Setting up default values for globalEntry
	gblEntry.IntfConfig.IpSubnet = ""
	gblEntry.IntfConfig.Netmask = ""
	gblEntry.IntfConfig.IfIndex = ifName
	gblEntry.IntfConfig.AgentSubType = 0
	gblEntry.IntfConfig.Enable = false
	gblEntry.dhcprelayConfigMutex = sync.RWMutex{}
	// Stats information
	gblEntry.StateDebugInfo.stats = make([]string, 150)
	DhcpRelayAgentUpdateStats(ifName, gblEntry)
	DhcpRelayAgentUpdateStats("Global Init Done", gblEntry)

	dhcprelayGblInfo[ifName] = gblEntry

}
