package relayServer

import (
	"database/sql"
	_ "dhcprelayd"
	"flag"
	"fmt"
	_ "utils/dbutils"
)

func DhcpRelayAgentInitDB() error {
	logger.Info("DRA: initializing SQL DB")
	params_dir := flag.String("params", "", "Directory Location for config files")
	flag.Parse()
	paramsDir = *params_dir
	dbName := paramsDir + USR_CONF_DB
	logger.Info("DRA: location of DB is " + dbName)
	dhcprelayDbHdl, err := sql.Open("sqlite3", dbName)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to create db handle", err))
		return err
	}

	if err = dhcprelayDbHdl.Ping(); err != nil {
		logger.Err(fmt.Sprintln("Failed to keep db connection alive", err))
		return err
	}
	return err
}

func DhcpRelayAgentReadDB() {
	logger.Info("DRA: Populate Dhcp Relay Info from DB entries")
	readIfIndex := make([]int32, 0)
	rows, err := dhcprelayDbHdl.Query("SELECT * FROM DhcpRelayIntfConfig")
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Unable to querry DB:", err))
		return
	}

	for rows.Next() {
		var IfIndex int32
		var Enable int
		err = rows.Scan(&IfIndex, &Enable)
		if err != nil {
			logger.Info(fmt.Sprintln("DRA: Unable to scan entries from DB",
				err))
			return
		}
		logger.Info(fmt.Sprintln("DRA: ifindex:", IfIndex,
			"enabled:", Enable))
		DhcpRelayAgentInitGblHandling(IfIndex, (Enable != 0))
		DhcpRelayAgentInitIntfState(IfIndex)
		readIfIndex = append(readIfIndex, IfIndex)
	}

	rows, err = dhcprelayDbHdl.Query("SELECT * FROM DhcpRelayIntfConfigServer")
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Unable to querry DB:", err))
		return
	}

	for rows.Next() {
		var IfIndex int32
		var serverIp string
		err = rows.Scan(&IfIndex, &serverIp)
		if err != nil {
			logger.Info(fmt.Sprintln("DRA: Unable to scan entried from DB",
				err))
			return
		}
		logger.Info(fmt.Sprintln("DRA: ifindex:", IfIndex, "Server Ip:",
			serverIp))
		DhcpRelayAgentUpdateIntfServerIp(IfIndex, serverIp)
	}

	if len(readIfIndex) > 0 {
		go DhcpRelayAgentUpdateIntfIpAddr(readIfIndex)
	}
}
