package server

import (
	"encoding/binary"
	"fmt"
	"l3/ospf/config"
	"net"
	"time"
)

type NbrMsgType uint32

const (
	NBRADD = 0
	NBRUPD = 1
	NBRDEL = 2
)

/* Nbr struct for get bulk
type  OspfNbrEntryState struct{
 	NbrIpAddrKey net.IP
	NbrAddressLessIndexKey int
	NbrRtrId string
	NbrOptions int
	NbrState config.nbrState
	NbrEvents int
	NbrLsRetransQLen int
	NbmaNbrPermanence int
	NbrHelloSuppressed bool
	NbrRestartHelperStatus int
	NbrRestartHelperAge int
	NbrRestartHelperExitReason int
}
*/

type NeighborConfKey struct {
	OspfNbrRtrId uint32
}

var INVALID_NEIGHBOR_CONF_KEY uint32
var neighborBulkSlice []NeighborConfKey

type OspfNeighborEntry struct {
	//	OspfNbrRtrId           uint32
	OspfNbrIPAddr          net.IP
	OspfRtrPrio            uint8
	intfConfKey            IntfConfKey
	OspfNbrOptions         int
	OspfNbrState           config.NbrState
	OspfNbrInactivityTimer time.Time
	OspfNbrDeadTimer       time.Duration
	NbrDeadTimer           *time.Timer
	isDRBDR                bool
	ospfNbrSeqNum          uint32
	isMaster               bool
	ospfNbrDBDTicker       *time.Ticker
}

type ospfNeighborConfMsg struct {
	ospfNbrConfKey NeighborConfKey
	ospfNbrEntry   OspfNeighborEntry
	nbrMsgType     NbrMsgType
}

type ospfNeighborDBDMsg struct {
	ospfNbrConfKey NeighborConfKey
	ospfNbrDBDData ospfDatabaseDescriptionData
}

/*@fn UpdateNeighborConf
Thread to update/add/delete neighbor global struct.
*/
func (server *OSPFServer) UpdateNeighborConf() {
	for {
		select {
		case nbrMsg := <-(server.neighborConfCh):
			server.logger.Info(fmt.Sprintln("Update neighbor conf.  received"))
			nbrConf := OspfNeighborEntry{
				//OspfNbrRtrId:           nbrMsg.ospfNbrConfKey.OspfNbrRtrId,
				OspfNbrIPAddr:          nbrMsg.ospfNbrEntry.OspfNbrIPAddr,
				OspfRtrPrio:            nbrMsg.ospfNbrEntry.OspfRtrPrio,
				intfConfKey:            nbrMsg.ospfNbrEntry.intfConfKey,
				OspfNbrOptions:         0,
				OspfNbrState:           nbrMsg.ospfNbrEntry.OspfNbrState,
				OspfNbrDeadTimer:       nbrMsg.ospfNbrEntry.OspfNbrDeadTimer,
				OspfNbrInactivityTimer: time.Now(),
			}

			if nbrMsg.nbrMsgType == NBRADD {
				//	nbrConf.NbrDeadTimer = time.NewTimer(nbrMsg.ospfNbrEntry.OspfNbrDeadTimer)
				server.neighborBulkSlice = append(server.neighborBulkSlice, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				server.neighborDeadTimerEvent(nbrMsg.ospfNbrConfKey)
			}

			//server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
			server.logger.Info(fmt.Sprintln("Updated neighbor with nbr id - ",
				nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			nbrConf = server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId]
			if nbrMsg.nbrMsgType == NBRUPD {
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				nbrConf.NbrDeadTimer.Stop()
				nbrConf.NbrDeadTimer.Reset(nbrMsg.ospfNbrEntry.OspfNbrDeadTimer)
				server.logger.Info(fmt.Sprintln("UPDATE neighbor with nbr id - ",
					nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			}
			if nbrMsg.nbrMsgType == NBRDEL {
				server.neighborBulkSlice = append(server.neighborBulkSlice, INVALID_NEIGHBOR_CONF_KEY)
				delete(server.NeighborConfigMap, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				//delete(server.neighborKeyToIdxMap, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.logger.Info(fmt.Sprintln("DELETE neighbor with nbr id - ",
					nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			}

		case state := <-(server.neighborConfStopCh):
			server.logger.Info("Exiting update neighbor config thread..")
			if state == true {
				return
			}
		}
	}
}

func (server *OSPFServer) InitNeighborStateMachine() {

	server.neighborBulkSlice = []uint32{}
	INVALID_NEIGHBOR_CONF_KEY = 0

	go server.refreshNeighborSlice()
	server.logger.Info("NBRINIT: Neighbor FSM init done..")
}
func (server *OSPFServer) adjacancyEstablishementCheck(isNbrDRBDR bool, isRtrDRBDR bool) (result bool) {
	if isNbrDRBDR || isRtrDRBDR {
		return true
	}
	/* return true if n/w is p2p , p2mp, virtual link */
	return false
}

func (server *OSPFServer) processDBDEvent(nbrKey NeighborConfKey, nbrDbPkt ospfDatabaseDescriptionData) {
	_, exists := server.NeighborConfigMap[nbrKey.OspfNbrRtrId]
	if exists {
		nbrConf := server.NeighborConfigMap[nbrKey.OspfNbrRtrId]
		//intfConfKey := nbrConf.intfConfKey
		//intfConf :=
		switch nbrConf.OspfNbrState {
		case config.NbrAttempt:
			/* reject packet */
			return
		case config.NbrInit:
		case config.NbrExchangeStart:
			//intfKey := nbrConf.intfConfKey
			var isAdjacent bool
			if nbrConf.OspfNbrState == config.NbrInit {
				isAdjacent = server.adjacancyEstablishementCheck(nbrConf.isDRBDR, true)
			}
			if isAdjacent || nbrConf.OspfNbrState == config.NbrExchangeStart {
				// change nbr state
				nbrConf.OspfNbrState = config.NbrExchangeStart
				/* The initialize(I), more (M) and master(MS) bits are set,
				   the contents of the packet are empty, and the neighbor's
				   Router ID is larger than the router's own.  In this case
				   the router is now Slave.  Set the master/slave bit to
				   slave, and set the neighbor data structure's DD sequence
				   number to that specified by the master.*/
				if nbrDbPkt.ibit && nbrDbPkt.mbit && nbrDbPkt.msbit &&
					nbrKey.OspfNbrRtrId > binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId) {
					server.logger.Info(fmt.Sprintln("NBRDBD: SLAVE = self,  MASTER = ", nbrKey.OspfNbrRtrId))
				}

				/*   The initialize(I) and master(MS) bits are off, the
				     packet's DD sequence number equals the neighbor data
				     structure's DD sequence number (indicating
				     acknowledgment) and the neighbor's Router ID is smaller
				     than the router's own.  In this case the router is
				     Master. */
				if nbrDbPkt.ibit == false && nbrDbPkt.msbit == false &&
					nbrDbPkt.dd_sequence_number == nbrConf.ospfNbrSeqNum &&
					nbrKey.OspfNbrRtrId < binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId) {
					nbrConf.isMaster = true
					server.logger.Info(fmt.Sprintln("NBRDBD: SLAVE = ", nbrKey.OspfNbrRtrId, "MASTER = SELF"))
				} else {
					nbrConf.isMaster = false
				}

			} else {
				nbrConf.OspfNbrState = config.NbrTwoWay
			}
			nbrConfMsg := ospfNeighborConfMsg{
				ospfNbrConfKey: NeighborConfKey{
					OspfNbrRtrId: nbrKey.OspfNbrRtrId,
				},
				ospfNbrEntry: OspfNeighborEntry{
					//OspfNbrRtrId:           nbrConf.OspfNbrRtrId,
					OspfNbrIPAddr:          nbrConf.OspfNbrIPAddr,
					OspfRtrPrio:            nbrConf.OspfRtrPrio,
					intfConfKey:            nbrConf.intfConfKey,
					OspfNbrOptions:         0,
					OspfNbrState:           nbrConf.OspfNbrState,
					OspfNbrInactivityTimer: time.Now(),
					OspfNbrDeadTimer:       nbrConf.OspfNbrDeadTimer,
				},
				nbrMsgType: NBRUPD,
			}
			server.neighborConfCh <- nbrConfMsg
		case config.NbrTwoWay:
			/* ignore packet */
			server.logger.Info(fmt.Sprintln("NBRDBD: Ignore packet as NBR state is two way"))
			return
		case config.NbrDown:
			/* ignore packet. */
			server.logger.Info(fmt.Sprintln("NBRDBD: Ignore packet . NBR is down"))
			return
		} // end of switch
	} else { //nbr doesnt exist
		server.logger.Info(fmt.Sprintln("Ignore DB packet. Nbr doesnt exist ", nbrKey))
		return
	}
}
func (server *OSPFServer) ProcessHelloPktEvent() {
	for {
		select {
		case nbrDbPkt := <-(server.neighborDBDEventCh):
			server.logger.Info(fmt.Sprintln("NBREVENT: DBD received  ", nbrDbPkt))
			server.processDBDEvent(nbrDbPkt.ospfNbrConfKey, nbrDbPkt.ospfNbrDBDData)

		case nbrData := <-(server.neighborHelloEventCh):
			server.logger.Info(fmt.Sprintln("NBREVENT: Received hellopkt event for nbrId ", nbrData.RouterId))
			var nbrConf OspfNeighborEntry
			intfConf := server.IntfConfMap[nbrData.IntfConfKey]
			//var nbrList list.List

			//Check if neighbor exists
			_, exists := server.NeighborConfigMap[nbrData.RouterId]
			if exists {
				fmt.Println("NBREVENT:Nbr ", nbrData.RouterId, "exists in the global list.")

				nbrConf = server.NeighborConfigMap[nbrData.RouterId]
				if nbrData.TwoWayStatus { // update the state
					startAdjacency := server.adjacancyEstablishementCheck(nbrConf.isDRBDR, true)
					if startAdjacency {
						var dbd_mdata ospfDatabaseDescriptionData
						nbrConf.OspfNbrState = config.NbrExchangeStart

						if nbrConf.ospfNbrSeqNum == 0 {
							nbrConf.ospfNbrSeqNum = uint32(time.Now().Unix())
							dbd_mdata.dd_sequence_number = nbrConf.ospfNbrSeqNum
							dbd_mdata.msbit = true // i am master
							nbrConf.isMaster = true
						} else {
							dbd_mdata.dd_sequence_number = nbrConf.ospfNbrSeqNum
							dbd_mdata.msbit = false
							nbrConf.isMaster = false
						}
						dbd_mdata.interface_mtu = INTF_MTU_MIN
						server.logger.Info(fmt.Sprintln("NBRHELLO: Send, seq no ", dbd_mdata.dd_sequence_number,
							"msbit ", dbd_mdata.msbit))
						// send dbd packets
						server.BuildAndSendDBDPkt(nbrData.IntfConfKey, intfConf, nbrConf, dbd_mdata)
					} else { // no adjacency
						nbrConf.OspfNbrState = config.NbrTwoWay
					}
				} else {
					nbrConf.OspfNbrState = config.NbrInit
				}

				nbrConfMsg := ospfNeighborConfMsg{
					ospfNbrConfKey: NeighborConfKey{
						OspfNbrRtrId: nbrData.RouterId,
					},
					ospfNbrEntry: OspfNeighborEntry{
						//OspfNbrRtrId:           nbrConf.OspfNbrRtrId,
						OspfNbrIPAddr:          nbrConf.OspfNbrIPAddr,
						OspfRtrPrio:            nbrConf.OspfRtrPrio,
						intfConfKey:            nbrConf.intfConfKey,
						OspfNbrOptions:         0,
						OspfNbrState:           nbrConf.OspfNbrState,
						OspfNbrInactivityTimer: time.Now(),
						OspfNbrDeadTimer:       nbrConf.OspfNbrDeadTimer,
					},
					nbrMsgType: NBRUPD,
				}
				server.neighborConfCh <- nbrConfMsg
			} else { //neighbor doesnt exist
				var nbrState config.NbrState
				var dbd_mdata ospfDatabaseDescriptionData
				server.logger.Info(fmt.Sprintln("NBREVENT: Create new neighbor with id ", nbrData.RouterId))
				if nbrData.TwoWayStatus { // update the state
					startAdjacency := server.adjacancyEstablishementCheck(false, true)
					if startAdjacency {
						var dbd_mdata ospfDatabaseDescriptionData
						nbrState = config.NbrExchangeStart
						//nbrConf.ospfNbrSeqNum = uint32(time.Now().Nanosecond())
						dbd_mdata.dd_sequence_number = uint32(time.Now().Nanosecond())
						dbd_mdata.msbit = true // i am master
						dbd_mdata.interface_mtu = INTF_MTU_MIN
						// send dbd packets
						server.BuildAndSendDBDPkt(nbrData.IntfConfKey, intfConf, nbrConf, dbd_mdata)
						server.logger.Info(fmt.Sprintln("NBRHELLO: Send, seq no ", dbd_mdata.dd_sequence_number,
							"msbit ", dbd_mdata.msbit))
					} else { // no adjacency
						nbrState = config.NbrTwoWay
					}
				} else {
					nbrState = config.NbrInit
				}

				nbrConfMsg := ospfNeighborConfMsg{
					ospfNbrConfKey: NeighborConfKey{
						OspfNbrRtrId: nbrData.RouterId,
					},
					ospfNbrEntry: OspfNeighborEntry{
						//OspfNbrRtrId:           nbrData.NeighborIP,
						OspfNbrIPAddr:          nbrData.NeighborIP,
						OspfRtrPrio:            nbrData.RtrPrio,
						intfConfKey:            nbrData.IntfConfKey,
						OspfNbrOptions:         0,
						OspfNbrState:           nbrState,
						OspfNbrInactivityTimer: time.Now(),
						OspfNbrDeadTimer:       nbrData.nbrDeadTimer,
						ospfNbrSeqNum:          dbd_mdata.dd_sequence_number,
					},
					nbrMsgType: NBRADD,
				}
				server.neighborConfCh <- nbrConfMsg

				//fill up neighbor config datastruct
			}

			server.logger.Info(fmt.Sprintln("NBREVENT: ADD Nbr ", nbrData.RouterId, "state ", nbrConf.OspfNbrState))

		case state := <-server.neighborFSMCtrlCh:
			if state == false {
				return
			}
		}
	} // end of for
}

func (server *OSPFServer) neighborDeadTimerEvent(nbrConfKey NeighborConfKey) {
	var nbr_entry_dead_func func()

	nbr_entry_dead_func = func() {
		server.logger.Info(fmt.Sprintln("NBRSCAN: DEAD ", nbrConfKey.OspfNbrRtrId))
		nbrStateChangeData := NbrStateChangeMsg{
			RouterId: nbrConfKey.OspfNbrRtrId,
		}

		_, exists := server.NeighborConfigMap[nbrConfKey.OspfNbrRtrId]
		if exists {
			nbrConf := server.NeighborConfigMap[nbrConfKey.OspfNbrRtrId]

			nbrConfMsg := ospfNeighborConfMsg{
				ospfNbrConfKey: NeighborConfKey{
					OspfNbrRtrId: nbrConfKey.OspfNbrRtrId,
				},
				ospfNbrEntry: OspfNeighborEntry{
					//OspfNbrRtrId:           nbrData.NeighborIP,
					OspfNbrIPAddr:          nbrConf.OspfNbrIPAddr,
					OspfRtrPrio:            nbrConf.OspfRtrPrio,
					intfConfKey:            nbrConf.intfConfKey,
					OspfNbrOptions:         0,
					OspfNbrState:           config.NbrDown,
					OspfNbrInactivityTimer: time.Now(),
					OspfNbrDeadTimer:       nbrConf.OspfNbrDeadTimer,
				},
				nbrMsgType: NBRDEL,
			}
			// update neighbor map
			server.neighborConfCh <- nbrConfMsg
			intfConf := server.IntfConfMap[nbrConf.intfConfKey]
			intfConf.NbrStateChangeCh <- nbrStateChangeData
		}
	} // end of afterFunc callback

	_, exists := server.NeighborConfigMap[nbrConfKey.OspfNbrRtrId]
	if exists {
		nbrConf := server.NeighborConfigMap[nbrConfKey.OspfNbrRtrId]
		nbrConf.NbrDeadTimer = time.AfterFunc(nbrConf.OspfNbrDeadTimer, nbr_entry_dead_func)
		server.NeighborConfigMap[nbrConfKey.OspfNbrRtrId] = nbrConf
	}

}

func (server *OSPFServer) printIntfNeighbors(nbrId uint32) {
	_, list_exists := server.NeighborConfigMap[nbrId]
	if !list_exists {
		fmt.Println("No neighbor with neighbor id - ", nbrId)
		return
	}
	fmt.Println("Printing neighbors for - ", nbrId)

}

func (server *OSPFServer) refreshNeighborSlice() {
	go func() {
		for t := range server.neighborSliceRefCh.C {

			server.neighborBulkSlice = []uint32{}
			//server.neighborKeyToIdxMap = make(map[uint32]uint32)
			idx := 0
			for nbrKey, _ := range server.NeighborConfigMap {
				server.neighborBulkSlice = append(server.neighborBulkSlice, nbrKey)
				//server.neighborKeyToIdxMap[nbrKey] = uint32(idx)
				idx++
			}

			fmt.Println("Tick at", t)
		}
	}()

}
