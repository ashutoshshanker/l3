package rpc

import (
	//    "ospfd"
	//    "fmt"
	//    "l3/ospf/config"
	"l3/ospf/server"
	"utils/logging"
	//    "net"
)

type OSPFHandler struct {
	server *server.OSPFServer
	logger *logging.Writer
}

func NewOSPFHandler(server *server.OSPFServer, logger *logging.Writer) *OSPFHandler {
	h := new(OSPFHandler)
	h.server = server
	h.logger = logger
	return h
}
