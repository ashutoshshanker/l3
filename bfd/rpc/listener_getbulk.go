package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/bfddCommonDefs"
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

func (h *BFDHandler) convertIntfStateToThrift(ent server.IntfState) *bfdd.BfdIntfState {
	intfState := bfdd.NewBfdIntfState()
	intfState.InterfaceId = int32(ent.InterfaceId)
	intfState.Enabled = ent.Enabled
	intfState.NumSessions = int32(ent.NumSessions)
	intfState.LocalMultiplier = int32(ent.LocalMultiplier)
	intfState.DesiredMinTxInterval = int32(ent.DesiredMinTxInterval)
	intfState.RequiredMinRxInterval = int32(ent.RequiredMinRxInterval)
	intfState.RequiredMinEchoRxInterval = int32(ent.RequiredMinEchoRxInterval)
	intfState.DemandEnabled = ent.DemandEnabled
	intfState.AuthenticationEnabled = ent.AuthenticationEnabled
	intfState.AuthenticationType = int32(ent.AuthenticationType)
	intfState.AuthenticationKeyId = int32(ent.AuthenticationKeyId)
	intfState.SequenceNumber = int32(ent.SequenceNumber)
	intfState.AuthenticationData = string(ent.AuthenticationData)
	return intfState
}

func (h *BFDHandler) GetBulkBfdIntfState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdIntfStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get BFD interface state"))
	nextIdx, currCount, bfdIntfStates := h.server.GetBulkBfdIntfStates(int(fromIdx), int(count))
	if bfdIntfStates == nil {
		err := errors.New("Bfd server is busy")
		return nil, err
	}
	bfdIntfResponse := make([]*bfdd.BfdIntfState, len(bfdIntfStates))
	/*
		for idx, item := range bfdIntfStates {
			bfdIntfResponse[idx] = h.convertIntfStateToThrift(item)
		}
	*/
	BfdIntfStateGetInfo := bfdd.NewBfdIntfStateGetInfo()
	BfdIntfStateGetInfo.Count = bfdd.Int(currCount)
	BfdIntfStateGetInfo.StartIdx = bfdd.Int(fromIdx)
	BfdIntfStateGetInfo.EndIdx = bfdd.Int(nextIdx)
	BfdIntfStateGetInfo.More = (nextIdx != 0)
	BfdIntfStateGetInfo.BfdIntfStateList = bfdIntfResponse
	return BfdIntfStateGetInfo, nil

}

func (h *BFDHandler) convertBfdSessionProtocolsToString(Protocols []bool) string {
	var protocols string
	if Protocols[bfddCommonDefs.BGP] {
		protocols += "bgp, "
	}
	if Protocols[bfddCommonDefs.OSPF] {
		protocols += "ospf, "
	}
	return protocols
}

func (h *BFDHandler) convertSessionStateToThrift(ent server.SessionState) *bfdd.BfdSessionState {
	sessionState := bfdd.NewBfdSessionState()
	sessionState.SessionId = int32(ent.SessionId)
	sessionState.LocalIpAddr = string(ent.LocalIpAddr)
	sessionState.RemoteIpAddr = string(ent.RemoteIpAddr)
	sessionState.InterfaceId = int32(ent.InterfaceId)
	sessionState.RegisteredProtocols = h.convertBfdSessionProtocolsToString(ent.RegisteredProtocols)
	sessionState.SessionState = int32(ent.SessionState)
	sessionState.RemoteSessionState = int32(ent.RemoteSessionState)
	sessionState.LocalDiscriminator = int32(ent.LocalDiscriminator)
	sessionState.RemoteDiscriminator = int32(ent.RemoteDiscriminator)
	sessionState.LocalDiagType = int32(ent.LocalDiagType)
	sessionState.DesiredMinTxInterval = int32(ent.DesiredMinTxInterval)
	sessionState.RequiredMinRxInterval = int32(ent.RequiredMinRxInterval)
	sessionState.RemoteMinRxInterval = int32(ent.RemoteMinRxInterval)
	sessionState.DetectionMultiplier = int32(ent.DetectionMultiplier)
	sessionState.DemandMode = ent.DemandMode
	sessionState.RemoteDemandMode = ent.RemoteDemandMode
	sessionState.AuthType = int32(ent.AuthType)
	sessionState.AuthSeqKnown = ent.AuthSeqKnown
	sessionState.ReceivedAuthSeq = int32(ent.ReceivedAuthSeq)
	sessionState.SentAuthSeq = int32(ent.SentAuthSeq)
	return sessionState
}

func (h *BFDHandler) GetBulkBfdSessionState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdSessionStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get session states"))
	nextIdx, currCount, bfdSessionStates := h.server.GetBulkBfdSessionStates(int(fromIdx), int(count))
	if bfdSessionStates == nil {
		err := errors.New("Bfd server is busy")
		return nil, err
	}
	bfdSessionResponse := make([]*bfdd.BfdSessionState, len(bfdSessionStates))
	for idx, item := range bfdSessionStates {
		bfdSessionResponse[idx] = h.convertSessionStateToThrift(item)
	}
	BfdSessionStateGetInfo := bfdd.NewBfdSessionStateGetInfo()
	BfdSessionStateGetInfo.Count = bfdd.Int(currCount)
	BfdSessionStateGetInfo.StartIdx = bfdd.Int(fromIdx)
	BfdSessionStateGetInfo.EndIdx = bfdd.Int(nextIdx)
	BfdSessionStateGetInfo.More = (nextIdx != 0)
	BfdSessionStateGetInfo.BfdSessionStateList = bfdSessionResponse
	return BfdSessionStateGetInfo, nil
}
