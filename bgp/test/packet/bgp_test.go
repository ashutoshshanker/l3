//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

// bgp_test.go
package packettest

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"l3/bgp/packet"
	"math"
	"testing"
)

func TestBGPUpdatePacketsSliceBoundOutOfRange(t *testing.T) {
	strPkts := make([]string, 0)
	strPkts = append(strPkts, "000000204001010140020602011908b10a4003040a0a00c280040400000000c01c000100080a")
	strPkts = append(strPkts, "0000002a400101015002feff02011908b10a4003040a0a00c280040400000000800e0b000101040a0a00c200080a")
	strPkts = append(strPkts, "00000401c0fcfe020080000008800000108000001880000020800000288000003080000038800000408000004880000050800000588000006080000068800000708000"+
		"007880000080800000888000009080000098800000a0800000a8800000b0800000b8800000c0800000c8800000d0800000d8800000e0800000e8800000f0800000f8800001008000010880000110800001188000012080000128800001308000013880000140800001488000015"+
		"0800001588000016080000168800001708000017880000180800001888000019080000198800001a0800001a8800001b0800001b8800001c0800001c8800001d0800001d8800001e0800001e8800001f0800001f880000200800002088000021080000218800002208000022880"+
		"00023080000238800002408000024880000250800002588000026080000268800002708000027880000280800002888000029080000298800002a0800002a8800002b0800002b8800002c0800002c8800002d0800002d8800002e0800002e8800002f0800002f88000030080000"+
		"308800003108000031880000320800003288000033080000338800003408000034880000350800003588000036080000368800003708000037880000380800003888000039080000398800003a0800003a8800003b0800003b8800003c0800003c8800003d0800003d8800003e0"+
		"800003e8800003f0800003f88000040080000408800004108000041880000420800004288000043080000438800004408000044880000450800004588000046080000468800004708000047880000480800004888000049080000498800004a0800004a8800004b0800004b8800"+
		"004c0800004c8800004d0800004d8800004e0800004e8800004f0800004f880000500800005088000051080000518800005208000052880000530800005388000054080000548800005508000055880000560800005688000057080000578800005808000058880000590800005"+
		"98800005a0800005a8800005b0800005b8800005c0800005c8800005d0800005d8800005e0800005e8800005f0800005f88000060080000608800006108000061880000620800006288000063080000638800006408000064880000650800006588000066080000668800006708"+
		"000067880000680800006888000069080000698800006a0800006a8800006b0800006b8800006c0800006c8800006d0800006d8800006e0800006e8800006f0800006f8800007008000070880000710800007188000072080000728800007308000073880000740800007488000"+
		"0750800007588000076080000768800007708000077880000780800007888000079080000798800007a0800007a8800007b0800007b8800007c0800007c8800007d0800007d8800007e0800007e8800007f0800007f8")
	strPkts = append(strPkts, "0000000940022a02011908b10a")
	strPkts = append(strPkts, "0000042b4001010140020602011908b10ad011000802008000000880000010800000188000002080000028800000308000003880000040800000488000005080000058"+
		"8000006080000068800000708000007880000080800000888000009080000098800000a0800000a8800000b0800000b8800000c0800000c8800000d0800000d8800000e0800000e8800000f0800000f880000100800001088000011080000118800001208000012880000130800"+
		"00138800001408000014880000150800001588000016080000168800001708000017880000180800001888000019080000198800001a0800001a8800001b0800001b8800001c0800001c8800001d0800001d8800001e0800001e8800001f0800001f88000020080000208800002"+
		"108000021880000220800002288000023080000238800002408000024880000250800002588000026080000268800002708000027880000280800002888000029080000298800002a0800002a8800002b0800002b8800002c0800002c8800002d0800002d8800002e0800002e88"+
		"00002f0800002f88000030080000308800003108000031880000320800003288000033080000338800003408000034880000350800003588000036080000368800003708000037880000380800003888000039080000398800003a0800003a8800003b0800003b8800003c08000"+
		"03c8800003d0800003d8800003e0800003e8800003f0800003f88000040080000408800004108000041880000420800004288000043080000438800004408000044880000450800004588000046080000468800004708000047880000480800004888000049080000498800004a"+
		"0800004a8800004b0800004b8800004c0800004c8800004d0800004d8800004e0800004e8800004f0800004f88000050080000508800005108000051880000520800005288000053080000538800005408000054880000550800005588000056080000568800005708000057880"+
		"000580800005888000059080000598800005a0800005a8800005b0800005b8800005c0800005c8800005d0800005d8800005e0800005e8800005f0800005f8800006008000060880000610800006188000062080000628800006308000063880000640800006488000065080000"+
		"6588000066080000668800006708000067880000680800006888000069080000698800006a0800006a8800006b0800006b8800006c0800006c8800006d0800006d8800006e0800006e8800006f0800006f880000700800007088000071080000718800007208000072880000730"+
		"80000738800007408000074880000750800007588000076080000768800007708000077880000780800007888000079080000798800007a0800007a8800007b0800007b8800007c0800007c8800007d0800007d8800007e0800007e8800007f0800007f84003040a0a00c280040"+
		"400000000800e0b000101040a0a00c200080a")
	strPkts = append(strPkts, "000000204001010140020602011908b10a4003040a0a00c280040400000000c01c000100080a")
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
			0x00, 0x00, 0x00, 0x00, 0x02}
		copy(header[16:18], pktLen)
		fmt.Printf("packet header = %x, len = %d\n", header, len(header))

		bgpHeader := packet.NewBGPHeader()
		err = bgpHeader.Decode(header)
		if err != nil {
			t.Fatal("BGP packet header decode failed with error", err)
		}

		peerAttrs := packet.BGPPeerAttrs{
			ASSize:           4,
			AddPathsRxActual: true,
		}
		bgpMessage := packet.NewBGPMessage()
		err = bgpMessage.Decode(bgpHeader, hexPkt, peerAttrs)
		if err == nil {
			t.Fatal("BGP update message decode called... expected failure, got NO error")
		} else {
			t.Log("BGP update message decode called... expected failure, error:", err)
		}
	}
}

func TestBGPOpenPacketsIndexOutOfRange(t *testing.T) {
	strPkts := make([]string, 0)
	strPkts = append(strPkts, "045ba0000a0a0a00c21d020682070001010101f003020601040001000102020200020440020000")
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

		peerAttrs := packet.BGPPeerAttrs{
			ASSize:           2,
			AddPathsRxActual: false,
		}
		bgpMessage := packet.NewBGPMessage()
		err = bgpMessage.Decode(bgpHeader, hexPkt, peerAttrs)
		if err == nil {
			t.Fatal("BGP open message decode called... expected failure, got NO error")
		} else {
			t.Log("BGP open message decode called... expected failure, error:", err)
		}
	}
}

func TestBGPUpdatePathAttrsBadFlags(t *testing.T) {
	strPkts := make([]string, 0)
	strPkts = append(strPkts, "0000001c40010100100200060201000002584003045a01010280040400000000183c010118500101184701011846010218460101183c0102")
	strPkts = append(strPkts, "0000001c40010100500200060201000002582003045a01010280040400000000183c010118500101184701011846010218460101183c0102")
	strPkts = append(strPkts, "0000001c40010100500200060201000002584003045a010102A0040400000000183c010118500101184701011846010218460101183c0102")

	pktPathAttrs := "0000002040010100500200060201000002584003045a01010280040400000000"
	nlri := "183c010118500101184701011846010218460101183c0102"
	pathAttrs := []string{"00000100", "20000100", "60000100", "A0000100"}
	for _, pa := range pathAttrs {
		pa = pa[:2] + fmt.Sprintf("%02x", packet.BGPPathAttrTypeUnknown) + pa[4:]
		strPkts = append(strPkts, pktPathAttrs+pa+nlri)
	}

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
			0x00, 0x00, 0x00, 0x00, 0x02}
		copy(header[16:18], pktLen)
		fmt.Printf("packet header = %x, len = %d\n", header, len(header))

		bgpHeader := packet.NewBGPHeader()
		err = bgpHeader.Decode(header)
		if err != nil {
			t.Fatal("BGP packet header decode failed with error", err)
		}

		peerAttrs := packet.BGPPeerAttrs{
			ASSize:           4,
			AddPathsRxActual: true,
		}
		bgpMessage := packet.NewBGPMessage()
		err = bgpMessage.Decode(bgpHeader, hexPkt, peerAttrs)
		if err == nil {
			t.Error("BGP update message decode called... expected failure, got NO error")
		} else {
			t.Log("BGP update message decode called... expected failure, error:", err)
		}
	}
}

func TestBGPUpdatePathAttrsBadLength(t *testing.T) {
	strPkts := make([]string, 0)

	pktPathAttrs := "0000001c40010100500200060201000002584003045a01010280040400000000"
	nlri := "183c010118500101184701011846010218460101183c0102"
	pathAttrs := []string{"80000100"}
	for _, pa := range pathAttrs {
		pa = pa[:2] + fmt.Sprintf("%02x", packet.BGPPathAttrTypeUnknown) + pa[4:]
		strPkts = append(strPkts, pktPathAttrs+pa+nlri)
	}

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
			0x00, 0x00, 0x00, 0x00, 0x02}
		copy(header[16:18], pktLen)
		fmt.Printf("packet header = %x, len = %d\n", header, len(header))

		bgpHeader := packet.NewBGPHeader()
		err = bgpHeader.Decode(header)
		if err != nil {
			t.Fatal("BGP packet header decode failed with error", err)
		}

		peerAttrs := packet.BGPPeerAttrs{
			ASSize:           4,
			AddPathsRxActual: false,
		}
		bgpMessage := packet.NewBGPMessage()
		err = bgpMessage.Decode(bgpHeader, hexPkt, peerAttrs)
		if err == nil {
			t.Error("BGP update message decode called... expected failure, got NO error")
		} else {
			t.Log("BGP update message decode called... expected failure, error:", err)
		}
	}
}
