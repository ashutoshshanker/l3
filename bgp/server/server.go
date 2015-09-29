// server.go
package server

import (
    "fmt"
    "net"
)

const IP string = "localhost" //"192.168.1.1"
const BGP_PORT string = "179"

type BgpServer struct {
    BgpConfig Bgp
    GlobalConfigCh chan GlobalConfig
    AddPeerCh chan PeerConfig
    RemPeerCh chan PeerConfig
    
    PeerMap map[string]*Peer
}

func NewBgpServer() *BgpServer {
    bgpServer := BgpServer{}
    bgpServer.GlobalConfigCh = make(chan GlobalConfig)
    bgpServer.AddPeerCh = make(chan PeerConfig)
    bgpServer.RemPeerCh = make(chan PeerConfig)
    bgpServer.PeerMap = make(map[string]*Peer)
    return &bgpServer
}

func listenForPeers(acceptCh chan *net.TCPConn) {
    addr := IP + ":" + BGP_PORT
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

func (server *BgpServer) StartServer() {
    gConf := <-server.GlobalConfigCh
    server.BgpConfig.GlobalConfig = gConf
    
    fmt.Println("Setting up Peer connections")
    acceptCh := make(chan *net.TCPConn)
    go listenForPeers(acceptCh)
    
    for {
        select {
            case addPeer := <-server.AddPeerCh:
                fmt.Println("Add Peer ip[%s]", addPeer.IP.String())
                peer := NewPeer(server.BgpConfig.GlobalConfig, addPeer)
                _, ok := server.PeerMap[addPeer.IP.String()]
                if ok {
                    fmt.Println("Failed to add peer, Peer at that address already exists, %v", addPeer.IP.String())
                }
                
                server.PeerMap[addPeer.IP.String()] = peer
            case remPeer := <-server.RemPeerCh:
                fmt.Println("Remove Peer")
                _, ok := server.PeerMap[remPeer.IP.String()]
                if !ok {
                    fmt.Println("Failed to remove peer, Peer at that address does not exist, %v", remPeer.IP.String())
                }
            case tcpConn := <-acceptCh:
                fmt.Println("Connected to", tcpConn.RemoteAddr().String())
                host, _, _ := net.SplitHostPort(tcpConn.RemoteAddr().String())
                peer, ok := server.PeerMap[host]
                if !ok {
                    fmt.Println("Can't accept connection, Peer is not configured yet, %v", host)
                    tcpConn.Close()
                }
                fmt.Println("send keep alives to peer...")
                go peer.SendKeepAlives(tcpConn)
        }
    }

}