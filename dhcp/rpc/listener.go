package rpc

import (
	"l3/dhcp/server"
	"utils/logging"
)

type DHCPHandler struct {
	server *server.DHCPServer
	logger *logging.Writer
}

func NewDHCPHandler(server *server.DHCPServer, logger *logging.Writer) *DHCPHandler {
	h := new(DHCPHandler)
	h.server = server
	h.logger = logger
	return h
}
