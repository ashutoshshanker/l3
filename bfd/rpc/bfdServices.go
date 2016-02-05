package rpc

import (
	"bfdd"
	"l3/bfd/bfddCommonDefs"
)

func (h *BFDHandler) ExecuteBfdCommand(IpAddr string, Cmd bfdd.Int, Owner bfdd.Int) (bool, error) {
	bfdSessionCommand := bfddCommonDefs.BfdSessionConfig{
		DestIp:    IpAddr,
		Protocol:  int(Owner),
		Operation: int(Cmd),
	}
	h.server.SessionConfigCh <- bfdSessionCommand
	return true, nil
}
