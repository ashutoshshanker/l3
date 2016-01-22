package server

import (
	"fmt"
	"l3/ospf/config"
	"net"
	"time"
)

type NeighborConfKey struct {
	IPAddr  config.IpAddress
	IntfIdx config.InterfaceIndexOrZero
}

type OspfNeighborEntry struct {
	OspfNbrRtrId           uint32
	OspfNbrIPAddr          net.IP
	OspfRtrPrio            uint8
	intfConfKey            IntfConfKey
	OspfNbrOptions         int
	OspfNbrState           config.NbrState
	OspfNbrInactivityTimer time.Time
	OspfNbrDeadTimer       time.Duration
}

func (server *OSPFServer) ProcessHelloPktEvent() {
	nbrData := <-(server.neighborHelloEventCh)
	for {
		fmt.Println("NBREVENT: Received hellopkt event for nbrId ", nbrData.RouterId)
		var nbrConf OspfNeighborEntry
		var nbrState config.NbrState
		//var nbrList list.List

		/*
			intfConfKey := IntfConfKey{
				IPAddr:  intfIPaddr,
				IntfIdx: intfIndex,
			} */
		//Check if neighbor exists
		_, exists := server.NeighborConfigMap[nbrData.RouterId]
		if exists {
			fmt.Println("NBREVENT:Nbr ", nbrData.RouterId, "exists in the global list.")
			server.neighborConfMutex.Lock()
			nbrConf = server.NeighborConfigMap[nbrData.RouterId]
			if nbrData.TwoWayStatus { // update the state
				nbrConf.OspfNbrState = config.NbrTwoWay
			} else {
				nbrConf.OspfNbrState = config.NbrInit
			}
			nbrConf.OspfNbrInactivityTimer = time.Now()
			server.NeighborConfigMap[nbrData.RouterId] = nbrConf
			server.neighborConfMutex.Unlock()
		} else { //neighbor doesnt exist
			fmt.Println("NBREVENT: Create new neighbor with id ", nbrData.RouterId)
			server.neighborConfMutex.Lock()
			if nbrData.TwoWayStatus {
				nbrState = config.NbrTwoWay
			} else {
				nbrState = config.NbrInit
			}
			//fill up neighbor config datastruct
			nbrConf = OspfNeighborEntry{
				OspfNbrRtrId:           nbrData.RouterId,
				OspfNbrIPAddr:          nbrData.NeighborIP,
				OspfRtrPrio:            nbrData.RtrPrio,
				intfConfKey:            nbrData.IntfConfKey,
				OspfNbrOptions:         0,
				OspfNbrState:           nbrState,
				OspfNbrDeadTimer:       nbrData.nbrDeadTimer,
				OspfNbrInactivityTimer: time.Now(),
			}
			server.NeighborConfigMap[nbrData.RouterId] = nbrConf
			server.neighborConfMutex.Unlock()
		}
		fmt.Println("NBREVENT: ADD Nbr ", nbrData.RouterId, "state ", nbrConf.OspfNbrState)

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
	} // end of for
}

/*
 * @fn scanNeighborDeadTimers
 * 	This API will scan the global neighbor entries
 *  and if timers elapsed declare them as dead.
 * IntfConf needs to know state changes. ( to recalculate
 * DR and BDR
 */

func (server *OSPFServer) scanNeighborDeadTimers() {
	server.neighborConfMutex.Lock()
	for neighborKey, nbrConf := range server.NeighborConfigMap {
		//check elapsed time and compare with dead timer
		elapsed := time.Since(nbrConf.OspfNbrInactivityTimer)
		if elapsed.Seconds() > nbrConf.OspfNbrDeadTimer.Seconds() &&
			nbrConf.OspfNbrState != config.NbrDown {
			fmt.Println("Neighbor id ", neighborKey, "is DEAD")
			//TODO - inform interfaceConf
			nbrStateChangeData := NbrStateChangeMsg{
				RouterId: nbrConf.OspfNbrRtrId,
			}
			// update neighbor map
			nbrConf.OspfNbrInactivityTimer = time.Now()
			nbrConf.OspfNbrState = config.NbrDown
			server.NeighborConfigMap[neighborKey] = nbrConf

			intfConf := server.IntfConfMap[nbrConf.intfConfKey]
			intfConf.NbrStateChangeCh <- nbrStateChangeData
		} // end of if
	}
	server.neighborConfMutex.Unlock()
}

func (server *OSPFServer) printIntfNeighbors(nbrId uint32) {
	_, list_exists := server.NeighborConfigMap[nbrId]
	if !list_exists {
		fmt.Println("No neighbor with neighbor id - ", nbrId)
		return
	}
	fmt.Println("Printing neighbors for - ", nbrId)

}
