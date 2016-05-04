package packettest

import (
	"bytes"
	"l3/vrrp/server"
	"net"
	"testing"
)

func TestVRRPDEncode(t *testing.T) {
	expectedOutput := []byte{0x21, 0x01, 0x64, 0x01, 0x00, 0x01, 0xba, 0x52, 0xc0, 0xa8,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	server := &vrrpServer.VrrpServer{}
	vrrpHeader := vrrpServer.VrrpPktHeader{
		Version:       vrrpServer.VRRP_VERSION2,
		Type:          vrrpServer.VRRP_PKT_TYPE_ADVERTISEMENT,
		VirtualRtrId:  1,
		Priority:      100,
		CountIPv4Addr: 1,
		Rsvd:          vrrpServer.VRRP_RSVD,
		MaxAdverInt:   1,
		CheckSum:      0,
	}
	ip := net.ParseIP("192.168.0.1")
	vrrpHeader.IPv4Addr = append(vrrpHeader.IPv4Addr, ip)
	encoded, _ := server.VrrpEncodeHeader(vrrpHeader)
	if bytes.Compare(expectedOutput, encoded) != 0 {
		t.Error("Encoding vrrp header failed as the bytes are not equal")
	}
}
