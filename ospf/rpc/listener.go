package rpc

import (
//    "ospfd"
//    "fmt"
//    "l3/ospf/config"
    "l3/ospf/server"
    "log/syslog"
//    "net"
)

type OSPFHandler struct {
    server        *server.OSPFServer
    logger        *syslog.Writer
}

func NewOSPFHandler(server *server.OSPFServer, logger *syslog.Writer) *OSPFHandler {
    h := new(OSPFHandler)
    h.server = server
    h.logger = logger
    return h
}
