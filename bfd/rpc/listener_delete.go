package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"l3/bfd/server"
)

func (h *BFDHandler) SendBfdSessionDeleteConfig(bfdSessionConfig *bfdd.BfdSessionConfig) bool {
	sessionConf := server.SessionConfig{
		DestIp:    bfdSessionConfig.IpAddr,
		PerLink:   bfdSessionConfig.PerLink,
		Protocol:  bfddCommonDefs.ConvertBfdSessionOwnerStrToVal(bfdSessionConfig.Owner),
		Operation: bfddCommonDefs.DELETE,
	}
	h.server.SessionConfigCh <- sessionConf
	return true
}

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
	if bfdSessionConf == nil {
		err := errors.New("Invalid Session Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create session config attrs:", bfdSessionConf))
	return h.SendBfdSessionDeleteConfig(bfdSessionConf), nil
}
