// bgp_client.go
package main

import (
	"fmt"
    "l3/bgp/server"
    "net"
    "net/rpc"
)

const CONF_IP string = "localhost" //"10.0.2.15"
const CONF_PORT string = "2001"

func main() {
	fmt.Println("Starting the BGP client...")
    client, err := rpc.Dial("tcp", CONF_IP + ":" + CONF_PORT)
    if err != nil {
        fmt.Println("Connection to server failed")
    }
    
    var reply bool
    globalConfigArgs := &server.GlobalConfigAttrs{10001}
    err = client.Call("ConfigInterface.SetBGPConfig", globalConfigArgs, &reply)
    if err != nil {
        fmt.Println("ConfigInterface.SetBGPConfig FAILED with err:", err)
    }
    
    peerConfigArgs := &server.PeerConfigAttrs{net.ParseIP("127.0.0.1"), 10002}
    err = client.Call("ConfigInterface.AddPeer", peerConfigArgs, &reply)
    if err != nil {
        fmt.Println("ConfigInterface.AddPeer FAILED with err:", err)
    }
}
