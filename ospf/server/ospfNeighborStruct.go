package server

import (
	"encoding/binary"
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

type LsaOp uint8

const (
	LSAFLOOD    = 0 // flood when FULL state reached
	LSASELFLOOD = 1 // flood for received LSA
)

type NeighborConfKey struct {
	OspfNbrRtrId uint32
}

var INVALID_NEIGHBOR_CONF_KEY uint32
var neighborBulkSlice []NeighborConfKey

type OspfNeighborEntry struct {
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
	nbrEvent               config.NbrEvent
	isSeqNumUpdate         bool
	isMaster               bool
	ospfNbrDBDTickerCh     *time.Ticker

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

type ospfFloodMsg struct {
	key    uint32
	areaId uint32
	lsType uint8
	linkid uint32
	lsOp   uint8
	pkt    []byte //LSA flood packet received from another neighbor
}

type ospfNbrMdata struct {
	isDR    bool
	areaId  uint32
	intf    IntfConfKey
	nbrList []uint32
}

func newospfNbrMdata() *ospfNbrMdata {
	return &ospfNbrMdata{}
}

/*
	Global structures for Neighbor
*/
var OspfNeighborLastDbd map[NeighborConfKey]ospfDatabaseDescriptionData
var ospfNeighborIPToMAC map[uint32]net.HardwareAddr

/* neighbor lists each indexed by neighbor router id. */
var ospfNeighborRequest_list map[uint32][]*ospfNeighborReq
var ospfNeighborDBSummary_list map[uint32][]*ospfNeighborDBSummary
var ospfNeighborRetx_list map[uint32][]*ospfNeighborRetx

/* List of Neighbors per interface instance */
var ospfIntfToNbrMap map[IntfConfKey]ospfNbrMdata

func (server *OSPFServer) InitNeighborStateMachine() {

	server.neighborBulkSlice = []uint32{}
	INVALID_NEIGHBOR_CONF_KEY = 0
	OspfNeighborLastDbd = make(map[NeighborConfKey]ospfDatabaseDescriptionData)
	ospfNeighborIPToMAC = make(map[uint32]net.HardwareAddr)
	ospfIntfToNbrMap = make(map[IntfConfKey]ospfNbrMdata)
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
			//server.logger.Info(fmt.Sprintln("Update neighbor conf.  received"))
			if nbrMsg.nbrMsgType == NBRUPD {
				nbrConf = server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId]
			}
			nbrConf.OspfNbrState = nbrMsg.ospfNbrEntry.OspfNbrState
			nbrConf.OspfNbrDeadTimer = nbrMsg.ospfNbrEntry.OspfNbrDeadTimer
			nbrConf.OspfNbrInactivityTimer = time.Now()
			if nbrMsg.ospfNbrEntry.isSeqNumUpdate {
				nbrConf.ospfNbrSeqNum = nbrMsg.ospfNbrEntry.ospfNbrSeqNum
			}
			nbrConf.ospfNbrDBDTickerCh = nbrMsg.ospfNbrEntry.ospfNbrDBDTickerCh
			nbrConf.isMaster = nbrMsg.ospfNbrEntry.isMaster
			nbrConf.ospfNbrLsaReqIndex = nbrMsg.ospfNbrEntry.ospfNbrLsaReqIndex
			nbrConf.nbrEvent = nbrMsg.ospfNbrEntry.nbrEvent

			if nbrMsg.nbrMsgType == NBRADD {
				nbrConf.OspfNbrIPAddr = nbrMsg.ospfNbrEntry.OspfNbrIPAddr
				nbrConf.OspfRtrPrio = nbrMsg.ospfNbrEntry.OspfRtrPrio
				nbrConf.intfConfKey = nbrMsg.ospfNbrEntry.intfConfKey
				nbrConf.OspfNbrOptions = 0
				server.neighborBulkSlice = append(server.neighborBulkSlice, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				nbrConf.req_list_mutex = &sync.Mutex{}
				nbrConf.db_summary_list_mutex = &sync.Mutex{}
				nbrConf.retx_list_mutex = &sync.Mutex{}
				updateLSALists(nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				if nbrMsg.ospfNbrEntry.OspfNbrState >= config.NbrTwoWay {
					server.ConstructAndSendDbdPacket(nbrMsg.ospfNbrConfKey, true, true, true,
						INTF_OPTIONS, uint32(time.Now().Nanosecond()), false, false)
					nbrConf.OspfNbrState = config.NbrExchangeStart
					nbrConf.nbrEvent = config.Nbr2WayReceived
				}
				server.neighborDeadTimerEvent(nbrMsg.ospfNbrConfKey)

			}

			if nbrMsg.nbrMsgType == NBRUPD {
				server.NeighborConfigMap[nbrMsg.ospfNbrConfKey.OspfNbrRtrId] = nbrConf
				nbrConf.NbrDeadTimer.Stop()
				nbrConf.NbrDeadTimer.Reset(nbrMsg.ospfNbrEntry.OspfNbrDeadTimer)
				/*server.logger.Info(fmt.Sprintln("UPDATE neighbor with nbr id - ",
				nbrMsg.ospfNbrConfKey.OspfNbrRtrId)) */
			}
			if nbrMsg.nbrMsgType == NBRDEL {
				server.neighborBulkSlice = append(server.neighborBulkSlice, INVALID_NEIGHBOR_CONF_KEY)
				delete(server.NeighborConfigMap, nbrMsg.ospfNbrConfKey.OspfNbrRtrId)
				server.logger.Info(fmt.Sprintln("DELETE neighbor with nbr id - ",
					nbrMsg.ospfNbrConfKey.OspfNbrRtrId))
			}

			//server.logger.Info(fmt.Sprintln("NBR UPDATE: Nbr , seq_no ", nbrMsg.ospfNbrConfKey.OspfNbrRtrId, nbrConf.ospfNbrSeqNum))
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

func (server *OSPFServer) initNeighborMdata(intf IntfConfKey) {
	nbrMdata := newospfNbrMdata()
	nbrMdata.nbrList = []uint32{}
	nbrMdata.intf = intf
	ospfIntfToNbrMap[intf] = *nbrMdata
}

func (server *OSPFServer) updateNeighborMdata(intf IntfConfKey, nbr uint32) {
	nbrMdata, exists := ospfIntfToNbrMap[intf]
	intfData := server.IntfConfMap[intf]
	if !exists {
		server.initNeighborMdata(intf)
		nbrMdata = ospfIntfToNbrMap[intf]
	}
	nbrMdata.areaId = binary.BigEndian.Uint32(intfData.IfAreaId)
	nbrMdata.isDR = bytesEqual(intfData.IfDRIp, intfData.IfIpAddr.To4())
	for inst := range nbrMdata.nbrList {
		if nbrMdata.nbrList[inst] == nbr {
			// nbr already exist no need to add.
			return
		}
	}
	nbrMdata.nbrList = append(nbrMdata.nbrList, nbr)
	ospfIntfToNbrMap[intf] = nbrMdata
}

func (server *OSPFServer) sendLsdbToNeighborEvent(key uint32, areaId uint32, lsType uint8,
	linkId uint32, op uint8) {
	_, exists := server.NeighborConfigMap[key]
	if !exists {
		server.logger.Info(fmt.Sprintln("Nbr-LSDB: Failed to get nbr instance key ", key))
		return
	}
	msg := ospfFloodMsg{
		key:    key,
		areaId: areaId,
		lsType: lsType,
		linkid: linkId,
		lsOp:   op,
	}
	server.ospfNbrLsaUpdSendCh <- msg
	//server.logger.Info("Send flood data to Tx thread")
}

func (server *OSPFServer) calculateDBLsaAttach(nbrKey NeighborConfKey, nbrConf OspfNeighborEntry) (last_exchange bool, lsa_attach uint8) {
	last_exchange = true
	lsa_attach = 0

	max_lsa_headers := calculateMaxLsaHeaders()
	db_list := ospfNeighborDBSummary_list[nbrKey.OspfNbrRtrId]
	slice_len := len(db_list)
	server.logger.Info(fmt.Sprintln("DBD: slice_len ", slice_len, "max_lsa_header ", max_lsa_headers,
		"nbrConf.lsa_index ", nbrConf.ospfNbrLsaIndex))
	if slice_len == int(nbrConf.ospfNbrLsaIndex) {
		return
	}
	if max_lsa_headers > (uint8(slice_len) - uint8(nbrConf.ospfNbrLsaIndex)) {
		lsa_attach = uint8(slice_len) - uint8(nbrConf.ospfNbrLsaIndex)
	} else {
		lsa_attach = max_lsa_headers
	}
	if (nbrConf.ospfNbrLsaIndex + lsa_attach) >= uint8(slice_len) {
		// the last slice in the list being sent
		server.logger.Info(fmt.Sprintln("DBD:  Send the last dd packet with nbr/state ", nbrKey.OspfNbrRtrId, nbrConf.OspfNbrState))
		last_exchange = true
	}
	return last_exchange, 0
}
