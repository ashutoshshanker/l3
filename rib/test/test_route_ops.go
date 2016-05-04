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
