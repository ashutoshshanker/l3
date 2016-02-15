package relayServer

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

func DhcpRelayAgentInitDB() error {
	logger.Info("DRA: initializing SQL DB")
	var err error
	params_dir := flag.String("params", "", "Directory Location for config files")
	flag.Parse()
	paramsDir = *params_dir
	dbName := paramsDir + USR_CONF_DB
	logger.Info("DRA: location of DB is " + dbName)
	dhcprelayDbHdl, err = sql.Open("sqlite3", dbName)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to create db handle", err))
		return err
	}

	if err = dhcprelayDbHdl.Ping(); err != nil {
		logger.Err(fmt.Sprintln("Failed to keep db connection alive", err))
		return err
	}
	logger.Info("DRA: SQL DB init success")
	return err
}

func DhcpRelayAgentReadDB() {
	dbCmd := "SELECT * FROM DhcpRelayIntfConfig"
	logger.Info("DRA: Populate Dhcp Relay Info via " + dbCmd)
	rows, err := dhcprelayDbHdl.Query(dbCmd)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Unable to querry DB:", err))
		dhcprelayDbHdl.Close()
		return
	}

	readIfIndex := make([]int32, 0)
	for rows.Next() {
		var IfIndex int32
		var Enable int
		err = rows.Scan(&IfIndex, &Enable)
		if err != nil {
			logger.Info(fmt.Sprintln("DRA: Unable to scan entries from DB",
				err))
			dhcprelayDbHdl.Close()
			return
		}
		logger.Info(fmt.Sprintln("DRA: ifindex:", IfIndex,
			"enabled:", Enable))
		DhcpRelayAgentInitGblHandling(IfIndex, (Enable != 0))
		DhcpRelayAgentInitIntfState(IfIndex)
		readIfIndex = append(readIfIndex, IfIndex)
	}
	dbCmd = "SELECT * FROM DhcpRelayIntfConfigServer"
	logger.Info("DRA: Populate Dhcp Relay Server Info via " + dbCmd)
	rows, err = dhcprelayDbHdl.Query(dbCmd)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Unable to querry DB:", err))
		dhcprelayDbHdl.Close()
		return
	}

	for rows.Next() {
		var IfIndex int32
		var serverIp string
		err = rows.Scan(&IfIndex, &serverIp)
		if err != nil {
			logger.Info(fmt.Sprintln("DRA: Unable to scan entried from DB",
				err))
			dhcprelayDbHdl.Close()
			return
		}
		logger.Info(fmt.Sprintln("DRA: ifindex:", IfIndex, "Server Ip:",
			serverIp))
		DhcpRelayAgentUpdateIntfServerIp(IfIndex, serverIp)
	}

	if len(readIfIndex) > 0 {
		go DhcpRelayAgentUpdateIntfIpAddr(readIfIndex)
	} else {
		dhcprelayDbHdl.Close()
	}
}
