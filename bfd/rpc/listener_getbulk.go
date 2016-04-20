package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"l3/bfd/server"
	"strconv"
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

func (h *BFDHandler) convertIntfStateToThrift(ent server.IntfState) *bfdd.BfdInterfaceState {
	intfState := bfdd.NewBfdInterfaceState()
	intfState.IfIndex = int32(ent.InterfaceId)
	intfState.Enabled = ent.Enabled
	intfState.NumSessions = int32(ent.NumSessions)
	intfState.LocalMultiplier = int32(ent.LocalMultiplier)
	intfState.DesiredMinTxInterval = string(strconv.Itoa(int(ent.DesiredMinTxInterval)) + "(us)")
	intfState.RequiredMinRxInterval = string(strconv.Itoa(int(ent.RequiredMinRxInterval)) + "(us)")
	intfState.RequiredMinEchoRxInterval = string(strconv.Itoa(int(ent.RequiredMinEchoRxInterval)) + "(us)")
	intfState.DemandEnabled = ent.DemandEnabled
	intfState.AuthenticationEnabled = ent.AuthenticationEnabled
	intfState.AuthenticationType = string(h.server.ConvertBfdAuthTypeValToStr(ent.AuthenticationType))
	intfState.AuthenticationKeyId = int32(ent.AuthenticationKeyId)
	intfState.AuthenticationData = string(ent.AuthenticationData)
	return intfState
}

func (h *BFDHandler) GetBulkBfdInterfaceState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdInterfaceStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get BFD interface state"))
	nextIdx, currCount, bfdIntfStates := h.server.GetBulkBfdIntfStates(int(fromIdx), int(count))
	if bfdIntfStates == nil {
		err := errors.New("Bfd server is busy")
		return nil, err
	}
	bfdIntfResponse := make([]*bfdd.BfdInterfaceState, len(bfdIntfStates))
	for idx, item := range bfdIntfStates {
		bfdIntfResponse[idx] = h.convertIntfStateToThrift(item)
	}
	BfdIntfStateGetInfo := bfdd.NewBfdInterfaceStateGetInfo()
	BfdIntfStateGetInfo.Count = bfdd.Int(currCount)
	BfdIntfStateGetInfo.StartIdx = bfdd.Int(fromIdx)
	BfdIntfStateGetInfo.EndIdx = bfdd.Int(nextIdx)
	BfdIntfStateGetInfo.More = (nextIdx != 0)
	BfdIntfStateGetInfo.BfdInterfaceStateList = bfdIntfResponse
	return BfdIntfStateGetInfo, nil

}

func (h *BFDHandler) convertBfdSessionProtocolsToString(Protocols []bool) string {
	var protocols string
	if Protocols[bfddCommonDefs.DISC] {
		protocols += "doscover, "
	}
	if Protocols[bfddCommonDefs.USER] {
		protocols += "user, "
	}
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
	sessionState.IpAddr = string(ent.IpAddr)
	sessionState.IfIndex = int32(ent.InterfaceId)
	sessionState.PerLinkSession = ent.PerLinkSession
	sessionState.LocalMacAddr = string(ent.LocalMacAddr.String())
	sessionState.RemoteMacAddr = string(ent.RemoteMacAddr.String())
	sessionState.RegisteredProtocols = string(h.convertBfdSessionProtocolsToString(ent.RegisteredProtocols))
	sessionState.SessionState = string(h.server.ConvertBfdSessionStateValToStr(ent.SessionState))
	sessionState.RemoteSessionState = string(h.server.ConvertBfdSessionStateValToStr(ent.RemoteSessionState))
	sessionState.LocalDiscriminator = int32(ent.LocalDiscriminator)
	sessionState.RemoteDiscriminator = int32(ent.RemoteDiscriminator)
	sessionState.LocalDiagType = string(h.server.ConvertBfdSessionDiagValToStr(ent.LocalDiagType))
	sessionState.DesiredMinTxInterval = string(strconv.Itoa(int(ent.DesiredMinTxInterval)) + "(us)")
	sessionState.RequiredMinRxInterval = string(strconv.Itoa(int(ent.RequiredMinRxInterval)) + "(us)")
	sessionState.RemoteMinRxInterval = string(strconv.Itoa(int(ent.RemoteMinRxInterval)) + "(us)")
	sessionState.DetectionMultiplier = int32(ent.DetectionMultiplier)
	sessionState.DemandMode = ent.DemandMode
	sessionState.RemoteDemandMode = ent.RemoteDemandMode
	sessionState.AuthType = string(h.server.ConvertBfdAuthTypeValToStr(ent.AuthType))
	sessionState.AuthSeqKnown = ent.AuthSeqKnown
	sessionState.ReceivedAuthSeq = int32(ent.ReceivedAuthSeq)
	sessionState.SentAuthSeq = int32(ent.SentAuthSeq)
	sessionState.NumTxPackets = int32(ent.NumTxPackets)
	sessionState.NumRxPackets = int32(ent.NumRxPackets)
	return sessionState
}

func (h *BFDHandler) convertSessionParamStateToThrift(ent server.SessionParamState) *bfdd.BfdSessionParamState {
	sessionParamState := bfdd.NewBfdSessionParamState()
	sessionParamState.Name = string(ent.Name)
	sessionParamState.NumSessions = int32(ent.NumSessions)
	sessionParamState.LocalMultiplier = int32(ent.LocalMultiplier)
	sessionParamState.DesiredMinTxInterval = string(strconv.Itoa(int(ent.DesiredMinTxInterval)) + "(us)")
	sessionParamState.RequiredMinRxInterval = string(strconv.Itoa(int(ent.RequiredMinRxInterval)) + "(us)")
	sessionParamState.RequiredMinEchoRxInterval = string(strconv.Itoa(int(ent.RequiredMinEchoRxInterval)) + "(us)")
	sessionParamState.DemandEnabled = ent.DemandEnabled
	sessionParamState.AuthenticationEnabled = ent.AuthenticationEnabled
	sessionParamState.AuthenticationType = string(h.server.ConvertBfdAuthTypeValToStr(ent.AuthenticationType))
	sessionParamState.AuthenticationKeyId = int32(ent.AuthenticationKeyId)
	sessionParamState.AuthenticationData = string(ent.AuthenticationData)
	return sessionParamState
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

func (h *BFDHandler) GetBulkBfdSessionParamState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdSessionParamStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get session param states"))
	nextIdx, currCount, bfdSessionParamStates := h.server.GetBulkBfdSessionParamStates(int(fromIdx), int(count))
	if bfdSessionParamStates == nil {
		err := errors.New("Bfd server is busy")
		return nil, err
	}
	bfdSessionParamResponse := make([]*bfdd.BfdSessionParamState, len(bfdSessionParamStates))
	for idx, item := range bfdSessionParamStates {
		bfdSessionParamResponse[idx] = h.convertSessionParamStateToThrift(item)
	}
	BfdSessionParamStateGetInfo := bfdd.NewBfdSessionParamStateGetInfo()
	BfdSessionParamStateGetInfo.Count = bfdd.Int(currCount)
	BfdSessionParamStateGetInfo.StartIdx = bfdd.Int(fromIdx)
	BfdSessionParamStateGetInfo.EndIdx = bfdd.Int(nextIdx)
	BfdSessionParamStateGetInfo.More = (nextIdx != 0)
	BfdSessionParamStateGetInfo.BfdSessionParamStateList = bfdSessionParamResponse
	return BfdSessionParamStateGetInfo, nil
}
