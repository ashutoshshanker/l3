package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/server"
	//    "log/syslog"
	//    "net"
)

func (h *BFDHandler) convertGlobalStateToThrift(ent server.GlobalState) *bfdd.BfdGlobalState {
	gState := bfdd.NewBfdGlobalState()
	gState.Enable = ent.Enable
	gState.NumInterfaces = int32(ent.NumInterfaces)
	gState.NumUpSessions = int32(ent.NumUpSessions)
	gState.NumDownSessions = int32(ent.NumDownSessions)
	gState.NumAdminDownSessions = int32(ent.NumAdminDownSessions)
	return gState
}

func (h *BFDHandler) GetBulkBfdGlobalState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdGlobalStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get BFD global state"))

	if fromIdx != 0 {
		err := errors.New("Invalid range")
		return nil, err
	}
	bfdGlobalState := h.server.GetBfdGlobalState()
	bfdGlobalStateResponse := make([]*bfdd.BfdGlobalState, 1)
	bfdGlobalStateResponse[0] = h.convertGlobalStateToThrift(*bfdGlobalState)
	bfdGlobalStateGetInfo := bfdd.NewBfdGlobalStateGetInfo()
	bfdGlobalStateGetInfo.Count = bfdd.Int(1)
	bfdGlobalStateGetInfo.StartIdx = bfdd.Int(0)
	bfdGlobalStateGetInfo.EndIdx = bfdd.Int(0)
	bfdGlobalStateGetInfo.More = false
	bfdGlobalStateGetInfo.BfdGlobalStateList = bfdGlobalStateResponse
	return bfdGlobalStateGetInfo, nil
}

func (h *BFDHandler) GetBulkBfdSessionState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdSessionStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get Neighbor attrs"))
	bfdSessionResponse := bfdd.NewBfdSessionStateGetInfo()
	return bfdSessionResponse, nil
}
