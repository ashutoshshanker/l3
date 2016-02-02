package rpc

import (
	"bfdd"
	"errors"
	"fmt"
	"l3/bfd/server"
	//"log/syslog"
	//"net"
)

func (h *BFDHandler) SendBfdGlobalConfig(bfdGlobalConfig *bfdd.BfdGlobalConfig) bool {
	gConf := server.GlobalConfig{
		Enable: bfdGlobalConfig.Enable,
	}
	h.server.GlobalConfigCh <- gConf
	return true
}

func (h *BFDHandler) SendBfdIntfConfig(bfdIntfConfig *bfdd.BfdIntfConfig) bool {
	ifConf := server.IntfConfig{
		InterfaceId:               bfdIntfConfig.Interface,
		LocalMultiplier:           bfdIntfConfig.LocalMultiplier,
		DesiredMinTxInterval:      bfdIntfConfig.DesiredMinTxInterval,
		RequiredMinRxInterval:     bfdIntfConfig.RequiredMinRxInterval,
		RequiredMinEchoRxInterval: bfdIntfConfig.RequiredMinEchoRxInterval,
		DemandEnabled:             bfdIntfConfig.DemandEnabled,
		AuthenticationEnabled:     bfdIntfConfig.AuthenticationEnabled,
		AuthenticationType:        bfdIntfConfig.AuthType,
		AuthenticationKeyId:       bfdIntfConfig.AuthKeyId,
		SequenceNumber:            bfdIntfConfig.SequenceNumber,
		AuthenticationData:        bfdIntfConfig.AuthData,
	}
	h.server.IntfConfigCh <- ifConf
	return true
}

func (h *BFDHandler) CreateBfdGlobalConfig(bfdGlobalConf *bfdd.BfdGlobalConfig) (bool, error) {
	if bfdGlobalConf == nil {
		err := errors.New("Invalid Global Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bfdGlobalConf))
	return h.SendBfdGlobalConfig(bfdGlobalConf), nil
}

func (h *BFDHandler) CreateBfdIntfConfig(bfdIntfConf *bfdd.BfdIntfConfig) (bool, error) {
	if bfdIntfConf == nil {
		err := errors.New("Invalid Interface Configuration")
		return false, err
	}
	h.logger.Info(fmt.Sprintln("Create interface config attrs:", bfdIntfConf))
	return h.SendBfdIntfConfig(bfdIntfConf), nil
}