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
	"fmt"
	"l3/rib/testutils"
	"ribd"
	"ribdInt"
	"strconv"
	"time"
	"utils/ipcutils"
)

func handleClient(client *ribd.RIBDServicesClient) (err error) {
	fmt.Println("handleClient")
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
		route.OutgoingInterface = "4"
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
	byte1 := "42"
	byte4 := "0"
	start := time.Now()
	var route ribdInt.IPv4Route
	var routeList []*ribdInt.IPv4Route
	routeList = make([]*ribdInt.IPv4Route, 5000)
	var temprouteList [5000]ribdInt.IPv4Route
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
		temprouteList[curr] = route
		routeList[curr] = &temprouteList[curr]
		curr++
		if curr == 5000 {
			fmt.Println("calling count ", count, "routes")
			rv := client.OnewayCreateBulkIPv4Route(routeList)
			if rv == nil {
				count += 5000
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
	ribdClient := testutils.GetRIBdClient()
	if ribdClient == nil {
		fmt.Println("RIBd client nil")
		return
	}
	handleClient(ribdClient) //ribd.NewRIBDServicesClientFactory(transport, protocolFactory))
	//handleBulkClient(ribd.NewRIBDServicesClientFactory(transport, protocolFactory))
}
