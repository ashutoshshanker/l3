// conn.go
package fsm

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"math/rand"
	"net"
	"time"
	"utils/logging"

	"golang.org/x/net/ipv4"
)

type OutTCPConn struct {
	fsm          *FSM
	logger       *logging.Writer
	fsmConnCh    chan net.Conn
	fsmConnErrCh chan error
	StopConnCh   chan bool
	id           uint32
}

func NewOutTCPConn(fsm *FSM, fsmConnCh chan net.Conn, fsmConnErrCh chan error) *OutTCPConn {
	outConn := OutTCPConn{
		fsm:          fsm,
		logger:       fsm.logger,
		fsmConnCh:    fsmConnCh,
		fsmConnErrCh: fsmConnErrCh,
		StopConnCh:   make(chan bool),
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	outConn.id = r.Uint32()
	fsm.logger.Info(fmt.Sprintln("Neighbor:", fsm.pConf.NeighborAddress, "FSM", fsm.id,
		"Creating new out TCP conn with id", outConn.id))
	return &outConn
}

func (o *OutTCPConn) Connect(seconds uint32, addr string, connCh chan net.Conn, errCh chan error) {
	reachableCh := make(chan bool)
	reachabilityInfo := config.ReachabilityInfo{
		IP:          o.fsm.pConf.NeighborAddress.String(),
		ReachableCh: reachableCh,
	}
	o.fsm.Manager.reachabilityCh <- reachabilityInfo
	reachable := <-reachableCh
	if !reachable {
		duration := uint32(3)
		for {
			select {
			case <-time.After(time.Duration(duration) * time.Second):
				o.fsm.Manager.reachabilityCh <- reachabilityInfo
				reachable = <-reachableCh
			}
			seconds -= duration
			if reachable || seconds <= duration {
				break
			}
		}
		if !reachable {
			errCh <- config.AddressNotResolvedError{"Neighbor is not reachable"}
			return
		}
	}

	o.logger.Info(fmt.Sprintln("Neighbor:", o.fsm.pConf.NeighborAddress, "FSM", o.fsm.id,
		"Connect called... calling DialTimeout with", seconds, "second timeout", "OutTCPCOnn id", o.id))
	conn, err := net.DialTimeout("tcp", addr, time.Duration(seconds)*time.Second)
	if err != nil {
		errCh <- err
	} else {
		packetConn := ipv4.NewConn(conn)
		ttl := 1
		if o.fsm.pConf.MultiHopEnable {
			ttl = int(o.fsm.pConf.MultiHopTTL)
		}
		if err = packetConn.SetTTL(ttl); err != nil {
			conn.Close()
			errCh <- err
			return
		}
		connCh <- conn
	}
}

func (o *OutTCPConn) ConnectToPeer(seconds uint32, addr string) {
	var stopConn bool = false
	connCh := make(chan net.Conn)
	errCh := make(chan error)

	o.logger.Info(fmt.Sprintln("Neighbor:", o.fsm.pConf.NeighborAddress, "FSM", o.fsm.id, "ConnectToPeer called",
		"OutTCPCOnn id", o.id))
	connTime := seconds - 3
	if connTime <= 0 {
		connTime = seconds
	}

	go o.Connect(seconds, addr, connCh, errCh)

	for {
		select {
		case conn := <-connCh:
			o.logger.Info(fmt.Sprintln("Neighbor:", o.fsm.pConf.NeighborAddress, "FSM", o.fsm.id,
				"ConnectToPeer: Connected to peer", addr, "OutTCPCOnn id", o.id))
			if stopConn {
				conn.Close()
				return
			}

			o.fsmConnCh <- conn
			return

		case err := <-errCh:
			o.logger.Info(fmt.Sprintln("Neighbor:", o.fsm.pConf.NeighborAddress, "FSM", o.fsm.id,
				"ConnectToPeer: Failed to connect to peer", addr, "with error:", err, "OutTCPCOnn id", o.id))
			if stopConn {
				return
			}

			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				o.logger.Info(fmt.Sprintln("Neighbor:", o.fsm.pConf.NeighborAddress, "FSM", o.fsm.id,
					"Connect to peer timed out, retrying...", "OutTCPCOnn id", o.id))
				go o.Connect(3, addr, connCh, errCh)
			} else if _, ok := err.(config.AddressNotResolvedError); ok {
				go o.Connect(3, addr, connCh, errCh)
			} else {
				o.fsmConnErrCh <- err
			}

		case <-o.StopConnCh:
			o.logger.Info(fmt.Sprintln("Neighbor:", o.fsm.pConf.NeighborAddress, "FSM", o.fsm.id,
				"ConnectToPeer: Recieved stop connecting to peer", addr, "OutTCPCOnn id", o.id))
			stopConn = true
		}
	}
}

type PeerConn struct {
	fsm       *FSM
	logger    *logging.Writer
	dir       config.ConnDir
	conn      *net.Conn
	peerAttrs packet.BGPPeerAttrs

	readCh chan bool
	stopCh chan bool
	exitCh chan bool
}

func NewPeerConn(fsm *FSM, dir config.ConnDir, conn *net.Conn) *PeerConn {
	peerConn := PeerConn{
		fsm:    fsm,
		logger: fsm.logger,
		dir:    dir,
		conn:   conn,
		peerAttrs: packet.BGPPeerAttrs{
			ASSize:           2,
			AddPathsRxActual: false,
		},
		readCh: make(chan bool),
		stopCh: make(chan bool),
		exitCh: make(chan bool),
	}

	return &peerConn
}

func (p *PeerConn) StartReading() {
	stopReading := false
	readError := false
	doneReadingCh := make(chan bool)
	stopReadingCh := make(chan bool)
	exitCh := make(chan bool)

	p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id, "conn:StartReading called"))
	go p.ReadPkt(doneReadingCh, stopReadingCh, exitCh)
	p.readCh <- true

	for {
		select {
		case <-p.stopCh:
			stopReading = true
			if readError {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"readError is true, send stopReadingCh"))
				stopReadingCh <- true
			}

		case <-exitCh:
			p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
				"conn: exit channel"))
			p.exitCh <- true

		case readOk := <-doneReadingCh:
			if stopReading {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"stopReading is true, send stopReadingCh"))
				stopReadingCh <- true
			} else {
				if readOk {
					p.readCh <- true
				} else {
					p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
						"read failed, set readError to true"))
					readError = true
				}
			}
		}
	}
}

func (p *PeerConn) StopReading() {
	p.stopCh <- true
}

func (p *PeerConn) readPartialPkt(length int) ([]byte, error) {
	buf := make([]byte, length)
	var totalRead int = 0
	var read int = 0
	var err error
	for totalRead < length {
		read, err = (*p.conn).Read(buf[totalRead:])
		if err != nil {
			return buf, err
		}
		totalRead += read
		p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id, "conn:readPartialPkt -",
			"read", read, "bytes, total read", totalRead, "bytes, lenght =", length))
	}
	return buf, err
}

func (p *PeerConn) ReadPkt(doneCh chan bool, stopCh chan bool, exitCh chan bool) {
	p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id, "conn:ReadPkt called"))
	var t time.Time
	var msg *packet.BGPMessage
	var msgErr *packet.BGPMessageError
	var header *packet.BGPHeader
	for {
		select {
		case <-p.readCh:
			msg = nil
			msgErr = nil
			header = nil
			(*p.conn).SetReadDeadline(time.Now().Add(time.Duration(3) * time.Second))
			buf, err := p.readPartialPkt(int(packet.BGPMsgHeaderLen))
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					doneCh <- true
					continue
				} else {
					p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
						"readPartialPkt DID NOT time out, returned err:", err, "nerr:", nerr))
					p.fsm.outConnErrCh <- err
					doneCh <- false
					break
				}
			}

			header = packet.NewBGPHeader()
			err = header.Decode(buf)
			if err != nil {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"BGP packet header decode failed"))
				//bgpPktInfo := packet.NewBGPPktInfo(nil, err.(*packet.BGPMessageError))
				p.fsm.pktRxCh <- packet.NewBGPPktInfo(nil, err.(*packet.BGPMessageError))
				doneCh <- false
				continue
			}

			if header.Type != packet.BGPMsgTypeKeepAlive {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"Recieved BGP packet type=", header.Type, "len=", header.Len()))
			}

			(*p.conn).SetReadDeadline(t)
			if header.Len() > packet.BGPMsgHeaderLen {
				buf, err = p.readPartialPkt(int(header.Len() - packet.BGPMsgHeaderLen))
				if err != nil {
					p.fsm.outConnErrCh <- err
					doneCh <- false
					break
				}
			} else {
				buf = make([]byte, 0)
			}

			if header.Type != packet.BGPMsgTypeKeepAlive {
				p.logger.Info(fmt.Sprintf("Neighbor:%s FSM %d Received BGP packet %x", p.fsm.pConf.NeighborAddress,
					p.fsm.id, buf))
			}

			msg = packet.NewBGPMessage()
			err = msg.Decode(header, buf, p.peerAttrs)
			//bgpPktInfo := packet.NewBGPPktInfo(msg, nil)
			msgOk := true
			if header.Type == packet.BGPMsgTypeNotification {
				msgOk = false
			}

			if err != nil {
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
					"BGP packet body decode failed, err:", err))
				//bgpPktInfo = packet.NewBGPPktInfo(msg, err.(*packet.BGPMessageError))
				msgErr = err.(*packet.BGPMessageError)
				msgOk = false
			} else if header.Type == packet.BGPMsgTypeOpen {
				p.peerAttrs.ASSize = packet.GetASSize(msg.Body.(*packet.BGPOpen))
				p.peerAttrs.AddPathFamily = packet.GetAddPathFamily(msg.Body.(*packet.BGPOpen))
				addPathsTxFarEnd := packet.IsAddPathsTxEnabledForIPv4(p.peerAttrs.AddPathFamily)
				p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress,
					"Far end can send add paths"))
				if addPathsTxFarEnd && p.fsm.pConf.AddPathsRx {
					p.peerAttrs.AddPathsRxActual = true
					p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress,
						"negotiated to recieve add paths from far end"))
				}
			}
			p.fsm.pktRxCh <- packet.NewBGPPktInfo(msg, msgErr)
			doneCh <- msgOk

		case <-stopCh:
			p.logger.Info(fmt.Sprintln("Neighbor:", p.fsm.pConf.NeighborAddress, "FSM", p.fsm.id,
				"Closing the peer connection"))
			if p.conn != nil {
				(*p.conn).Close()
			}
			exitCh <- true
			return
		}
	}
}
