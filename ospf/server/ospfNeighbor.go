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

const (
	RxDBDInterval = 5
)

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
	ospfNbrDBDTickerCh     *time.Ticker
	ospfNbrDBDSendCh       chan ospfDatabaseDescriptionData
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

/* @fn exchangePacketDiscardCheck
    Function to check SeqNumberMismatch
	for dbd exchange state packets.
*/
func (server *OSPFServer) exchangePacketDiscardCheck(nbrConf OspfNeighborEntry, nbrDbPkt ospfDatabaseDescriptionData) (isDiscard bool) {
	if nbrDbPkt.msbit != nbrConf.isMaster {
		server.logger.Info(fmt.Sprintln("NBREVENT: SeqNumberMismatch. Nbr should be master"))
		return false
	}

	if nbrDbPkt.ibit == true {
		server.logger.Info("NBREVENT:SeqNumberMismatch . Nbr ibit is true ")
		return false
	}
	if nbrDbPkt.options != INTF_OPTIONS {
		server.logger.Info(fmt.Sprintln("NBREVENT:SeqNumberMismatch. Nbr options dont match. Nbr options ", nbrDbPkt.options,
			" dbd oackts options", nbrDbPkt.options))
		return false
	}

	if nbrConf.isMaster {
		if nbrDbPkt.dd_sequence_number != nbrConf.ospfNbrSeqNum {
			server.logger.Info(fmt.Sprintln("NBREVENT:SeqNumberMismatch : Nbr is master but dbd packet has M bit 0"))
			return false
		}
	} else {
		if nbrDbPkt.dd_sequence_number != nbrConf.ospfNbrSeqNum+1 {
			server.logger.Info(fmt.Sprintln("NBREVENT:SeqNumberMismatch : Nbr is slave but dbd packet has M bit 1"))

			return false
		}
	}

	return true
}

/*@fn UpdateNeighborConf
Thread to update/add/delete neighbor global struct.
*/
func (server *OSPFServer) UpdateNeighborConf() {
	for {
		select {
		case nbrMsg := <-(server.neighborConfCh):
			var nbrConf OspfNeighborEntry
			server.logger.Info(fmt.Sprintln("Update neighbor conf.  received"))
			if nbrMsg.nbrMsgType == NBRUPD {
				nbrConf = server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId]
			}
			nbrConf.OspfNbrIPAddr = nbrMsg.ospfNbrEntry.OspfNbrIPAddr
			nbrConf.OspfRtrPrio = nbrMsg.ospfNbrEntry.OspfRtrPrio
			nbrConf.intfConfKey = nbrMsg.ospfNbrEntry.intfConfKey
			nbrConf.OspfNbrOptions = 0
			nbrConf.OspfNbrState = nbrMsg.ospfNbrEntry.OspfNbrState
			nbrConf.OspfNbrDeadTimer = nbrMsg.ospfNbrEntry.OspfNbrDeadTimer
			nbrConf.OspfNbrInactivityTimer = time.Now()
			nbrConf.ospfNbrSeqNum = nbrMsg.ospfNbrEntry.ospfNbrSeqNum
			nbrConf.ospfNbrDBDTickerCh = nbrMsg.ospfNbrEntry.ospfNbrDBDTickerCh

			if nbrMsg.nbrMsgType == NBRADD {
				nbrConf.ospfNbrDBDSendCh = nbrMsg.ospfNbrEntry.ospfNbrDBDSendCh
				server.neighborBulkSlice = append(server.neighborBulkSlice, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				server.neighborDeadTimerEvent(nbrMsg.ospfNbrConfKey)
			}

			server.logger.Info(fmt.Sprintln("Updated neighbor with nbr id - ",
				nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			//nbrConf = server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId]
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
			var negotiationDone bool
			var dbd_mdata ospfDatabaseDescriptionData
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
				server.logger.Info(fmt.Sprintln("NBRDBD: nbr rtr id ", nbrKey.OspfNbrRtrId,
					" my router id ", binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId),
					" nbr_seq ", nbrConf.ospfNbrSeqNum, "dbd_seq no ", nbrDbPkt.dd_sequence_number))
				if nbrDbPkt.ibit && nbrDbPkt.mbit && nbrDbPkt.msbit &&
					nbrKey.OspfNbrRtrId > binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId) {
					server.logger.Info(fmt.Sprintln("NBRDBD: SLAVE = self,  MASTER = ", nbrKey.OspfNbrRtrId))
					nbrConf.isMaster = true
					server.logger.Info("NBREVENT: Negotiation done..")
					negotiationDone = true
					//nbrConf.ospfNbrDBDTickerCh.Stop()
					nbrConf.OspfNbrState = config.NbrExchange
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
					nbrConf.isMaster = false
					server.logger.Info(fmt.Sprintln("NBRDBD: SLAVE = ", nbrKey.OspfNbrRtrId, "MASTER = SELF"))
					server.logger.Info("NBREVENT: Negotiation done..")
					negotiationDone = true
					//nbrConf.ospfNbrDBDTickerCh.Stop()
					nbrConf.OspfNbrState = config.NbrExchange
				}

			} else {
				nbrConf.OspfNbrState = config.NbrTwoWay
			}

			dbd_mdata.interface_mtu = INTF_MTU_MIN
			dbd_mdata.options = INTF_OPTIONS
			if negotiationDone {
				if nbrConf.isMaster != true {
					dbd_mdata.dd_sequence_number = nbrDbPkt.dd_sequence_number + 1
					dbd_mdata.ibit = false
					dbd_mdata.mbit = true
					dbd_mdata.msbit = true
					// send  DBD with LSA description
				} else {
					// send acknowledgement DBD with I and MS bit false , mbit = 1
					dbd_mdata.dd_sequence_number = nbrDbPkt.dd_sequence_number
					dbd_mdata.ibit = false
					dbd_mdata.mbit = true
					dbd_mdata.msbit = false
				}
			} else { // negotiation not done
				dbd_mdata.ibit = true
				dbd_mdata.mbit = true
				if nbrConf.isMaster {
					dbd_mdata.msbit = false
					dbd_mdata.dd_sequence_number = nbrDbPkt.dd_sequence_number
				} else {
					dbd_mdata.msbit = true
					dbd_mdata.dd_sequence_number = nbrConf.ospfNbrSeqNum
				}
			}
			nbrConf.ospfNbrDBDSendCh <- dbd_mdata

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
					ospfNbrSeqNum:          dbd_mdata.dd_sequence_number,
					isMaster:               nbrConf.isMaster,
				},
				nbrMsgType: NBRUPD,
			}
			server.neighborConfCh <- nbrConfMsg

		case config.NbrExchange:
			isDiscard := server.exchangePacketDiscardCheck(nbrConf, nbrDbPkt)
			if isDiscard {
				server.logger.Info(fmt.Sprintln("NBRDBD: Discard packet. nbr", nbrKey.OspfNbrRtrId,
					" nbr state ", nbrConf.OspfNbrState))
				// dont update nbr state as it remains in exchange state.
				return
			}
			if nbrConf.isMaster != true {
				/* send DBD with seq num + 1 , ibit = 0 ,  ms = 1
				 * if this is the last DBD for LSA description set mbit = 0
				 */
			} else {
				/* send acknowledgement DBD with I and MS bit false and mbit same as
				rx packet */
			}

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
			var send_dbd bool
			var dbd_mdata ospfDatabaseDescriptionData

			//Check if neighbor exists
			_, exists := server.NeighborConfigMap[nbrData.RouterId]
			send_dbd = false
			if exists {
				fmt.Println("NBREVENT:Nbr ", nbrData.RouterId, "exists in the global list.")

				nbrConf = server.NeighborConfigMap[nbrData.RouterId]
				if nbrData.TwoWayStatus { // update the state
					startAdjacency := server.adjacancyEstablishementCheck(nbrConf.isDRBDR, true)
					if startAdjacency && nbrConf.OspfNbrState == config.NbrTwoWay {

						nbrConf.OspfNbrState = config.NbrExchangeStart

						if nbrConf.ospfNbrSeqNum == 0 {
							nbrConf.ospfNbrSeqNum = uint32(time.Now().Unix())
							dbd_mdata.dd_sequence_number = nbrConf.ospfNbrSeqNum
							dbd_mdata.msbit = true // i am master
							dbd_mdata.ibit = true
							dbd_mdata.mbit = true
							nbrConf.isMaster = true
							go server.SendDBDPkt(nbrData.RouterId)
						} else {
							dbd_mdata.dd_sequence_number = nbrConf.ospfNbrSeqNum
							dbd_mdata.msbit = false
							nbrConf.isMaster = false
						}
						dbd_mdata.interface_mtu = INTF_MTU_MIN
						server.logger.Info(fmt.Sprintln("NBRHELLO: Send, seq no ", dbd_mdata.dd_sequence_number,
							"msbit ", dbd_mdata.msbit))
						// send dbd packets
						//	nbrConf.ospfNbrDBDTickerCh = time.NewTicker(time.Second * 10)

						send_dbd = true
					} else { // no adjacency
						if nbrConf.OspfNbrState < config.NbrTwoWay {
							nbrConf.OspfNbrState = config.NbrTwoWay
						}
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
						ospfNbrDBDTickerCh:     nbrConf.ospfNbrDBDTickerCh,
						ospfNbrSeqNum:          nbrConf.ospfNbrSeqNum,
					},
					nbrMsgType: NBRUPD,
				}
				server.neighborConfCh <- nbrConfMsg

				if send_dbd {
					nbrConf.ospfNbrDBDSendCh <- dbd_mdata
				}
			} else { //neighbor doesnt exist
				var ticker *time.Ticker
				var nbrState config.NbrState
				var dbd_mdata ospfDatabaseDescriptionData
				var send_dbd bool
				server.logger.Info(fmt.Sprintln("NBREVENT: Create new neighbor with id ", nbrData.RouterId))
				if nbrData.TwoWayStatus { // update the state
					startAdjacency := server.adjacancyEstablishementCheck(false, true)
					if startAdjacency {
						var dbd_mdata ospfDatabaseDescriptionData
						nbrState = config.NbrExchangeStart
						//nbrConf.ospfNbrSeqNum = uint32(time.Now().Nanosecond())
						dbd_mdata.dd_sequence_number = uint32(time.Now().Nanosecond())
						dbd_mdata.msbit = true // i am master
						dbd_mdata.ibit = true
						dbd_mdata.mbit = true
						dbd_mdata.interface_mtu = INTF_MTU_MIN
						// send dbd packets
						ticker = time.NewTicker(time.Second * 10)
						go server.SendDBDPkt(nbrData.RouterId)
						//go server.BuildAndSendDBDPkt(nbrData.IntfConfKey, intfConf, nbrConf, dbd_mdata)
						send_dbd = true
						server.logger.Info(fmt.Sprintln("NBRHELLO: Send, seq no ", dbd_mdata.dd_sequence_number,
							"msbit ", dbd_mdata.msbit))
					} else { // no adjacency
						nbrState = config.NbrTwoWay
						send_dbd = false
					}
				} else {
					nbrState = config.NbrInit
					send_dbd = false
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
						ospfNbrDBDTickerCh:     ticker,
						ospfNbrDBDSendCh:       make(chan ospfDatabaseDescriptionData),
					},
					nbrMsgType: NBRADD,
				}
				server.neighborConfCh <- nbrConfMsg
				if send_dbd {
					nbrConf.ospfNbrDBDSendCh <- dbd_mdata
				}
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
