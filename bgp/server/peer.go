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
	Neighbor   *config.Neighbor
	fsmManager *FSMManager
	BGPId      uint32
}

func NewPeer(server *BGPServer, globalConf config.GlobalConfig, peerConf config.NeighborConfig) *Peer {
	peer := Peer{
		Server: server,
		logger: server.logger,
		Global: &globalConf,
		Neighbor: &config.Neighbor{
			NeighborAddress: peerConf.NeighborAddress,
			Config: peerConf,
		},
		BGPId:  0,
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
	return p.Neighbor.Config.PeerAS == p.Neighbor.Config.LocalAS
}

func (p *Peer) IsExternal() bool {
	return p.Neighbor.Config.LocalAS != p.Neighbor.Config.PeerAS
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

func (p *Peer) updatePathAttrs(bgpMsg *packet.BGPMessage, path *Path) bool {
	if p.Neighbor.Transport.Config.LocalAddress == nil {
		p.logger.Err(fmt.Sprintf("Can't send Update message, local address not set for peer[%s]", p.Neighbor.NeighborAddress))
		return false
	}

	if p.IsInternal() {
		packet.PrependAS(bgpMsg, 0, false)
		packet.SetNextHop(bgpMsg, p.Neighbor.Transport.Config.LocalAddress)
		packet.SetLocalPref(bgpMsg, path.GetPreference())
	} else {
		packet.PrependAS(bgpMsg, p.Neighbor.Config.LocalAS, true)
		packet.SetNextHop(bgpMsg, p.Neighbor.Transport.Config.LocalAddress)
		packet.RemoveMultiExitDisc(bgpMsg)
		packet.RemoveLocalPref(bgpMsg)
	}

	return true
}

func (p *Peer) PeerConnEstablished(conn *net.Conn) {
	host, _, err := net.SplitHostPort((*conn).LocalAddr().String())
	if err != nil {
		p.logger.Err(fmt.Sprintf("Can't find local address from the peer connection: %s", (*conn).LocalAddr()))
		return
	}
	p.Neighbor.Transport.Config.LocalAddress = net.ParseIP(host)
}

func (p *Peer) PeerConnBroken() {
	p.Neighbor.Transport.Config.LocalAddress = nil
}

func (p *Peer) SendUpdate(bgpMsg packet.BGPMessage, path *Path) {
	p.logger.Info(fmt.Sprintf("Peer: Send update message to peer %s", p.Neighbor.NeighborAddress))
	bgpMsgRef := &bgpMsg
	if p.updatePathAttrs(bgpMsgRef, path) {
		p.fsmManager.SendUpdateMsg(bgpMsgRef)
	}
}
