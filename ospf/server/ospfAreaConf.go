package server

import (
    "l3/ospf/config"
)

type AreaConfKey struct {
    AreaId          config.AreaId
}

type AreaConf struct {
    AuthType                                config.AuthType
    ImportAsExtern                          config.ImportAsExtern
    AreaSummary                             config.AreaSummary
    AreaNssaTranslatorRole                  config.NssaTranslatorRole
    AreaNssaTranslatorStabilityInterval     config.PositiveInteger
}

func (server *OSPFServer)processAreaConfig(areaConf config.AreaConf) {
    areaConfKey := AreaConfKey {
        AreaId:             areaConf.AreaId,
    }

    ent, _ := server.AreaConfMap[areaConfKey]
    ent.AuthType = areaConf.AuthType
    ent.ImportAsExtern = areaConf.ImportAsExtern
    ent.AreaSummary = areaConf.AreaSummary
    ent.AreaNssaTranslatorRole = areaConf.AreaNssaTranslatorRole
    ent.AreaNssaTranslatorStabilityInterval = areaConf.AreaNssaTranslatorStabilityInterval
    server.AreaConfMap[areaConfKey] = ent
}
