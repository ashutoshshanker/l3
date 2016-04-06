package vxlan_test

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/vishvananda/netlink"
	vxlan "l3/tunnel/vxlan/protocol"
	"net"
	"testing"
	"time"
	"utils/logging"
)

var logger *logging.Writer

func Setup() {
	logger, _ = logging.NewLogger("./", "vxland", "TEST")
	vxlan.SetLogger(logger)
}

func CreateTestLbPort(name string) error {

	var linkAttrs netlink.LinkAttrs
	//create loopbacki/f
	liLink, err := netlink.LinkByName(name)
	if err != nil {
		linkAttrs.Name = name
		linkAttrs.Flags = net.FlagLoopback
		linkAttrs.HardwareAddr = net.HardwareAddr{
			0x00, 0x00, 0x64, 0x01, 0x01, 0x01,
		}
		liLink = &netlink.Dummy{linkAttrs} //,"loopback"}
		err = netlink.LinkAdd(liLink)
		if err != nil {
			logger.Err(fmt.Sprintf("SS: LinkAdd call failed during CreateTestLbPort() ", err))
			return err
		}
		time.Sleep(5 * time.Second)
		link, err2 := netlink.LinkByName(name)
		if err2 != nil {
			logger.Err(fmt.Sprintf("SS: 2 LinkByName call failed during CreateTestLbPort()", err2))
			return err2
		}
		err = netlink.LinkSetUp(link)
		if err != nil {
			logger.Err(fmt.Sprintf("SS: LinkSetUp call failed during CreateTestLbPort()", err))
			return err
		}
	}
	// need to delay some time to let the interface create to happen
	time.Sleep(5 * time.Second)
	return nil
}

func CreateTestTxHandle(ifname string) *pcap.Handle {
	handle, err := pcap.OpenLive(ifname, 65536, false, 50*time.Millisecond)
	if err != nil {
		logger.Err(fmt.Sprintf("SS: FAiled during OpenLive()", err))
		return nil
	}
	return handle
}

func CreateVxlanArpFrame(vni [3]uint8) gopacket.SerializeBuffer {
	// send an ARP frame
	// Set up all the layers' fields we can.
	tunnelsrcmac, _ := net.ParseMAC("00:00:64:01:01:02")
	tunneldstmac, _ := net.ParseMAC("00:00:64:01:01:01")
	tunnelsrcip := net.ParseIP("100.1.1.2")
	tunneldstip := net.ParseIP("100.1.1.1")
	// outer ethernet header
	eth := layers.Ethernet{
		SrcMAC:       tunneldstmac,
		DstMAC:       tunnelsrcmac,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := layers.IPv4{
		Version:    4,
		IHL:        20,
		TOS:        0,
		Length:     120,
		Id:         0xd2c0,
		Flags:      layers.IPv4DontFragment, //IPv4Flag
		FragOffset: 0,                       //uint16
		TTL:        255,
		Protocol:   layers.IPProtocolUDP, //IPProtocol
		SrcIP:      tunnelsrcip,
		DstIP:      tunneldstip,
	}

	udp := layers.UDP{
		SrcPort: 4789,
		DstPort: 4789,
		Length:  100,
	}
	udp.SetNetworkLayerForChecksum(&ip)

	vxlan := layers.VXLAN{
		Flags: 0x08,
		VNI:   vni,
	}

	dstmac, _ := net.ParseMAC("FF:FF:FF:FF:FF:FF")
	srcmac, _ := net.ParseMAC("00:01:02:03:04:05")

	// inner ethernet header
	ieth := layers.Ethernet{
		SrcMAC:       dstmac,
		DstMAC:       srcmac,
		EthernetType: layers.EthernetTypeARP,
	}

	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeARP,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         1,
		SourceHwAddress:   []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
		SourceProtAddress: []byte{0xA, 0x01, 0x01, 0x01},
		DstHwAddress:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    []byte{0x14, 0x01, 0x01, 0x011},
	}

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, &vxlan, &ieth, &arp)

	p := gopacket.NewPacket(buf.Bytes(), layers.LinkTypeEthernet, gopacket.Default)
	fmt.Println("created packet", p)
	return buf
}

func SendPacket(handle *pcap.Handle, buf gopacket.SerializeBuffer, t *testing.T) {
	if err := handle.WritePacketData(buf.Bytes()); err != nil {
		t.Error("Error writing packet to interface")
		t.FailNow()
	}
}

func TestRxArpPacket(t *testing.T) {

	Setup()

	// setup
	vteplbname := "lo"
	srcip := net.ParseIP("100.1.1.1")
	srcmac, _ := net.ParseMAC("00:00:64:01:01:01")
	dstip := net.ParseIP("100.1.1.2")
	dstmac, _ := net.ParseMAC("00:00:64:01:01:02")

	vxlanconfig := &vxlan.VxlanConfig{
		VNI:    500,
		VlanId: 100, // used to tag inner ethernet frame when egressing
		MTU:    1550,
	}

	vxlan.CreateVxLAN(vxlanconfig)

	vtepconfig := &vxlan.VtepConfig{
		VtepId:    10,
		VxlanId:   500,
		VtepName:  "vtep10",
		SrcIfName: vteplbname,
		UDP:       4789,
		TTL:       255,
		TOS:       0,
		InnerVlanHandlingMode: 0,
		Learning:              false,
		Rsc:                   false,
		L2miss:                false,
		L3miss:                false,
		TunnelSrcIp:           srcip,
		TunnelDstIp:           dstip,
		VlanId:                100,
		TunnelSrcMac:          srcmac,
		TunnelDstMac:          dstmac,
	}

	if vteplbname != "lo" {
		// create linux loopback interface to which the vtep will be associated with
		err := CreateTestLbPort(vteplbname)
		if err != nil {
			t.Error("Failed to Create test looopback interface")
			t.FailNow()
		}
	}

	handle := CreateTestTxHandle(vteplbname)
	if handle == nil {
		t.Error("Failed to Create pcap handle")
		t.FailNow()
	}

	// create vtep interface and which will listen on vtep interface
	vtep := vxlan.CreateVtep(vtepconfig)

	// send an ARP frame
	// Set up all the layers' fields we can.
	arppktbuf := CreateVxlanArpFrame([3]uint8{uint8(vtepconfig.VxlanId >> 16 & 0xff), uint8(vtepconfig.VxlanId >> 8 & 0xff), uint8(vtepconfig.VxlanId >> 0 & 0xff)})
	fmt.Println("Sending packet")
	SendPacket(handle, arppktbuf, t)

	time.Sleep(15 * time.Second)
	if vtep.GetRxStats() == 0 {
		t.Error("Failed to Receive a packet")
		t.FailNow()
	}

}
