package vrrpServer

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"vrrpd"
)

func (svr *VrrpServer) VrrpInitDB() error {
	svr.logger.Info("Initializing SQL DB")
	var err error
	dbName := svr.paramsDir + VRRP_USR_CONF_DB
	svr.logger.Info("VRRP: location for DB is " + dbName)
	svr.vrrpDbHdl, err = sql.Open("sqlite3", dbName)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
		return err
	}

	if err = svr.vrrpDbHdl.Ping(); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to keep db connection alive", err))
		return err
	}
	svr.logger.Info("DB connection is established")
	return err
}

func (svr *VrrpServer) VrrpCloseDB() {
	svr.logger.Info("Closed vrrp db")
	svr.vrrpDbHdl.Close()
}

func (svr *VrrpServer) VrrpReadDB() error {
	svr.logger.Info("Reading from Database")
	dbCmd := "SELECT * FROM " + VRRP_INTF_DB
	rows, err := svr.vrrpDbHdl.Query(dbCmd)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Unable to querry DB:", err))
		return err
	}

	for rows.Next() {
		var config vrrpd.VrrpIntf
		err = rows.Scan(&config.IfIndex, &config.VRID,
			&config.Priority, &config.VirtualIPv4Addr,
			&config.AdvertisementInterval, &config.PreemptMode,
			&config.AcceptMode)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("scanning rows failed", err))
		} else {
			svr.VrrpCreateGblInfo(config)
		}
	}
	svr.logger.Info("Done reading from DB")
	return err
}
