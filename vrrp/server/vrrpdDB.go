package vrrpServer

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

func (svr *VrrpServer) VrrpInitDB() error {
	svr.logger.Info("VRRP: Initializing SQL DB")
	var err error
	dbName := svr.paramsDir + VRRP_USR_CONF_DB
	svr.logger.Info("VRRP: location for DB is " + dbName)
	svr.vrrpDbHdl, err = sql.Open("sqlite3", dbName)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("VRRP: Failed to Create DB Handle", err))
		return err
	}

	if err = svr.vrrpDbHdl.Ping(); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to keep db connection alive", err))
		return err
	}
	svr.logger.Info("VRRP: DB connection is established")
	return err
}

func (svr *VrrpServer) VrrpReadDB() error {
	svr.logger.Info("VRRP: Reading from Database")
	dbCmd := "SELECT * FROM VrrpIntfConfig"
	rows, err := svr.vrrpDbHdl.Query(dbCmd)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("VRRP: Unable to querry DB:", err))
		svr.vrrpDbHdl.Close()
		return err
	}

	for rows.Next() {
		//@TODO: finish implementation
	}
	svr.vrrpDbHdl.Close()
	return err
}
