// main.go
package main

import (
    "flag"
    "fmt"
    "l3/ospf/rpc"
    "l3/ospf/server"
    "log/syslog"
)

func main() {
    fmt.Println("Start the logger")
    logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR OSPF")
    if err != nil {
        fmt.Println("Failed to start the logger. Exiting!!")
        return
    }

    logger.Info("Started the logger successfully.")

    paramsDir := flag.String("params", "./params", "Params directory")
    flag.Parse()
    fileName := *paramsDir
    if fileName[len(fileName) - 1] != '/' {
        fileName = fileName + "/"
    }
    fileName = fileName + "clients.json"

    logger.Info(fmt.Sprintln("Starting OSPF Server..."))
    ospfServer := server.NewOSPFServer(logger)
    go ospfServer.StartServer(fileName)

    logger.Info(fmt.Sprintln("Starting Config listener..."))
    confIface := rpc.NewOSPFHandler(ospfServer, logger)
    rpc.StartServer(logger, confIface, fileName)
}
