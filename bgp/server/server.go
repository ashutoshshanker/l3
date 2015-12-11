// server.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"ribd"
	"sync"
	"sync/atomic"
	"time"
)

const IP string = "12.1.12.202" //"192.168.1.1"
const BGPPort string = "179"

type BGPServer struct {
	logger          *syslog.Writer
	ribdClient      *ribd.RouteServiceClient
	BgpConfig       config.Bgp
	GlobalConfigCh  chan config.GlobalConfig
	AddPeerCh       chan config.NeighborConfig
	RemPeerCh       chan config.NeighborConfig
	PeerCommandCh   chan config.PeerCommand
	BGPPktSrc       chan *packet.BGPPktSrc
	connRoutesTimer *time.Timer

	NeighborMutex  sync.RWMutex
	PeerMap        map[string]*Peer
	Neighbors      []*Peer
	adjRib         *AdjRib
	connRoutesPath *Path
}

func NewBGPServer(logger *syslog.Writer, ribdClient *ribd.RouteServiceClient) *BGPServer {
	bgpServer := &BGPServer{}
	bgpServer.logger = logger
	bgpServer.ribdClient = ribdClient
	bgpServer.GlobalConfigCh = make(chan config.GlobalConfig)
	bgpServer.AddPeerCh = make(chan config.NeighborConfig)
	bgpServer.RemPeerCh = make(chan config.NeighborConfig)
	bgpServer.PeerCommandCh = make(chan config.PeerCommand)
	bgpServer.BGPPktSrc = make(chan *packet.BGPPktSrc)
	bgpServer.NeighborMutex = sync.RWMutex{}
	bgpServer.PeerMap = make(map[string]*Peer)
	bgpServer.Neighbors = make([]*Peer, 0)
	bgpServer.adjRib = NewAdjRib(bgpServer)
	bgpServer.connRoutesTimer = time.NewTimer(time.Duration(10) * time.Second)
	bgpServer.connRoutesTimer.Stop()

	return bgpServer
}

func (server *BGPServer) listenForPeers(acceptCh chan *net.TCPConn) {
	addr := ":" + BGPPort
	server.logger.Info(fmt.Sprintf("Listening for incomig connections on %s\n", addr))
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("ResolveTCPAddr failed with", err))
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("ListenTCP failed with", err))
	}

	for {
		tcpConn, err := listener.AcceptTCP()
		server.logger.Info(fmt.Sprintln("Waiting for peer connections..."))
		if err != nil {
			server.logger.Info(fmt.Sprintln("AcceptTCP failed with", err))
		}
		acceptCh <- tcpConn
	}
}

func (server *BGPServer) IsPeerLocal(peerIp string) bool {
	return server.PeerMap[peerIp].Neighbor.Config.PeerAS == server.BgpConfig.Global.Config.AS
}

func (server *BGPServer) SendUpdate(updated map[*Path][]packet.IPPrefix, withdrawn []packet.IPPrefix) {
	firstMsg := true
	for path, dest := range updated {
		updateMsg := packet.NewBGPUpdateMessage(make([]packet.IPPrefix, 0), path.pathAttrs, dest)
		if firstMsg {
			firstMsg = false
			updateMsg.Body.(*packet.BGPUpdate).WithdrawnRoutes = withdrawn
		}

		for _, peer := range server.PeerMap {
			// If we recieve the route from IBGP peer, don't send it to other IBGP peers
			if path.peer != nil {
				if peer.IsInternal() {
					if path.peer.IsInternal() {
						continue
					}
				}

				// Don't send the update to the peer that sent the update.
				if peer.Neighbor.Config.NeighborAddress.String() == path.peer.Neighbor.Config.NeighborAddress.String() {
					continue
				}
			}

			peer.SendUpdate(*updateMsg.Clone(), path)
		}
	}
}

func (server *BGPServer) ProcessUpdate(pktInfo *packet.BGPPktSrc) {
	peer, ok := server.PeerMap[pktInfo.Src]
	if !ok {
		server.logger.Err(fmt.Sprintln("BgpServer:ProcessUpdate - Peer not found, address:", pktInfo.Src))
		return
	}

	atomic.AddUint32(&peer.Neighbor.State.Queues.Input, ^uint32(0))
	peer.Neighbor.State.Messages.Received.Update++
	updated, withdrawn := server.adjRib.ProcessUpdate(peer, pktInfo)
	server.SendUpdate(updated, withdrawn)
}

func (server *BGPServer) ProcessConnectedRoutes(routes []*ribd.Routes) {
	dest := make([]packet.IPPrefix, 0, len(routes))
	for _, r := range routes {
		ipPrefix := packet.ConstructIPPrefix(r.Ipaddr, r.Mask)
		dest = append(dest, *ipPrefix)
	}

	updated, withdrawn := server.adjRib.ProcessConnectedRoutes(server.BgpConfig.Global.Config.RouterId.String(),
		server.connRoutesPath, dest, make([]packet.IPPrefix, 0))
	server.SendUpdate(updated, withdrawn)
}

func (server *BGPServer) addPeerToList(peer *Peer) {
	server.Neighbors = append(server.Neighbors, peer)
}

func (server *BGPServer) removePeerFromList(peer *Peer) {
	for idx, item := range server.Neighbors {
		if item == peer {
			server.Neighbors[idx] = server.Neighbors[len(server.Neighbors)-1]
			server.Neighbors[len(server.Neighbors)-1] = nil
			server.Neighbors = server.Neighbors[:len(server.Neighbors)-1]
			break
		}
	}
}

func (server *BGPServer) StartServer() {
	gConf := <-server.GlobalConfigCh
	server.logger.Info(fmt.Sprintln("Recieved global conf:", gConf))
	server.BgpConfig.Global.Config = gConf

	pathAttrs := packet.ConstructPathAttrForConnRoutes(gConf.RouterId, gConf.AS)
	server.connRoutesPath = NewPath(server, nil, pathAttrs, false, false, RouteTypeConnected)

	server.logger.Info(fmt.Sprintln("Setting up Peer connections"))
	acceptCh := make(chan *net.TCPConn)
	go server.listenForPeers(acceptCh)
	server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

	for {
		select {
		case addPeer := <-server.AddPeerCh:
			_, ok := server.PeerMap[addPeer.NeighborAddress.String()]
			if ok {
				server.logger.Info(fmt.Sprintln("Failed to add peer. Peer at that address already exists,",
					addPeer.NeighborAddress.String()))
			}
			server.logger.Info(fmt.Sprintln("Add Peer ip:", addPeer.NeighborAddress.String()))
			peer := NewPeer(server, server.BgpConfig.Global.Config, addPeer)
			server.PeerMap[addPeer.NeighborAddress.String()] = peer
			server.NeighborMutex.Lock()
			server.addPeerToList(peer)
			server.NeighborMutex.Unlock()
			peer.Init()

		case remPeer := <-server.RemPeerCh:
			server.logger.Info(fmt.Sprintln("Remove Peer"))
			peer, ok := server.PeerMap[remPeer.NeighborAddress.String()]
			if !ok {
				server.logger.Info(fmt.Sprintln("Failed to remove peer. Peer at that address does not exist,",
					remPeer.NeighborAddress.String()))
			}
			server.NeighborMutex.Lock()
			server.removePeerFromList(peer)
			server.NeighborMutex.Unlock()
			delete(server.PeerMap, remPeer.NeighborAddress.String())
			peer.Cleanup()

		case tcpConn := <-acceptCh:
			server.logger.Info(fmt.Sprintln("Connected to", tcpConn.RemoteAddr().String()))
			host, _, _ := net.SplitHostPort(tcpConn.RemoteAddr().String())
			peer, ok := server.PeerMap[host]
			if !ok {
				server.logger.Info(fmt.Sprintln("Can't accept connection. Peer is not configured yet", host))
				tcpConn.Close()
				server.logger.Info(fmt.Sprintln("Closed connection from", host))
				break
			}
			peer.AcceptConn(tcpConn)
			//server.logger.Info(fmt.Sprintln("send keep alives to peer..."))
			//go peer.SendKeepAlives(tcpConn)

		case peerCommand := <-server.PeerCommandCh:
			server.logger.Info(fmt.Sprintln("Peer Command received", peerCommand))
			peer, ok := server.PeerMap[peerCommand.IP.String()]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to apply command %s. Peer at that address does not exist, %v\n",
					peerCommand.Command, peerCommand.IP.String()))
			}
			peer.Command(peerCommand.Command)

		case pktInfo := <-server.BGPPktSrc:
			server.logger.Info(fmt.Sprintln("Received BGP message", pktInfo.Msg))
			server.ProcessUpdate(pktInfo)

		case <-server.connRoutesTimer.C:
			routes, _ := server.ribdClient.GetConnectedRoutesInfo()
			server.ProcessConnectedRoutes(routes)
			server.connRoutesTimer.Reset(time.Duration(10) * time.Second)
		}
	}

}

func (s *BGPServer) GetBGPGlobalState() config.GlobalState {
	return s.BgpConfig.Global.State
}

func (s *BGPServer) GetBGPNeighborState(neighborIP string) *config.NeighborState {
	peer, ok := s.PeerMap[neighborIP]
	if !ok {
		s.logger.Err(fmt.Sprintf("GetBGPNeighborState - Neighbor not found for address:%s", neighborIP))
		return nil
	}
	return &peer.Neighbor.State
}

func (s *BGPServer) BulkGetBGPNeighbors(index int, count int) (int, int, []*config.NeighborState) {
	defer s.NeighborMutex.RUnlock()

	s.NeighborMutex.RLock()
	if index + count > len(s.Neighbors) {
		count = len(s.Neighbors) - index
	}

	result := make([]*config.NeighborState, count)
	for i := 0; i < count; i++ {
		result[i] = &s.Neighbors[i + index].Neighbor.State
	}

	index += count
	if index >= len(s.Neighbors) {
		index = 0
	}
	return index, count, result
}
