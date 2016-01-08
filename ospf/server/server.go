package server

import (
        "fmt"
//        "l3/ospf/config"
        "time"
        "log/syslog"
        "ribd"
)

type OSPFServer struct {
    logger              *syslog.Writer
    ribdClient          *ribd.RouteServiceClient
}

func NewOSPFServer(logger *syslog.Writer, ribdClient *ribd.RouteServiceClient) *OSPFServer {
    ospfServer := &OSPFServer{}
    ospfServer.logger = logger
    ospfServer.ribdClient = ribdClient
    return ospfServer
}

func (server *OSPFServer) StartServer() {
    server.logger.Info(fmt.Sprintln("Starting Ospf Server"))
    for {
        time.Sleep(time.Duration(1) * time.Minute)
    }
}
