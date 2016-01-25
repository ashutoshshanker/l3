package rpc

import (
	"bfdd"
	"fmt"
	//    "l3/bfd/config"
	//    "l3/bfd/server"
	//    "log/syslog"
	//    "net"
)

func (h *BFDHandler) UpdateBfdGlobalConfig(origConf *bfdd.BfdGlobalConfig, newConf *bfdd.BfdGlobalConfig, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original global config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New global config attrs:", newConf))
	return true, nil
}

func (h *BFDHandler) UpdateBfdIntfConfig(origConf *bfdd.BfdIntfConfig, newConf *bfdd.BfdIntfConfig, attrset []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Original interface config attrs:", origConf))
	h.logger.Info(fmt.Sprintln("New interface config attrs:", newConf))
	return true, nil
}
