package rpc

import (
	"l3/arp/server"
	"utils/logging"
)

type ARPHandler struct {
	server *server.ARPServer
	logger *logging.Writer
}

func NewARPHandler(server *server.ARPServer, logger *logging.Writer) *ARPHandler {
	h := new(ARPHandler)
	h.server = server
	h.logger = logger
	return h
}
