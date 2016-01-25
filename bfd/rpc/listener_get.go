package rpc

import (
	"bfdd"
	"fmt"
	//    "l3/bfd/config"
	//    "l3/bfd/server"
	//    "log/syslog"
	//    "net"
)

func (h *BFDHandler) GetBfdGlobalState() (*bfdd.BfdGlobalState, error) {
	h.logger.Info(fmt.Sprintln("Get global attrs"))
	bfdGlobalResponse := bfdd.NewBfdGlobalState()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBfdSessionState(ifIpAddress string, addressLessIf int32) (*bfdd.BfdSessionState, error) {
	h.logger.Info(fmt.Sprintln("Get Interface attrs"))
	bfdSessionResponse := bfdd.NewBfdSessionState()
	return bfdSessionResponse, nil
}
