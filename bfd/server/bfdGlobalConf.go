package server

import (
//"fmt"
)

func (server *BFDServer) updateGlobalConf(gConf GlobalConfig) {
	if gConf.Enable {
		server.logger.Info("Enabled BFD")
	} else {
		server.logger.Info("Disabled BFD")
	}
	server.bfdGlobal.Enabled = gConf.Enable
}

func (server *BFDServer) initBfdGlobalConfDefault() {
}

func (server *BFDServer) processGlobalConfig(gConf GlobalConfig) {
	if gConf.Enable {
		server.logger.Info("Enabled BFD")
	} else {
		server.logger.Info("Disabled BFD")
	}
	server.bfdGlobal.Enabled = gConf.Enable
}
