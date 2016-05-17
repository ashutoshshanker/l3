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

package main

import (
	"flag"
	"fmt"
	"l3/rib/rpc"
	"l3/rib/server"
	"utils/dbutils"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	var err error
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("ribd", "RIB", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	dbHdl := dbutils.NewDBUtil(logger)
	err = dbHdl.Connect()
	if err != nil {
		logger.Err("Failed to dial out to Redis server")
		return
	}
	routeServer := server.NewRIBDServicesHandler(dbHdl, logger)
	if routeServer == nil {
		logger.Println("routeServer nil")
		return
	}
	go routeServer.StartDBServer()
	go routeServer.StartPolicyServer()
	go routeServer.NotificationServer()
	go routeServer.StartAsicdServer()
	go routeServer.StartArpdServer()
	go routeServer.StartServer(*paramsDir)
	up := <-routeServer.ServerUpCh
	//dbHdl.Close()
	logger.Info(fmt.Sprintln("RIBD server is up: ", up))
	if !up {
		logger.Err(fmt.Sprintln("Exiting!!"))
		return
	}

	// Start keepalive routine
	go keepalive.InitKeepAlive("ribd", fileName)

	ribdServicesHandler := rpc.NewRIBdHandler(logger, routeServer)
	rpc.NewRIBdRPCServer(logger, ribdServicesHandler, fileName)
}
