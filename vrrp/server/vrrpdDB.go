package vrrpServer

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"models"
	"vrrpd"
)

const ()

func (svr *VrrpServer) VrrpInitDB() error {
	svr.logger.Info("Initializing SQL DB")
	var err error
	svr.vrrpDbHdl, err = redis.Dial("tcp", VRRP_REDDIS_DB_PORT)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
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
	svr.logger.Info("Reading VrrpIntf Config from DB")
	if svr.vrrpDbHdl == nil {
		return nil
	}
	var dbObj models.VrrpIntf
	objList, err := dbObj.GetAllObjFromDb(svr.vrrpDbHdl)
	if err != nil {
		svr.logger.Warning("DB querry failed for VrrpIntf Config")
		return err
	}
	for idx := 0; idx < len(objList); idx++ {
		obj := vrrpd.NewVrrpIntf()
		dbObject := objList[idx].(models.VrrpIntf)
		models.ConvertvrrpdVrrpIntfObjToThrift(&dbObject, obj)
		svr.VrrpCreateGblInfo(*obj)
	}
	svr.logger.Info("Done reading from DB")
	return err
}
