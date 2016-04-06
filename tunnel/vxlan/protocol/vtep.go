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
		VtepName:  c.VtepName,
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

	vtep := saveVtepConfigData(c)

	// create vtep resources in hw
	asicDCreateVtep(c)

	// lets create the packet listener for the rx/tx packets
	vtep.createVtepSenderListener()

	return vtep
}

func DeleteVtep(c *VtepConfig) {

	// create vtep resources in hw
	asicDCreateVtep(c)

	key := vtepDbKey{
		VtepId: c.VtepId,
	}

	if vtep, ok := vtepDB[key]; ok {
		vtep.handle.Close()
		delete(vtepDB, key)
	}
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

// lets determine if this packet is for us based on mac/ip/udp/vni
func (vtep *VtepDbEntry) IsVxlanPkMine(packet gopacket.Packet) {

}

func (vtep *VtepDbEntry) IsMyVxlanPkt(packet gopacket.Packet) bool {
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udp := udpLayer.(*layers.UDP)
		if udp.DstPort == layers.UDPPort(vtep.UDP) {
			vxlanLayer := packet.Layer(layers.LayerTypeVxlan)
			if vxlanLayer != nil {
				vxlan := vxlanLayer.(*layers.VXLAN)
				logger.Info(fmt.Sprintf("Vxlan packet Rx: %#v looking for VNI %d", vxlan, vtep.VxlanId))
				if vxlan.VNI[0] == uint8(vtep.VxlanId>>16&0xff) &&
					vxlan.VNI[1] == uint8(vtep.VxlanId>>8&0xff) &&
					vxlan.VNI[2] == uint8(vtep.VxlanId>>0&0xff) {
					return true
				} else {
					logger.Warning(fmt.Sprintf("%s: Received VXLAN packet whos VNI %#v was not mine %d %s", vtep.VtepName, vxlan.VNI, vtep.VxlanId, packet))
				}
			} else {
				logger.Warning(fmt.Sprintf("%s: Received VXLAN packet without VXLAN header %s", vtep.VtepName, packet))
			}
		} else {
			logger.Warning(fmt.Sprintf("%s: Received UDP %d packet expected %d %s", vtep.VtepName, udp.DstPort, vtep.UDP, packet))
		}
	} else {
		logger.Warning(fmt.Sprintf("%s: Received non-UDP packet %s ", vtep.VtepName, packet))
	}
	return false
}

func (vtep *VtepDbEntry) createVtepSenderListener() error {

	handle, err := pcap.OpenLive(vtep.VtepName, 65536, false, 50*time.Millisecond)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Creating VXLAN Listener for intf", vtep.VtepName))
	vtep.handle = handle
	src := gopacket.NewPacketSource(vtep.handle, layers.LayerTypeEthernet)
	in := src.Packets()

	go func(rxchan chan gopacket.Packet) {
		for {
			select {
			case packet, ok := <-rxchan:
				if ok {
					if vtep.IsMyVxlanPkt(packet) {
						fmt.Println(packet)
						vtep.rxpkts++
					}
				} else {
					// channel closed
					return
				}
			}
		}
	}(in)

	return nil
}

func (vtep *VtepDbEntry) GetRxStats() uint64 {
	return vtep.rxpkts
}

func (vtep *VtepDbEntry) GetTxStats() uint64 {
	return vtep.txpkts
}
