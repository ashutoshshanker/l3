package vrrpServer

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

func VrrpInitDB() error {
	logger.Info("VRRP: Initializing SQL DB")
	var err error
	dbName := paramsDir + USR_CONF_DB
	logger.Info("VRRP: location for DB is " + dbName)
	vrrpDbHdl, err = sql.Open("sqlite3", dbName)
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Failed to Create DB Handle", err))
		return err
	}

	if err = vrrpDbHdl.Ping(); err != nil {
		logger.Err(fmt.Sprintln("Failed to keep db connection alive", err))
		return err
	}
	logger.Info("VRRP: DB connection is established")
	return err
}

func VrrpReadDB() error {
	logger.Info("VRRP: Reading from Database")
	dbCmd := "SELECT * FROM VrrpIntfConfig"
	rows, err := vrrpDbHdl.Query(dbCmd)
	if err != nil {
		logger.Err(fmt.Sprintln("VRRP: Unable to querry DB:", err))
		vrrpDbHdl.Close()
		return err
	}

	for rows.Next() {
		//@TODO: finish implementation
	}
	vrrpDbHdl.Close()
	return err
}
