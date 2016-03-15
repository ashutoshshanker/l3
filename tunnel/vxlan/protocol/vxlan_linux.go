// vxlan_linux.go
package vxlan

import (
	"github.com/vishvananda/netlink"
	"net"
)

var VxlanDB map[uint32]VxlanDbEntry

type VxlanDbEntry struct {
	VNI    uint32
	VlanId uint16 // used to tag inner ethernet frame when egressing
	Group  net.IP // multicast group IP
	MTU    uint32 // MTU size for each VTEP
}

type VxlanLinux struct {
}

func init() {
	VxlanDB = make(map[uint32]VxlanDbEntry)
}

// createVxLAN is the equivalent to creating a bridge in the linux
// The VNI is actually associated with the VTEP so lets just create a bridge
// if necessary
func (v *VxlanLinux) createVxLAN(c *VxlanConfig) {

	if _, ok := VxlanDB[c.VNI]; !ok {
		VxlanDB[c.VNI] = VxlanDbEntry{
			VNI:    c.VNI,
			VlanId: c.VlanId,
			Group:  c.Group,
			MTU:    c.MTU,
		}
	}
}

func (v *VxlanLinux) deleteVxLAN(c *VxlanConfig) {
}

func (v *VxlanLinux) createVtep(c *VtepConfig) {

	vtep := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: c.VtepName,
		},
		VxlanId:      int(c.VxlanId),
		VtepDevIndex: int(c.SrcIfIndex),
		SrcAddr:      c.TunnelSourceIp,
		Group:        VxlanDB[c.VxlanId].Group,
		TTL:          int(c.TTL),
		TOS:          int(c.TOS),
		Learning:     c.Learning,
		Proxy:        false,
		RSC:          c.Rsc,
		L2miss:       false,
		L3miss:       false,
		UDPCSum:      true,
		NoAge:        false,
		GBP:          false,
		Age:          300,
		Limit:        256,
		Port:         int(c.UDP),
		PortLow:      int(c.UDP),
		PortHigh:     int(c.UDP),
	}

	if err := netlink.LinkAdd(vtep); err != nil {
		panic(err)
	}
}

func (v *VxlanLinux) deleteVtep(c *VtepConfig) {
}
