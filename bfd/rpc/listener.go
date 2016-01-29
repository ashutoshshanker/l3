package rpc

import (
	//    "bfdd"
	//    "fmt"
	//    "l3/bfd/config"
	"l3/bfd/server"
	"log/syslog"
	//    "net"
)

type BFDHandler struct {
	server *server.BFDServer
	logger *syslog.Writer
}

func NewBFDHandler(logger *syslog.Writer, server *server.BFDServer) *BFDHandler {
	h := new(BFDHandler)
	h.server = server
	h.logger = logger
	return h
}
