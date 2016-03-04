package rpc

import (
	"l3/bfd/server"
	"utils/logging"
)

type BFDHandler struct {
	server *server.BFDServer
	logger *logging.Writer
}

func NewBFDHandler(logger *logging.Writer, server *server.BFDServer) *BFDHandler {
	h := new(BFDHandler)
	h.server = server
	h.logger = logger
	return h
}
