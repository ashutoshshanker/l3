// server.go
package rpc

import (
    "fmt"
	"l3/bgp/config"
	"l3/bgp/server"
	"generated/src/bgpd"
    "net"
)

type PeerConfigCommands struct {
    IP net.IP
    Command int
}

type BgpHandler struct {
    GlobalConfigCh chan GlobalConfigAttrs
    AddPeerConfigCh chan PeerConfigAttrs
    RemPeerConfigCh chan PeerConfigAttrs
    PeerCommandCh chan PeerConfigCommands
	server *server.BgpServer
}

func NewBgpHandler(server *server.BgpServer) *BgpHandler {
    h := new(BgpHandler)
    h.GlobalConfigCh = make(chan GlobalConfigAttrs)
    h.AddPeerConfigCh = make(chan PeerConfigAttrs)
    h.RemPeerConfigCh = make(chan PeerConfigAttrs)
    h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
    return h
}

func (h *BgpHandler) CreateBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
    fmt.Println("Create global config attrs:", bgpGlobal)
	gConf := config.GlobalConfig{AS: uint32(bgpGlobal.AS)}
    h.server.GlobalConfigCh <- gConf
    return true, nil
}

func (h *BgpHandler) UpdateBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
    fmt.Println("Update global config attrs:", bgpGlobal)
	return true, nil
}

func (h *BgpHandler) DeleteBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
    fmt.Println("Delete global config attrs:", bgpGlobal)
	return true, nil
}

func (h *BgpHandler) CreatePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    fmt.Println("Create peer attrs:", peerConfig)
	ip := net.ParseIP(peerConfig.NeighborAddress)
	if ip == nil {
		fmt.Println("CreatePeer - IP is not valid:", peerConfig.NeighborAddress)
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
    fmt.Println("Update peer attrs:", peerConfig)
    return true, nil
}

func (h *BgpHandler) DeletePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    fmt.Println("Delete peer attrs:", peerConfig)
	ip := net.ParseIP(peerConfig.NeighborAddress)
	if ip == nil {
		fmt.Println("CreatePeer - IP is not valid:", peerConfig.NeighborAddress)
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
    fmt.Println("Good peer command:", in)
    *out = true
    return nil
}
