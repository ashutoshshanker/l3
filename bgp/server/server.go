// server.go
package server

import (
    "fmt"
	"l3/bgp/packet"
    "net"
)

const IP string = "12.1.12.202" //"192.168.1.1"
const BGPPort string = "179"

type BgpServer struct {
    BgpConfig Bgp
    GlobalConfigCh chan GlobalConfig
    AddPeerCh chan PeerConfig
    RemPeerCh chan PeerConfig
    PeerCommandCh chan PeerCommand
	BGPPktSrc chan *packet.BGPPktSrc

    PeerMap map[string]*Peer
	adjRib *AdjRib
}

func NewBgpServer() *BgpServer {
    bgpServer := &BgpServer{}
    bgpServer.GlobalConfigCh = make(chan GlobalConfig)
    bgpServer.AddPeerCh = make(chan PeerConfig)
    bgpServer.RemPeerCh = make(chan PeerConfig)
    bgpServer.PeerCommandCh = make(chan PeerCommand)
	bgpServer.BGPPktSrc = make(chan *packet.BGPPktSrc)
    bgpServer.PeerMap = make(map[string]*Peer)
	bgpServer.adjRib = NewAdjRib(bgpServer)
    return bgpServer
}

func listenForPeers(acceptCh chan *net.TCPConn) {
    addr := ":" + BGPPort
    fmt.Printf("Listening for incomig connections on %s\n", addr)
    tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
    if err != nil {
        fmt.Println("ResolveTCPAddr failed with", err)
    }

    listener, err := net.ListenTCP("tcp", tcpAddr)
    if err != nil {
        fmt.Println("ListenTCP failed with", err)
    }

    for {
        tcpConn, err := listener.AcceptTCP()
        fmt.Println("Waiting for peer connections...")
        if err != nil {
            fmt.Println("AcceptTCP failed with", err)
        }
        acceptCh <- tcpConn
    }
}

func (server *BgpServer) IsPeerLocal(peerIp string) bool {
	return server.PeerMap[peerIp].Peer.AS == server.BgpConfig.GlobalConfig.AS
}

func (server *BgpServer) ProcessUpdate(pktInfo *packet.BGPPktSrc) {
	server.adjRib.ProcessUpdate(pktInfo)
}

func (server *BgpServer) StartServer() {
    gConf := <-server.GlobalConfigCh
    server.BgpConfig.GlobalConfig = gConf

    fmt.Println("Setting up Peer connections")
    acceptCh := make(chan *net.TCPConn)
    go listenForPeers(acceptCh)

	for {
		select {
		case addPeer := <-server.AddPeerCh:
			_, ok := server.PeerMap[addPeer.IP.String()]
			if ok {
			    fmt.Println("Failed to add peer. Peer at that address already exists,", addPeer.IP.String())
			}
			fmt.Println("Add Peer ip[%s]", addPeer.IP.String())
			peer := NewPeer(server, server.BgpConfig.GlobalConfig, addPeer)
			server.PeerMap[addPeer.IP.String()] = peer
			peer.Init()

		case remPeer := <-server.RemPeerCh:
			fmt.Println("Remove Peer")
			peer, ok := server.PeerMap[remPeer.IP.String()]
			if !ok {
			    fmt.Println("Failed to remove peer. Peer at that address does not exist, %v", remPeer.IP.String())
			}
			peer.Cleanup()
			delete(server.PeerMap, remPeer.IP.String())

		case tcpConn := <-acceptCh:
			fmt.Println("Connected to", tcpConn.RemoteAddr().String())
			host, _, _ := net.SplitHostPort(tcpConn.RemoteAddr().String())
			peer, ok := server.PeerMap[host]
			if !ok {
			    fmt.Println("Can't accept connection. Peer is not configured yet", host)
			    tcpConn.Close()
				fmt.Println("Closed connection from", host)
				break
			}
			peer.AcceptConn(tcpConn)
			//fmt.Println("send keep alives to peer...")
			//go peer.SendKeepAlives(tcpConn)

		case peerCommand := <- server.PeerCommandCh:
			fmt.Println("Peer Command received", peerCommand)
			peer, ok := server.PeerMap[peerCommand.IP.String()]
			if !ok {
			    fmt.Printf("Failed to apply command %s. Peer at that address does not exist, %v\n", peerCommand.Command, peerCommand.IP.String())
			}
			peer.Command(peerCommand.Command)

		case pktInfo := <- server.BGPPktSrc:
			fmt.Println("Received BGP message", pktInfo.Msg)
			server.ProcessUpdate(pktInfo)
        }
    }

}
