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
	Peer       *config.NeighborConfig
	fsmManager *FSMManager
}

func NewPeer(server *BGPServer, globalConf config.GlobalConfig, peerConf config.NeighborConfig) *Peer {
	peer := Peer{
		Server: server,
		logger: server.logger,
		Global: &globalConf,
		Peer:   &peerConf,
	}
	peer.fsmManager = NewFSMManager(&peer, &globalConf, &peerConf)
	return &peer
}

func (peer *Peer) Init() {
	go peer.fsmManager.Init()
}

func (peer *Peer) Cleanup() {}

func (peer *Peer) AcceptConn(conn *net.TCPConn) {
	peer.fsmManager.acceptCh <- conn
}

func (peer *Peer) Command(command int) {
	peer.fsmManager.commandCh <- command
}

func (peer *Peer) SendKeepAlives(conn *net.TCPConn) {
	bgpKeepAliveMsg := packet.NewBGPKeepAliveMessage()
	var num int
	var err error

	for {
		select {
		case <-time.After(time.Second * 1):
			peer.logger.Info(fmt.Sprintln("send the packet ..."))
			packet, _ := bgpKeepAliveMsg.Encode()
			num, err = conn.Write(packet)
			if err != nil {
				peer.logger.Info(fmt.Sprintln("Conn.Write failed with error:", err))
			}
			peer.logger.Info(fmt.Sprintln("Conn.Write succeeded. sent %d", num, "bytes"))
		}
	}
}
