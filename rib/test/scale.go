package main

import (
	"fmt"
	"ribd"
	"strconv"
	"time"
	"utils/ipcutils"
)

func handleClient(client *ribd.RIBDServicesClient) (err error) {
	var count int = 1
	var maxCount int = 30000
	intByt2 := 1
	intByt3 := 1
	byte1 := "22"
	byte4 := "0"

	start := time.Now()
	var route ribd.IPv4Route
	for {
		if intByt3 > 254 {
			intByt3 = 1
			intByt2++
		} else {
			intByt3++
		}
		if intByt2 > 254 {
			intByt2 = 1
		} //else {
			//intByt2++
		//}

		route = ribd.IPv4Route{}
		byte2 := strconv.Itoa(intByt2)
		byte3 := strconv.Itoa(intByt3)
		rtNet := byte1 + "." + byte2 + "." + byte3 + "." + byte4
		route.DestinationNw = rtNet
		route.NetworkMask = "255.255.255.0"
		route.NextHopIp = "7.0.1.2"
		route.OutgoingInterface = "0"
		route.OutgoingIntfType = "Loopback"
		route.Protocol = "STATIC"
		//fmt.Println("Creating Route ", route)
		 rv := client.OnewayCreateIPv4Route(&route)
		if rv == nil {
			count++
		} else {
			fmt.Println("Call failed", rv, "count: ", count)
	        elapsed := time.Since(start)
	        fmt.Println(" ## Elapsed time is ", elapsed)
			return nil
		}
		if maxCount == count {
			fmt.Println("Done. Total calls executed", count)
			break
		}

	}
	elapsed := time.Since(start)
	fmt.Println(" ## Elapsed time is ", elapsed)
	return nil
}

func main() {
	transport, protocolFactory, err := ipcutils.CreateIPCHandles("localhost:5000")
	fmt.Println("### Calling client ", transport, protocolFactory, err)
	handleClient(ribd.NewRIBDServicesClientFactory(transport, protocolFactory))
}
