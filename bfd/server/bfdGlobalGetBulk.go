package server

import (
	"fmt"
)

func (server *BFDServer) GetBfdGlobalState() *GlobalState {
	result := new(GlobalState)
	ent := server.bfdGlobal

	result.Enable = ent.Enabled
	result.NumSessions = ent.NumSessions
	result.NumUpSessions = ent.NumUpSessions
	result.NumDownSessions = ent.NumDownSessions
	result.NumAdminDownSessions = ent.NumAdminDownSessions

	server.logger.Info(fmt.Sprintln("Global State:", result))
	return result
}
