package server

import (
	//"fmt"
	"l3/bfd/config"
)

func (server *BFDServer) updateGlobalConf(gConf config.GlobalConfig) {
	if gConf.Enable {
		server.logger.Info("Enabled BFD")
	} else {
		server.logger.Info("Disabled BFD")
	}
	server.bfdGlobal.Enabled = gConf.Enable
}

func (server *BFDServer) initBfdGlobalConfDefault() {
}

func (server *BFDServer) processGlobalConfig(gConf config.GlobalConfig) {
	if gConf.Enable {
		server.logger.Info("Enabled BFD")
	} else {
		server.logger.Info("Disabled BFD")
	}
	server.bfdGlobal.Enabled = gConf.Enable
}
