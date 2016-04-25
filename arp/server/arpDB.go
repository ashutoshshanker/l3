package server

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strconv"
)

type arpDbEntry struct {
	IpAddr string
	Port   int
}

func (server *ARPServer) initiateDB(dbName string) error {
	var err error
	server.dbHdl, err = redis.Dial("tcp", ":6379")
	if err != nil {
		server.logger.Err("Failed to create the DB handle")
		return err
	}
	return nil
}

func (server *ARPServer) updateArpCacheFromDB() {
	server.logger.Info("Populate ARP Cache from DB entries")
	if server.dbHdl != nil {
		keyPattern := fmt.Sprintln("ArpCacheEntry#*")
		keys, err := redis.Strings(redis.Values(server.dbHdl.Do("KEYS", keyPattern)))
		if err != nil {
			server.logger.Err(fmt.Sprintln("Failed to get all keys from DB"))
			return
		}
		for idx := 0; idx < len(keys); idx++ {
			var obj arpDbEntry
			val, err := redis.Values(server.dbHdl.Do("HGETALL", keys[idx]))
			if err != nil {
				server.logger.Err(fmt.Sprintln("Failed to get ARP entry for key:", keys[idx]))
				continue
			}
			err = redis.ScanStruct(val, &obj)
			if err != nil {
				server.logger.Err(fmt.Sprintln("Failed to get values corresponding to ARP entry key:", keys[idx]))
				continue
			}
			server.logger.Info(fmt.Sprintln("Data Retrived From DB IP:", obj.IpAddr, "port:", obj.Port))
			server.logger.Info(fmt.Sprintln("Adding arp cache entry for ", obj.IpAddr))
			ent := server.arpCache[obj.IpAddr]
			ent.MacAddr = "incomplete"
			ent.Counter = (server.minCnt + server.retryCnt + 1)
			//ent.Valid = false
			ent.PortNum = obj.Port
			server.arpCache[obj.IpAddr] = ent
		}
	} else {
		server.logger.Err("DB handler is nil")
	}
	server.logger.Info(fmt.Sprintln("Arp Cache after restoring: ", server.arpCache))
}

func (server *ARPServer) refreshArpDB() {
	if server.dbHdl != nil {
		keyPattern := fmt.Sprintln("ArpCacheEntry#*")
		keys, err := redis.Strings(redis.Values(server.dbHdl.Do("KEYS", keyPattern)))
		if err != nil {
			server.logger.Err(fmt.Sprintln("Failed to get all keys from DB"))
		}
		for idx := 0; idx < len(keys); idx++ {
			_, err := server.dbHdl.Do("DEL", keys[idx])
			if err != nil {
				server.logger.Err(fmt.Sprintln("Failed to Delete ARP entry for key:", keys[idx]))
				continue
			}
		}
	} else {
		server.logger.Err("DB handler is nil")
	}
}

func (server *ARPServer) deleteArpEntryInDB(ipAddr string) {
	if server.dbHdl != nil {
		key := fmt.Sprintln("ArpCacheEntry#", ipAddr, "*")
		_, err := server.dbHdl.Do("DEL", key)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Failed to Delete ARP entries from DB for:", ipAddr))
			return
		}
	} else {
		server.logger.Err("DB handler is nil")
	}
}

func (server *ARPServer) storeArpEntryInDB(ip string, port int) {
	if server.dbHdl != nil {
		key := fmt.Sprintln("ArpCacheEntry#", ip, "#", strconv.Itoa(port))
		obj := arpDbEntry{
			IpAddr: ip,
			Port:   port,
		}
		_, err := server.dbHdl.Do("HMSET", redis.Args{}.Add(key).AddFlat(&obj)...)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Failed to add entry to db : ", ip, port, err))
			return
		}
		return
	} else {
		server.logger.Err("DB handler is nil")
	}
}
