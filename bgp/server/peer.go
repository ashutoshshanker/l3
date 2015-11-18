// peer.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"time"
)

type Peer struct {
	Server     *BGPServer
	logger     *syslog.Writer
	Global     *config.GlobalConfig
	Peer       *config.NeighborConfig
	fsmManager *FSMManager
	BGPId      uint32
	adjRibIn   map[string]*Path
}

func NewPeer(server *BGPServer, globalConf config.GlobalConfig, peerConf config.NeighborConfig) *Peer {
	peer := Peer{
		Server: server,
		logger: server.logger,
		Global: &globalConf,
		Peer:   &peerConf,
		BGPId:  0,
		adjRibIn: make(map[string]*Path),
	}
	peer.fsmManager = NewFSMManager(&peer, &globalConf, &peerConf)
	return &peer
}

func (p *Peer) Init() {
	go p.fsmManager.Init()
}

func (p *Peer) Cleanup() {}

func (p *Peer) AcceptConn(conn *net.TCPConn) {
	p.fsmManager.acceptCh <- conn
}

func (p *Peer) Command(command int) {
	p.fsmManager.commandCh <- command
}

func (p *Peer) IsInternal() bool {
	return p.Peer.PeerAS == p.Peer.LocalAS
}

func (p *Peer) IsExternal() bool {
	return p.Peer.LocalAS != p.Peer.PeerAS
}

func (p *Peer) SendKeepAlives(conn *net.TCPConn) {
	bgpKeepAliveMsg := packet.NewBGPKeepAliveMessage()
	var num int
	var err error

	for {
		select {
		case <-time.After(time.Second * 1):
			p.logger.Info(fmt.Sprintln("send the packet ..."))
			packet, _ := bgpKeepAliveMsg.Encode()
			num, err = conn.Write(packet)
			if err != nil {
				p.logger.Info(fmt.Sprintln("Conn.Write failed with error:", err))
			}
			p.logger.Info(fmt.Sprintln("Conn.Write succeeded. sent %d", num, "bytes"))
		}
	}
}

func (p *Peer) SetBGPId(bgpId uint32) {
	p.BGPId = bgpId
}
