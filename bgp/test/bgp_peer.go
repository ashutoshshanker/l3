// bgp_client.go
package main

import (
	"fmt"
    "net"
)

const CONF_IP string = "localhost" //"10.0.2.15"
const CONF_PORT string = "179"

func main() {
	fmt.Println("Starting the BGP peer...")
    
    tcpAddr, err := net.ResolveTCPAddr("tcp", CONF_IP + ":" + CONF_PORT)
    if err != nil {
        fmt.Println("ResolveTCPAddr failed with", err)
    }

    lAddr, err := net.ResolveTCPAddr("tcp", CONF_IP + ":" + "10001")
    if err != nil {
        fmt.Println("ResolveTCPAddr failed with", err)
    }
        
    client, err := net.DialTCP("tcp", lAddr, tcpAddr)
    if err != nil {
        fmt.Println("Connection to server failed")
    }

    packet := make([]byte, 80)
    var num int
    
    for {
        num, err = client.Read(packet)
        if err != nil {
            fmt.Println("Read failed with error:%s", err)
        }
        fmt.Println("bytes received:", num)
        fmt.Println("Received packet:", packet)
    }
    client.Close()
}
