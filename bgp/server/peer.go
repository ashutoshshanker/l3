// peer.go
package server

import (
	"fmt"
    "net"
    "time"
)

const (
    _ = iota
    BGP_FSM_IDLE
    BGP_FSM_CONNECT
    BGP_FSM_ACTIVE
    BGP_FSM_OPENSENT
    BGP_FSM_OPENCONFIRM
    BGP_FSM_ESTABLISHED
)

type Peer struct {
    Global *GlobalConfig
    Peer *PeerConfig
    FSM *FSM
    Conn net.TCPConn
}

func NewPeer(globalConf GlobalConfig, peerConf PeerConfig) *Peer {
    peer := Peer{
        Global: &globalConf,
        Peer: &peerConf,
    }
    peerConf.SessionState = uint32(BGP_FSM_IDLE)
    peer.FSM = NewFSM(&globalConf, &peerConf)
    return &peer
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