//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//

package server

import (
	"fmt"
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
		RouterId:        config.RouterId(conf.RouterId),
		ASBdrRtrStatus:  conf.ASBdrRtrStatus,
		TOSSupport:      conf.TOSSupport,
		RestartSupport:  config.RestartSupport(conf.RestartSupport),
		RestartInterval: conf.RestartInterval,
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
		AreaId:                 config.AreaId(conf.AreaId),
		AuthType:               config.AuthType(conf.AuthType),
		ImportAsExtern:         config.ImportAsExtern(conf.ImportAsExtern),
		AreaSummary:            config.AreaSummary(conf.AreaSummary),
		AreaNssaTranslatorRole: config.NssaTranslatorRole(conf.AreaNssaTranslatorRole),
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
		IfIpAddress:       config.IpAddress(conf.IfIpAddress),
		AddressLessIf:     config.InterfaceIndexOrZero(conf.AddressLessIf),
		IfAreaId:          config.AreaId(conf.IfAreaId),
		IfType:            config.IfType(conf.IfType),
		IfRtrPriority:     config.DesignatedRouterPriority(conf.IfRtrPriority),
		IfTransitDelay:    config.UpToMaxAge(conf.IfTransitDelay),
		IfRetransInterval: config.UpToMaxAge(conf.IfRetransInterval),
		IfHelloInterval:   config.HelloRange(conf.IfHelloInterval),
		IfRtrDeadInterval: config.PositiveInteger(conf.IfRtrDeadInterval),
		IfPollInterval:    config.PositiveInteger(conf.IfPollInterval),
		IfAuthKey:         conf.IfAuthKey,
		IfAuthType:        config.AuthType(conf.IfAuthType),
	}

	err := server.processIntfConfig(ifConf)
	if err != nil {
		server.logger.Err("Error Configuring Ospf Area Configuration")
		err := errors.New("Error Configuring Ospf Area Configuration")
		return err
	}
	return nil

}

func (server *OSPFServer) AddIPv4RoutesState(entry RoutingTblEntryKey) error {
	server.logger.Info(fmt.Sprintln("DB: Add IPv4 entry to db. ", entry))
	rEntry, exist := server.GlobalRoutingTbl[entry]
	if !exist {
		server.logger.Info(fmt.Sprintln("DB: Routing table entry doesnt exist key ", entry))
		return nil
	}
	var dbObj models.OspfIPv4Routes
	obj := ospfd.NewOspfIPv4Routes()
	/* Fill in obj values */
	obj.DestId = convertUint32ToIPv4(entry.DestId)
	obj.AddrMask = convertUint32ToIPv4(entry.AddrMask)
	obj.DestType = string(entry.DestType)
	obj.OptCapabilities = int32(rEntry.RoutingTblEnt.OptCapabilities)
	obj.AreaId = convertUint32ToIPv4(rEntry.AreaId)
	obj.PathType = string(rEntry.RoutingTblEnt.PathType)
	obj.Cost = int32(rEntry.RoutingTblEnt.Cost)
	obj.Type2Cost = int32(rEntry.RoutingTblEnt.Type2Cost)
	obj.NumOfPaths = int32(rEntry.RoutingTblEnt.NumOfPaths)
	nh_local := rEntry.RoutingTblEnt.NextHops
	obj.NextHops = make([]*ospfd.OspfNextHop, 0)
	nh_list := make([]ospfd.OspfNextHop, len(nh_local))
	index := 0
	/*
	    type OspfNextHop struct {
		IfIPAddr  string `DESCRIPTION: O/P interface IP address`
		IfIdx     uint32 `DESCRIPTION: Interface index `
		NextHopIP string `DESCRIPTION: Nexthop ip address`
		AdvRtr    string `DESCRIPTION: Advertising router id`
	}
	*/
	for nh_val, _ := range nh_local {
		nh_list[index].IfIPAddr = convertUint32ToIPv4(nh_val.IfIPAddr)
		nh_list[index].IfIdx = int32(nh_val.IfIdx)
		nh_list[index].NextHopIP = convertUint32ToIPv4(nh_val.NextHopIP)
		nh_list[index].AdvRtr = convertUint32ToIPv4(nh_val.AdvRtr)
		obj.NextHops = append(obj.NextHops, &nh_list[index])
		index++
	}
	obj.LSOrigin = &ospfd.OspfLsaKey{}
	obj.LSOrigin.LSType = int8(rEntry.RoutingTblEnt.LSOrigin.LSType)
	obj.LSOrigin.LSId = int32(rEntry.RoutingTblEnt.LSOrigin.LSId)
	obj.LSOrigin.AdvRouter = int32(rEntry.RoutingTblEnt.LSOrigin.AdvRouter)

	models.ConvertThriftToospfdOspfIPv4RoutesObj(obj, &dbObj)
	err := dbObj.StoreObjectInDb(server.dbHdl)
	if err != nil {
		server.logger.Err(fmt.Sprintln("DB: Failed to add object in db , err ", err))
		return errors.New(fmt.Sprintln("Failed to add OspfIPv4Routes db : ", entry))
	}

	return nil
}

func (server *OSPFServer) DelIPv4RoutesState(entry RoutingTblEntryKey) error {
	server.logger.Info(fmt.Sprintln("DB: Delete IPv4 entry from db ", entry))

	var dbObj models.OspfIPv4Routes
	obj := ospfd.NewOspfIPv4Routes()

	obj.LSOrigin = &ospfd.OspfLsaKey{}
	obj.DestId = convertUint32ToIPv4(entry.DestId)
	obj.AddrMask = convertUint32ToIPv4(entry.AddrMask)
	obj.DestType = string(entry.DestType)

	models.ConvertThriftToospfdOspfIPv4RoutesObj(obj, &dbObj)
	err := dbObj.DeleteObjectFromDb(server.dbHdl)
	if err != nil {
		server.logger.Err(fmt.Sprintln("DB: Failed to add object in db , err ", err))
		return errors.New(fmt.Sprintln("Failed to DEL OspfIPv4Routes db : ", entry))
	}
	return nil
}

func (server *OSPFServer) AddLsdbEntry(entry LsdbSliceEnt) error {
	server.logger.Info(fmt.Sprintln("DB: Add lsdb entry. ", entry))
	var lsaEnc []byte
	var lsaMd LsaMetadata

	var dbObj models.OspfLsdbEntryState
	//  var obj *ospfd.OspfLsdbEntryState
	obj := ospfd.NewOspfLsdbEntryState()
	lsdbKey := LsdbKey{
		AreaId: entry.AreaId,
	}
	lsDbEnt, exist := server.AreaLsdb[lsdbKey]
	if !exist {
		return nil
	}

	lsaKey := LsaKey{
		LSType:    entry.LSType,
		LSId:      entry.LSId,
		AdvRouter: entry.AdvRtr,
	}

	if entry.LSType == RouterLSA {
		lsa, exist := lsDbEnt.RouterLsaMap[lsaKey]
		if !exist {
			return nil
		}
		lsaEnc = encodeRouterLsa(lsa, lsaKey)
		lsaMd = lsa.LsaMd
	} else if entry.LSType == NetworkLSA {
		lsa, exist := lsDbEnt.NetworkLsaMap[lsaKey]
		if !exist {
			return nil
		}
		lsaEnc = encodeNetworkLsa(lsa, lsaKey)
		lsaMd = lsa.LsaMd
	} else if entry.LSType == Summary3LSA {
		lsa, exist := lsDbEnt.Summary3LsaMap[lsaKey]
		if !exist {
			return nil
		}
		lsaEnc = encodeSummaryLsa(lsa, lsaKey)
		lsaMd = lsa.LsaMd
	} else if entry.LSType == Summary4LSA {
		lsa, exist := lsDbEnt.Summary4LsaMap[lsaKey]
		if !exist {
			return nil
		}
		lsaEnc = encodeSummaryLsa(lsa, lsaKey)
		lsaMd = lsa.LsaMd
	} else if entry.LSType == ASExternalLSA {
		lsa, exist := lsDbEnt.ASExternalLsaMap[lsaKey]
		if !exist {
			return nil
		}
		lsaEnc = encodeASExternalLsa(lsa, lsaKey)
		lsaMd = lsa.LsaMd
	}
	adv := convertByteToOctetString(lsaEnc[OSPF_LSA_HEADER_SIZE:])

	obj.LsdbAreaId = convertUint32ToIPv4(lsdbKey.AreaId)
	obj.LsdbType = int32(lsaKey.LSType)
	obj.LsdbLsid = convertUint32ToIPv4(lsaKey.LSId)
	obj.LsdbRouterId = convertUint32ToIPv4(lsaKey.AdvRouter)
	obj.LsdbSequence = int32(lsaMd.LSSequenceNum)
	obj.LsdbAge = int32(lsaMd.LSAge)
	obj.LsdbChecksum = int32(lsaMd.LSChecksum)
	obj.LsdbAdvertisement = adv

	models.ConvertThriftToospfdOspfLsdbEntryStateObj(obj, &dbObj)
	server.logger.Info(fmt.Sprintln("DB: Db obj received ", dbObj))
	err := dbObj.StoreObjectInDb(server.dbHdl)
	if err != nil {
		server.logger.Err(fmt.Sprintln("DB: lsdb Failed to add object in db , err ", err))
		return errors.New(fmt.Sprintln("Failed to add OspfLsdbEntryStaten in db : ", entry))
	}

	return nil
}

func (server *OSPFServer) DelLsdbEntry(entry LsdbSliceEnt) error {
	server.logger.Info(fmt.Sprintln("DB: Delete LSDB entry from db ", entry))

	var dbObj models.OspfLsdbEntryState
	obj := ospfd.NewOspfLsdbEntryState()

	obj.LsdbAreaId = convertUint32ToIPv4(entry.AreaId)
	obj.LsdbType = int32(entry.LSType)
	obj.LsdbLsid = convertUint32ToIPv4(entry.LSId)
	obj.LsdbRouterId = convertUint32ToIPv4(entry.AdvRtr)

	models.ConvertThriftToospfdOspfLsdbEntryStateObj(obj, &dbObj)
	err := dbObj.DeleteObjectFromDb(server.dbHdl)
	if err != nil {
		server.logger.Err(fmt.Sprintln("DB: LSDB Failed to delete object in db , err ", err))
		return errors.New(fmt.Sprintln("Failed to DEL OspfLsdbEntryState db : ", entry))
	}
	return nil
}
