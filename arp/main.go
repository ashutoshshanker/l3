package main

import (
        "fmt"
	"flag"
	"log/syslog"
        "l3/arp/rpc"
        "l3/arp/server"
)


func main() {
        fmt.Println("Start the logger")
        logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR ARP")
        if err != nil {
                fmt.Println("Failed to start the logger. Exiting!!")
                return
        }

        logger.Info("Start the logger successfully.")
        paramsDir := flag.String("params", "./params", "Params directory")
        flag.Parse()
/*
        fileName := *paramsDir
        if fileName[len(fileName) - 1] != '/' {
                fileName = fileName + "/"
        }
        fileName = fileName + "clients.json"
*/

        logger.Info(fmt.Sprintln("Starting ARP server..."))
        arpServer := server.NewARPServer(logger)
        //go arpServer.StartServer(fileName)
        go arpServer.StartServer(*paramsDir)

        logger.Info(fmt.Sprintln("Starting Config listener..."))
        confIface := rpc.NewARPHandler(arpServer, logger)
        rpc.StartServer(logger, confIface, *paramsDir)
}
