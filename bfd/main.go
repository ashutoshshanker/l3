package main

import (
	"flag"
	"fmt"
	"l3/bfd/config"
	"l3/bfd/rpc"
	"l3/bfd/server"
	"log/syslog"
)

var gBfdCB *BfdCB

type BfdCB struct {
	State                bool
	NumInterfaces        uint32
	Interfaces           map[int32]config.IntfConfig
	NumSessions          uint32
	Sessions             map[int32]config.SessionState
	NumUpSessions        uint32
	NumDownSessions      uint32
	NumAdminDownSessions uint32
	logger               *syslog.Writer
}

func NewBfdCB(logger *syslog.Writer, paramsDir string) *BfdCB {
	bfdCB := new(BfdCB)
	if bfdCB == nil {
		return nil
	}
	bfdCB.State = false
	bfdCB.NumInterfaces = 0
	bfdCB.Interfaces = make(map[int32]config.IntfConfig)
	bfdCB.NumSessions = 0
	bfdCB.Sessions = make(map[int32]config.SessionState)
	bfdCB.NumUpSessions = 0
	bfdCB.NumDownSessions = 0
	bfdCB.NumAdminDownSessions = 0
	logger.Info("Initialization Done")
	return bfdCB
}

func main() {
	fmt.Println("Start the logger")
	logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "SR BFD")
	if err != nil {
		fmt.Println("Failed to start the logger. Exiting!!")
		return
	}

	logger.Info("Started the logger successfully.")

	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	gBfdCB = NewBfdCB(logger, *paramsDir)
	if gBfdCB == nil {
		return
	}

	logger.Info(fmt.Sprintln("Starting BFD Server..."))
	bfdServer := server.NewBFDServer(logger)
	go bfdServer.StartServer(fileName)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewBFDHandler(logger, bfdServer)
	rpc.StartServer(logger, confIface, fileName)
}
