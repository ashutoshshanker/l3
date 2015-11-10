// server.go
package server

import (
    "fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
    "net"
	"ribd"
)

const IP string = "12.1.12.202" //"192.168.1.1"
const BGPPort string = "179"

type BGPServer struct {
	logger *syslog.Writer
	ribdClient *ribd.RouteServiceClient
    BgpConfig config.Bgp
    GlobalConfigCh chan config.GlobalConfig
    AddPeerCh chan config.NeighborConfig
    RemPeerCh chan config.NeighborConfig
    PeerCommandCh chan config.PeerCommand
	BGPPktSrc chan *packet.BGPPktSrc

    PeerMap map[string]*Peer
	adjRib *AdjRib
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
    bgpServer.PeerMap = make(map[string]*Peer)
	bgpServer.adjRib = NewAdjRib(bgpServer)
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
	return server.PeerMap[peerIp].Peer.PeerAS == server.BgpConfig.Global.Config.AS
}

func (server *BGPServer) ProcessUpdate(pktInfo *packet.BGPPktSrc) {
	server.adjRib.ProcessUpdate(pktInfo)
}

func (server *BGPServer) StartServer() {
    gConf := <-server.GlobalConfigCh
    server.BgpConfig.Global.Config = gConf

    server.logger.Info(fmt.Sprintln("Setting up Peer connections"))
    acceptCh := make(chan *net.TCPConn)
    go server.listenForPeers(acceptCh)

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
			peer.Init()

		case remPeer := <-server.RemPeerCh:
			server.logger.Info(fmt.Sprintln("Remove Peer"))
			peer, ok := server.PeerMap[remPeer.NeighborAddress.String()]
			if !ok {
			    server.logger.Info(fmt.Sprintln("Failed to remove peer. Peer at that address does not exist,",
					remPeer.NeighborAddress.String()))
			}
			peer.Cleanup()
			delete(server.PeerMap, remPeer.NeighborAddress.String())

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

		case peerCommand := <- server.PeerCommandCh:
			server.logger.Info(fmt.Sprintln("Peer Command received", peerCommand))
			peer, ok := server.PeerMap[peerCommand.IP.String()]
			if !ok {
			    server.logger.Info(fmt.Sprintf("Failed to apply command %s. Peer at that address does not exist, %v\n",
					peerCommand.Command, peerCommand.IP.String()))
			}
			peer.Command(peerCommand.Command)

		case pktInfo := <- server.BGPPktSrc:
			server.logger.Info(fmt.Sprintln("Received BGP message", pktInfo.Msg))
			server.ProcessUpdate(pktInfo)
        }
    }

}
