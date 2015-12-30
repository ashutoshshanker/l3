// server.go
package rpc

import (
	"bgpd"
	"fmt"
	"l3/bgp/config"
	"l3/bgp/server"
	"log/syslog"
	"net"
)

type PeerConfigCommands struct {
	IP      net.IP
	Command int
}

type BGPHandler struct {
	PeerCommandCh chan PeerConfigCommands
	server        *server.BGPServer
	logger        *syslog.Writer
}

func NewBGPHandler(server *server.BGPServer, logger *syslog.Writer) *BGPHandler {
	h := new(BGPHandler)
	h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
	h.logger = logger
	return h
}

func (h *BGPHandler) CreateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bgpGlobal))
	if bgpGlobal.RouterId == "localhost" {
		bgpGlobal.RouterId = "127.0.0.1"
	}

	ip := net.ParseIP(bgpGlobal.RouterId)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreateBGPGlobal - IP is not valid:", bgpGlobal.RouterId))
		return false, nil
	}

	gConf := config.GlobalConfig{
		AS:       uint32(bgpGlobal.AS),
		RouterId: ip,
	}
	h.server.GlobalConfigCh <- gConf
	return true, nil
}

func (h *BGPHandler) GetBGPGlobal() (*bgpd.BGPGlobalState, error) {
	bgpGlobal := h.server.GetBGPGlobalState()
	bgpGlobalResponse := bgpd.NewBGPGlobalState()
	bgpGlobalResponse.AS = int32(bgpGlobal.AS)
	bgpGlobalResponse.RouterId = bgpGlobal.RouterId.String()
	bgpGlobalResponse.TotalPaths = int32(bgpGlobal.TotalPaths)
	bgpGlobalResponse.TotalPrefixes = int32(bgpGlobal.TotalPrefixes)
	return bgpGlobalResponse, nil
}

func (h *BGPHandler) UpdateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) DeleteBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) CreateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create BGP neighbor attrs:", bgpNeighbor))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintf("Can't create BGP neighbor - IP[%s] not valid", bgpNeighbor.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS:          uint32(bgpNeighbor.PeerAS),
		LocalAS:         uint32(bgpNeighbor.LocalAS),
		AuthPassword:    bgpNeighbor.AuthPassword,
		Description:     bgpNeighbor.Description,
		NeighborAddress: ip,
		RouteReflectorClusterId: uint32(bgpNeighbor.RouteReflectorClusterId),
		RouteReflectorClient: bgpNeighbor.RouteReflectorClient,
	}
	h.server.AddPeerCh <- pConf
	return true, nil
}

func (h *BGPHandler) convertToThriftNeighbor(neighborState *config.NeighborState) *bgpd.BGPNeighborState {
	bgpNeighborResponse := bgpd.NewBGPNeighborState()
	bgpNeighborResponse.PeerAS = int32(neighborState.PeerAS)
	bgpNeighborResponse.LocalAS = int32(neighborState.LocalAS)
	bgpNeighborResponse.AuthPassword = neighborState.AuthPassword
	bgpNeighborResponse.PeerType = bgpd.PeerType(neighborState.PeerType)
	bgpNeighborResponse.Description = neighborState.Description
	bgpNeighborResponse.NeighborAddress = neighborState.NeighborAddress.String()
	bgpNeighborResponse.SessionState = int32(neighborState.SessionState)

	received := bgpd.NewBgpCounters()
	received.Notification = int64(neighborState.Messages.Received.Notification)
	received.Update = int64(neighborState.Messages.Received.Update)
	sent := bgpd.NewBgpCounters()
	sent.Notification = int64(neighborState.Messages.Sent.Notification)
	sent.Update = int64(neighborState.Messages.Sent.Update)
	messages := bgpd.NewBGPMessages()
	messages.Received = received
	messages.Sent = sent
	bgpNeighborResponse.Messages = messages

	queues := bgpd.NewBGPQueues()
	queues.Input = int32(neighborState.Queues.Input)
	queues.Output = int32(neighborState.Queues.Output)
	bgpNeighborResponse.Queues = queues

	return bgpNeighborResponse
}

func (h *BGPHandler) GetBGPNeighbor(neighborAddress string) (*bgpd.BGPNeighborState, error) {
	bgpNeighborState := h.server.GetBGPNeighborState(neighborAddress)
	bgpNeighborResponse := h.convertToThriftNeighbor(bgpNeighborState)
	return bgpNeighborResponse, nil
}

func (h *BGPHandler) BulkGetBGPNeighbors(index int64, count int64) (*bgpd.BGPNeighborStateBulk, error) {
	nextIdx, currCount, bgpNeighbors := h.server.BulkGetBGPNeighbors(int(index), int(count))
	bgpNeighborsResponse := make([]*bgpd.BGPNeighborState, len(bgpNeighbors))
	for idx, item := range bgpNeighbors {
		bgpNeighborsResponse[idx] = h.convertToThriftNeighbor(item)
	}

	bgpNeighborStateBulk := bgpd.NewBGPNeighborStateBulk()
	bgpNeighborStateBulk.NextIndex = int64(nextIdx)
	bgpNeighborStateBulk.Count = int64(currCount)
	bgpNeighborStateBulk.More = (nextIdx != 0)
	bgpNeighborStateBulk.StateList = bgpNeighborsResponse

	return bgpNeighborStateBulk, nil
}

func (h *BGPHandler) UpdateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update peer attrs:", bgpNeighbor))
	return true, nil
}

func (h *BGPHandler) DeleteBGPNeighbor(neighborAddress string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete BGP neighbor:", neighborAddress))
	ip := net.ParseIP(neighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintf("Can't delete BGP neighbor - IP[%s] not valid", neighborAddress))
		return false, nil
	}
	h.server.RemPeerCh <- neighborAddress
	return true, nil
}

func (h *BGPHandler) PeerCommand(in *PeerConfigCommands, out *bool) error {
	h.PeerCommandCh <- *in
	h.logger.Info(fmt.Sprintln("Good peer command:", in))
	*out = true
	return nil
}
