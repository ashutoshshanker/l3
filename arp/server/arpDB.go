package server

import (
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "utils/dbutils"
    "database/sql"
)


func (server *ARPServer) initiateDB(dbName string) error {
    var err error
    err = nil
    //DbName := paramsFile + server.usrConfDbName
        //TODO
    //DbName := server.usrConfDbName
    //logger.Println("DB Location: ", DbName)
    server.logger.Info(fmt.Sprintln("DB Location: ", dbName))
    server.dbHdl, err = sql.Open("sqlite3", dbName)
    if err != nil {
        server.logger.Err("Failed to create the handle")
        return err
    }

    if err = server.dbHdl.Ping(); err != nil {
        server.logger.Err("Failed to keep DB connection alive")
        return err
    }

    dbCmd := "CREATE TABLE IF NOT EXISTS ARPCache " +
            "(ipAddr string PRIMARY KEY ," +
            "portNum int)"

    _, err = dbutils.ExecuteSQLStmt(dbCmd, server.dbHdl)
    if err != nil {
        server.logger.Err("Failed to create ARPCache Table in DB")
        return err
    }

    return err
}

func (server *ARPServer)updateArpCacheFromDB() {
        var ip          string
        var port        int
        server.logger.Info("Populate ARP Cache from DB entries")
        rows, err := server.dbHdl.Query("SELECT * FROM ARPCache")
        if err != nil {
            server.logger.Err(fmt.Sprintf("Unable to Query DB:", err))
            return
        }
        for rows.Next() {
            err = rows.Scan(&ip, &port)
            if err != nil {
                server.logger.Err(fmt.Sprintf("Unable to Scan entry from DB:", err))
                return
            }
            server.logger.Info(fmt.Sprintln("Data Retrived From DB IP:", ip, "port:", port))

            server.logger.Info(fmt.Sprintln("Adding arp cache entry for ", ip))
            ent := server.arpCache[ip]
                ent.MacAddr = "incomplete"
            ent.Counter = (server.minCnt + server.retryCnt + 1)
            //ent.Valid = false
                ent.PortNum = port
            server.arpCache[ip] = ent
        }

}

func (server *ARPServer) refreshArpDB() {
        var dbCmd string
        dbCmd = "DELETE FROM ARPCache ;"
        server.logger.Info(dbCmd)
        if server.dbHdl != nil {
            server.logger.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
            _, err := dbutils.ExecuteSQLStmt(dbCmd, server.dbHdl)
            if err != nil {
                server.logger.Err(fmt.Sprintln("Failed to Delete all ARP entries from DB"))
                return
            }
        } else {
            server.logger.Err("DB handler is nil");
        }
}

func (server *ARPServer) deleteArpEntryInDB(ipAddr string) {
        dbCmd := fmt.Sprintf(`DELETE FROM ARPCache WHERE ipAddr='%s';`, ipAddr)
        server.logger.Info(dbCmd)
        if server.dbHdl != nil {
            server.logger.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
            _, err := dbutils.ExecuteSQLStmt(dbCmd, server.dbHdl)
            if err != nil {
                server.logger.Err(fmt.Sprintln("Failed to Delete ARP entries from DB for:", ipAddr))
                return
            }
        } else {
            server.logger.Err("DB handler is nil");
        }
}

func (server *ARPServer) storeArpEntryInDB(ip string, port int) {
    dbCmd := fmt.Sprintf(`INSERT INTO ARPCache (portNum, ipAddr) VALUES ('%d', '%s') ;`, port, ip)
    if server.dbHdl != nil {
        _, err := dbutils.ExecuteSQLStmt(dbCmd, server.dbHdl)
        if err != nil {
            server.logger.Err(fmt.Sprintln("Failed to Insert entry for", ip, "in DB"))
            return
        }
    } else {
        server.logger.Err("DB handler is nil");
    }
    return
}
