// config.go
// Config entry is based on thrift data structures.
package vxlan

import (
	"errors"
	"fmt"
	"l3/tunnel/vxlan/vxlan_linux"
	"net"
	"reflect"
	"vxland"
)

type VxLanConfigChannels struct {
	Vxlancreate chan VxlanConfig
	Vxlandelete chan VxlanConfig
	Vxlanupdate chan VxlanUpdate
	Vtepcreate  chan VtepConfig
	Vtepdelete  chan VtepConfig
	Vtepupdate  chan VtepUpdate
}

type VxlanUpdate struct {
	Oldconfig VxlanConfig
	Newconfig VxlanConfig
	Attr      []bool
}

type VtepUpdate struct {
	Oldconfig VtepConfig
	Newconfig VtepConfig
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
	SrcIfName             string           //Source interface ifIndex.
	UDP                   uint16           //vxlan udp port.  Deafult is the iana default udp port
	TTL                   uint16           //TTL of the Vxlan tunnel
	TOS                   uint16           //Type of Service
	InnerVlanHandlingMode bool             //The inner vlan tag handling mode.
	Learning              bool             //specifies if unknown source link layer  addresses and IP addresses are entered into the VXLAN  device forwarding database.
	Rsc                   bool             //specifies if route short circuit is turned on.
	L2miss                bool             //specifies if netlink LLADDR miss notifications are generated.
	L3miss                bool             //specifies if netlink IP ADDR miss notifications are generated.
	TunnelSrcIp           net.IP           //Source IP address for the static VxLAN tunnel
	TunnelDstIp           net.IP           //Destination IP address for the static VxLAN tunnel
	VlanId                uint16           //Vlan Id to encapsulate with the vtep tunnel ethernet header
	TunnelSrcMac          net.HardwareAddr //Src Mac assigned to the VTEP within this VxLAN. If an address is not assigned the the local switch address will be used.
	TunnelDstMac          net.HardwareAddr
}

func ConvertInt32ToBool(val int32) bool {
	if val == 0 {
		return false
	}
	return true
}

func (s *VXLANServer) ConvertVxlanInstanceToVxlanConfig(c *vxland.VxlanInstance) (*VxlanConfig, error) {

	return &VxlanConfig{
		VNI:    uint32(c.VxlanId),
		VlanId: uint16(c.VlanId),
		Group:  net.ParseIP(c.McDestIp),
		MTU:    uint32(c.Mtu),
	}, nil
}

func (s *VXLANServer) ConvertVxlanVtepInstanceToVtepConfig(c *vxland.VxlanVtepInstances) (*VtepConfig, error) {

	var mac net.HardwareAddr
	var ip net.IP
	var name string
	var ok bool
	if c.SrcIp == "" || c.SrcMac == "" {
		ok, name, mac, ip = s.getLoopbackInfo()
		if !ok {
			errorstr := "VTEP: Src Tunnel Info not provisioned yet, loopback intf needed"
			s.logger.Info(errorstr)
			return &VtepConfig{}, errors.New(errorstr)
		}
		fmt.Println("loopback info:", name, mac, ip)
		if c.SrcMac != "" {
			mac, _ = net.ParseMAC(c.SrcMac)
		}
		if c.SrcIp != "" {
			ip = net.ParseIP(c.SrcIp)
		}

	}

	srcName := s.getLinuxIfName(c.SrcIfIndex)
	DstNetMac, _ := net.ParseMAC(c.DstMac)

	s.logger.Info(fmt.Sprintf("Forcing Vtep %s to use Lb %s SrcMac %s Ip %s", c.VtepName, name, mac, ip))
	return &VtepConfig{
		VtepId:    uint32(c.VtepId),
		VxlanId:   uint32(c.VxlanId),
		VtepName:  string(c.VtepName),
		SrcIfName: srcName,
		UDP:       uint16(c.UDP),
		TTL:       uint16(c.TTL),
		TOS:       uint16(c.TOS),
		InnerVlanHandlingMode: ConvertInt32ToBool(c.InnerVlanHandlingMode),
		Learning:              c.Learning,
		Rsc:                   c.Rsc,
		L2miss:                c.L2miss,
		L3miss:                c.L3miss,
		TunnelSrcIp:           ip,
		TunnelDstIp:           net.ParseIP(c.DstIp),
		VlanId:                uint16(c.VlanId),
		TunnelSrcMac:          mac,
		TunnelDstMac:          DstNetMac,
	}, nil
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

func (s *VXLANServer) ConfigListener() {

	s.Configchans = &VxLanConfigChannels{
		Vxlancreate: make(chan VxlanConfig, 0),
		Vxlandelete: make(chan VxlanConfig, 0),
		Vxlanupdate: make(chan VxlanUpdate, 0),
		Vtepcreate:  make(chan VtepConfig, 0),
		Vtepdelete:  make(chan VtepConfig, 0),
		Vtepupdate:  make(chan VtepUpdate, 0),
	}

	softswitch := vxlan_linux.NewVxlanLinux(s.logger)

	go func(cc *VxLanConfigChannels, ss *vxlan_linux.VxlanLinux) {
		for {
			select {
			case vxlan := <-cc.Vxlancreate:
				s.saveVxLanConfigData(&vxlan)
				c := vxlan_linux.VxlanConfig(vxlan)
				ss.CreateVxLAN(&c)

			case vxlan := <-cc.Vxlandelete:
				c := vxlan_linux.VxlanConfig(vxlan)
				ss.DeleteVxLAN(&c)

			case <-cc.Vxlanupdate:
				//s.UpdateThriftVxLAN(&vxlan)

			case vtep := <-cc.Vtepcreate:
				s.saveVtepConfigData(&vtep)
				c := vxlan_linux.VtepConfig(vtep)
				ss.CreateVtep(&c)

			case vtep := <-cc.Vtepdelete:
				c := vxlan_linux.VtepConfig(vtep)
				ss.DeleteVtep(&c)

			case <-cc.Vtepupdate:
				//s.UpdateThriftVtep(&vtep)
			}
		}
	}(s.Configchans, softswitch)
}
