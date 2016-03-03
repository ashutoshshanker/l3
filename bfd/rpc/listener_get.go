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

func (h *BFDHandler) GetBfdIntfState() (*bfdd.BfdIntfState, error) {
	h.logger.Info(fmt.Sprintln("Get Interface attrs"))
	bfdGlobalResponse := bfdd.NewBfdIntfState()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBfdSessionState(ifIpAddress string, addressLessIf int32) (*bfdd.BfdSessionState, error) {
	h.logger.Info(fmt.Sprintln("Get Session attrs"))
	bfdSessionResponse := bfdd.NewBfdSessionState()
	return bfdSessionResponse, nil
}
