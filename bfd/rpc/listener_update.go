package rpc

import (
	"bfdd"
	"errors"
	"fmt"
)

func (h *BFDHandler) UpdateBfdGlobalConfig(origConf *bfdd.BfdGlobalConfig, newConf *bfdd.BfdGlobalConfig, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original global config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New global config attrs:", newConf))
	return true, nil
}

func (h *BFDHandler) UpdateBfdIntfConfig(origConf *bfdd.BfdIntfConfig, newConf *bfdd.BfdIntfConfig, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original interface config attrs:", origConf))
	if newConf == nil {
		err := errors.New("Invalid Interface Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Updated interface config attrs:", newConf))
	return h.SendBfdIntfConfig(newConf), nil
}

func (h *BFDHandler) UpdateBfdSessionConfig(origConf *bfdd.BfdSessionConfig, newConf *bfdd.BfdSessionConfig, attrset []bool) (bool, error) {
	if newConf == nil {
		err := errors.New("Invalid Session Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Update session config attrs:", newConf))
	return true, nil
}
