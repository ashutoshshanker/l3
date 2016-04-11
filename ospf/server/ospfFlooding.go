package server

import (
	"encoding/binary"
	"fmt"
	"l3/ospf/config"
	"net"
)

const (
	FloodLsa uint8 = LsdbNoAction + 1
)

/* Flood message structure to be sent
for tx LSAUPD channel
*/
type ospfFloodMsg struct {
	nbrKey        uint32
	intfKey       IntfConfKey
	areaId        uint32
	lsType        uint8
	linkid        uint32
	lsOp          uint8  // indicates whether to flood on all interfaces or selective ones.
	pkt           []byte //LSA flood packet received from another neighbor
	summaryUpdMsg summaryLsaUpdMsg
}

var maxAgeLsaMap map[LsaKey][]byte

/*@fn SendSelfOrigLSA
Api is called
When adjacency is established
DR/BDR change

*/
func (server *OSPFServer) SendSelfOrigLSA(areaId uint32, intfKey IntfConfKey) []byte {
	server.logger.Info("Flood: Start flooding as Nbr is in full state")
	lsdbKey := LsdbKey{
		AreaId: areaId,
	}
	intConf, _ := server.IntfConfMap[intfKey]
	ospfLsaPkt := newospfNeighborLSAUpdPkt()
	var lsaEncPkt []byte
	LsaEnc := []byte{}

	selfOrigLsaEnt, exist := server.AreaSelfOrigLsa[lsdbKey]
	if !exist {
		return nil
	}
	pktLen := 0
	total_len := 0
	ospfLsaPkt.no_lsas = 0

	for key, valid := range selfOrigLsaEnt {
		if valid {
			switch key.LSType {
			case RouterLSA:
				entry, _ := server.getRouterLsaFromLsdb(areaId, key)
				LsaEnc = encodeRouterLsa(entry, key)
				checksumOffset := uint16(14)
				checkSum := computeFletcherChecksum(LsaEnc[2:], checksumOffset)
				binary.BigEndian.PutUint16(LsaEnc[16:18], checkSum)
				pktLen = len(LsaEnc)
				binary.BigEndian.PutUint16(LsaEnc[18:20], uint16(pktLen))
				lsaid := convertUint32ToIPv4(key.LSId)
				server.logger.Info(fmt.Sprintln("Flood: router  LSA = ", lsaid))
				ospfLsaPkt.lsa = append(ospfLsaPkt.lsa, LsaEnc...)
				ospfLsaPkt.no_lsas++
				total_len += pktLen

			case NetworkLSA:
				rtr_id := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
				server.logger.Info(fmt.Sprintln("Flood: intfRouterid ", intConf.IfDRtrId, " globalrtrId ", rtr_id))
				if intConf.IfDRtrId == rtr_id {
					server.logger.Info(fmt.Sprintln("Flood: I am DR. Send Nw LSA."))
					entry, _ := server.getNetworkLsaFromLsdb(areaId, key)

					server.logger.Info(fmt.Sprintln("Flood: Network lsa for key ", key, " lsa ", entry))
					LsaEnc = encodeNetworkLsa(entry, key)
					checksumOffset := uint16(14)
					checkSum := computeFletcherChecksum(LsaEnc[2:], checksumOffset)
					binary.BigEndian.PutUint16(LsaEnc[16:18], checkSum)
					pktLen = len(LsaEnc)
					binary.BigEndian.PutUint16(LsaEnc[18:20], uint16(pktLen))
					//server.logger.Info(fmt.Sprintln("Flood: Encoded LSA = ", LsaEnc))
					ospfLsaPkt.lsa = append(ospfLsaPkt.lsa, LsaEnc...)
					ospfLsaPkt.no_lsas++
					total_len += pktLen

				}

			} // end of case
		}
	}

	lsa_pkt_len := total_len + OSPF_NO_OF_LSA_FIELD
	//server.logger.Info(fmt.Sprintln("Flood: Total length ", lsa_pkt_len, "total lsas ", ospfLsaPkt.no_lsas))
	if lsa_pkt_len == OSPF_NO_OF_LSA_FIELD {
		server.logger.Info(fmt.Sprintln("Flood: No LSA to send"))
		return nil
	}
	lsas_enc := make([]byte, 4)

	binary.BigEndian.PutUint32(lsas_enc, ospfLsaPkt.no_lsas)
	lsaEncPkt = append(lsaEncPkt, lsas_enc...)
	lsaEncPkt = append(lsaEncPkt, ospfLsaPkt.lsa...)

	//server.logger.Info(fmt.Sprintln("Flood: LSA pkt with #lsas = ", lsaEncPkt))

	return lsaEncPkt
}

/* @fn processFloodMsg
When new LSA is received on the interfaces
flood is based on different checks
LSAFLOOD - Flood router LSA and n/w LSA.
LSASELFLOOD - Flood on selective interface.
LSAINTF - LSA sent over the interface for LSAREQ
*/

func (server *OSPFServer) processFloodMsg(lsa_data ospfFloodMsg) {

	intConf := server.IntfConfMap[lsa_data.intfKey]
	dstMac := net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x05}
	dstIp := net.IP{224, 0, 0, 5}

	switch lsa_data.lsOp {
	case LSAFLOOD: // flood router LSAs and n/w LSA if DR
		server.logger.Info(fmt.Sprintln("FLOOD: Flood request received from interface key ",
			lsa_data.intfKey, " nbr ", lsa_data.nbrKey))
		for key, intf := range server.IntfConfMap {
			flood_lsa := server.interfaceFloodCheck(key)
			if !flood_lsa {
				continue // dont flood if no nbr is full for this interface
			}
			lsa_upd_pkt := server.SendSelfOrigLSA(lsa_data.areaId, lsa_data.intfKey)
			lsa_pkt_len := len(lsa_upd_pkt)
			if lsa_pkt_len == 0 {
				return
			}
			pkt := server.BuildLsaUpdPkt(key, intf,
				dstMac, dstIp, lsa_pkt_len, lsa_upd_pkt)
			server.SendOspfPkt(key, pkt)
			server.logger.Info(fmt.Sprintln("FLOOD: Nbr FULL intf ", intf.IfIpAddr))
		}

	case LSASELFLOOD: //Flood received LSA on selective interfaces.
		nbrConf := server.NeighborConfigMap[lsa_data.nbrKey]
		rxIntf := server.IntfConfMap[nbrConf.intfConfKey]
		lsid := convertUint32ToIPv4(lsa_data.linkid)
		server.logger.Info(fmt.Sprintln("LSASELFLOOD: Received lsid ", lsid, " lstype ", lsa_data.lsType))
		var lsaEncPkt []byte
		for key, intf := range server.IntfConfMap {
			if intf.IfIpAddr.Equal(rxIntf.IfIpAddr) {
				server.logger.Info(fmt.Sprintln("LSASELFLOOD:Dont flood on rx intf ", rxIntf.IfIpAddr))
				continue // dont flood the LSA on the interface it is received.
			}
			send := server.nbrFloodCheck(lsa_data.nbrKey, key, intf, lsa_data.lsType)
			if send {
				if lsa_data.pkt != nil {
					server.logger.Info(fmt.Sprintln("LSASELFLOOD: Unicast LSA interface ", intf.IfIpAddr, " lsid ", lsid, " lstype ", lsa_data.lsType))
					lsas_enc := make([]byte, 4)
					var no_lsa uint32
					no_lsa = 1
					binary.BigEndian.PutUint32(lsas_enc, no_lsa)
					lsaEncPkt = append(lsaEncPkt, lsas_enc...)
					lsaEncPkt = append(lsaEncPkt, lsa_data.pkt...)
					lsa_pkt_len := len(lsaEncPkt)
					pkt := server.BuildLsaUpdPkt(key, intf,
						dstMac, dstIp, lsa_pkt_len, lsaEncPkt)
					server.SendOspfPkt(key, pkt)
				}
			}
		}
	case LSAINTF: //send the LSA on specific interface for reply to the LSAREQ
		nbrConf, exists := server.NeighborConfigMap[lsa_data.nbrKey]
		if !exists {
			server.logger.Info(fmt.Sprintln("Flood: LSAINTF Neighbor doesnt exist ", lsa_data.nbrKey))
			return
		}
		lsid := convertUint32ToIPv4(lsa_data.linkid)
		var lsaEncPkt []byte
		if lsa_data.pkt != nil {
			lsas_enc := make([]byte, 4)
			var no_lsa uint32
			no_lsa = 1
			binary.BigEndian.PutUint32(lsas_enc, no_lsa)
			lsaEncPkt = append(lsaEncPkt, lsas_enc...)
			lsaEncPkt = append(lsaEncPkt, lsa_data.pkt...)
			lsa_pkt_len := len(lsaEncPkt)
			pkt := server.BuildLsaUpdPkt(nbrConf.intfConfKey, intConf,
				dstMac, dstIp, lsa_pkt_len, lsaEncPkt)
			server.logger.Info(fmt.Sprintln("LSAINTF: Send  LSA to interface ", intConf.IfIpAddr,
				" lsid ", lsid, " lstype ", lsa_data.lsType))
			server.SendOspfPkt(nbrConf.intfConfKey, pkt)

		}

	case LSASUMMARYFLOOD:
		server.logger.Info(fmt.Sprintln("Flood: Summary LSA flood msg received."))
		server.processSummaryLSAFlood(lsa_data.summaryUpdMsg)
	case LSAAGE: // Flood aged LSAs
		server.constructAndSendLsaAgeFlood()

	}
}

func (server *OSPFServer) constructAndSendLsaAgeFlood() {
	dstMac := net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x05}
	dstIp := net.IP{224, 0, 0, 5}
	lsas_enc := make([]byte, 4)
	var lsaEncPkt []byte
	var lsasWithHeader []byte
	var no_lsa uint32
	no_lsa = 0
	total_len := 0
	for lsaKey, lsaPkt := range maxAgeLsaMap {
		if lsaPkt != nil {
			no_lsa++
			checksumOffset := uint16(14)
			checkSum := computeFletcherChecksum(lsaPkt[2:], checksumOffset)
			binary.BigEndian.PutUint16(lsaPkt[16:18], checkSum)
			pktLen := len(lsaPkt)
			binary.BigEndian.PutUint16(lsaPkt[18:20], uint16(pktLen))
			lsaEncPkt = append(lsaEncPkt, lsaPkt...)
			total_len += pktLen
			server.logger.Info(fmt.Sprintln("FLUSH: Added to flush list lsakey",
				lsaKey.AdvRouter, lsaKey.LSId, lsaKey.LSId))
		}
		msg := maxAgeLsaMsg{
			lsaKey:   lsaKey,
			msg_type: delMaxAgeLsa,
		}
		server.maxAgeLsaCh <- msg
		//delete(maxAgeLsaMap, lsaKey)
	}
	lsa_pkt_len := total_len + OSPF_NO_OF_LSA_FIELD
	if lsa_pkt_len == OSPF_NO_OF_LSA_FIELD {
		return
	}
	binary.BigEndian.PutUint32(lsas_enc, no_lsa)
	lsasWithHeader = append(lsasWithHeader, lsas_enc...)
	lsasWithHeader = append(lsasWithHeader, lsaEncPkt...)

	/* flood on all eligible interfaces */
	for key, intConf := range server.IntfConfMap {
		server.logger.Info(fmt.Sprintln("FLUSH: Send flush message ", intConf.IfIpAddr))
		pkt := server.BuildLsaUpdPkt(key, intConf,
			dstMac, dstIp, lsa_pkt_len, lsasWithHeader)
		server.SendOspfPkt(key, pkt)
	}

}

/* @fn interfaceFloodCheck
Check if we need to flood the LSA on the interface
*/
func (server *OSPFServer) nbrFloodCheck(nbrKey uint32, key IntfConfKey, intf IntfConf, lsType uint8) bool {
	/* Check neighbor state */
	flood_check := true
	nbrConf := server.NeighborConfigMap[nbrKey]
	//rtrid := convertIPv4ToUint32(server.ospfGlobalConf.RouterId)
	if nbrConf.intfConfKey == key && nbrConf.isDRBDR && lsType != Summary3LSA && lsType != Summary4LSA {
		server.logger.Info(fmt.Sprintln("IF FLOOD: Nbr is DR/BDR.   flood on this interface . nbr - ", nbrKey, nbrConf.OspfNbrIPAddr))
		return false
	}
	flood_check = server.interfaceFloodCheck(key)
	return flood_check
}

func (server *OSPFServer) interfaceFloodCheck(key IntfConfKey) bool {
	flood_check := false
	nbrData, exist := ospfIntfToNbrMap[key]
	if !exist {
		server.logger.Info(fmt.Sprintln("FLOOD: Intf to nbr map doesnt exist."))
		return false
	}
	if nbrData.nbrList != nil {
		for index := range nbrData.nbrList {
			nbrId := nbrData.nbrList[index]
			nbrConf := server.NeighborConfigMap[nbrId]
			if nbrConf.OspfNbrState < config.NbrExchange {
				server.logger.Info(fmt.Sprintln("FLOOD: Nbr < exchange . ", nbrConf.OspfNbrIPAddr))
				flood_check = false
				continue
			}
			flood_check = true
			/* TODO - add check if nbrstate is loading - check its retransmission list
			   add LSA to the adjacency list of neighbor with FULL state.*/
		}
	} else {
		server.logger.Info(fmt.Sprintln("FLOOD: nbr list is null for interface ", key.IPAddr))
	}
	return flood_check
}

/*
@fn processSummaryLSAFlood
This API takes care of flooding new summary LSAs that is added in the LSDB
*/
func (server *OSPFServer) processSummaryLSAFlood(msg summaryLsaUpdMsg) {
	ospfLsaPkt := newospfNeighborLSAUpdPkt()
	var lsaEncPkt []byte
	LsaEnc := []byte{}

	ospfLsaPkt.no_lsas = 0

	count := len(msg.lsa_data)
	server.logger.Info(fmt.Sprintln("Summary: Received msg. Count ", count))
	for i := 0; i < count; i++ {
		lsamdata := msg.lsa_data[i]
		server.logger.Info(fmt.Sprintln("Summary: Start flooding. Area ",
			lsamdata.areaId, " lsa ", lsamdata.lsaKey))
		LsaEnc = server.encodeSummaryLsa(lsamdata.areaId, lsamdata.lsaKey)
		//ospfLsaPkt.lsa = append(ospfLsaPkt.lsa, LsaEnc...)
		no_lsas := uint32(1)
		lsas_enc := make([]byte, 4)
		binary.BigEndian.PutUint32(lsas_enc, no_lsas)
		lsaEncPkt = append(lsaEncPkt, lsas_enc...)
		lsaEncPkt = append(lsaEncPkt, LsaEnc...)
		server.logger.Info(fmt.Sprintln("SUMMARY: Send for flooding ",
			lsamdata.areaId, " adv_router ", lsamdata.lsaKey.AdvRouter, " lsid ",
			lsamdata.lsaKey.LSId))
		server.floodSummaryLsa(lsaEncPkt, lsamdata.areaId)
	}

}

func (server *OSPFServer) encodeSummaryLsa(areaid uint32, lsakey LsaKey) []byte {
	entry, ret := server.getSummaryLsaFromLsdb(areaid, lsakey)
	if ret == LsdbEntryNotFound {
		server.logger.Info(fmt.Sprintln("Summary LSA: Lsa not found . Area",
			areaid, " LSA key ", lsakey))
		return nil
	}
	LsaEnc := encodeSummaryLsa(entry, lsakey)
	pktLen := len(LsaEnc)
	checksumOffset := uint16(14)
	checkSum := computeFletcherChecksum(LsaEnc[2:], checksumOffset)
	binary.BigEndian.PutUint16(LsaEnc[16:18], checkSum)
	binary.BigEndian.PutUint16(LsaEnc[18:20], uint16(pktLen))
	return LsaEnc

}

func (server *OSPFServer) floodSummaryLsa(pkt []byte, areaid uint32) {
	dstMac := net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x05}
	dstIp := net.IP{224, 0, 0, 5}
	if server.ospfGlobalConf.isABR {
		/*TODO - add the logic later . */
		server.logger.Info("Recieved flood summary message")
	} else {
		for key, intf := range server.IntfConfMap {
			ifArea := convertIPv4ToUint32(intf.IfAreaId)
			if ifArea == areaid {
				// flood to your own area
				nbrMdata, ok := ospfIntfToNbrMap[key]
				if ok && len(nbrMdata.nbrList) > 0 {
					send_pkt := server.BuildLsaUpdPkt(key, intf, dstMac, dstIp, len(pkt), pkt)
					server.logger.Info(fmt.Sprintln("SUMMARY: Send  LSA to interface ", intf.IfIpAddr, " area ", areaid))
					server.SendOspfPkt(key, send_pkt)
				}
			}
		}
	}
}

/*
func (server *OSPFServer) ProcessOspfFlood(nbrKey uint32) {

}*/
