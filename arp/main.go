Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
package main

import (
	"flag"
	"fmt"
	"l3/arp/rpc"
	"l3/arp/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting arp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("arpd", "ARP", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	logger.Info(fmt.Sprintln("Starting ARP server..."))
	arpServer := server.NewARPServer(logger)
	plugin := "Flexswitch" // Flexswitch/OvsDB
	go arpServer.StartServer(*paramsDir, plugin)

	<-arpServer.InitDone

	// Start keepalive routine
	go keepalive.InitKeepAlive("arpd", fileName)

	logger.Info(fmt.Sprintln("Starting Config listener..."))
	confIface := rpc.NewARPHandler(arpServer, logger)
	rpc.StartServer(logger, confIface, *paramsDir)
}
