package server

import (
//"fmt"
)

func (server *BFDServer) GetBulkBfdSessionStates(idx int, cnt int) (int, int, []SessionState) {
	var nextIdx int
	var count int
	length := len(server.bfdGlobal.SessionsIdSlice)
	if idx+cnt >= length {
		count = length - idx
		nextIdx = 0
	} else {
		nextIdx = idx + count + 1
	}
	result := make([]SessionState, count)
	for i := idx; i < count; i++ {
		sessionId := server.bfdGlobal.SessionsIdSlice[i]
		result[i] = server.bfdGlobal.Sessions[sessionId].state
	}
	return nextIdx, count, result
}
