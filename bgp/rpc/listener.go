// server.go
package rpc

import (
    "fmt"
	"generated/src/bgpd"
    "net"
)

type GlobalConfigAttrs struct {
    AS int32
}

type PeerConfigAttrs struct {
    IP net.IP
    AS int32
}

type PeerConfigCommands struct {
    IP net.IP
    Command int
}

type BgpHandler struct {
    GlobalConfigCh chan GlobalConfigAttrs
    AddPeerConfigCh chan PeerConfigAttrs
    RemPeerConfigCh chan PeerConfigAttrs
    PeerCommandCh chan PeerConfigCommands
}

func NewBgpHandler() *BgpHandler {
    h := new(BgpHandler)
    h.GlobalConfigCh = make(chan GlobalConfigAttrs)
    h.AddPeerConfigCh = make(chan PeerConfigAttrs)
    h.RemPeerConfigCh = make(chan PeerConfigAttrs)
    h.PeerCommandCh = make(chan PeerConfigCommands)
    return h
}

func (h *BgpHandler) CreateBgp(bgpGlobal *bgpd.BgpGlobal) (bool, error) {
    fmt.Println("Create global config attrs:", bgpGlobal)
	gConf := GlobalConfigAttrs{AS: bgpGlobal.AS}
    h.GlobalConfigCh <- gConf
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
	ip := net.ParseIP(peerConfig.IP)
	if ip == nil {
		fmt.Println("CreatePeer - IP is not valid:", peerConfig.IP)
	}
	pConf := PeerConfigAttrs{AS: peerConfig.AS, IP: ip}
    h.AddPeerConfigCh <- pConf
    return true, nil
}

func (h *BgpHandler) UpdatePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    fmt.Println("Update peer attrs:", peerConfig)
    return true, nil
}

func (h *BgpHandler) DeletePeer(peerConfig *bgpd.BgpPeer) (bool, error) {
    fmt.Println("Delete peer attrs:", peerConfig)
    return true, nil
}

func (h *BgpHandler) PeerCommand(in *PeerConfigCommands, out *bool) error {
    h.PeerCommandCh <- *in
    fmt.Println("Good peer command:", in)
    *out = true
    return nil
}
