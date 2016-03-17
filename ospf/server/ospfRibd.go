package server

import (
//    "fmt"
    "ribd"
//    nanomsg "github.com/op/go-nanomsg"
//    "encoding/json"
//    "l3/rib/ribdCommonDefs"
)

type RibdClient struct {
        OspfClientBase
        ClientHdl *ribd.RIBDServicesClient
}

/*
func (server *OSPFServer) listenForRIBUpdates(address string) error {
    var err error
    if server.ribSubSocket, err = nanomsg.NewSubSocket(); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to create RIB subscribe socket, error:", err))
        return err
    }

    if err = server.ribSubSocket.Subscribe(""); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on RIB subscribe socket, error:", err))
        return err
    }

    if _, err = server.ribSubSocket.Connect(address); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to connect to RIB publisher socket, address:", address, "error:", err))
        return err
    }

    server.logger.Info(fmt.Sprintln("Connected to RIB publisher at address:", address))
    if err = server.ribSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to set the buffer size for RIB publisher socket, error:", err))
        return err
    }
    return nil
}

func (server *OSPFServer)createRIBSubscriber() {
    for {
        server.logger.Info("Read on RIB subscriber socket...")
        ribrxBuf, err := server.ribSubSocket.Recv(0)
        if err != nil {
            server.logger.Err(fmt.Sprintln("Recv on RIB subscriber socket failed with error:", err))
            server.ribSubSocketErrCh <- err
            continue
        }
        server.logger.Info(fmt.Sprintln("RIB subscriber recv returned:", ribrxBuf))
        server.ribSubSocketCh <- ribrxBuf
    }
}

func (server *OSPFServer)processRibdNotification(ribrxBuf []byte) {
    var route ribdCommonDefs.RoutelistInfo
    routes := make([]*ribd.Routes, 0, 1)
    reader := bytes.NewReader(ribrxBuf)
    decoder := json.NewDecoder(reader)
    msg := ribdCommonDefs.RibdNotifyMsg{}
    for err := decoder.Decode(&msg); err == nil; err = decoder.Decode(&msg) {
        err = json.Unmarshal(msg.MsgBuf, &route)
        if err != nil {
                server.logger.Err("Err in processing routes from RIB")
        }
        server.logger.Info(fmt.Sprintln("Remove connected route, dest:", route.RouteInfo.Ipaddr, "netmask:", route.RouteInfo.Mask, "nexthop:", route.RouteInfo.NextHopIp))
        routes = append(routes, &route.RouteInfo)
    }
    //server.ProcessConnectedRoutes(make([]*ribd.Routes, 0), routes)
}
*/
