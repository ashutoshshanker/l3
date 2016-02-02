package rpc

import (
	"bfdd"
	"fmt"
	//    "l3/bfd/server"
	//    "log/syslog"
	//    "net"
)

func (h *BFDHandler) DeleteBfdGlobalConfig(bfdGlobalConf *bfdd.BfdGlobalConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bfdGlobalConf))
	return true, nil
}

func (h *BFDHandler) DeleteBfdIntfConfig(bfdIfConf *bfdd.BfdIntfConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete interface config attrs:", bfdIfConf))
	return true, nil
}
