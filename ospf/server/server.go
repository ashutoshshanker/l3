package server

import (
        "fmt"
        "l3/ospf/config"
        "log/syslog"
        "ribd"
)

type OSPFServer struct {
    logger              *syslog.Writer
    ribdClient          *ribd.RouteServiceClient
    OspfGlobalConfig    config.GlobalConfig
    OspfAreaConfig      config.AreaConfig
    GlobalConfigCh      chan config.GlobalConfig
    AreaConfigCh        chan config.AreaConfig
}

func NewOSPFServer(logger *syslog.Writer, ribdClient *ribd.RouteServiceClient) *OSPFServer {
    ospfServer := &OSPFServer{}
    ospfServer.logger = logger
    ospfServer.ribdClient = ribdClient
    ospfServer.GlobalConfigCh = make(chan config.GlobalConfig)
    ospfServer.AreaConfigCh = make(chan config.AreaConfig)
    return ospfServer
}

func (server *OSPFServer) StartServer() {
    gConf := <-server.GlobalConfigCh
    server.logger.Info(fmt.Sprintln("Received global conf:", gConf))
    server.OspfGlobalConfig = gConf

    areaConf := <-server.AreaConfigCh
    server.logger.Info(fmt.Sprintln("Received area conf:", areaConf))
    server.OspfAreaConfig = areaConf
    for {

    }
}
