package rpc

import (
	"bfdd"
	"fmt"
)

func (h *BFDHandler) GetBfdGlobalState() (*bfdd.BfdGlobalState, error) {
	h.logger.Info(fmt.Sprintln("Get Global attrs"))
	bfdGlobalResponse := bfdd.NewBfdGlobalState()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBfdInterfaceState() (*bfdd.BfdInterfaceState, error) {
	h.logger.Info(fmt.Sprintln("Get Interface attrs"))
	bfdGlobalResponse := bfdd.NewBfdInterfaceState()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBfdSessionState(ifIpAddress string, addressLessIf int32) (*bfdd.BfdSessionState, error) {
	h.logger.Info(fmt.Sprintln("Get Session attrs"))
	bfdSessionResponse := bfdd.NewBfdSessionState()
	return bfdSessionResponse, nil
}
