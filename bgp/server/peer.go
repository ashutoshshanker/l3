// peer.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"time"
	"sync/atomic"
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
			Config:          peerConf,
		},
		BGPId: 0,
	}

	peer.Neighbor.State = config.NeighborState{
		PeerAS: peerConf.PeerAS,
		LocalAS: peerConf.LocalAS,
		AuthPassword: peerConf.AuthPassword,
		Description: peerConf.Description,
		NeighborAddress: peerConf.NeighborAddress,
	}

	if peerConf.LocalAS == peerConf.PeerAS {
		peer.Neighbor.State.PeerType = config.PeerTypeInternal
	} else {
		peer.Neighbor.State.PeerType = config.PeerTypeExternal
	}

	peer.fsmManager = NewFSMManager(&peer, &globalConf, &peerConf)
	return &peer
}

func (p *Peer) Init() {
	go p.fsmManager.Init()
}

func (p *Peer) Cleanup() {
	p.fsmManager.closeCh <- true
}

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
	if bgpMsg == nil || bgpMsg.Body.(*packet.BGPUpdate).PathAttributes == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Path attrs not found in BGP Update message", p.Neighbor.NeighborAddress))
		return true
	}

	if p.Neighbor.Transport.Config.LocalAddress == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Can't send Update message, FSM is not in Established state",
			p.Neighbor.NeighborAddress))
		return false
	}

	if p.IsInternal() {
		packet.SetNextHop(bgpMsg, p.Neighbor.Transport.Config.LocalAddress)
		packet.SetLocalPref(bgpMsg, path.GetPreference())
	} else {
		// Do change these path attrs for local routes
		if path.peer != nil {
			packet.RemoveMultiExitDisc(bgpMsg)
		}
		packet.PrependAS(bgpMsg, p.Neighbor.Config.LocalAS)
		packet.SetNextHop(bgpMsg, p.Neighbor.Transport.Config.LocalAddress)
		packet.RemoveLocalPref(bgpMsg)
	}

	return true
}

func (p *Peer) PeerConnEstablished(conn *net.Conn) {
	host, _, err := net.SplitHostPort((*conn).LocalAddr().String())
	if err != nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Can't find local address from the peer connection: %s", p.Neighbor.NeighborAddress, (*conn).LocalAddr()))
		return
	}
	p.Neighbor.Transport.Config.LocalAddress = net.ParseIP(host)
}

func (p *Peer) PeerConnBroken() {
	p.Neighbor.Transport.Config.LocalAddress = nil
}

func (p *Peer) FSMStateChange(state BGPFSMState) {
	p.Neighbor.State.SessionState = uint32(state)
}

func (p *Peer) SendUpdate(bgpMsg packet.BGPMessage, path *Path) {
	p.logger.Info(fmt.Sprintf("Neighbor %s: Send update message valid routes:%s, withdraw routes:%s", p.Neighbor.NeighborAddress, bgpMsg.Body.(*packet.BGPUpdate).NLRI, bgpMsg.Body.(*packet.BGPUpdate).WithdrawnRoutes))
	bgpMsgRef := &bgpMsg
	if p.updatePathAttrs(bgpMsgRef, path) {
		atomic.AddUint32(&p.Neighbor.State.Queues.Output, 1)
		p.fsmManager.SendUpdateMsg(bgpMsgRef)
	}
}
