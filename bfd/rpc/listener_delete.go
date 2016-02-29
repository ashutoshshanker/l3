package rpc

import (
	"bfdd"
	"fmt"
)

func (h *BFDHandler) DeleteBfdGlobalConfig(bfdGlobalConf *bfdd.BfdGlobalConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bfdGlobalConf))
	return true, nil
}

func (h *BFDHandler) DeleteBfdIntfConfig(bfdIfConf *bfdd.BfdIntfConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete interface config attrs:", bfdIfConf))
	ifIndex := bfdIfConf.IfIndex
	h.server.IntfConfigDeleteCh <- ifIndex
	return true, nil
}

func (h *BFDHandler) DeleteBfdSessionConfig(bfdSessionConf *bfdd.BfdSessionConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete session config attrs:", bfdSessionConf))
	return true, nil
}
