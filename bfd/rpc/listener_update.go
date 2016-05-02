package rpc

import (
	"bfdd"
	"errors"
	"fmt"
)

func (h *BFDHandler) UpdateBfdGlobal(origConf *bfdd.BfdGlobal, newConf *bfdd.BfdGlobal, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original global config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New global config attrs:", newConf))
	return true, nil
}

func (h *BFDHandler) UpdateBfdSession(origConf *bfdd.BfdSession, newConf *bfdd.BfdSession, attrset []bool) (bool, error) {
	if newConf == nil {
		err := errors.New("Invalid Session Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Update session config attrs:", newConf))
	return true, nil
}

func (h *BFDHandler) UpdateBfdSessionParam(origConf *bfdd.BfdSessionParam, newConf *bfdd.BfdSessionParam, attrset []bool) (bool, error) {
	if newConf == nil {
		err := errors.New("Invalid Session Param Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Update session Param config attrs:", newConf))
	return h.SendBfdSessionParamConfig(newConf), nil
}
