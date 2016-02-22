package server

import (
	"fmt"
	"l3/ospf/config"
	"net"
	"sync"
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
	ospfNbrLsaSendCh       chan []ospfLSAReq
	ospfNbrLsaUpdSendCh    chan []byte
	ospfNbrDBDStopCh       chan bool
	ospfNbrLsaIndex        uint8       // db_summary list index
	ospfNbrLsaReqIndex     uint8       //req_list index
	ospfNeighborLsaRxTimer *time.Timer // retx interval timer
	req_list_mutex         *sync.Mutex
	db_summary_list_mutex  *sync.Mutex
	retx_list_mutex        *sync.Mutex
}

/* LSA lists */
type ospfNeighborReq struct {
	lsa_headers ospfLSAHeader
	valid       bool // entry is valid or not
}

func newospfNeighborReq() *ospfNeighborReq {
	return &ospfNeighborReq{}
}

type ospfNeighborDBSummary struct {
	lsa_headers ospfLSAHeader
	valid       bool
}

func newospfNeighborDBSummary() *ospfNeighborDBSummary {
	return &ospfNeighborDBSummary{}
}

type ospfNeighborRetx struct {
	lsa_headers ospfLSAHeader
	valid       bool
}

func newospfNeighborRetx() *ospfNeighborRetx {
	return &ospfNeighborRetx{}
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

var OspfNeighborLastDbd map[NeighborConfKey]ospfDatabaseDescriptionData
var ospfNeighborIPToMAC map[uint32]net.HardwareAddr
var ospfNeighborRequest_list map[uint32][]*ospfNeighborReq
var ospfNeighborDBSummary_list map[uint32][]*ospfNeighborDBSummary
var ospfNeighborRetx_list map[uint32][]*ospfNeighborRetx

func (server *OSPFServer) InitNeighborStateMachine() {

	server.neighborBulkSlice = []uint32{}
	INVALID_NEIGHBOR_CONF_KEY = 0
	OspfNeighborLastDbd = make(map[NeighborConfKey]ospfDatabaseDescriptionData)
	ospfNeighborIPToMAC = make(map[uint32]net.HardwareAddr)
	ospfNeighborRequest_list = make(map[uint32][]*ospfNeighborReq)
	ospfNeighborDBSummary_list = make(map[uint32][]*ospfNeighborDBSummary)
	ospfNeighborRetx_list = make(map[uint32][]*ospfNeighborRetx)
	go server.refreshNeighborSlice()
	server.logger.Info("NBRINIT: Neighbor FSM init done..")
}

func calculateMaxLsaHeaders() (max_headers uint8) {
	rem := INTF_MTU_MIN - (OSPF_DBD_MIN_SIZE + OSPF_HEADER_SIZE)
	max_headers = uint8(rem / OSPF_LSA_HEADER_SIZE)
	return max_headers
}

func calculateMaxLsaReq() (max_req uint8) {
	rem := INTF_MTU_MIN - OSPF_HEADER_SIZE
	max_req = uint8(rem / OSPF_LSA_REQ_SIZE)
	return max_req
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
				nbrConf.ospfNbrDBDStopCh = make(chan bool)
				nbrConf.ospfNbrLsaSendCh = make(chan []ospfLSAReq)
				nbrConf.ospfNbrLsaUpdSendCh = make(chan []byte)
				nbrConf.req_list_mutex = &sync.Mutex{}
				nbrConf.db_summary_list_mutex = &sync.Mutex{}
				nbrConf.retx_list_mutex = &sync.Mutex{}
				updateLSALists(nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				server.neighborDeadTimerEvent(nbrMsg.ospfNbrConfKey)
				if nbrMsg.ospfNbrEntry.OspfNbrState >= config.NbrTwoWay {
					server.ConstructAndSendDbdPacket(nbrMsg.ospfNbrConfKey, 0, true, true, true, uint32(time.Now().Nanosecond()), false, false)
				}
			}

			server.logger.Info(fmt.Sprintln("Updated neighbor with nbr id - ",
				nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			if nbrMsg.nbrMsgType == NBRUPD {
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				nbrConf.NbrDeadTimer.Stop()
				nbrConf.NbrDeadTimer.Reset(nbrMsg.ospfNbrEntry.OspfNbrDeadTimer)
				server.logger.Info(fmt.Sprintln("UPDATE neighbor with nbr id - ",
					nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			}
			if nbrMsg.nbrMsgType == NBRDEL {
				server.neighborBulkSlice = append(server.neighborBulkSlice, INVALID_NEIGHBOR_CONF_KEY)
				nbrConf.ospfNbrDBDStopCh <- true
				delete(server.NeighborConfigMap, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
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

func updateLSALists(id uint32) {
	ospfNeighborRequest_list[id] = []*ospfNeighborReq{}
	ospfNeighborDBSummary_list[id] = []*ospfNeighborDBSummary{}
	ospfNeighborRetx_list[id] = []*ospfNeighborRetx{}
}

func (server *OSPFServer) sendNeighborConf(nbrKey uint32, nbr OspfNeighborEntry, op NbrMsgType) {

	nbrConfMsg := ospfNeighborConfMsg{
		ospfNbrConfKey: NeighborConfKey{
			OspfNbrRtrId: nbrKey,
		},
		ospfNbrEntry: OspfNeighborEntry{
			OspfNbrIPAddr:          nbr.OspfNbrIPAddr,
			OspfRtrPrio:            nbr.OspfRtrPrio,
			intfConfKey:            nbr.intfConfKey,
			OspfNbrOptions:         0,
			OspfNbrState:           nbr.OspfNbrState,
			OspfNbrInactivityTimer: time.Now(),
			OspfNbrDeadTimer:       nbr.OspfNbrDeadTimer,
		},
		nbrMsgType: op,
	}

	server.neighborConfCh <- nbrConfMsg
}

func (server *OSPFServer) neighborExist(nbrKey uint32) bool {
	_, exists := server.NeighborConfigMap[nbrKey]
	if exists {
		return true
	}
	return false
}
