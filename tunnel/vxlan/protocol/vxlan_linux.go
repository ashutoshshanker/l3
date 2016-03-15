// vxlan_linux.go
package vxlan

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
	"utils/logging"
)

var VxlanDB map[uint32]VxlanDbEntry

type VxlanDbEntry struct {
	VNI    uint32
	VlanId uint16 // used to tag inner ethernet frame when egressing
	Group  net.IP // multicast group IP
	MTU    uint32 // MTU size for each VTEP
	Brg    *netlink.Bridge
	Links  []*netlink.Link
}

type VxlanLinux struct {
	logger *logging.Writer
}

func NewVxlanLinux() *VxlanLinux {
	initVxlanDB()
	return &VxlanLinux{}

}

func initVxlanDB() {
	if VxlanDB == nil {
		VxlanDB = make(map[uint32]VxlanDbEntry)
	}
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
			Links:  make([]*netlink.Link, 0),
		}
		// lets create a bridge if it does not exists
		// bridge should be based on the VLAN used by a
		// customer.
		brname := fmt.Sprintf("br%d", c.VNI)
		bridge := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: brname,
				MTU:  int(c.MTU),
			},
		}

		if err := netlink.LinkAdd(bridge); err != nil {
			panic(err)
		}

		link, err := netlink.LinkByName(bridge.Attrs().Name)
		if err != nil {
			panic(err)
		}

		vxlanDbEntry := VxlanDB[c.VNI]
		vxlanDbEntry.Brg = link.(*netlink.Bridge)
		VxlanDB[c.VNI] = vxlanDbEntry
		// lets set the vtep interface to up
		if err := netlink.LinkSetUp(bridge); err != nil {
			panic(err)
		}
	}
}

func (v *VxlanLinux) deleteVxLAN(c *VxlanConfig) {

	if vxlan, ok := VxlanDB[c.VNI]; ok {
		for _, link := range vxlan.Links {
			// lets set the vtep interface to up
			if err := netlink.LinkSetDown(*link); err != nil {
				panic(err)
			}
			if err := netlink.LinkDel(*link); err != nil {
				panic(err)
			}
		}
	}
}

func (v *VxlanLinux) createVtep(c *VtepConfig) {

	vtep := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: c.VtepName,
			//MasterIndex: VxlanDB[c.VxlanId].Brg.Attrs().Index,
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

	//equivalent to linux command:
	// ip link add DEVICE type vxlan id ID [ dev PHYS_DEV  ] [ { group
	//         | remote } IPADDR ] [ local IPADDR ] [ ttl TTL ] [ tos TOS ] [
	//          port MIN MAX ] [ [no]learning ] [ [no]proxy ] [ [no]rsc ] [
	//          [no]l2miss ] [ [no]l3miss ]
	if err := netlink.LinkAdd(vtep); err != nil {
		panic(err)
	}

	link, err := netlink.LinkByName(vtep.Name)
	if err != nil {
		panic(err)
	}

	// equivalent to linux command:
	/* bridge fdb add - add a new fdb entry
	       This command creates a new fdb entry.

	       LLADDR the Ethernet MAC address.

	       dev DEV
	              the interface to which this address is associated.

	              self - the address is associated with a software fdb (default)

	              embedded - the address is associated with an offloaded fdb

	              router - the destination address is associated with a router.
	              Valid if the referenced device is a VXLAN type device and has
	              route shortcircuit enabled.

	      The next command line parameters apply only when the specified device
	      DEV is of type VXLAN.

	       dst IPADDR
	              the IP address of the destination VXLAN tunnel endpoint where
	              the Ethernet MAC ADDRESS resides.

	       vni VNI
	              the VXLAN VNI Network Identifier (or VXLAN Segment ID) to use to
	              connect to the remote VXLAN tunnel endpoint.  If omitted the
	              value specified at vxlan device creation will be used.

	       port PORT
	              the UDP destination PORT number to use to connect to the remote
	              VXLAN tunnel endpoint.  If omitted the default value is used.

	       via DEVICE
	              device name of the outgoing interface for the VXLAN device
	              driver to reach the remote VXLAN tunnel endpoint.


			// values taken from linux/neighbour.h

	if c.TunnelDestinationIp != nil &&
		c.DestHostMac != nil {
		neigh := netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			Family:       7,   // NDA_VNI
			State:        192, // NUD_NOARP (0x40) | NUD_PERMANENT (0x80)
			Type:         1,
			Flags:        2, // NTF_SELF
			IP:           c.TunnelDestinationIp,
			HardwareAddr: c.DestHostMac,
		}
		if err := netlink.NeighAppend(neigh); err != nil {
			panic(err)
		}
	}
	*/

	vxlanDbEntry := VxlanDB[uint32(vtep.VxlanId)]
	vxlanDbEntry.Links = append(vxlanDbEntry.Links, &link)
	VxlanDB[uint32(vtep.VxlanId)] = vxlanDbEntry

	// lets set the vtep interface to up
	if err := netlink.LinkSetUp(link); err != nil {
		panic(err)
	}

	if err := netlink.LinkSetMaster(link, vxlanDbEntry.Brg); err != nil {
		panic(err)
	}

}

func (v *VxlanLinux) deleteVtep(c *VtepConfig) {

	if vxlan, ok := VxlanDB[c.VxlanId]; ok {
		for i, link := range vxlan.Links {
			if (*link).(*netlink.Vxlan).Attrs().Name == c.VtepName {
				vxlan.Links = append(vxlan.Links[:i], vxlan.Links[i+1:]...)
				break
			}
		}
	}

	link, err := netlink.LinkByName(c.VtepName)
	if err != nil {
		panic(err)
	}
	if err := netlink.LinkSetDown(link); err != nil {
		panic(err)
	}

	if err := netlink.LinkDel(link); err != nil {
		panic(err)
	}
}
