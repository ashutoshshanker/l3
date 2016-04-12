// port.go
package vxlan

import (
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"time"
)

var portDB map[string]*VxlanPort

type VxlanPort struct {
	IfName string
	// only going to listen for specific vxlan ports
	// IANA 4789, Linux 8472
	UDP        []uint16
	VtepRefCnt uint32
	handle     *pcap.Handle
	rxPkts     uint64
	txPkts     uint64
}

func (p *VxlanPort) GetVtepRefCnt() uint32 {
	return p.VtepRefCnt
}
func (p *VxlanPort) GetRxStats() uint64 {
	return p.rxPkts
}

func (p *VxlanPort) GetTxStats() uint64 {
	return p.txPkts
}

func CreatePort(ifname string, udpport uint16) {

	if p, ok := portDB[ifname]; !ok {
		portDB[ifname] = &VxlanPort{
			IfName: ifname,
			UDP:    make([]uint16, 0),
		}

		p = portDB[ifname]
		p.UDP = append(p.UDP, udpport)
		p.createPortSenderListener()
		p.createVxlanUdpFilter()
	} else {
		p.VtepRefCnt++
		foundUdpPort := false
		for _, udp := range p.UDP {
			if udpport == udp {
				foundUdpPort = true
				break
			}
		}
		if !foundUdpPort {
			p.UDP = append(p.UDP, udpport)
			p.createVxlanUdpFilter()
		}
	}
}

func DeletePort(ifname string, udpport uint16) {
	if p, ok := portDB[ifname]; ok {
		p.VtepRefCnt--
		if p.VtepRefCnt == 0 {
			logger.Info(fmt.Sprintf("Deleting Port %s from vxland", ifname))
			// TODO
			delete(portDB, ifname)
			p.handle.Close()
		}
	}
}

func (p *VxlanPort) createVxlanUdpFilter() error {
	filter := ""
	for i, udp := range p.UDP {
		if i == 0 {
			filter = filter + fmt.Sprintf("udp dst port %d", udp)
		} else {
			filter = filter + fmt.Sprintf("or udp dst port %d", udp)
		}
	}
	if err := p.handle.SetBPFFilter(filter); err != nil {
		logger.Err(fmt.Sprintf("%s: Error setting pcap filter %s %s", p.IfName, filter, err))
		return err
	}
	return nil
}

func (p *VxlanPort) IsMyVtepPkt(packet gopacket.Packet) (*VtepDbEntry, bool) {
	vxlanLayer := packet.Layer(layers.LayerTypeVxlan)
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if vxlanLayer != nil &&
		ethernetLayer != nil &&
		ipLayer != nil {
		eth := ethernetLayer.(*layers.Ethernet)
		ip := ipLayer.(*layers.IPv4)
		vxlan := vxlanLayer.(*layers.VXLAN)
		for _, vtep := range vtepDB {
			// Only support unicast packets for now
			// 1) Dst MAC
			// 2) Dst IP
			// 3) VNI
			logger.Info(fmt.Sprintf("pkg mac %#v config mac %#v", eth.DstMAC, vtep.SrcMac))
			logger.Info(fmt.Sprintf("pkg ip %#v config ip %#v", ip.DstIP, vtep.SrcIp))
			logger.Info(fmt.Sprintf("pkt vni %#v config vni %d", vxlan.VNI, vtep.VxlanId))
			if bytes.Compare(eth.DstMAC, vtep.SrcMac) == 0 &&
				bytes.Compare(ip.DstIP, vtep.SrcIp) == 0 &&
				CompareVNI(vtep.VxlanId, vxlan.VNI) == 0 {
				return vtep, true
			}
			logger.Warning(fmt.Sprintf("%s: Received VXLAN packet not meant for us %s", p.IfName, packet))

		}
	} else {
		logger.Warning(fmt.Sprintf("%s: Received improper VXLAN packet %s", p.IfName, packet))
	}
	return nil, false
}

func (p *VxlanPort) createPortSenderListener() error {

	handle, err := pcap.OpenLive(p.IfName, 65536, false, 50*time.Millisecond)
	if err != nil {
		logger.Err(fmt.Sprintf("%s: Error opening pcap.OpenLive %s", p.IfName, err))
		return err
	}
	// only want to capture ingress frames
	handle.SetDirection(pcap.DirectionIn)
	//logger.Info(fmt.Sprintf("Creating VXLAN Listener for intf ", vtep.VtepName, "with filter", filter))
	logger.Info(fmt.Sprintf("Creating VXLAN Port Listener for intf ", p.IfName))
	p.handle = handle
	src := gopacket.NewPacketSource(p.handle, layers.LayerTypeEthernet)
	in := src.Packets()

	go func(rxchan chan gopacket.Packet) {
		for {
			select {
			case packet, ok := <-rxchan:
				if ok {
					fmt.Println("PORT Rx: ", packet)
					if vtep, ok := p.IsMyVtepPkt(packet); ok {
						//fmt.Println("FOUND MY PACKET: ", packet)
						p.rxPkts++
						go vtep.decapAndDispatchPkt(packet)
					}
					//}
				} else {
					// channel closed
					return
				}
			}
		}
	}(in)

	return nil
}
