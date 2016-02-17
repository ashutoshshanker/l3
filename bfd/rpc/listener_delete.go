package rpc

import (
	"bfdd"
	"fmt"
	//    "l3/bfd/server"
	//    "log/syslog"
	//    "net"
	"l3/bfd/bfddCommonDefs"
)

func (h *BFDHandler) DeleteBfdGlobalConfig(bfdGlobalConf *bfdd.BfdGlobalConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bfdGlobalConf))
	return true, nil
}

func (h *BFDHandler) DeleteBfdIntfConfig(bfdIfConf *bfdd.BfdIntfConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete interface config attrs:", bfdIfConf))
	return true, nil
}

func (h *BFDHandler) DeleteBfdSessionConfig(bfdSessionConf *bfdd.BfdSessionConfig) (bool, error) {
	bfdSessionCommand := bfddCommonDefs.BfdSessionConfig{
		DestIp:    bfdSessionConf.IpAddr,
		Protocol:  int(bfdSessionConf.Owner),
		Operation: int(bfdSessionConf.Operation),
	}
	h.server.SessionConfigCh <- bfdSessionCommand
	return true, nil
}
