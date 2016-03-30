package rpc

import (
	"bfdd"
	"fmt"
)

func (h *BFDHandler) GetBfdGlobalState(bfd string) (*bfdd.BfdGlobalState, error) {
	h.logger.Info(fmt.Sprintln("Get Global attrs"))
	bfdGlobalResponse := bfdd.NewBfdGlobalState()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBfdInterfaceState(ifIndex int32) (*bfdd.BfdInterfaceState, error) {
	h.logger.Info(fmt.Sprintln("Get Interface attrs"))
	bfdGlobalResponse := bfdd.NewBfdInterfaceState()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBfdSessionState(ipAdd string) (*bfdd.BfdSessionState, error) {
	h.logger.Info(fmt.Sprintln("Get Session attrs"))
	bfdSessionResponse := bfdd.NewBfdSessionState()
	return bfdSessionResponse, nil
}
