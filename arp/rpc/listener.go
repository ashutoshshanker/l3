package rpc

import (
    "l3/arp/server"
    "log/syslog"
)

type ARPHandler struct {
    server        *server.ARPServer
    logger        *syslog.Writer
}

func NewARPHandler(server *server.ARPServer, logger *syslog.Writer) *ARPHandler {
    h := new(ARPHandler)
    h.server = server
    h.logger = logger
    return h
}
