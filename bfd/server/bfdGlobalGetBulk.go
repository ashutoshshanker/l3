package server

import (
	"fmt"
	"l3/bfd/config"
)

func (server *BFDServer) GetBfdGlobalState() *config.GlobalState {
	result := new(config.GlobalState)
	ent := server.bfdGlobal

	result.Enable = ent.Enabled
	result.NumInterfaces = ent.NumInterfaces
	result.NumSessions = ent.NumSessions
	result.NumUpSessions = ent.NumUpSessions
	result.NumDownSessions = ent.NumDownSessions
	result.NumAdminDownSessions = ent.NumAdminDownSessions

	server.logger.Info(fmt.Sprintln("Global State:", result))
	return result
}
