// main.go
package main

import (
    "fmt"
	"l3/bgp/rpc"
    "l3/bgp/server"
)

const IP string = "localhost" //"10.0.2.15"
const BGPPort string = "179"
const CONF_PORT string = "2001"
const BGPConfPort string = "2001"

func main() {
    fmt.Println("Start BGP Server")
    bgpServer := server.NewBgpServer()
    go bgpServer.StartServer()

    fmt.Println("Start config listener")
	confIface := rpc.NewBgpHandler(bgpServer)
	rpc.StartServer(confIface, BGPConfPort)
}

