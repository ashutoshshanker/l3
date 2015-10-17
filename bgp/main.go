// main.go
package main

import (
    "fmt"
    "l3/bgp/server"
)

const IP string = "localhost" //"10.0.2.15"
const BGPPort string = "179"
const CONF_PORT string = "2001"

func main() {
    //var globalConfig GlobalConfig{as: 65000}
    //var peerConfig = []PeerConfig {{"192.168.1.2"}, {"192.168.1.3"}}
    //var configInterface ConfigInterface

    fmt.Println("Start config listener")
    confIface := server.NewConfigInterface()
    go server.StartConfigListener(confIface, IP, CONF_PORT)


    fmt.Println("Start BGP Server")
    bgpServer := server.NewBgpServer()
    go bgpServer.StartServer()

    for {
        select {
            case globalConfigAttrs := <-confIface.GlobalConfigCh:
                globalConfig := server.GlobalConfig{
                    AS: uint16(globalConfigAttrs.AS),
                }
                bgpServer.GlobalConfigCh <- globalConfig
            case peerConfigAttrs := <-confIface.AddPeerConfigCh:
                peerConfig := server.PeerConfig{
                    AS: uint16(peerConfigAttrs.AS),
                    IP: peerConfigAttrs.IP,
                }
                bgpServer.AddPeerCh <- peerConfig
            case peerConfigAttrs := <-confIface.RemPeerConfigCh:
                peerConfig := server.PeerConfig{
                    AS: uint16(peerConfigAttrs.AS),
                    IP: peerConfigAttrs.IP,
                }
                bgpServer.RemPeerCh <- peerConfig
            case peercommandAttrs := <-confIface.PeerCommandCh:
                peerCommand := server.PeerCommand{
                    IP: peercommandAttrs.IP,
                    Command: peercommandAttrs.Command,
                }
                bgpServer.PeerCommandCh <- peerCommand
        }
    }
}

