package server

import (
//"fmt"
)

func (server *BFDServer) initBfdGlobalConfDefault() error {
	return nil
}

func (server *BFDServer) processGlobalConfig(gConf GlobalConfig) {
	if gConf.Enable {
		server.logger.Info("Enabled BFD")
	} else {
		server.logger.Info("Disabled BFD")
	}
	wasEnabled := server.bfdGlobal.Enabled
	server.bfdGlobal.Enabled = gConf.Enable
	isEnabled := server.bfdGlobal.Enabled
	length := len(server.bfdGlobal.SessionsIdSlice)
	for i := 0; i < length; i++ {
		sessionId := server.bfdGlobal.SessionsIdSlice[i]
		session := server.bfdGlobal.Sessions[sessionId]
		if !wasEnabled && isEnabled {
			// Bfd enabled globally. Restart all the sessions
			session.StartBfdSession()
		}
		if wasEnabled && !isEnabled {
			// Bfd disabled globally. Stop all the sessions
			session.StopBfdSession()
		}
	}
}
