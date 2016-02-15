package server

import (
	"encoding/binary"
	"fmt"
	"l3/ospf/config"
	"time"
)

/* @fn exchangePacketDiscardCheck
    Function to check SeqNumberMismatch
	for dbd exchange state packets.
*/
func (server *OSPFServer) exchangePacketDiscardCheck(nbrConf OspfNeighborEntry, nbrDbPkt ospfDatabaseDescriptionData) (isDiscard bool) {
	if nbrDbPkt.msbit != nbrConf.isMaster {
		server.logger.Info(fmt.Sprintln("NBREVENT: SeqNumberMismatch. Nbr should be master"))
		return true
	}

	if nbrDbPkt.ibit == true {
		server.logger.Info("NBREVENT:SeqNumberMismatch . Nbr ibit is true ")
		return true
	}
	if nbrDbPkt.options != INTF_OPTIONS {
		server.logger.Info(fmt.Sprintln("NBREVENT:SeqNumberMismatch. Nbr options dont match. Nbr options ", nbrDbPkt.options,
			" dbd oackts options", nbrDbPkt.options))
		return true
	}

	if nbrConf.isMaster {
		if nbrDbPkt.dd_sequence_number != nbrConf.ospfNbrSeqNum {
			server.logger.Info(fmt.Sprintln("NBREVENT:SeqNumberMismatch : Nbr is master but dbd packet seq no doesnt match. dbd seq ",
				nbrDbPkt.dd_sequence_number, "nbr seq ", nbrConf.ospfNbrSeqNum))
			return true
		}
	} else {
		if nbrDbPkt.dd_sequence_number != nbrConf.ospfNbrSeqNum {
			server.logger.Info(fmt.Sprintln("NBREVENT:SeqNumberMismatch : Nbr is slave but dbd packet seq no doesnt match.dbd seq ",
				nbrDbPkt.dd_sequence_number, "nbr seq ", nbrConf.ospfNbrSeqNum))
			return true
		}
	}

	return false
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
			nbrConf.OspfNbrState = nbrMsg.ospfNbrEntry.OspfNbrState
			nbrConf.OspfNbrDeadTimer = nbrMsg.ospfNbrEntry.OspfNbrDeadTimer
			nbrConf.OspfNbrInactivityTimer = time.Now()
			nbrConf.ospfNbrSeqNum = nbrMsg.ospfNbrEntry.ospfNbrSeqNum
			nbrConf.ospfNbrDBDTickerCh = nbrMsg.ospfNbrEntry.ospfNbrDBDTickerCh
			nbrConf.isMaster = nbrMsg.ospfNbrEntry.isMaster
			nbrConf.ospfNbrLsaReqIndex = nbrMsg.ospfNbrEntry.ospfNbrLsaReqIndex

			if nbrMsg.nbrMsgType == NBRADD {
				nbrConf.OspfNbrIPAddr = nbrMsg.ospfNbrEntry.OspfNbrIPAddr
				nbrConf.OspfRtrPrio = nbrMsg.ospfNbrEntry.OspfRtrPrio
				nbrConf.intfConfKey = nbrMsg.ospfNbrEntry.intfConfKey
				nbrConf.OspfNbrOptions = 0

				nbrConf.ospfNbrDBDSendCh = nbrMsg.ospfNbrEntry.ospfNbrDBDSendCh
				server.neighborBulkSlice = append(server.neighborBulkSlice, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				nbrConf.ospf_db_summary_list = []ospfLSAHeader{}
				nbrConf.ospf_db_req_list = []ospfLSAHeader{}
				nbrConf.ospfNbrDBDStopCh = make(chan bool)
				nbrConf.ospfNbrLsaSendCh = make(chan []ospfLSAReq)
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				server.neighborDeadTimerEvent(nbrMsg.ospfNbrConfKey)
				if nbrMsg.ospfNbrEntry.OspfNbrState >= config.NbrTwoWay {
					server.ConstructAndSendDbdPacket(nbrMsg.ospfNbrConfKey, 0, true, true, true, uint32(time.Now().Nanosecond()), false, false)
				}
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
				//nbrConf.ospfNbrDBDTickerCh.Stop()
				nbrConf.ospfNbrDBDStopCh <- true
				delete(server.NeighborConfigMap, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				//delete(server.neighborKeyToIdxMap, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.logger.Info(fmt.Sprintln("DELETE neighbor with nbr id - ",
					nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			}

			server.logger.Info(fmt.Sprintln("NBR UPDATE: Nbr , seq_no ", nbrMsg.ospfNbrConfKey.OspfNbrRtrId, nbrConf.ospfNbrSeqNum))
		case state := <-(server.neighborConfStopCh):
			server.logger.Info("Exiting update neighbor config thread..")
			if state == true {
				return
			}
		}
	}
}

func (server *OSPFServer) adjacancyEstablishementCheck(isNbrDRBDR bool, isRtrDRBDR bool) (result bool) {
	if isNbrDRBDR || isRtrDRBDR {
		return true
	}
	/* return true if n/w is p2p , p2mp, virtual link */
	return false
}

func (server *OSPFServer) lsaAddCheck(lsaheader ospfLSAHeader) (result bool) {
	/*
		TODO check if the entry exist in LSDB.
	*/
	return true
}

func (server *OSPFServer) ConstructAndSendDbdPacket(nbrKey NeighborConfKey, lsa_attach uint8,
	ibit bool, mbit bool, msbit bool,
	seq uint32, append_lsa bool, is_duplicate bool) (dbd_mdata ospfDatabaseDescriptionData) {
	nbrCon, exists := server.NeighborConfigMap[nbrKey.OspfNbrRtrId]
	if !exists {
		server.logger.Info(fmt.Sprintln("DBD: Failed to send initial dbd packet as nbr doesnt exist. nbr", nbrKey.OspfNbrRtrId))
		return dbd_mdata
	}
	dbd_mdata.ibit = ibit
	dbd_mdata.mbit = mbit
	dbd_mdata.msbit = msbit

	dbd_mdata.interface_mtu = INTF_MTU_MIN
	dbd_mdata.options = INTF_OPTIONS
	dbd_mdata.dd_sequence_number = seq

	if append_lsa && exists {
		dbd_mdata.lsa_headers = []ospfLSAHeader{}
		fmt.Sprintln("DBD: add lsa to the packet..")
		var index uint8
		for index = 0; index < lsa_attach; index++ {
			dbd_mdata.lsa_headers[index] = nbrCon.ospf_db_summary_list[nbrCon.ospfNbrLsaIndex+index]
		}

	}
	server.logger.Info(fmt.Sprintln("DBDSEND: nbr state ", nbrCon.OspfNbrState,
		" imms ", dbd_mdata.ibit, dbd_mdata.mbit, dbd_mdata.msbit,
		" seq num ", seq, "options ", dbd_mdata.options))
	nbrCon.ospfNbrDBDSendCh <- dbd_mdata
	return dbd_mdata
}

func (server *OSPFServer) processDBDEvent(nbrKey NeighborConfKey, nbrDbPkt ospfDatabaseDescriptionData) {
	_, exists := server.NeighborConfigMap[nbrKey.OspfNbrRtrId]
	var dbd_mdata ospfDatabaseDescriptionData
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

			//if nbrConf.OspfNbrState == config.NbrInit {
			isAdjacent = server.adjacancyEstablishementCheck(nbrConf.isDRBDR, true)
			//}
			if isAdjacent || nbrConf.OspfNbrState == config.NbrExchangeStart {
				// change nbr state
				nbrConf.OspfNbrState = config.NbrExchangeStart
				// decide master slave relation
				if nbrKey.OspfNbrRtrId > binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId) {
					nbrConf.isMaster = true
				} else {
					nbrConf.isMaster = false
				}
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
					server.logger.Info(fmt.Sprintln("DBD: (ExStart/slave) SLAVE = self,  MASTER = ", nbrKey.OspfNbrRtrId))
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
					server.logger.Info(fmt.Sprintln("DBD:(ExStart) SLAVE = ", nbrKey.OspfNbrRtrId, "MASTER = SELF"))
					server.logger.Info("NBREVENT: Negotiation done..")
					negotiationDone = true
					//nbrConf.ospfNbrDBDTickerCh.Stop()
					nbrConf.OspfNbrState = config.NbrExchange
				}

			} else {
				nbrConf.OspfNbrState = config.NbrTwoWay
				//nbrConf.ospfNbrDBDStopCh <- true // stop nbr thread
			}

			if negotiationDone {
				server.logger.Info(fmt.Sprintln("DBD: (Exstart) lsa_headers = ", len(nbrDbPkt.lsa_headers)))
				if nbrConf.isMaster != true { // i am the master
					dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, 0, false, true, true,
						nbrDbPkt.dd_sequence_number+1, false, false)
					// send  DBD with LSA description
				} else {
					// send acknowledgement DBD with I and MS bit false , mbit = 1
					/* TODO - check if LSA needs to be sent else mark m bit as 0 and
					   state as exchange. */
					dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, 0, false, false, false,
						nbrDbPkt.dd_sequence_number, false, false)
					dbd_mdata.dd_sequence_number++

				}
			} else { // negotiation not done
				if nbrConf.isMaster {
					dbd_mdata.dd_sequence_number = nbrDbPkt.dd_sequence_number
					dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, 0, true, true, false,
						nbrDbPkt.dd_sequence_number, false, false)
					dbd_mdata.dd_sequence_number++
				} else {
					//start with new seq number
					dbd_mdata.dd_sequence_number = uint32(time.Now().Nanosecond()) //nbrConf.ospfNbrSeqNum
					dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, 0, true, true, true,
						nbrDbPkt.dd_sequence_number, false, false)
				}
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
					ospfNbrSeqNum:          dbd_mdata.dd_sequence_number,
					isMaster:               nbrConf.isMaster,
					ospfNbrLsaIndex:        0,
				},
				nbrMsgType: NBRUPD,
			}
			server.neighborConfCh <- nbrConfMsg
			OspfNeighborLastDbd[nbrKey] = dbd_mdata

		case config.NbrExchange:
			//TODO - Fix THIS
			isDiscard := server.exchangePacketDiscardCheck(nbrConf, nbrDbPkt)
			last_exchange := false
			if isDiscard {
				server.logger.Info(fmt.Sprintln("NBRDBD: Discard packet. nbr", nbrKey.OspfNbrRtrId,
					" nbr state ", nbrConf.OspfNbrState))
				//update neighbor to exchange start state and send dbd

				nbrConf.OspfNbrState = config.NbrExchangeStart
				nbrConf.ospf_db_req_list = []ospfLSAHeader{}
				nbrConf.ospf_db_summary_list = []ospfLSAHeader{}
				//nbrConf.isMaster = false
				nbrConf.ospfNbrDBDSendCh <- OspfNeighborLastDbd[nbrKey]
				//dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, 0, true, true, true, nbrConf.ospfNbrSeqNum+1, false, false)
			} else { // process exchange state
				/* 1) get lsa headers update in req_list */
				headers_len := len(nbrDbPkt.lsa_headers)
				server.logger.Info(fmt.Sprintln("DBD: (Exchange) Received . nbr,total_lsa ", nbrKey.OspfNbrRtrId, headers_len))

				for i := 0; i < headers_len; i++ {
					var lsaheader ospfLSAHeader
					lsaheader = nbrDbPkt.lsa_headers[i]
					result := server.lsaAddCheck(lsaheader) // check lsdb
					if result {
						nbrConf.ospf_db_req_list = append(nbrConf.ospf_db_req_list, lsaheader)
						items := len(nbrConf.ospf_db_req_list)
						server.logger.Info(fmt.Sprintln("DBD:(Exchange) Total elements in req_list ", items))
						server.logger.Info(fmt.Sprintln("DBD: (Exchange) Append nbr, lsa_header_id", nbrKey.OspfNbrRtrId,
							lsaheader.adv_router_id))
					}
				}

				/* 2) Add lsa_headers to db packet from db_summary list */
				max_lsa_headers := calculateMaxLsaHeaders()
				slice_len := len(nbrConf.ospf_db_summary_list)
				var lsa_attach uint8
				if max_lsa_headers > (uint8(slice_len) - uint8(nbrConf.ospfNbrLsaIndex)) {
					lsa_attach = uint8(slice_len) - uint8(nbrConf.ospfNbrLsaIndex)
				} else {
					lsa_attach = max_lsa_headers
				}
				if (nbrConf.ospfNbrLsaIndex + lsa_attach) >= uint8(slice_len) {
					// the last slice in the list being sent
					server.logger.Info(fmt.Sprintln("DBD: (Exchange) Send the last dd packet with nbr ", nbrKey.OspfNbrRtrId))
					last_exchange = true
				}

				if nbrConf.isMaster != true { // i am master
					/* send DBD with seq num + 1 , ibit = 0 ,  ms = 1
					 * if this is the last DBD for LSA description set mbit = 0
					 */
					if nbrDbPkt.dd_sequence_number == nbrConf.ospfNbrSeqNum {
						server.logger.Info(fmt.Sprintln("DBD: (master/Exchange) Send next packet in the exchange  to nbr ", nbrKey.OspfNbrRtrId))
						dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, lsa_attach, false, !last_exchange, true, nbrDbPkt.dd_sequence_number+1, true, false)
						nbrConf.ospfNbrLsaIndex = nbrConf.ospfNbrLsaIndex + lsa_attach
						OspfNeighborLastDbd[nbrKey] = dbd_mdata
					} else {
						// send old packet
						server.logger.Info(fmt.Sprintln("DBD: (master/exchange) Duplicated dbd. Resend . dbd_seq , nbr_seq_num ",
							nbrDbPkt.dd_sequence_number, nbrConf.ospfNbrSeqNum))
						nbrConf.ospfNbrDBDSendCh <- OspfNeighborLastDbd[nbrKey]
					}
				} else { // i am slave
					/* send acknowledgement DBD with I and MS bit false and mbit same as
					rx packet */
					server.logger.Info(fmt.Sprintln("DBD: (slave/Exchange) Send next packet in the exchange  to nbr ", nbrKey.OspfNbrRtrId))
					if nbrDbPkt.dd_sequence_number == nbrConf.ospfNbrSeqNum {
						dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, lsa_attach, false, !last_exchange, false, nbrDbPkt.dd_sequence_number, true, false)
						nbrConf.ospfNbrLsaIndex = nbrConf.ospfNbrLsaIndex + lsa_attach
						OspfNeighborLastDbd[nbrKey] = dbd_mdata
						dbd_mdata.dd_sequence_number++
					} else {
						server.logger.Info(fmt.Sprintln("DBD: (slave/exchange) Duplicated dbd. Resend . dbd_seq , nbr_seq_num ",
							nbrDbPkt.dd_sequence_number, nbrConf.ospfNbrSeqNum))
						// send old ACK
						nbrConf.ospfNbrDBDSendCh <- OspfNeighborLastDbd[nbrKey]
						dbd_mdata = OspfNeighborLastDbd[nbrKey]

					}
				}
			}
			if !nbrDbPkt.mbit || last_exchange {
				server.logger.Info(fmt.Sprintln("DBD: Exchange done with nbr ", nbrKey.OspfNbrRtrId))
				nbrConf.OspfNbrState = config.NbrLoading
				//nbrConf.ospfNbrLsaReqIndex = server.BuildAndSendLSAReq(nbrConf)
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
					ospfNbrSeqNum:          dbd_mdata.dd_sequence_number,
					isMaster:               nbrConf.isMaster,
					ospfNbrLsaReqIndex:     nbrConf.ospfNbrLsaReqIndex,
				},
				nbrMsgType: NBRUPD,
			}
			server.neighborConfCh <- nbrConfMsg

		case config.NbrLoading:
			server.logger.Info(fmt.Sprintln("DBD: Loading . Nbr ", nbrKey.OspfNbrRtrId))
			isDiscard := server.exchangePacketDiscardCheck(nbrConf, nbrDbPkt)
			if isDiscard {
				server.logger.Info(fmt.Sprintln("NBRDBD:Loading  Discard packet. nbr", nbrKey.OspfNbrRtrId,
					" nbr state ", nbrConf.OspfNbrState))
				//update neighbor to exchange start state and send dbd

				nbrConf.OspfNbrState = config.NbrExchangeStart
				nbrConf.ospf_db_req_list = []ospfLSAHeader{}
				nbrConf.ospf_db_summary_list = []ospfLSAHeader{}
				nbrConf.isMaster = false
				dbd_mdata = server.ConstructAndSendDbdPacket(nbrKey, 0, true, true, true, nbrConf.ospfNbrSeqNum+1, false, false)
			} else {
				/* dbd received in this stage is duplicate.
				    slave - Send the old dbd packet.
					master - discard
				*/
				if nbrConf.isMaster {
					nbrConf.ospfNbrDBDSendCh <- OspfNeighborLastDbd[nbrKey]
				}
				nbrConf.ospfNbrLsaReqIndex = server.BuildAndSendLSAReq(nbrConf)
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
					ospfNbrSeqNum:          dbd_mdata.dd_sequence_number,
					isMaster:               nbrConf.isMaster,
					ospfNbrLsaReqIndex:     nbrConf.ospfNbrLsaReqIndex,
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

/*
func (server *OSPFServer) ProcessLinkStatePktEvent() {
	nbrConf, exists := server.NeighborConfigMap[nbrKey.OspfNbrRtrId]
	if exists{
		switch nbrConf.OspfNbrState {
			case config.NbrTwoway:
			case config.NbrOneway:
			case config.NbrExchangeStart
			return
			case config.NbrExchange:
			case config.NbrLoading:


		}
	}

}
*/
func (server *OSPFServer) ProcessHelloPktEvent() {
	for {
		select {
		case nbrData := <-(server.neighborHelloEventCh):
			server.logger.Info(fmt.Sprintln("NBREVENT: Received hellopkt event for nbrId ", nbrData.RouterId, " two_way", nbrData.TwoWayStatus))
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
							nbrConf.isMaster = false
							//go server.SendDBDPkt(nbrData.RouterId)
						} else {
							dbd_mdata.dd_sequence_number = nbrConf.ospfNbrSeqNum
							dbd_mdata.msbit = true
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
						isMaster:               nbrConf.isMaster,
					},
					nbrMsgType: NBRUPD,
				}
				server.neighborConfCh <- nbrConfMsg

				if send_dbd {
					server.ConstructAndSendDbdPacket(nbrConfMsg.ospfNbrConfKey, 0, true, true, true,
						nbrConf.ospfNbrSeqNum, false, false)
					//nbrConf.ospfNbrDBDSendCh <- dbd_mdata
				}
				server.logger.Info(fmt.Sprintln("NBREVENT: update Nbr ", nbrData.RouterId, "state ", nbrConf.OspfNbrState))

			} else { //neighbor doesnt exist
				var ticker *time.Ticker
				var nbrState config.NbrState
				var dbd_mdata ospfDatabaseDescriptionData
				var send_dbd bool
				server.logger.Info(fmt.Sprintln("NBREVENT: Create new neighbor with id ", nbrData.RouterId))

				go server.SendDBDPkt(nbrData.RouterId)
				if nbrData.TwoWayStatus { // update the state
					startAdjacency := server.adjacancyEstablishementCheck(false, true)
					if startAdjacency {
						nbrState = config.NbrExchangeStart
						//nbrConf.ospfNbrSeqNum = uint32(time.Now().Nanosecond())
						dbd_mdata.dd_sequence_number = uint32(time.Now().Nanosecond())
						// send dbd packets
						ticker = time.NewTicker(time.Second * 10)
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
				/* add the stub entry so that till the update thread updates the data
				valid entry will be present in the map */
				server.NeighborConfigMap[nbrData.RouterId] = nbrConfMsg.ospfNbrEntry
				server.neighborConfCh <- nbrConfMsg
				if send_dbd {
					dbd_mdata.ibit = true
					dbd_mdata.mbit = true
					dbd_mdata.msbit = true

					dbd_mdata.interface_mtu = INTF_MTU_MIN
					dbd_mdata.options = INTF_OPTIONS

					//nbrConfMsg.ospfNbrEntry.ospfNbrDBDSendCh <- dbd_mdata
				}
				server.logger.Info(fmt.Sprintln("NBREVENT: ADD Nbr ", nbrData.RouterId, "state ", nbrState))
			}

		case nbrDbPkt := <-(server.neighborDBDEventCh):
			server.logger.Info(fmt.Sprintln("NBREVENT: DBD received  ", nbrDbPkt))
			server.processDBDEvent(nbrDbPkt.ospfNbrConfKey, nbrDbPkt.ospfNbrDBDData)

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
