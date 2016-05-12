package server

import (
	//"fmt"
	//"github.com/garyburd/redigo/redis"
	"models"
	"ospfd"
	//"strconv"
	"errors"
	"l3/ospf/config"
	"utils/dbutils"
)

func (server *OSPFServer) InitializeDB() error {
	server.dbHdl = dbutils.NewDBUtil(server.logger)
	err := server.dbHdl.Connect()
	if err != nil {
		server.logger.Err("Failed to create the DB Handle")
		return err
	}
	return nil
}

func (server *OSPFServer) ReadOspfCfgFromDB() {
	server.readGlobalConfFromDB()
	server.readAreaConfFromDB()
	server.readIntfConfFromDB()
}

func (server *OSPFServer) readGlobalConfFromDB() {
	server.logger.Info("Reading global object from DB")
	var dbObj models.OspfGlobal

	objList, err := server.dbHdl.GetAllObjFromDb(dbObj)
	if err != nil {
		server.logger.Err("DB query failed for OspfGlobal")
		return
	}
	for idx := 0; idx < len(objList); idx++ {
		obj := ospfd.NewOspfGlobal()
		dbObject := objList[idx].(models.OspfGlobal)
		models.ConvertospfdOspfGlobalObjToThrift(&dbObject, obj)
		err := server.applyOspfGlobalConf(obj)
		if err != nil {
			server.logger.Err("Error applying Ospf Global Configuration")
		}
	}
}

func (server *OSPFServer) applyOspfGlobalConf(conf *ospfd.OspfGlobal) error {
	gConf := config.GlobalConf{
		RouterId:                 config.RouterId(conf.RouterId),
		ASBdrRtrStatus:           conf.ASBdrRtrStatus,
		TOSSupport:               conf.TOSSupport,
		RestartSupport:           config.RestartSupport(conf.RestartSupport),
		RestartInterval:          conf.RestartInterval,
	}
	err := server.processGlobalConfig(gConf)
	if err != nil {
		server.logger.Err("Error Configuring Ospf Global Configuration")
		err := errors.New("Error Configuring Ospf Global Configuration")
		return err
	}

	return nil
}

func (server *OSPFServer) readAreaConfFromDB() {
	server.logger.Info("Reading area object from DB")
	var dbObj models.OspfAreaEntry

	objList, err := server.dbHdl.GetAllObjFromDb(dbObj)
	if err != nil {
		server.logger.Err("DB query failed for OspfAreaEntry")
		return
	}
	for idx := 0; idx < len(objList); idx++ {
		obj := ospfd.NewOspfAreaEntry()
		dbObject := objList[idx].(models.OspfAreaEntry)
		models.ConvertospfdOspfAreaEntryObjToThrift(&dbObject, obj)
		err := server.applyOspfAreaConf(obj)
		if err != nil {
			server.logger.Err("Error applying Ospf Area Configuration")
		}
	}
}

func (server *OSPFServer) applyOspfAreaConf(conf *ospfd.OspfAreaEntry) error {
	aConf := config.AreaConf{
		AreaId:                              config.AreaId(conf.AreaId),
		AuthType:                            config.AuthType(conf.AuthType),
		ImportAsExtern:                      config.ImportAsExtern(conf.ImportAsExtern),
		AreaSummary:                         config.AreaSummary(conf.AreaSummary),
		AreaNssaTranslatorRole:              config.NssaTranslatorRole(conf.AreaNssaTranslatorRole),
	}
	err := server.processAreaConfig(aConf)
	if err != nil {
		server.logger.Err("Error Configuring Ospf Area Configuration")
		err := errors.New("Error Configuring Ospf Area Configuration")
		return err
	}
	return nil
}

func (server *OSPFServer) readIntfConfFromDB() {
	server.logger.Info("Reading interface object from DB")
	var dbObj models.OspfIfEntry

	objList, err := server.dbHdl.GetAllObjFromDb(dbObj)
	if err != nil {
		server.logger.Err("DB query failed for OspfIfEntry")
		return
	}
	for idx := 0; idx < len(objList); idx++ {
		obj := ospfd.NewOspfIfEntry()
		dbObject := objList[idx].(models.OspfIfEntry)
		models.ConvertospfdOspfIfEntryObjToThrift(&dbObject, obj)
		err := server.applyOspfIntfConf(obj)
		if err != nil {
			server.logger.Err("Error applying Ospf Area Configuration")
		}
	}
}

func (server *OSPFServer) applyOspfIntfConf(conf *ospfd.OspfIfEntry) error {
	ifConf := config.InterfaceConf{
		IfIpAddress:           config.IpAddress(conf.IfIpAddress),
		AddressLessIf:         config.InterfaceIndexOrZero(conf.AddressLessIf),
		IfAreaId:              config.AreaId(conf.IfAreaId),
		IfType:                config.IfType(conf.IfType),
		IfRtrPriority:         config.DesignatedRouterPriority(conf.IfRtrPriority),
		IfTransitDelay:        config.UpToMaxAge(conf.IfTransitDelay),
		IfRetransInterval:     config.UpToMaxAge(conf.IfRetransInterval),
		IfHelloInterval:       config.HelloRange(conf.IfHelloInterval),
		IfRtrDeadInterval:     config.PositiveInteger(conf.IfRtrDeadInterval),
		IfPollInterval:        config.PositiveInteger(conf.IfPollInterval),
		IfAuthKey:             conf.IfAuthKey,
		IfAuthType:            config.AuthType(conf.IfAuthType),
	}

	err := server.processIntfConfig(ifConf)
	if err != nil {
		server.logger.Err("Error Configuring Ospf Area Configuration")
		err := errors.New("Error Configuring Ospf Area Configuration")
		return err
	}
	return nil

}
