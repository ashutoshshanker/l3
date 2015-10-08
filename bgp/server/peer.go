// peer.go
package server

import (
	"fmt"
    "net"
    "time"
)

type Peer struct {
    Global *GlobalConfig
    Peer *PeerConfig
    fsmManager *FsmManager
}

func NewPeer(globalConf GlobalConfig, peerConf PeerConfig) *Peer {
    peer := Peer{
        Global: &globalConf,
        Peer: &peerConf,
    }
    peer.fsmManager = NewFsmManager(&globalConf, &peerConf)
    return &peer
}

func (peer *Peer) Init() {
    go peer.fsmManager.Init()
}

func (peer *Peer) SendKeepAlives(conn *net.TCPConn) {
    peer.Conn = *conn

    bgpKeepAliveMsg := NewBGPKeepAliveMessage()
    var num int
    var err error

    for {
        select {
            case <-time.After(time.Second * 1):
                fmt.Println("send the packet ...")
                packet, _ := bgpKeepAliveMsg.Serialize()
                num, err = peer.Conn.Write(packet)
                if err != nil {
                    fmt.Println("Conn.Write failed with error:", err)
                }
                fmt.Println("Conn.Write succeeded. sent %d", num, "bytes")
        }
    }
}
