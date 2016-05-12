package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/server"
)

func (h *BFDHandler) UpdateBfdGlobal(origConf *bfdd.BfdGlobal, newConf *bfdd.BfdGlobal, attrset []bool, op string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original global config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New global config attrs:", newConf))
	gConf := server.GlobalConfig{
		Enable: newConf.Enable,
	}
	h.server.GlobalConfigCh <- gConf
	return true, nil
}

func (h *BFDHandler) UpdateBfdSession(origConf *bfdd.BfdSession, newConf *bfdd.BfdSession, attrset []bool, op string) (bool, error) {
	if newConf == nil {
		err := errors.New("Invalid Session Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Update session config attrs:", newConf))
	return h.SendBfdSessionConfig(newConf), nil
}

func (h *BFDHandler) UpdateBfdSessionParam(origConf *bfdd.BfdSessionParam, newConf *bfdd.BfdSessionParam, attrset []bool, op string) (bool, error) {
	if newConf == nil {
		err := errors.New("Invalid Session Param Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Update session Param config attrs:", newConf))
	return h.SendBfdSessionParamConfig(newConf), nil
}
