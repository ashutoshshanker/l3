package main

import (
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "arpd"
    "utils/dbutils"
    "database/sql"
)

func storeArpTableInDB(ifType int, vlanid int, ifName string, portid int, dest_ip string, src_ip string, dest_mac string) error {
    var dbCmd string
    dbCmd = fmt.Sprintf(`INSERT INTO ARPCache (ifType, vlanid, ifName, portid, src_ip, mac, key) VALUES ('%d', '%d', '%s', '%d', '%s', '%s', '%s') ;`, ifType, vlanid, ifName, portid, src_ip, dest_mac, dest_ip)
//    logger.Println(dbCmd)
    //logWriter.Info(dbCmd)
    if dbHdl != nil {
//        logger.Println("Executing DB Command:", dbCmd)
//        logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
        _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
        if err != nil {
            logWriter.Err(fmt.Sprintln("Failed to Insert entry for", dest_ip, "in DB"))
            return err
        }
    } else {
        //logger.Println("DB handler is nil");
        logWriter.Err("DB handler is nil");
    }
    return nil
}

func updateArpTableInDB(ifType int, vlanid int, ifName string, portid int, dest_ip string, src_ip string, dest_mac string) error {
    var dbCmd string
    dbCmd = fmt.Sprintf(`UPDATE ARPCache SET ifType='%d', vlanid='%d', ifName='%s', portid='%d', src_ip='%s', mac='%s' WHERE key='%s' ;`, ifType, vlanid, ifName, portid, src_ip, dest_mac, dest_ip)
//    logger.Println(dbCmd)
    //logWriter.Info(dbCmd)
    if dbHdl != nil {
//        logger.Println("Executing DB Command:", dbCmd)
//        logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
        _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
        if err != nil {
            logWriter.Err(fmt.Sprintln("Failed to Update entry for", dest_ip, "in DB"))
            return err
        }
    } else {
        //logger.Println("DB handler is nil");
        logWriter.Err("DB handler is nil");
    }
    return nil
}

func intantiateDB() error {
    var err error
    err = nil
    DbName := params_dir + UsrConfDbName
    //logger.Println("DB Location: ", DbName)
    logWriter.Info(fmt.Sprintln("DB Location: ", DbName))
    dbHdl, err = sql.Open("sqlite3", DbName)
    if err != nil {
        logWriter.Err("Failed to create the handle")
        return err
    }

    if err = dbHdl.Ping(); err != nil {
        logWriter.Err("Failed to keep DB connection alive")
        return err
    }

    dbCmd := "CREATE TABLE IF NOT EXISTS ARPCache " +
            "(key string PRIMARY KEY ," +
            "ifType int, vlanid int, ifName string, portid int, src_ip string, mac string)"

    _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
    if err != nil {
        logWriter.Err("Failed to create ARPCache Table in DB")
        return err
    }

    return err
}

func updateARPCacheFromDB() {
        var ent arpEntry
        var port_prop_ent portProperty
        var ip      string
        var ifType  int
        var vlanid  int
        var ifName  string
        var portid  int
        var src_ip  string
        var mac     string
        //var dbCmd string

        //logger.Println("Populate ARP Cache from DB entries")
        logWriter.Info("Populate ARP Cache from DB entries")
        rows, err := dbHdl.Query("SELECT * FROM ARPCache")
        if err != nil {
            logWriter.Err(fmt.Sprintf("Unable to Query DB:", err))
            return
        }
        for rows.Next() {
            err = rows.Scan(&ip, &ifType, &vlanid, &ifName, &portid, &src_ip, &mac)
            if err != nil {
                logWriter.Err(fmt.Sprintf("Unable to Scan entry from DB:", err))
                return
            }
            //logger.Println("Data Retrived From DB IP:", ip, "IFTYPE:", ifType, "VLANID:", vlanid, "IFNAME:", ifName, "PORTID:", portid, "SRC_IP:", src_ip)
            logWriter.Info(fmt.Sprintln("Data Retrived From DB IP:", ip, "IFTYPE:", ifType, "VLANID:", vlanid, "IFNAME:", ifName, "PORTID:", portid, "SRC_IP:", src_ip, "MAC:", mac))

            if mac == "incomplete" {
                continue
            }
            logger.Println("Adding arp cache entry for ", ip)
            ent = arp_cache.arpMap[ip]
            ent.ifType = arpd.Int(ifType)
            ent.vlanid = arpd.Int(vlanid)
            ent.ifName = ifName
            ent.port = portid
            ent.localIP = src_ip
            ent.counter = (min_cnt + retry_cnt + 1)
            ent.valid = true
            arp_cache.arpMap[ip] = ent
            port_prop_ent = port_property_map[portid]
            port_prop_ent.untagged_vlanid = arpd.Int(vlanid)
            port_property_map[portid] = port_prop_ent
        }

}

func refreshARPDB() {
        var dbCmd string
        dbCmd = "DELETE FROM ARPCache ;"
        //logger.Println(dbCmd)
        logWriter.Info(dbCmd)
        if dbHdl != nil {
            //logger.Println("Executing DB Command:", dbCmd)
            logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
            _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
            if err != nil {
                logWriter.Err(fmt.Sprintln("Failed to Delete all ARP entries from DB"))
                return
            }
        } else {
            //logger.Println("DB handler is nil");
            logWriter.Err("DB handler is nil");
        }
}

