package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"l3/bfd/server"
)

func (h *BFDHandler) SendBfdSessionDeleteConfig(bfdSessionConfig *bfdd.BfdSession) bool {
	sessionConf := server.SessionConfig{
		DestIp:    bfdSessionConfig.IpAddr,
		PerLink:   bfdSessionConfig.PerLink,
		Protocol:  bfddCommonDefs.ConvertBfdSessionOwnerStrToVal(bfdSessionConfig.Owner),
		Operation: bfddCommonDefs.DELETE,
	}
	h.server.SessionConfigCh <- sessionConf
	return true
}

func (h *BFDHandler) DeleteBfdGlobal(bfdGlobalConf *bfdd.BfdGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bfdGlobalConf))
	return true, nil
}

func (h *BFDHandler) DeleteBfdSession(bfdSessionConf *bfdd.BfdSession) (bool, error) {
	if bfdSessionConf == nil {
		err := errors.New("Invalid Session Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Delete session config attrs:", bfdSessionConf))
	return h.SendBfdSessionDeleteConfig(bfdSessionConf), nil
}

func (h *BFDHandler) DeleteBfdSessionParam(bfdSessionParamConf *bfdd.BfdSessionParam) (bool, error) {
	if bfdSessionParamConf == nil {
		err := errors.New("Invalid Session Param Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Delete session param config attrs:", bfdSessionParamConf))
	paramName := bfdSessionParamConf.Name
	h.server.SessionParamDeleteCh <- paramName
	return true, nil
}
