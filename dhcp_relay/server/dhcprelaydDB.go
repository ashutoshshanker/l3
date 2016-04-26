package relayServer

import (
	"dhcprelayd"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"models"
)

func DhcpRelayAgentInitDB() error {
	logger.Info("DRA: initializing SQL DB")
	var err error
	dhcprelayDbHdl, err = redis.Dial("tcp", DHCP_REDDIS_DB_PORT)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to create db handle", err))
		return err
	}

	logger.Info("DRA: SQL DB init success")
	return err
}

func DhcpRelayAgentReadDB() {
	logger.Info("Reading Dhcp Relay Global Config from DB")
	if dhcprelayDbHdl == nil {
		return
	}
	/*  First reading Dhcp Relay Global Config
	 */
	var dbObj models.DhcpRelayGlobal
	objList, err := dbObj.GetAllObjFromDb(dhcprelayDbHdl)
	if err != nil {
		logger.Warning("DB querry failed for Dhcp Relay Global Config")
		return
	}
	for idx := 0; idx < len(objList); idx++ {
		obj := dhcprelayd.NewDhcpRelayGlobal()
		dbObject := objList[idx].(models.DhcpRelayGlobal)
		models.ConvertdhcprelaydDhcpRelayGlobalObjToThrift(&dbObject, obj)
		DhcpRelayGlobalInit(bool(obj.Enable))
	}

	/*  Reading Dhcp Relay Interface Config.
	 *  As we are using redis DB, we will get the server ip list automatically..
	 */
	readIfIndex := make([]int32, 0)
	var intfDbObj models.DhcpRelayIntf
	objList, err = intfDbObj.GetAllObjFromDb(dhcprelayDbHdl)
	if err != nil {
		logger.Warning("DB querry failed for Dhcp Relay Intf Config")
		return
	}
	for idx := 0; idx < len(objList); idx++ {
		obj := dhcprelayd.NewDhcpRelayIntf()
		dbObject := objList[idx].(models.DhcpRelayIntf)
		models.ConvertdhcprelaydDhcpRelayIntfObjToThrift(&dbObject, obj)
		IfIndex := int32(obj.IfIndex)
		Enable := bool(obj.Enable)
		DhcpRelayAgentInitGblHandling(IfIndex, Enable)
		DhcpRelayAgentInitIntfState(IfIndex)
		readIfIndex = append(readIfIndex, IfIndex)
		for _, serverIp := range obj.ServerIp {
			logger.Info(fmt.Sprintln("DRA: ifindex:", IfIndex, "Server Ip:",
				serverIp))
			DhcpRelayAgentUpdateIntfServerIp(IfIndex, serverIp)
		}
	}
	if len(readIfIndex) > 0 {
		// For all ifIndex recovered from DB.. get ip address from asicd
		go DhcpRelayAgentUpdateIntfIpAddr(readIfIndex)
	}
	dhcprelayDbHdl.Close()
}
