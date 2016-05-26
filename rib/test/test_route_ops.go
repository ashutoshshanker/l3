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

// test_route_ops
package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"l3/rib/server"
	"ribd"
	"utils/logging"
)

var route ribd.IPv4Route
var routeServer *server.RIBDServer

func CreateRoute(cfg *ribd.IPv4Route) {
	routeServer.ProcessRouteCreateConfig(cfg)
}
func DeleteRoute(cfg *ribd.IPv4Route) {
	routeServer.ProcessRouteDeleteConfig(cfg)
}
func main() {
	fmt.Println("Start logger")
	logger, err := logging.NewLogger("ribd", "RIB", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	dbHdl, err := redis.Dial("tcp", ":6379")
	if err != nil {
		logger.Err("Failed to dial out to Redis server")
		return
	}
	routeServer = server.NewRIBDServicesHandler(dbHdl, logger)
	if routeServer == nil {
		logger.Println("routeServer nil")
		return
	}
	route = ribd.IPv4Route{
		DestinationNw:     "40.0.1.1",
		NetworkMask:       "255.255.255.0",
		NextHopIp:         "1.1.1.1",
		OutgoingIntfType:  "Loopback",
		OutgoingInterface: "0",
		Protocol:          "STATIC",
	}
	CreateRoute(&route)
	DeleteRoute(&route)
}
