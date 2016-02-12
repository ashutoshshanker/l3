package server

import (
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
	ospfNbrLsaSendCh       chan []ospfLSAReq
	ospf_db_summary_list   []ospfLSAHeader
	ospf_db_req_list       []ospfLSAHeader
	ospfNbrDBDStopCh       chan bool
	ospfNbrLsaIndex        uint8 // db_summary list index
	ospfNbrLsaReqIndex     uint8 //req_list index
}

var OspfNeighborLastDbd map[NeighborConfKey]ospfDatabaseDescriptionData

type ospfNeighborConfMsg struct {
	ospfNbrConfKey NeighborConfKey
	ospfNbrEntry   OspfNeighborEntry
	nbrMsgType     NbrMsgType
}

type ospfNeighborDBDMsg struct {
	ospfNbrConfKey NeighborConfKey
	ospfNbrDBDData ospfDatabaseDescriptionData
}

func (server *OSPFServer) InitNeighborStateMachine() {

	server.neighborBulkSlice = []uint32{}
	INVALID_NEIGHBOR_CONF_KEY = 0
	OspfNeighborLastDbd = make(map[NeighborConfKey]ospfDatabaseDescriptionData)

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
