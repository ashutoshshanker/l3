// main.go
package main

import (
	"flag"
	"fmt"
	"l3/bgp/flexswitch"
	"l3/bgp/ovs"
	bgppolicy "l3/bgp/policy"
	"l3/bgp/rpc"
	"l3/bgp/server"
	"l3/bgp/utils"
	"utils/dbutils"
	"utils/keepalive"
	"utils/logging"
)

const (
	IP          string = "10.1.10.229"
	BGPPort     string = "179"
	CONF_PORT   string = "2001"
	BGPConfPort string = "4050"
	RIBConfPort string = "5000"

	OVSDB_PLUGIN = "ovsdb"
)

func main() {
	fmt.Println("Starting bgp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fmt.Println("Start logger")
	logger, err := logging.NewLogger("bgpd", "BGP", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")
	utils.SetLogger(logger)

	// Start DB Util
	dbUtil := dbutils.NewDBUtil(logger)
	err = dbUtil.Connect()
	if err != nil {
		logger.Err(fmt.Sprintf("DB connect failed with error %s. Exiting!!", err))
		return
	}

	// Start keepalive routine
	go keepalive.InitKeepAlive("bgpd", fileName)

	// starting bgp policy engine...
	logger.Info(fmt.Sprintln("Starting BGP policy engine..."))
	bgpPolicyEng := bgppolicy.NewBGPPolicyEngine(logger)
	go bgpPolicyEng.StartPolicyEngine()

	// @FIXME: Plugin name should come for json readfile...
	//plugin := OVSDB_PLUGIN
	plugin := ""
	switch plugin {
	case OVSDB_PLUGIN:
		// if plugin used is ovs db then lets start ovsdb client listener
		quit := make(chan bool)
		rMgr := ovsMgr.NewOvsRouteMgr()
		pMgr := ovsMgr.NewOvsPolicyMgr()
		iMgr := ovsMgr.NewOvsIntfMgr()
		bMgr := ovsMgr.NewOvsBfdMgr()

		bgpServer := server.NewBGPServer(logger, bgpPolicyEng, iMgr, pMgr,
			rMgr, bMgr)
		go bgpServer.StartServer()

		logger.Info(fmt.Sprintln("Starting config listener..."))
		confIface := rpc.NewBGPHandler(bgpServer, bgpPolicyEng, logger, dbUtil, fileName)
		dbUtil.Disconnect()

		// create and start ovsdb handler
		ovsdbManager, err := ovsMgr.NewBGPOvsdbHandler(logger, confIface)
		if err != nil {
			logger.Info(fmt.Sprintln("Starting OVDB client failed ERROR:", err))
			return
		}
		err = ovsdbManager.StartMonitoring()
		if err != nil {
			logger.Info(fmt.Sprintln("OVSDB Serve failed ERROR:", err))
			return
		}

		<-quit
	default:
		// flexswitch plugin lets connect to clients first and then
		// start flexswitch client listener
		iMgr, err := FSMgr.NewFSIntfMgr(logger, fileName)
		if err != nil {
			return
		}
		rMgr, err := FSMgr.NewFSRouteMgr(logger, fileName)
		if err != nil {
			return
		}
		bMgr, err := FSMgr.NewFSBfdMgr(logger, fileName)
		if err != nil {
			return
		}
		pMgr := FSMgr.NewFSPolicyMgr(logger, fileName)

		logger.Info(fmt.Sprintln("Starting BGP Server..."))

		bgpServer := server.NewBGPServer(logger, bgpPolicyEng, iMgr, pMgr,
			rMgr, bMgr)
		go bgpServer.StartServer()

		logger.Info(fmt.Sprintln("Starting config listener..."))
		confIface := rpc.NewBGPHandler(bgpServer, bgpPolicyEng, logger, dbUtil, fileName)
		dbUtil.Disconnect()

		rpc.StartServer(logger, confIface, fileName)
	}
}
