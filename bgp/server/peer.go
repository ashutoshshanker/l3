// peer.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"sync/atomic"
	"time"
)

type Peer struct {
	Server     *BGPServer
	logger     *syslog.Writer
	Global     *config.GlobalConfig
	Neighbor   *config.Neighbor
	fsmManager *FSMManager
	BGPId      net.IP
	ASSize     uint8
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
		BGPId: net.IP{},
	}

	peer.Neighbor.State = config.NeighborState{
		PeerAS:          peerConf.PeerAS,
		LocalAS:         peerConf.LocalAS,
		AuthPassword:    peerConf.AuthPassword,
		Description:     peerConf.Description,
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
	if p.fsmManager == nil {
		p.logger.Info(fmt.Sprintf("Instantiating new FSM Manager for neighbor %s", p.Neighbor.NeighborAddress))
		p.fsmManager = NewFSMManager(p, &p.Server.BgpConfig.Global.Config, &p.Neighbor.Config)
	}

	go p.fsmManager.Init()
}

func (p *Peer) Cleanup() {
	p.fsmManager.closeCh <- true
	p.fsmManager = nil
}

func (p *Peer) UpdateNeighborConf(nConf config.NeighborConfig) {
	p.Neighbor.NeighborAddress = nConf.NeighborAddress
	p.Neighbor.Config = nConf
}

func (p *Peer) AcceptConn(conn *net.TCPConn) {
	if p.fsmManager == nil {
		p.logger.Info(fmt.Sprintf("FSM Manager is not instantiated yet for neighbor %s", p.Neighbor.NeighborAddress))
		return
	}
	p.fsmManager.acceptCh <- conn
}

func (p *Peer) Command(command int) {
	if p.fsmManager == nil {
		p.logger.Info(fmt.Sprintf("FSM Manager is not instantiated yet for neighbor %s", p.Neighbor.NeighborAddress))
		return
	}
	p.fsmManager.commandCh <- command
}

func (p *Peer) IsInternal() bool {
	return p.Neighbor.Config.PeerAS == p.Neighbor.Config.LocalAS
}

func (p *Peer) IsExternal() bool {
	return p.Neighbor.Config.LocalAS != p.Neighbor.Config.PeerAS
}

func (p *Peer) IsRouteReflectorClient() bool {
	return p.Neighbor.Config.RouteReflectorClient
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

func (p *Peer) SetPeerAttrs(bgpId net.IP, asSize uint8) {
	p.BGPId = bgpId
	p.ASSize = asSize
}

func (p *Peer) updatePathAttrs(bgpMsg *packet.BGPMessage, path *Path) bool {
	if p.Neighbor.Transport.Config.LocalAddress == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Can't send Update message, FSM is not in Established state",
			p.Neighbor.NeighborAddress))
		return false
	}

	if bgpMsg == nil || bgpMsg.Body.(*packet.BGPUpdate).PathAttributes == nil {
		p.logger.Err(fmt.Sprintf("Neighbor %s: Path attrs not found in BGP Update message", p.Neighbor.NeighborAddress))
		return false
	}

	if p.ASSize == 2 {
		packet.Convert4ByteTo2ByteASPath(bgpMsg)
	}

	if p.IsInternal() {
		if path.peer != nil && (path.peer.IsRouteReflectorClient() || p.IsRouteReflectorClient()) {
			packet.AddOriginatorId(bgpMsg, path.peer.BGPId)
			packet.AddClusterId(bgpMsg, path.peer.Neighbor.Config.RouteReflectorClusterId)
		} else {
			packet.SetNextHop(bgpMsg, p.Neighbor.Transport.Config.LocalAddress)
			packet.SetLocalPref(bgpMsg, path.GetPreference())
		}
	} else {
		// Do change these path attrs for local routes
		if path.peer != nil {
			packet.RemoveMultiExitDisc(bgpMsg)
		}
		packet.PrependAS(bgpMsg, p.Neighbor.Config.LocalAS, p.ASSize)
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
	p.Server.PeerConnEstCh <- p.Neighbor.NeighborAddress.String()
}

func (p *Peer) PeerConnBroken(fsmCleanup bool) {
	if p.Neighbor.Transport.Config.LocalAddress != nil {
		p.Neighbor.Transport.Config.LocalAddress = nil
		p.Server.PeerConnBrokenCh <- p.Neighbor.NeighborAddress.String()
	}
}

func (p *Peer) FSMStateChange(state BGPFSMState) {
	p.Neighbor.State.SessionState = uint32(state)
}

func (p *Peer) SendUpdate(bgpMsg packet.BGPMessage, path *Path) {
	p.logger.Info(fmt.Sprintf("Neighbor %s: Send update message valid routes:%v, withdraw routes:%v", p.Neighbor.NeighborAddress, bgpMsg.Body.(*packet.BGPUpdate).NLRI, bgpMsg.Body.(*packet.BGPUpdate).WithdrawnRoutes))
	bgpMsgRef := &bgpMsg
	if p.updatePathAttrs(bgpMsgRef, path) {
		atomic.AddUint32(&p.Neighbor.State.Queues.Output, 1)
		p.fsmManager.SendUpdateMsg(bgpMsgRef)
	}
}
