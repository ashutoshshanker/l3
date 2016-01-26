package rpc

import (
	"bfdd"
	"fmt"
	//    "l3/bfd/config"
	//    "l3/bfd/server"
	//    "log/syslog"
	//    "net"
)

func (h *BFDHandler) GetBulkBfdGlobalState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdGlobalStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get Global attrs"))
	bfdGlobalResponse := bfdd.NewBfdGlobalStateGetInfo()
	return bfdGlobalResponse, nil
}

func (h *BFDHandler) GetBulkBfdSessionState(fromIdx bfdd.Int, count bfdd.Int) (*bfdd.BfdSessionStateGetInfo, error) {
	h.logger.Info(fmt.Sprintln("Get Neighbor attrs"))
	bfdSessionResponse := bfdd.NewBfdSessionStateGetInfo()
	return bfdSessionResponse, nil
}
