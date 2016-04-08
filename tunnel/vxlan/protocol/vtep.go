// vtepdb.go
package vxlan

import (
	"encoding/json"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"io/ioutil"
	"net"
	"time"
)

type vtepDbKey struct {
	VtepId uint32
}

var vtepDB map[vtepDbKey]*VtepDbEntry

var VxlanVtepSrcIp net.IP
var VxlanVtepSrcNetMac net.HardwareAddr
var VxlanVtepSrcMac [6]uint8

type vtepStatus string

const (
	VtepStatusUp         vtepStatus = "UP"
	VtepStatusDown                  = "DOWN"
	VtepStatusAdminDown             = "ADMIN DOWN"
	VtepStatusIncomplete            = "INCOMPLETE PROV"
)

type VtepDbEntry struct {
	VtepId                uint32
	VxlanId               uint32
	VtepName              string
	SrcIfName             string
	UDP                   uint16
	TTL                   uint16
	TOS                   uint16
	InnerVlanHandlingMode int32
	Learning              bool
	Rsc                   bool
	L2miss                bool
	L3miss                bool
	SrcIp                 net.IP
	DstIp                 net.IP
	VlanId                uint16
	SrcMac                net.HardwareAddr
	DstMac                net.HardwareAddr
	Status                vtepStatus
	VtepIfIndex           int32

	// handle used to rx/tx packets to linux if
	handle *pcap.Handle

	rxpkts uint64
	txpkts uint64
}

func (vtep *VtepDbEntry) GetRxStats() uint64 {
	return vtep.rxpkts
}

func (vtep *VtepDbEntry) GetTxStats() uint64 {
	return vtep.txpkts
}

func GetVtepDB() map[vtepDbKey]*VtepDbEntry {
	return vtepDB
}

/* TODO may need to keep a table to map customer macs to vtep
type srcMacVtepMap struct {
	SrcMac      net.HardwareAddr
	VtepIfIndex int32
}
*/

func NewVtepDbEntry(c *VtepConfig) *VtepDbEntry {
	vtep := &VtepDbEntry{
		VtepId:    c.VtepId,
		VxlanId:   c.VxlanId,
		VtepName:  c.VtepName + "Int",
		SrcIfName: c.SrcIfName,
		UDP:       c.UDP,
		TTL:       c.TTL,
		TOS:       c.TOS,
		InnerVlanHandlingMode: c.InnerVlanHandlingMode,
		Learning:              c.Learning,
		Rsc:                   c.Rsc,
		L2miss:                c.L2miss,
		L3miss:                c.L3miss,
		DstIp:                 c.TunnelDstIp,
		SrcIp:                 c.TunnelSrcIp,
		SrcMac:                c.TunnelSrcMac,
		DstMac:                c.TunnelDstMac,
		VlanId:                c.VlanId,
	}
	return vtep
}

func CreateVtep(c *VtepConfig) *VtepDbEntry {

	CreatePort(c.SrcIfName, c.UDP)

	vtep := saveVtepConfigData(c)

	// create vtep resources in hw
	asicDCreateVtep(c)

	// lets create the packet listener for the rx/tx packets
	vtep.createVtepSenderListener()

	return vtep
}

func DeleteVtep(c *VtepConfig) {

	key := vtepDbKey{
		VtepId: c.VtepId,
	}

	if vtep, ok := vtepDB[key]; ok {
		vtep.handle.Close()
		delete(vtepDB, key)
	}
	time.Sleep(2 * time.Second)
	// create vtep resources in hw
	asicDDeleteVtep(c)

	DeletePort(c.SrcIfName, c.UDP)

}

func saveVtepConfigData(c *VtepConfig) *VtepDbEntry {
	key := vtepDbKey{
		VtepId: c.VtepId,
	}
	vtep, ok := vtepDB[key]
	if !ok {
		vtep = NewVtepDbEntry(c)
		vtepDB[key] = vtep
	}
	return vtep
}

func SaveVtepSrcMacSrcIp(paramspath string) {
	var cfgFile cfgFileJson
	asicdconffilename := paramspath + "asicd.conf"
	cfgFileData, err := ioutil.ReadFile(asicdconffilename)
	if err != nil {
		logger.Info("Error reading config file - asicd.conf")
		return
	}
	err = json.Unmarshal(cfgFileData, &cfgFile)
	if err != nil {
		logger.Info("Error parsing config file")
		return
	}

	VxlanVtepSrcNetMac, _ := net.ParseMAC(cfgFile.SwitchMac)
	VxlanVtepSrcMac = [6]uint8{VxlanVtepSrcNetMac[0], VxlanVtepSrcNetMac[1], VxlanVtepSrcNetMac[2], VxlanVtepSrcNetMac[3], VxlanVtepSrcNetMac[4], VxlanVtepSrcNetMac[5]}

}

func (vtep *VtepDbEntry) createVtepSenderListener() error {

	handle, err := pcap.OpenLive(vtep.VtepName, 65536, false, 50*time.Millisecond)
	if err != nil {
		logger.Err(fmt.Sprintf("%s: Error opening pcap.OpenLive %s", vtep.VtepName, err))
		return err
	}
	logger.Info(fmt.Sprintf("Creating VXLAN Listener for intf ", vtep.VtepName))
	vtep.handle = handle
	src := gopacket.NewPacketSource(vtep.handle, layers.LayerTypeEthernet)
	in := src.Packets()

	go func(rxchan chan gopacket.Packet) {
		for {
			select {
			case packet, ok := <-rxchan:
				if ok {
					go vtep.encapAndDispatchPkt(packet)
				} else {
					// channel closed
					return
				}
			}
		}
	}(in)

	return nil
}

func (vtep *VtepDbEntry) decapAndDispatchPkt(packet gopacket.Packet) {

	vxlanLayer := packet.Layer(layers.LayerTypeVxlan)
	if vxlanLayer != nil {
		vxlan := vxlanLayer.(*layers.VXLAN)
		buf := vxlan.LayerPayload()
		logger.Info(fmt.Sprintf("Sending Packet to %s %#v", vtep.VtepName, buf))
		if err := vtep.handle.WritePacketData(buf); err != nil {
			logger.Err("Error writing packet to interface")
		}
	}
}

func (vtep *VtepDbEntry) encapAndDispatchPkt(packet gopacket.Packet) {
	// every vtep is tied to a port
	if p, ok := portDB[vtep.SrcIfName]; ok {
		phandle := p.handle
		// outer ethernet header
		eth := layers.Ethernet{
			SrcMAC:       vtep.SrcMac,
			DstMAC:       vtep.DstMac,
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
			SrcIP:      vtep.SrcIp,
			DstIP:      vtep.DstIp,
		}

		udp := layers.UDP{
			SrcPort: layers.UDPPort(vtep.UDP), // TODO need a src port
			DstPort: layers.UDPPort(vtep.UDP),
			Length:  100,
		}
		udp.SetNetworkLayerForChecksum(&ip)

		vxlan := layers.VXLAN{
			BaseLayer: layers.BaseLayer{
				Payload: packet.Data(),
			},
			Flags: 0x08,
		}
		vxlan.SetVNI(vtep.VxlanId)

		// Set up buffer and options for serialization.
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}
		// Send one packet for every address.
		gopacket.SerializeLayers(buf, opts, &eth, &ip, &udp, &vxlan)
		logger.Info(fmt.Sprintf("Rx Packet now encapsulating and sending packet to if", vtep.SrcIfName, buf))
		if err := phandle.WritePacketData(buf.Bytes()); err != nil {
			logger.Err("Error writing packet to interface")
			return
		}
		vtep.txpkts++
	}
}
