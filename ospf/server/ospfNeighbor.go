package server

import (
	"fmt"
	"l3/ospf/config"

	"time"
)

type NeighborConfKey struct {
	IPAddr  config.IpAddress
	IntfIdx config.InterfaceIndexOrZero
}

type OspfNeighborEntry struct {
	OspfNbrRtrId           uint32
	OspfNbrIPAddr          config.IpAddress
	OspfNbrOptions         int
	OspfNbrPriority        int
	OspfNbrState           config.NbrState
	OspfNbrInactivityTimer time.Time
	OspfNbrDeadTimer       time.Duration
}

func (server *OSPFServer) ProcessHelloPktEvent(nbrId int32, nbrIpaddr config.IpAddress,
	nbrIntfIndex config.InterfaceIndexOrZero,
	intfIPaddr config.IpAddress,
	intfIndex config.InterfaceIndexOrZero,
	nbrRtrId uint32, nbrPriority int, nbrDeadTimer time.Duration, two_way bool) {
	var nbrConf OspfNeighborEntry
	var nbrState config.NbrState
	//var nbrList list.List

	/*
		intfConfKey := IntfConfKey{
			IPAddr:  intfIPaddr,
			IntfIdx: intfIndex,
		} */
	//Check if neighbor exists
	_, exists := server.NeighborConfigMap[nbrId]
	if exists {
		server.neighborConfMutex.Lock()
		nbrConf = server.NeighborConfigMap[nbrId]
		if two_way { // update the state
			nbrConf.OspfNbrState = config.NbrTwoWay
		} else {
			nbrConf.OspfNbrState = config.NbrInit
		}
		nbrConf.OspfNbrInactivityTimer = time.Now()
		server.NeighborConfigMap[nbrId] = nbrConf
		server.neighborConfMutex.Unlock()
	} else { //neighbor doesnt exist
		server.neighborConfMutex.Lock()
		if two_way {
			nbrState = config.NbrTwoWay
		} else {
			nbrState = config.NbrInit
		}
		//fill up neighbor config datastruct
		nbrConf = OspfNeighborEntry{
			OspfNbrRtrId:           nbrRtrId,
			OspfNbrIPAddr:          nbrIpaddr,
			OspfNbrOptions:         0,
			OspfNbrPriority:        nbrPriority,
			OspfNbrState:           nbrState,
			OspfNbrDeadTimer:       nbrDeadTimer,
			OspfNbrInactivityTimer: time.Now(),
		}
		server.NeighborConfigMap[nbrId] = nbrConf
		server.neighborConfMutex.Unlock()
	}
	/*
		_, list_exists := server.NeighborListMap[intfConfKey]
		if !list_exists {
			//create a list and Nbrconf  object
			nbrList.PushBack(neighborKey)
			server.NeighborListMap[intfConfKey] = nbrList
		} else {
			nbrList = server.NeighborListMap[intfConfKey]
			nbrList.PushBack(neighborKey)
		}
	*/
}

func (server *OSPFServer) scanNeighborDeadTimers() {
	server.neighborConfMutex.Lock()
	for neighborKey, nbrConf := range server.NeighborConfigMap {
		//check elapsed time and compare with dead timer
		elapsed := time.Since(nbrConf.OspfNbrInactivityTimer)
		if elapsed.Seconds() > nbrConf.OspfNbrDeadTimer.Seconds() {
			fmt.Println("Neighbor id ", neighborKey, "is DEAD")
			//TODO - inform interfaceConf
		}
	}
	server.neighborConfMutex.Unlock()
	time.Sleep(10)
}

func (server *OSPFServer) printIntfNeighbors(nbrId int32) {
	_, list_exists := server.NeighborConfigMap[nbrId]
	if !list_exists {
		fmt.Println("No neighbor with neighbor id - ", nbrId)
		return
	}
	fmt.Println("Printing neighbors for - ", nbrId)

}
