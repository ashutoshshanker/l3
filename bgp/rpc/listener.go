// server.go
package rpc

import (
    "fmt"
	"l3/bgp/config"
	"l3/bgp/server"
	"log/syslog"
	"generated/src/bgpd"
    "net"
)

type PeerConfigCommands struct {
    IP net.IP
    Command int
}

type BgpHandler struct {
    PeerCommandCh chan PeerConfigCommands
	server *server.BgpServer
	logger *syslog.Writer
}

func NewBgpHandler(server *server.BgpServer, logger *syslog.Writer) *BgpHandler {
    h := new(BgpHandler)
    h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
	h.logger = logger
    return h
}

func (h *BgpHandler) CreateBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bgpGlobal))
	gConf := config.GlobalConfig{AS: uint32(bgpGlobal.AS)}
	h.server.GlobalConfigCh <- gConf
	return true, nil
}

func (h *BgpHandler) UpdateBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
    h.logger.Info(fmt.Sprintln("Update global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BgpHandler) DeleteBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BgpHandler) CreatePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    h.logger.Info(fmt.Sprintln("Create peer attrs:", peerConfig))
	ip := net.ParseIP(peerConfig.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreatePeer - IP is not valid:", peerConfig.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS: uint32(peerConfig.PeerAS),
		LocalAS: uint32(peerConfig.LocalAS),
		Description: peerConfig.Description,
		NeighborAddress: ip,
	}
    h.server.AddPeerCh <- pConf
    return true, nil
}

func (h *BgpHandler) UpdatePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    h.logger.Info(fmt.Sprintln("Update peer attrs:", peerConfig))
    return true, nil
}

func (h *BgpHandler) DeletePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    h.logger.Info(fmt.Sprintln("Delete peer attrs:", peerConfig))
	ip := net.ParseIP(peerConfig.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreatePeer - IP is not valid:", peerConfig.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS: uint32(peerConfig.PeerAS),
		LocalAS: uint32(peerConfig.LocalAS),
		Description: peerConfig.Description,
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
