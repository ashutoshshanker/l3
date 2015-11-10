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
    fsm *FSM
	logger *syslog.Writer
    dir config.ConnDir
    conn *net.Conn

    readCh chan bool
    stopCh chan bool
}

func NewPeerConn(fsm *FSM, dir config.ConnDir, conn *net.Conn) *PeerConn {
    peerConn := PeerConn{
        fsm: fsm,
		logger: fsm.logger,
        dir: dir,
        conn: conn,
		readCh: make(chan bool),
        stopCh: make(chan bool),
    }

    return &peerConn
}

func (p *PeerConn) StartReading() {
    stopReading := false
    doneReadingCh := make(chan bool)
    stopReadingCh := make(chan bool)

	p.logger.Info(fmt.Sprintln("conn:StartReading called"))
    go p.ReadPkt(doneReadingCh, stopReadingCh)
	p.readCh <- true

	for {
		select {
		case <- p.stopCh:
			if !stopReading {
			    stopReading = true
			    stopReadingCh <- true
				return
			}

		case readOk := <- doneReadingCh:
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
	p.logger.Info(fmt.Sprintln("read", length, "bytes from the TCP conn"))
	num, err := (*p.conn).Read(buf)
	p.logger.Info(fmt.Sprintln("conn.Read read ", num, "bytes, returned", err))
	return buf, err
}

func (p *PeerConn) ReadPkt(doneCh chan bool, stopCh chan bool) {
	p.logger.Info(fmt.Sprintln("conn:ReadPkt called"))
	var t time.Time
    for {
        select {
        case <- p.readCh:
			p.logger.Info(fmt.Sprintln("Start reading again..."))
			(*p.conn).SetReadDeadline(time.Now().Add(time.Duration(3) * time.Second))
			buf, err := p.readPartialPkt(packet.BGPMsgHeaderLen)
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					p.logger.Info(fmt.Sprintln("readPartialPkt timed out, returned err:", err, "neer:", nerr))
				    doneCh <- true
				    continue
				} else {
					p.logger.Info(fmt.Sprintln("readPartialPkt DID NOT time out, returned err:", err, "nerr:", nerr))
				    p.fsm.outConnErrCh <- err
				    break
				}
			}

			header := packet.BGPHeader{}
			err = header.Decode(buf)
			if err != nil {
				p.logger.Info(fmt.Sprintln("BGP packet header decode failed"))
				bgpPktInfo := packet.NewBGPPktInfo(nil, err.(*packet.BGPMessageError))
				p.fsm.pktRxCh <- bgpPktInfo
				doneCh <- false
				continue
			}

			p.logger.Info(fmt.Sprintln("Recieved BGP packet type=", header.Type))

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

			msg := &packet.BGPMessage{}
			err = msg.Decode(&header, buf)
			bgpPktInfo := packet.NewBGPPktInfo(msg, nil)
			msgOk := true
			if err != nil {
				p.logger.Info(fmt.Sprintln("BGP packet body decode failed"))
				bgpPktInfo = packet.NewBGPPktInfo(msg, err.(*packet.BGPMessageError))
				msgOk = false
			}

			p.fsm.pktRxCh <- bgpPktInfo
			doneCh <- msgOk

        case <- stopCh:
            return
        }
    }
}

