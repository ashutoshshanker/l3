package server

import (
	"l3/ospf/config"
	"time"
	"fmt"
)

type AreaConfKey struct {
	AreaId config.AreaId
}

/* TODO - Add list of interfaces for this Area */
type AreaConf struct {
	AuthType                            config.AuthType
	ImportAsExtern                      config.ImportAsExtern
	AreaSummary                         config.AreaSummary
	AreaNssaTranslatorRole              config.NssaTranslatorRole
	AreaNssaTranslatorStabilityInterval config.PositiveInteger
	TransitCapability                   bool
	IntfListMap                         map[IntfConfKey]bool
}

type AreaState struct {
	SpfRuns                  int32
	AreaBdrRtrCount          int32
	AsBdrRtrCount            int32
	AreaLsaCount             int32
	AreaLsaCksumSum          int32
	AreaNssaTranslatorState  config.NssaTranslatorState
	AreaNssaTranslatorEvents int32
}

func (server *OSPFServer) processAreaConfig(areaConf config.AreaConf) {
	areaConfKey := AreaConfKey{
		AreaId: areaConf.AreaId,
	}

	ent, _ := server.AreaConfMap[areaConfKey]
	ent.AuthType = areaConf.AuthType
	ent.ImportAsExtern = areaConf.ImportAsExtern
	ent.AreaSummary = areaConf.AreaSummary
	ent.AreaNssaTranslatorRole = areaConf.AreaNssaTranslatorRole
	ent.AreaNssaTranslatorStabilityInterval = areaConf.AreaNssaTranslatorStabilityInterval
	ent.IntfListMap = make(map[IntfConfKey]bool)
	server.AreaConfMap[areaConfKey] = ent
	server.initAreaStateSlice(areaConfKey)
	areaId := convertAreaOrRouterIdUint32(string(areaConf.AreaId))
	server.initLSDatabase(areaId)
}

func (server *OSPFServer) initAreaConfDefault() {
	server.logger.Info("Initializing default area config")
	areaConfKey := AreaConfKey{
		AreaId: "0.0.0.0",
	}
	ent, exist := server.AreaConfMap[areaConfKey]
	if !exist {
		ent.AuthType = config.NoAuth
		ent.ImportAsExtern = config.ImportExternal
		ent.AreaSummary = config.NoAreaSummary
		ent.AreaNssaTranslatorRole = config.Candidate
		ent.AreaNssaTranslatorStabilityInterval = config.PositiveInteger(40)
		ent.IntfListMap = make(map[IntfConfKey]bool)
		server.AreaConfMap[areaConfKey] = ent
	}
	server.initAreaStateSlice(areaConfKey)
	server.areaStateRefresh()
}

func (server *OSPFServer) initAreaStateSlice(key AreaConfKey) {
	//server.AreaStateMutex.Lock()
	server.logger.Info("Initializing area slice")
	ent, exist := server.AreaStateMap[key]
	ent.SpfRuns = 0
	ent.AreaBdrRtrCount = 0
	ent.AsBdrRtrCount = 0
	ent.AreaLsaCount = 0
	ent.AreaLsaCksumSum = 0
	ent.AreaNssaTranslatorState = config.NssaTranslatorDisabled
	ent.AreaNssaTranslatorEvents = 0
	server.AreaStateMap[key] = ent
	if !exist {
		server.AreaStateSlice = append(server.AreaStateSlice, key)
		server.AreaConfKeyToSliceIdxMap[key] = len(server.AreaStateSlice) - 1
	}
	//server.AreaStateMutex.Unlock()
}

func (server *OSPFServer) areaStateRefresh() {
	var areaStateRefFunc func()
	areaStateRefFunc = func() {
		//server.AreaStateMutex.Lock()
		server.logger.Info("Inside areaStateRefFunc()")
		server.AreaStateSlice = []AreaConfKey{}
		server.AreaConfKeyToSliceIdxMap = nil
		server.AreaConfKeyToSliceIdxMap = make(map[AreaConfKey]int)
		for key, _ := range server.AreaStateMap {
			server.AreaStateSlice = append(server.AreaStateSlice, key)
			server.AreaConfKeyToSliceIdxMap[key] = len(server.AreaStateSlice) - 1
		}
		//server.AreaStateMutex.Unlock()
		server.AreaStateTimer.Reset(server.RefreshDuration)
	}
	server.AreaStateTimer = time.AfterFunc(server.RefreshDuration, areaStateRefFunc)
}

func (server *OSPFServer) updateIntfToAreaMap(key IntfConfKey, areaId config.AreaId) config.AreaId {
	def_key := AreaConfKey{ 
		AreaId: "0.0.0.0",
	}
	areaConfKey := AreaConfKey{
		AreaId: areaId,
	}
	ent, exist := server.AreaConfMap[areaConfKey]
	if exist {
		// update the intf list
		ent.IntfListMap[key] = true
		server.AreaConfMap[areaConfKey] = ent
		def_key = areaConfKey
	} else {
		server.logger.Info(fmt.Sprintln("Add interface to the default area 0 . Intf - ", key.IPAddr))
		def_ent, exists := server.AreaConfMap[def_key]
		if !exists {
			server.initAreaConfDefault()
			def_ent, _ = server.AreaConfMap[def_key]
		}
		def_ent.IntfListMap[key] = true
		server.AreaConfMap[def_key] = def_ent
	}
	if !server.ospfGlobalConf.isABR {
		server.updateIfABR()
	} 
return def_key.AreaId
}

func (server *OSPFServer) updateIfABR() {
	index := 0
	for _, _ = range server.AreaConfMap {
		index++
	}
	if index > 1 {
	server.ospfGlobalConf.isABR = true
	}
}
