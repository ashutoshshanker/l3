package rpc

import (
	"bfdd"
	"github.com/garyburd/redigo/redis"
	"l3/bfd/server"
	"models"
	"utils/logging"
)

type BFDHandler struct {
	server *server.BFDServer
	logger *logging.Writer
}

func NewBFDHandler(logger *logging.Writer, server *server.BFDServer) *BFDHandler {
	h := new(BFDHandler)
	h.server = server
	h.logger = logger
	return h
}

func (h *BFDHandler) ReadGlobalConfigFromDB(dbHdl redis.Conn) error {
	h.logger.Info("Reading BfdGlobal")
	if dbHdl != nil {
		var dbObj models.BfdGlobal
		objList, err := dbObj.GetAllObjFromDb(dbHdl)
		if err != nil {
			h.logger.Err("DB query failed for global config")
			return err
		}
		for idx := 0; idx < len(objList); idx++ {
			obj := bfdd.NewBfdGlobal()
			dbObject := objList[idx].(models.BfdGlobal)
			models.ConvertbfddBfdGlobalObjToThrift(&dbObject, obj)
			rv, _ := h.CreateBfdGlobal(obj)
			if rv == false {
				h.logger.Err("BfdGlobal create failed")
				return nil
			}
		}
	}
	return nil
}

func (h *BFDHandler) ReadSessionParamConfigFromDB(dbHdl redis.Conn) error {
	h.logger.Info("Reading BfdSessionParam")
	if dbHdl != nil {
		var dbObj models.BfdSessionParam
		objList, err := dbObj.GetAllObjFromDb(dbHdl)
		if err != nil {
			h.logger.Err("DB query failed for session param config")
			return err
		}
		for idx := 0; idx < len(objList); idx++ {
			obj := bfdd.NewBfdSessionParam()
			dbObject := objList[idx].(models.BfdSessionParam)
			models.ConvertbfddBfdSessionParamObjToThrift(&dbObject, obj)
			rv, _ := h.CreateBfdSessionParam(obj)
			if rv == false {
				h.logger.Err("BfdSessionParam create failed")
				return nil
			}
		}
	}
	return nil
}

func (h *BFDHandler) ReadSessionConfigFromDB(dbHdl redis.Conn) error {
	h.logger.Info("Reading BfdSession")
	if dbHdl != nil {
		var dbObj models.BfdSession
		objList, err := dbObj.GetAllObjFromDb(dbHdl)
		if err != nil {
			h.logger.Err("DB query failed for session config")
			return err
		}
		for idx := 0; idx < len(objList); idx++ {
			obj := bfdd.NewBfdSession()
			dbObject := objList[idx].(models.BfdSession)
			models.ConvertbfddBfdSessionObjToThrift(&dbObject, obj)
			rv, _ := h.CreateBfdSession(obj)
			if rv == false {
				h.logger.Err("BfdSession create failed")
				return nil
			}
		}
	}
	return nil
}

func (h *BFDHandler) ReadConfigFromDB(dbHdl redis.Conn) error {
	// BfdGlobalConfig
	h.ReadGlobalConfigFromDB(dbHdl)
	// BfdIntfConfig
	h.ReadSessionParamConfigFromDB(dbHdl)
	// BfdSessionConfig
	h.ReadSessionConfigFromDB(dbHdl)
	return nil
}
