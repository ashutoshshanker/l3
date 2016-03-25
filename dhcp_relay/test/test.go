package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	_ "log/syslog"
	_ "net"
	"os"
	_ "strings"
	_ "time"
	"utils/dbutils"

	_ "github.com/mattn/go-sqlite3"
	_ "golang.org/x/net/ipv4"
)

type DhcpRelayIntf struct {
	IpSubnet string `SNAPROUTE: "KEY"` // Ip Address of the interface
	Netmask  string `SNAPROUTE: "KEY"` // NetMaks of the interface
	IfIndex  string `SNAPROUTE: "KEY"` // Unique If Id of the interface
	// Use below field for agent sub-type
	AgentSubType int32
	Enable       bool
	// To make life easy for testing first pass lets have only 1 server
	ServerIp []string
	//ServerIp string
}

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

func executeSQL(dbCmd string, dbHdl *sql.DB) (driver.Result, error) {
	var result driver.Result
	txn, err := dbHdl.Begin()
	if err != nil {
		fmt.Println("### Failed to strart db transaction for command", dbCmd)
		return result, err
	}
	result, err = dbHdl.Exec(dbCmd)
	if err != nil {
		fmt.Println("### Failed to execute command ", dbCmd, err)
		return result, err
	}
	err = txn.Commit()
	if err != nil {
		fmt.Println("### Failed to Commit transaction for command", dbCmd, err)
		return result, err
	}
	return result, err

}

func testDB() {
	obj := DhcpRelayIntf{
		IpSubnet:     "100.0.1.1",
		Netmask:      "255.255.255.0",
		IfIndex:      "9",
		AgentSubType: 0,
		Enable:       true,
		ServerIp:     []string{"90.0.1.2", "80.0.1.2"},
	}
	fmt.Println(obj)
	fmt.Println("Hello, playground")
	err := os.Remove("/home/jgheewala/dummy.sqlite")
	if err != nil {
		fmt.Println(err)
	}
	db, err := sql.Open("sqlite3", "/home/jgheewala/dummy.sqlite")
	if err != nil {
		fmt.Println(err)
	}

	dbCmd := "PRAGMA foreign_keys = ON;"
	_, err = executeSQL(dbCmd, db)
	if err != nil {
		fmt.Sprintln("dbCmd", dbCmd, "failed with err", err)
	}
	dbCmd = "CREATE TABLE IF NOT EXISTS DhcpRelayIntfConfig " +
		"( " +
		"IpSubnet TEXT, " +
		"Netmask TEXT, " +
		"IfIndex TEXT, " +
		"AgentSubType INTEGER, " +
		"Enable INTEGER, " +
		"PRIMARY KEY(IpSubnet, Netmask, IfIndex) " +
		")"
	_, err = executeSQL(dbCmd, db)
	dbCmd = "CREATE TABLE IF NOT EXISTS DhcpRelayIntfConfigServer " +
		"( " +
		"IpSubnet TEXT, " +
		"Netmask TEXT, " +
		"IfIndex TEXT, " +
		"ServerIp TEXT,\n" +
		`CONSTRAINT FK_DhcpRelayServerList
           FOREIGN KEY (IpSubnet, Netmask, IfIndex)
	       REFERENCES DhcpRelayIntfConfig (IpSubnet, Netmask, IfIndex)
	       ON DELETE CASCADE` +
		")"
	_, err = executeSQL(dbCmd, db)
	if err != nil {
		fmt.Sprintln("dbCmd", dbCmd, "failed with err", err)
	}
	/*
		dbCmd = "ALTER TABLE DhcpRelayIntfConfigServer\n" +
			"CHECK CONSTRAINT FK_DhcpRelayServerList"
		fmt.Println(dbCmd)
		trimS := strings.TrimSpace(dbCmd)
		_, err = executeSQL(trimS, db)
		if err != nil {
			fmt.Sprintln("dbCmd", dbCmd, "failed with err", err)
		}
	*/
	dbCmd = fmt.Sprintf("INSERT INTO DhcpRelayIntfConfig (IpSubnet, Netmask, IfIndex, AgentSubType, Enable) VALUES ('%v', '%v', '%v', '%v', '%v') ;",
		obj.IpSubnet, obj.Netmask, obj.IfIndex, obj.AgentSubType, dbutils.ConvertBoolToInt(obj.Enable))
	result, err := executeSQL(dbCmd, db)
	if err != nil {
		fmt.Println("**** Failed to Create table", err)
	} else {
		_, err = result.LastInsertId()
		if err != nil {
			fmt.Println("### Failed to return last object id", err)
		}
	}
	for i := 0; i < len(obj.ServerIp); i++ {
		dbCmd = fmt.Sprintf("INSERT INTO DhcpRelayIntfConfigServer (IpSubnet, Netmask, IfIndex, ServerIp) VALUES ('%v', '%v', '%v', '%v') ;",
			obj.IpSubnet, obj.Netmask, obj.IfIndex, obj.ServerIp[i])
		result, err := executeSQL(dbCmd, db)
		if err != nil {
			fmt.Println("**** Failed to Create table", err)
		} else {
			_, err = result.LastInsertId()
			if err != nil {
				fmt.Println("### Failed to return last object id", err)
			}
		}
	}
}

func main() {
	testDB()
	/*
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
	/*
		}

			addr := "localhost:7000"
			err := StartTestClient(addr)
			if err != nil {
				fmt.Println("Failed to start test client.. Exiting!!!!")
				return
			}
	*/
}
