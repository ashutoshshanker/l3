package rpc

import (
	"bfdd"
	"fmt"
)

func (h *BFDHandler) GetBfdGlobalState(bfd string) (*bfdd.BfdGlobalState, error) {
	h.logger.Info(fmt.Sprintln("Get Global attrs"))
	bfdGlobalStateResponse := bfdd.NewBfdGlobalState()
	gState := h.server.GetBfdGlobalState()
	bfdGlobalState := h.convertGlobalStateToThrift(*gState)
	bfdGlobalStateResponse = bfdGlobalState
	return bfdGlobalStateResponse, nil
}

func (h *BFDHandler) GetBfdInterfaceState(ifIndex int32) (*bfdd.BfdInterfaceState, error) {
	h.logger.Info(fmt.Sprintln("Get Interface attrs for IfIndex ", ifIndex))
	bfdInterfaceStateResponse := bfdd.NewBfdInterfaceState()
	intfState := h.server.GetBfdIntfState(ifIndex)
	bfdInterfaceState := h.convertIntfStateToThrift(*intfState)
	bfdInterfaceStateResponse = bfdInterfaceState
	return bfdInterfaceStateResponse, nil
}

func (h *BFDHandler) GetBfdSessionState(ipAddr string) (*bfdd.BfdSessionState, error) {
	h.logger.Info(fmt.Sprintln("Get Session attrs for neighbor ", ipAddr))
	bfdSessionStateResponse := bfdd.NewBfdSessionState()
	sessionState := h.server.GetBfdSessionState(ipAddr)
	bfdSessionState := h.convertSessionStateToThrift(*sessionState)
	bfdSessionStateResponse = bfdSessionState
	return bfdSessionStateResponse, nil
}

func (h *BFDHandler) GetBfdSessionParamState(paramName string) (*bfdd.BfdSessionParamState, error) {
	h.logger.Info(fmt.Sprintln("Get Session Params attrs for ", paramName))
	bfdSessionParamStateResponse := bfdd.NewBfdSessionParamState()
	sessionParamState := h.server.GetBfdSessionParamState(paramName)
	bfdSessionParamState := h.convertSessionParamStateToThrift(*sessionParamState)
	bfdSessionParamStateResponse = bfdSessionParamState
	return bfdSessionParamStateResponse, nil
}
