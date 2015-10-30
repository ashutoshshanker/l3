// conn.go
package server

import (
    "fmt"
    "net"
    "time"
)

type PeerConn struct {
    fsm *FSM
    dir ConnDir
    conn *net.Conn

    readCh chan bool
    stopCh chan bool
}

func NewPeerConn(fsm *FSM, dir ConnDir, conn *net.Conn) *PeerConn {
    peerConn := PeerConn{
        fsm: fsm,
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

	fmt.Println("conn:StartReading called")
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

func readPartialPkt(conn *net.Conn, length uint32) ([]byte, error) {
	buf := make([]byte, length)
	fmt.Println("read", length, "bytes from the TCP conn")
	num, err := (*conn).Read(buf)
	fmt.Println("conn.Read read ", num, "bytes, returned", err)
	return buf, err
}

func (p *PeerConn) ReadPkt(doneCh chan bool, stopCh chan bool) {
	fmt.Println("conn:ReadPkt called")
	var t time.Time
    for {
        select {
        case <- p.readCh:
			fmt.Println("Start reading again...")
			(*p.conn).SetReadDeadline(time.Now().Add(time.Duration(3) * time.Second))
			buf, err := readPartialPkt(p.conn, BGPMsgHeaderLen)
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					fmt.Println("readPartialPkt timed out, returned err:", err, "neer:", nerr)
				    doneCh <- true
				    continue
				} else {
					fmt.Println("readPartialPkt DID NOT time out, returned err:", err, "nerr:", nerr)
				    p.fsm.outConnErrCh <- err
				    break
				}
			}

			header := BGPHeader{}
			err = header.Decode(buf)
			if err != nil {
				fmt.Println("BGP packet header decode failed")
				bgpPktInfo := &BGPPktInfo{nil, err.(*BGPMessageError)}
				p.fsm.pktRxCh <- bgpPktInfo
				doneCh <- false
				continue
			}

			fmt.Println("Recieved BGP packet type=", header.Type)

			(*p.conn).SetReadDeadline(t)
			if header.Len() > BGPMsgHeaderLen {
				buf, err = readPartialPkt(p.conn, header.Len() - BGPMsgHeaderLen)
				if err != nil {
					p.fsm.outConnErrCh <- err
					break
				}
			} else {
				buf = make([]byte, 0)
			}

			msg := &BGPMessage{}
			err = msg.Decode(&header, buf)
			bgpPktInfo := &BGPPktInfo{msg, nil}
			msgOk := true
			if err != nil {
				fmt.Println("BGP packet body decode failed")
				bgpPktInfo = &BGPPktInfo{msg, err.(*BGPMessageError)}
				msgOk = false
			}

			p.fsm.pktRxCh <- bgpPktInfo
			doneCh <- msgOk

        case <- stopCh:
            return
        }
    }
}

