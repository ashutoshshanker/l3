// server.go
package vxlan

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"utils/logging"
)

var SwitchMac [6]uint8
var NetSwitchMac net.HardwareAddr

type VXLANServer struct {
	logger      *logging.Writer
	Configchans *VxLanChannels
	Paramspath  string // location of params path
}

type cfgFileJson struct {
	SwitchMac        string            `json:"SwitchMac"`
	PluginList       []string          `json:"PluginList"`
	IfNameMap        map[string]string `json:"IfNameMap"`
	IfNamePrefix     map[string]string `json:"IfNamePrefix"`
	SysRsvdVlanRange string            `json:"SysRsvdVlanRange"`
}

func SaveSwitchMac(asicdconffilename string) {
	var cfgFile cfgFileJson

	cfgFileData, err := ioutil.ReadFile(asicdconffilename)
	if err != nil {
		//StpLogger("ERROR", "Error reading config file - asicd.conf. Using defaults (linux plugin only)")
		return
	}
	err = json.Unmarshal(cfgFileData, &cfgFile)
	if err != nil {
		//StpLogger("ERROR", "Error parsing config file, using defaults (linux plugin only)")
		return
	}

	NetSwitchMac, _ = net.ParseMAC(cfgFile.SwitchMac)
	SwitchMac = [6]uint8{NetSwitchMac[0], NetSwitchMac[1], NetSwitchMac[2], NetSwitchMac[3], NetSwitchMac[4], NetSwitchMac[5]}

}

func NewVXLANServer(logger *logging.Writer, paramspath string) *VXLANServer {

	logger.Info(fmt.Sprintf("Params path: %s", paramspath))
	server := &VXLANServer{
		logger:     logger,
		Paramspath: paramspath,
	}

	// listen for config messages from server
	server.StartConfigListener()

	return server
}
