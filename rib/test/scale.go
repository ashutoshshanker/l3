package main

import (
	"fmt"
	"ribd"
	"ribdInt"
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
		route.NextHopIp = "40.0.1.2"
		route.OutgoingInterface = "0"
		route.OutgoingIntfType = "VLAN"
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
func handleBulkClient(client *ribd.RIBDServicesClient) (err error) {
	var count int = 1
	var maxCount int = 30000
	intByt2 := 1
	intByt3 := 1
	byte1 := "32"
	byte4 := "0"
	start := time.Now()
	var route ribdInt.IPv4Route
	var routeList [] *ribdInt.IPv4Route
	routeList = make([] 	*ribdInt.IPv4Route,1000)
	var temprouteList [1000] ribdInt.IPv4Route
	curr := 0
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

		route = ribdInt.IPv4Route{}
		byte2 := strconv.Itoa(intByt2)
		byte3 := strconv.Itoa(intByt3)
		rtNet := byte1 + "." + byte2 + "." + byte3 + "." + byte4
		route.DestinationNw = rtNet
		route.NetworkMask = "255.255.255.0"
		route.NextHopIp = "40.0.1.2"
		route.OutgoingInterface = "4"
		route.OutgoingIntfType = "VLAN"
		route.Protocol = "STATIC"
		//fmt.Println("Creating Route ", route)
		temprouteList[curr]=route
		routeList[curr] = &temprouteList[curr]
		curr++ 
		if curr == 1000 {
			fmt.Println("calling count ", count, "routes")
		    rv := client.OnewayCreateBulkIPv4Route(routeList)
		    if rv == nil {
			    count+=1000
		    } else {
			    fmt.Println("Call failed", rv, "count: ", count)
	            elapsed := time.Since(start)
	            fmt.Println(" ## Elapsed time is ", elapsed)
			    return nil
		    }
		    if maxCount <= count {
			    fmt.Println("Done. Total calls executed", count)
			    break
		    }
			curr = 0
		}

	}
	elapsed := time.Since(start)
	fmt.Println(" ## Elapsed time is ", elapsed)
	return nil
}

func main() {
	transport, protocolFactory, err := ipcutils.CreateIPCHandles("localhost:5000")
	fmt.Println("### Calling client ", transport, protocolFactory, err)
	//handleClient(ribd.NewRIBDServicesClientFactory(transport, protocolFactory))
	handleBulkClient(ribd.NewRIBDServicesClientFactory(transport, protocolFactory))
}
