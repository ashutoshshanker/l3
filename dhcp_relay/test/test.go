package main

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"log/syslog"
	"net"
)

/*
func StartTestClient(addr string) error {
	// create transport and protocol for server
	fmt.Println("Request for starting Dhcp Relay Test Client")
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	socket, err := thrift.NewTSocket(addr)
	if err != nil {
		fmt.Println("Error Opening Socket at addr %s", addr)
		return err
	}
	transport := transportFactory.GetTransport(socket)
	defer transport.Close()
	if err := transport.Open(); err != nil {
		return err
	}
	fmt.Printf("Transport for Test Client created successfully\n")
	fmt.Println("client started at %s", addr)

	//Create a client for communicating with the server
	client := dhcprelayd.NewDhcpRelayServerClientFactory(transport, protocolFactory)
	fmt.Println("DHCP RELAY TEST Client Started")
	fmt.Println("Calling add relay agent")

	//Create dhcprelay configuration structure
	globalConfigArgs := dhcprelayd.NewDhcpRelayConf()
	globalConfigArgs.IpSubnet = "10.10.1.1"
	globalConfigArgs.IfIndex = "Ethernet1/1"

	// Call add relay agent api for the client with configuration
	err = client.AddRelayAgent(globalConfigArgs)
	if err != nil {
		fmt.Println("Add Relay Agent returned error")
		return err
	}
	fmt.Println("addition of relay agent success")
	fmt.Println("calling update relay agent")
	err = client.UpdRelayAgent()
	if err != nil {
		fmt.Println("Update Relay Agent returned error")
		return err
	}
	fmt.Println("updation of relay agent success")
	fmt.Println("calling delete relay agent")
	err = client.DelRelayAgent()
	if err != nil {
		fmt.Println("Delete Relay Agent returned error")
		return err
	}
	fmt.Println("deletion of relay agent successful")
	return nil
}
*/
func main() {
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR DHCP RELAY")
	if err != nil {
		fmt.Println("Failed to start the logger... Exiting!!!")
		return
	}
	logger.Info("Started the logger successfully.")
	caddr := net.UDPAddr{
		Port: 68, //DHCP_CLIENT_PORT,
		IP:   net.ParseIP(""),
	}
	controlFlag := ipv4.FlagTTL | ipv4.FlagSrc | ipv4.FlagDst | ipv4.FlagInterface
	dhcprelayServerHandler, err := net.ListenUDP("udp", &caddr)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Opening udp port for server --> client failed", err))
		// do we need to close the client server communication??? ask
		// Hari/Adam
		return
	}
	dhcprelayServerConn := ipv4.NewPacketConn(dhcprelayServerHandler)
	err = dhcprelayServerConn.SetControlMessage(controlFlag, true)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA:Setting control flag for server failed..", err))
		return
	}
	logger.Info("DRA: Server Connection opened successfully")
	var buf []byte = make([]byte, 1500)
	for {
		bytesRead, cm, srcAddr, err := dhcprelayServerConn.ReadFrom(buf)
		if err != nil {
			logger.Err("DRA: reading buffer failed")
			continue
		}
		logger.Info("DRA: Received PACKET FROM SERVER")
		logger.Info(fmt.Sprintln("DRA: control message is ", cm))
		logger.Info(fmt.Sprintln("DRA: srcAddr is ", srcAddr))
		logger.Info(fmt.Sprintln("DRA: bytes read is ", bytesRead))
		//logger.Info(fmt.Sprintln("DRA: MessageType is ", mType))
		/*
			inReq, reqOptions, mType := DhcpRelayAgentDecodeInPkt(buf, bytesRead)
			if inReq == nil || reqOptions == nil {
				logger.Warning("DRA: Couldn't decode dhcp packet....continue")
				continue
			}
			// Get the interface from reverse mapping to send the unicast
			// packet...
			outIfId := dhcprelayReverseMap[inReq.GetCHAddr().String()]
			logger.Info(fmt.Sprintln("DRA: Send unicast packet to Interface Id:", outIfId))
			gblEntry, ok := dhcprelayGblInfo[outIfId]
			if !ok {
				// dropping the packet??
				logger.Err(fmt.Sprintln("DRA: dra is not enable on", outIfId, "??"))
				continue
			}
		*/
	}
	/*
		addr := "localhost:7000"
		err := StartTestClient(addr)
		if err != nil {
			fmt.Println("Failed to start test client.. Exiting!!!!")
			return
		}
	*/
}
