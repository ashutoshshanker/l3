// server.go
package vxlan

import (
	"fmt"
	"net"
	"utils/logging"
)

var SwitchMac [6]uint8
var NetSwitchMac net.HardwareAddr
var logger *logging.Writer

type VXLANServer struct {
	Configchans *VxLanConfigChannels
	Paramspath  string // location of params path
}

type cfgFileJson struct {
	SwitchMac        string            `json:"SwitchMac"`
	PluginList       []string          `json:"PluginList"`
	IfNameMap        map[string]string `json:"IfNameMap"`
	IfNamePrefix     map[string]string `json:"IfNamePrefix"`
	SysRsvdVlanRange string            `json:"SysRsvdVlanRange"`
}

func SetLogger(l *logging.Writer) {
	logger = l
}

func NewVXLANServer(l *logging.Writer, paramspath string) *VXLANServer {

	SetLogger(l)

	logger.Info(fmt.Sprintf("Params path: %s", paramspath))
	server := &VXLANServer{
		Paramspath: paramspath,
	}

	// save off the switch mac for use by the VTEPs
	//server.SaveVtepSrcMacSrcIp()

	// connect to the various servers
	ConnectToClients(paramspath + "clients.json")

	// listen for config messages from server
	server.ConfigListener()

	return server
}
