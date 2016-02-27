package server

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	FloodLsa uint8 = LsdbNoAction + 1
)

func (server *OSPFServer) SendRouterLsa(areaId uint32, nbrConf OspfNeighborEntry) {
	lsdbKey := LsdbKey{
		AreaId: areaId,
	}
	ospfLsaPkt := newospfNeighborLSAUpdPkt()
	var lsaEncPkt []byte
	LsaEnc := []byte{}

	intConf := server.IntfConfMap[nbrConf.intfConfKey]

	lsDbEnt, exists := server.AreaLsdb[lsdbKey]
	if !exists {
		server.logger.Info(fmt.Sprintln("Flood: Area lsdb doesnt exist for area id ", areaId))
		return
	}

	pktLen := 0
	total_len := 0
	ospfLsaPkt.no_lsas = 0

	for key, entry := range lsDbEnt.RouterLsaMap {
		server.logger.Info(fmt.Sprintln("Flood: Add lsa for key", key, " lsa ", entry))
		LsaEnc = encodeRouterLsa(entry, key)
		checksumOffset := uint16(14)
		checkSum := computeFletcherChecksum(LsaEnc[2:], checksumOffset)
		binary.BigEndian.PutUint16(LsaEnc[16:18], checkSum)
		pktLen = len(LsaEnc)
		binary.BigEndian.PutUint16(LsaEnc[18:20], uint16(pktLen))
		server.logger.Info(fmt.Sprintln("Flood: Encoded LSA = ", LsaEnc))
		ospfLsaPkt.lsa = append(ospfLsaPkt.lsa, LsaEnc...)
		ospfLsaPkt.no_lsas++
		total_len += pktLen
	}

	/* attach network LSA if I am DR. */

	rtr_id := binary.BigEndian.Uint32(server.ospfGlobalConf.RouterId)
	server.logger.Info(fmt.Sprintln("Flood: rtr_id ", rtr_id, " intConf.IfDRtrId ", intConf.IfDRtrId))
	if intConf.IfDRtrId == rtr_id {
		server.logger.Info(fmt.Sprintln("Flood: I am DR. Send Nw LSA."))
		for key, entry := range lsDbEnt.NetworkLsaMap {
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
	}

	lsa_pkt_len := total_len + OSPA_NO_OF_LSA_FIELD
	server.logger.Info(fmt.Sprintln("Flood: Total length ", lsa_pkt_len, "total lsas ", ospfLsaPkt.no_lsas))
	//lsaEncPkt = make([]byte, lsa_pkt_len)
	if lsa_pkt_len == OSPA_NO_OF_LSA_FIELD {
		server.logger.Info(fmt.Sprintln("Flood: No LSA to send"))
		return
	}
	lsas_enc := make([]byte, 4)

	binary.BigEndian.PutUint32(lsas_enc, ospfLsaPkt.no_lsas)
	lsaEncPkt = append(lsaEncPkt, lsas_enc...)
	lsaEncPkt = append(lsaEncPkt, ospfLsaPkt.lsa...)

	server.logger.Info(fmt.Sprintln("Flood: LSA pkt with #lsas = ", lsaEncPkt))
	dstMAC := net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x05}
	dstIp := net.IP{224, 0, 0, 5}

	pkt := server.BuildLsaUpdPkt(nbrConf.intfConfKey, intConf,
		nbrConf, dstMAC, dstIp, lsa_pkt_len, lsaEncPkt)
	server.logger.Info(fmt.Sprintln("Flood : LSA upd packet = ", lsaEncPkt))
	/* send the lsa update packet */
	nbrConf.ospfNbrLsaUpdSendCh <- pkt
}

/*
func (server *OSPFServer) ProcessOspfFlood(nbrKey uint32) {

}*/
