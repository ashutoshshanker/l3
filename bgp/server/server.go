// server.go
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bgp/config"
	"l3/bgp/packet"
	"l3/rib/ribdCommonDefs"
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
	RemPeerCh       chan string
	PeerCommandCh   chan config.PeerCommand
	BGPPktSrc       chan *packet.BGPPktSrc
	connRoutesTimer *time.Timer

	ribSubSocket    *nanomsg.SubSocket
	ribSubSocketCh   chan []byte
	ribSubSocketErrCh  chan error

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
	bgpServer.RemPeerCh = make(chan string)
	bgpServer.PeerCommandCh = make(chan config.PeerCommand)
	bgpServer.BGPPktSrc = make(chan *packet.BGPPktSrc)
	bgpServer.NeighborMutex = sync.RWMutex{}
	bgpServer.PeerMap = make(map[string]*Peer)
	bgpServer.Neighbors = make([]*Peer, 0)
	bgpServer.adjRib = NewAdjRib(bgpServer)
	bgpServer.connRoutesTimer = time.NewTimer(time.Duration(10) * time.Second)
	bgpServer.connRoutesTimer.Stop()
	bgpServer.ribSubSocketCh = make(chan []byte)
	bgpServer.ribSubSocketErrCh = make(chan error)
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

func (server *BGPServer) listenForRIBUpdates(address string) error {
	var err error
	if server.ribSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to create RIB subscribe socket, error:", err))
		return err
	}

	if err = server.ribSubSocket.Subscribe(""); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on RIB subscribe socket, error:", err))
		return err
	}

	if _, err = server.ribSubSocket.Connect(address); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to connect to RIB publisher socket, address:", address, "error:", err))
		return err
	}

	server.logger.Info(fmt.Sprintln("Connected to RIB publisher at address:", address))
	if err = server.ribSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to set the buffer size for RIB publisher socket, error:", err))
		return err
	}
	return nil
}

func (server *BGPServer) IsPeerLocal(peerIp string) bool {
	return server.PeerMap[peerIp].Neighbor.Config.PeerAS == server.BgpConfig.Global.Config.AS
}

func (server *BGPServer) sendUpdateMsgToAllPeers(msg *packet.BGPMessage, path *Path) {
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

		peer.SendUpdate(*msg.Clone(), path)
	}
}

func (server *BGPServer) SendUpdate(updated map[*Path][]packet.IPPrefix, withdrawn []packet.IPPrefix) {
	if len(withdrawn) > 0 {
		updateMsg := packet.NewBGPUpdateMessage(withdrawn, nil, nil)
		server.sendUpdateMsgToAllPeers(updateMsg, nil)
	}

	for path, dest := range updated {
		updateMsg := packet.NewBGPUpdateMessage(make([]packet.IPPrefix, 0), path.pathAttrs, dest)
		server.sendUpdateMsgToAllPeers(updateMsg, path)
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

func (server *BGPServer) convertDestIPToIPPrefix(routes []*ribd.Routes) []packet.IPPrefix {
	dest := make([]packet.IPPrefix, 0, len(routes))
	for _, r := range routes {
		ipPrefix := packet.ConstructIPPrefix(r.Ipaddr, r.Mask)
		dest = append(dest, *ipPrefix)
	}
	return dest
}

func (server *BGPServer) ProcessConnectedRoutes(installedRoutes []*ribd.Routes, withdrawnRoutes []*ribd.Routes) {
	server.logger.Info(fmt.Sprintln("valid routes:", installedRoutes, "invalid routes:", withdrawnRoutes))
	valid := server.convertDestIPToIPPrefix(installedRoutes)
	invalid := server.convertDestIPToIPPrefix(withdrawnRoutes)
	updated, withdrawn := server.adjRib.ProcessConnectedRoutes(server.BgpConfig.Global.Config.RouterId.String(),
		server.connRoutesPath, valid, invalid)
	server.SendUpdate(updated, withdrawn)
}

func (server *BGPServer) ProcessRemovePeer(peerIp string, peer *Peer) {
	updated, withdrawn := server.adjRib.RemoveUpdatesFromNeighbor(peerIp, peer)
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

	server.logger.Info("Listen for RIBd updates")
	server.listenForRIBUpdates(ribdCommonDefs.PUB_SOCKET_ADDR)

	server.logger.Info("Setting up Peer connections")
	acceptCh := make(chan *net.TCPConn)
	go server.listenForPeers(acceptCh)
	server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

	go func() {
		for {
			server.logger.Info("Read on RIB subscriber socket...")
			rxBuf, err := server.ribSubSocket.Recv(0)
			if err != nil {
				server.logger.Err(fmt.Sprintln("Recv on RIB subscriber socket failed with error:", err))
				server.ribSubSocketErrCh <- err
				continue
			}
			server.logger.Info(fmt.Sprintln("RIB subscriber recv returned:", rxBuf))
			server.ribSubSocketCh <- rxBuf
		}
	}()

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
			server.logger.Info(fmt.Sprintln("Remove Peer:", remPeer))
			peer, ok := server.PeerMap[remPeer]
			if !ok {
				server.logger.Info(fmt.Sprintln("Failed to remove peer. Peer at that address does not exist,", remPeer))
			}
			server.NeighborMutex.Lock()
			server.removePeerFromList(peer)
			server.NeighborMutex.Unlock()
			delete(server.PeerMap, remPeer)
			peer.Cleanup()
			server.ProcessRemovePeer(remPeer, peer)

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
			server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))
			server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

		case rxBuf := <-server.ribSubSocketCh:
			var route ribdCommonDefs.RoutelistInfo
			routes := make([]*ribd.Routes, 0, 1)
			reader := bytes.NewReader(rxBuf)
			decoder := json.NewDecoder(reader)
			msg := ribdCommonDefs.RibdNotifyMsg{}
			for err := decoder.Decode(&msg); err == nil; err = decoder.Decode(&msg) {
				err = json.Unmarshal(msg.MsgBuf, &route)
				if err != nil {
					server.logger.Err("Err in processing routes from RIB")
				}
				server.logger.Info(fmt.Sprintln("Remove connected route, dest:", route.RouteInfo.Ipaddr, "netmask:", route.RouteInfo.Mask, "nexthop:", route.RouteInfo.NextHopIp))
				routes = append(routes, &route.RouteInfo)
			}
			server.ProcessConnectedRoutes(make([]*ribd.Routes, 0), routes)

		case <-server.ribSubSocketErrCh:
			;
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
