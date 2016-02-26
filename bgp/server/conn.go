// conn.go
package server

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"log/syslog"
	"net"
	"time"
)

type PeerConn struct {
	fsm       *FSM
	logger    *syslog.Writer
	dir       config.ConnDir
	conn      *net.Conn
	peerAttrs packet.BGPPeerAttrs

	readCh chan bool
	stopCh chan bool
}

func NewPeerConn(fsm *FSM, dir config.ConnDir, conn *net.Conn) *PeerConn {
	peerConn := PeerConn{
		fsm:    fsm,
		logger: fsm.logger,
		dir:    dir,
		conn:   conn,
		peerAttrs: packet.BGPPeerAttrs{
			ASSize: 2,
		},
		readCh: make(chan bool),
		stopCh: make(chan bool),
	}

	return &peerConn
}

func (p *PeerConn) StartReading() {
	stopReading := false
	doneReadingCh := make(chan bool)
	stopReadingCh := make(chan bool)

	p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id, "conn:StartReading called"))
	go p.ReadPkt(doneReadingCh, stopReadingCh)
	p.readCh <- true

	for {
		select {
		case <-p.stopCh:
			if !stopReading {
				stopReading = true
				stopReadingCh <- true
				return
			}

		case readOk := <-doneReadingCh:
			if readOk && !stopReading {
				p.readCh <- true
			}
		}
	}
}

func (p *PeerConn) StopReading() {
	p.stopCh <- true
}

func (p *PeerConn) readPartialPkt(length uint32) ([]byte, error) {
	buf := make([]byte, length)
	_, err := (*p.conn).Read(buf)
	return buf, err
}

func (p *PeerConn) ReadPkt(doneCh chan bool, stopCh chan bool) {
	p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id, "conn:ReadPkt called"))
	var t time.Time
	for {
		select {
		case <-p.readCh:
			(*p.conn).SetReadDeadline(time.Now().Add(time.Duration(3) * time.Second))
			buf, err := p.readPartialPkt(packet.BGPMsgHeaderLen)
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					doneCh <- true
					continue
				} else {
					p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
						"readPartialPkt DID NOT time out, returned err:", err, "nerr:", nerr))
					p.fsm.outConnErrCh <- err
					break
				}
			}

			header := packet.BGPHeader{}
			err = header.Decode(buf)
			if err != nil {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"BGP packet header decode failed"))
				bgpPktInfo := packet.NewBGPPktInfo(nil, err.(*packet.BGPMessageError))
				p.fsm.pktRxCh <- bgpPktInfo
				doneCh <- false
				continue
			}

			if header.Type != packet.BGPMsgTypeKeepAlive {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"Recieved BGP packet type=", header.Type))
			}

			(*p.conn).SetReadDeadline(t)
			if header.Len() > packet.BGPMsgHeaderLen {
				buf, err = p.readPartialPkt(header.Len() - packet.BGPMsgHeaderLen)
				if err != nil {
					p.fsm.outConnErrCh <- err
					break
				}
			} else {
				buf = make([]byte, 0)
			}

			if header.Type != packet.BGPMsgTypeKeepAlive {
				p.logger.Info(fmt.Sprintf("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"Received BGP packet %x", buf))
			}

			msg := &packet.BGPMessage{}
			err = msg.Decode(&header, buf, p.peerAttrs)
			bgpPktInfo := packet.NewBGPPktInfo(msg, nil)
			msgOk := true
			if err != nil {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"BGP packet body decode failed, err:", err))
				bgpPktInfo = packet.NewBGPPktInfo(msg, err.(*packet.BGPMessageError))
				msgOk = false
			}

			if header.Type == packet.BGPMsgTypeOpen {
				p.peerAttrs.ASSize = packet.GetASSize(msg.Body.(*packet.BGPOpen))
				p.peerAttrs.AddPathFamily = packet.GetAddPathFamily(msg.Body.(*packet.BGPOpen))
			}
			p.fsm.pktRxCh <- bgpPktInfo
			doneCh <- msgOk

		case <-stopCh:
			if p.conn != nil {
				(*p.conn).Close()
			}
			return
		}
	}
}
