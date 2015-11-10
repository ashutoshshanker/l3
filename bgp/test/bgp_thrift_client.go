// bgp_client.go
package main

import (
	"fmt"
	"generated/src/bgpd"
	"git.apache.org/thrift.git/lib/go/thrift"
)

const CONF_IP string = "localhost" //"10.0.2.15"
const CONF_PORT string = "4050"

func main() {
	fmt.Println("Starting the BGP thrift client...")
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket("localhost:" + CONF_PORT)
	if err != nil {
		fmt.Println("NewTSocket failed with error:", err)
		return
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		fmt.Println("Failed to open the socket, error:", err)
	}

	client := bgpd.NewBgpServerClientFactory(clientTransport, protocolFactory)

	globalConfigArgs := bgpd.NewBgpGlobal()
	globalConfigArgs.AS = 5000
	globalConfigArgs.RouterId = "localhost"
	fmt.Println("calling CreateBgpGlobal with attr:", globalConfigArgs)
	ret,_ := client.CreateBgp(globalConfigArgs)
	if !ret {
		fmt.Println("CreateBgpGlobal FAILED")
	}
	fmt.Println("Created BGP global conf")

	peerConfigArgs := bgpd.NewBgpPeer()
	peerConfigArgs.NeighborAddress = "11.1.11.203"
	peerConfigArgs.LocalAS = 5000
	peerConfigArgs.PeerAS = 5000
	peerConfigArgs.Description = "IBGP Peer"
	fmt.Println("calling CreateBgpPeer with attr:", peerConfigArgs)
	ret, _ = client.CreatePeer(peerConfigArgs)
	if !ret {
		fmt.Println("CreateBgpPeer FAILED")
	}
	fmt.Println("Created BGP peer conf")

//	peerCommandArgs := &server.PeerConfigCommands{net.ParseIP("11.1.11.203"), 1}
//	err = client.Call("ConfigInterface.PeerCommand", peerCommandArgs, &reply)
//	if err != nil {
//		fmt.Println("ConfigInterface.AddPeer FAILED with err:", err)
//	}

}
