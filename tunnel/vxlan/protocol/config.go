// config.go
// Config entry is based on thrift data structures.
package vxlan

import (
	"net"
	"reflect"
	"vxland"
)

type VxLanChannels struct {
	Vxlancreate chan vxland.VxlanInstance
	Vxlandelete chan vxland.VxlanInstance
	Vxlanupdate chan VxlanUpdate
	Vtepcreate  chan vxland.VxlanVtepInstances
	Vtepdelete  chan vxland.VxlanVtepInstances
	Vtepupdate  chan VtepUpdate
}

type VxlanUpdate struct {
	Oldconfig vxland.VxlanInstance
	Newconfig vxland.VxlanInstance
	Attr      []bool
}

type VtepUpdate struct {
	Oldconfig vxland.VxlanVtepInstances
	Newconfig vxland.VxlanVtepInstances
	Attr      []bool
}

// bridge for the VNI
type VxlanConfig struct {
	VNI    uint32
	VlanId uint16 // used to tag inner ethernet frame when egressing
	Group  net.IP // multicast group IP
	MTU    uint32 // MTU size for each VTEP
}

// tunnel endpoint for the VxLAN
type VtepConfig struct {
	VtepId                uint32           `SNAPROUTE: KEY` //VTEP ID.
	VxlanId               uint32           `SNAPROUTE: KEY` //VxLAN ID.
	VtepName              string           //VTEP instance name.
	SrcIfIndex            int32            //Source interface ifIndex.
	UDP                   uint16           //vxlan udp port.  Deafult is the iana default udp port
	TTL                   uint16           //TTL of the Vxlan tunnel
	TOS                   uint16           //Type of Service
	InnerVlanHandlingMode bool             //The inner vlan tag handling mode.
	Learning              bool             //specifies if unknown source link layer  addresses and IP addresses are entered into the VXLAN  device forwarding database.
	Rsc                   bool             //specifies if route short circuit is turned on.
	L2miss                bool             //specifies if netlink LLADDR miss notifications are generated.
	L3miss                bool             //specifies if netlink IP ADDR miss notifications are generated.
	TunnelSourceIp        net.IP           //Source IP address for the static VxLAN tunnel
	TunnelDestinationIp   net.IP           //Destination IP address for the static VxLAN tunnel
	VlanId                uint16           //Vlan Id to encapsulate with the vtep tunnel ethernet header
	SrcMac                net.HardwareAddr //Src Mac assigned to the VTEP within this VxLAN. If an address is not assigned the the local switch address will be used.
}

func ConvertInt32ToBool(val int32) bool {
	if val == 0 {
		return false
	}
	return true
}

func (s *VXLANServer) ConvertVxlanInstanceToVxlanConfig(c *vxland.VxlanInstance) *VxlanConfig {

	return &VxlanConfig{
		VNI:    uint32(c.VxlanId),
		VlanId: uint16(c.VlanId),
		Group:  net.ParseIP(c.McDestIp),
		MTU:    uint32(c.Mtu),
	}
}

func (s *VXLANServer) ConvertVxlanVtepInstanceToVtepConfig(c *vxland.VxlanVtepInstances) *VtepConfig {
	mac, err := net.ParseMAC(c.SrcMac)
	if err != nil {
		mac = NetSwitchMac
	}
	return &VtepConfig{
		VtepId:     uint32(c.VtepId),
		VxlanId:    uint32(c.VxlanId),
		VtepName:   string(c.VtepName),
		SrcIfIndex: int32(c.SrcIfIndex),
		UDP:        uint16(c.UDP),
		TTL:        uint16(c.TTL),
		TOS:        uint16(c.TOS),
		InnerVlanHandlingMode: ConvertInt32ToBool(c.InnerVlanHandlingMode),
		Learning:              ConvertInt32ToBool(c.Learning),
		Rsc:                   ConvertInt32ToBool(c.Rsc),
		L2miss:                ConvertInt32ToBool(c.L2miss),
		L3miss:                ConvertInt32ToBool(c.L3miss),
		TunnelSourceIp:        net.ParseIP(c.TunnelSourceIp),
		TunnelDestinationIp:   net.ParseIP(c.TunnelDestinationIp),
		VlanId:                uint16(c.VlanId),
		SrcMac:                mac,
	}
}

func (s *VXLANServer) updateThriftVxLAN(c *VxlanUpdate) {
	objTyp := reflect.TypeOf(c.Oldconfig)

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		if c.Attr[i] {

			if objName == "VxlanId" {
				// TODO
			}
			if objName == "McDestIp" {
				// TODO
			}
			if objName == "VlanId" {
				// TODO
			}
			if objName == "Mtu" {
				// TODO
			}
		}
	}
}

func (s *VXLANServer) updateThriftVtep(c *VtepUpdate) {
	objTyp := reflect.TypeOf(c.Oldconfig)

	// important to note that the attrset starts at index 0 which is the BaseObj
	// which is not the first element on the thrift obj, thus we need to skip
	// this attribute
	for i := 0; i < objTyp.NumField(); i++ {
		objName := objTyp.Field(i).Name
		if c.Attr[i] {

			if objName == "InnerVlanHandlingMode" {
				// TODO
			}
			if objName == "UDP" {
				// TODO
			}
			if objName == "TunnelSourceIp" {
				// TODO
			}
			if objName == "SrcMac" {
				// TODO
			}
			if objName == "L2miss" {
				// TODO
			}
			if objName == "TOS" {
				// TODO
			}
			if objName == "VxlanId" {
				// TODO
			}
			if objName == "VtepName" {
				// TODO
			}
			if objName == "VlanId" {
				// TODO
			}
			if objName == "Rsc" {
				// TODO
			}
			if objName == "VtepId" {
				// TODO
			}
			if objName == "SrcIfIndex" {
				// TODO
			}
			if objName == "L3miss" {
				// TODO
			}
			if objName == "Learning" {
				// TODO
			}
			if objName == "TTL" {
				// TODO
			}
			if objName == "TunnelDestinationIp" {
				// TODO
			}
		}
	}
}

func (s *VXLANServer) StartConfigListener() {

	s.Configchans = &VxLanChannels{
		Vxlancreate: make(chan vxland.VxlanInstance, 0),
		Vxlandelete: make(chan vxland.VxlanInstance, 0),
		Vxlanupdate: make(chan VxlanUpdate, 0),
		Vtepcreate:  make(chan vxland.VxlanVtepInstances, 0),
		Vtepdelete:  make(chan vxland.VxlanVtepInstances, 0),
		Vtepupdate:  make(chan VtepUpdate, 0),
	}

	softswitch := &VxlanLinux{}

	go func(cc *VxLanChannels, ss *VxlanLinux) {
		for {
			select {
			case vxlan := <-cc.Vxlancreate:
				ss.createVxLAN(s.ConvertVxlanInstanceToVxlanConfig(&vxlan))

			case vxlan := <-cc.Vxlandelete:
				ss.deleteVxLAN(s.ConvertVxlanInstanceToVxlanConfig(&vxlan))

			case vxlan := <-cc.Vxlanupdate:
				s.updateThriftVxLAN(&vxlan)

			case vtep := <-cc.Vtepcreate:
				ss.createVtep(s.ConvertVxlanVtepInstanceToVtepConfig(&vtep))

			case vtep := <-cc.Vtepdelete:
				ss.deleteVtep(s.ConvertVxlanVtepInstanceToVtepConfig(&vtep))

			case vtep := <-cc.Vtepupdate:
				s.updateThriftVtep(&vtep)
			}
		}
	}(s.Configchans, softswitch)
}
