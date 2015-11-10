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

type BgpHandler struct {
	PeerCommandCh chan PeerConfigCommands
	server        *server.BgpServer
	logger        *syslog.Writer
}

func NewBgpHandler(server *server.BgpServer, logger *syslog.Writer) *BgpHandler {
	h := new(BgpHandler)
	h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
	h.logger = logger
	return h
}

func (h *BgpHandler) CreateBgpGlobal(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bgpGlobal))
	gConf := config.GlobalConfig{AS: uint32(bgpGlobal.AS)}
	h.server.GlobalConfigCh <- gConf
	return true, nil
}

func (h *BgpHandler) UpdateBgpGlobal(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BgpHandler) DeleteBgpGlobal(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BgpHandler) CreateBgpNeighbor(bgpNeighbor *bgpd.BgpNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create peer attrs:", bgpNeighbor))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreatePeer - IP is not valid:", bgpNeighbor.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS:          uint32(bgpNeighbor.PeerAS),
		LocalAS:         uint32(bgpNeighbor.LocalAS),
		Description:     bgpNeighbor.Description,
		NeighborAddress: ip,
	}
	h.server.AddPeerCh <- pConf
	return true, nil
}

func (h *BgpHandler) UpdateBgpNeighbor(bgpNeighbor *bgpd.BgpNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update peer attrs:", bgpNeighbor))
	return true, nil
}

func (h *BgpHandler) DeleteBgpNeighbor(bgpNeighbor *bgpd.BgpNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete peer attrs:", bgpNeighbor))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreatePeer - IP is not valid:", bgpNeighbor.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS:          uint32(bgpNeighbor.PeerAS),
		LocalAS:         uint32(bgpNeighbor.LocalAS),
		Description:     bgpNeighbor.Description,
		NeighborAddress: ip,
	}
	h.server.RemPeerCh <- pConf
	return true, nil
}

func (h *BgpHandler) PeerCommand(in *PeerConfigCommands, out *bool) error {
	h.PeerCommandCh <- *in
	h.logger.Info(fmt.Sprintln("Good peer command:", in))
	*out = true
	return nil
}
