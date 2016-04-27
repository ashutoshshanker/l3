// conn_test.go
package fsmtest

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"l3/bgp/baseobjects"
	"l3/bgp/config"
	"l3/bgp/fsm"
	"l3/bgp/packet"
	"math"
	"testing"
	"utils/logging"
)

func TestDecodeMessageBGPOpen(t *testing.T) {
	strPkts := make([]string, 0)
	strPkts = append(strPkts, "045ba0000a0a0a00c224020641041908b10a02060104000100010202020002080300050001010100020440028000")
	strPkts = append(strPkts, "045ba0000a0a0a00c21c020641041908b10a0206010400010001020202000206400280008005")
	strPkts = append(strPkts, "045ba0000a0a0a00c265020641041908b10a020601040001000102020200020440028000024901ffffffffffffffffffffffffffffffffffffffffffffffffffffffff"+
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	strPkts = append(strPkts, "045ba0000a0a0a00c21a020641041908b10a0206010400010001020f0200020440028000")

	logger, err := logging.NewLogger("bgpd", "BGP", true)
	if err != nil {
		t.Fatal("Failed to start the logger. Exiting!!")
	}

	gConf := &config.GlobalConfig{}
	var peerGroup *config.PeerGroupConfig = nil
	pConf := config.NeighborConfig{}
	nConf := base.NewNeighborConf(logger, gConf, peerGroup, pConf)
	fsmMgr := fsm.NewFSMManager(logger, nConf, make(chan *packet.BGPPktSrc), make(chan fsm.PeerFSMConn),
		make(chan config.ReachabilityInfo))
	stateMachine := fsm.NewFSM(fsmMgr, 0, nConf)
	peerConn := fsm.NewPeerConn(stateMachine, config.ConnDirOut, nil)

	for _, strPkt := range strPkts {
		hexPkt, err := hex.DecodeString(strPkt)
		fmt.Printf("packet = %x, len = %d\n", hexPkt, len(hexPkt))
		if err != nil {
			t.Fatal("Failed to decode the string to hex, string =", strPkt)
		}

		if len(hexPkt) > math.MaxUint16 {
			t.Fatal("Length of packet exceeded MAX size, packet len =", len(hexPkt))
		}

		pktLen := make([]byte, 2)
		binary.BigEndian.PutUint16(pktLen, uint16(len(hexPkt)+19))
		header := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x01}
		copy(header[16:18], pktLen)
		fmt.Printf("packet header = %x, len = %d\n", header, len(header))

		bgpHeader := packet.NewBGPHeader()
		err = bgpHeader.Decode(header)
		if err != nil {
			t.Fatal("BGP packet header decode failed with error", err)
		}

		_, bgpMsgErr, msgOK := peerConn.DecodeMessage(bgpHeader, hexPkt)
		if bgpMsgErr == nil || msgOK {
			t.Fatal("BGP open message decode called... expected BGP Message error, got bgp message err:", bgpMsgErr, "message OK:", msgOK)
		} else {
			t.Log("BGP open message decode called... expected BGP Message error, error:", bgpMsgErr, "message OK:", msgOK)
		}
	}
}
