package server

import (
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

/*
LSA request
 0                   1                   2                   3
        0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |   Version #   |       3       |         Packet length         |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                          Router ID                            |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                           Area ID                             |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |           Checksum            |             AuType            |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                       Authentication                          |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                       Authentication                          |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                          LS type                              |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                       Link State ID                           |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                     Advertising Router                        |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                              ...                              |
*/
type ospfLSAReq struct {
	ls_type       uint32
	link_state_id uint32
	adv_router_id uint32
}

/*
type ospfLSAUpd struct {
	ospf_lsas []byte
}

func newOspfLsaUpd() *ospfLSAUpd {
	return &ospfLSAUpd{}
}
*/
func encodeLSAReqPkt(lsa_data []ospfLSAReq) []byte {
	pkt := make([]byte, len(lsa_data)*3*8)
	for i := 0; i < len(lsa_data); i++ {
		binary.BigEndian.PutUint32(pkt[i:i+4], lsa_data[i].ls_type)
		binary.BigEndian.PutUint32(pkt[i:i+4], lsa_data[i].link_state_id)
		binary.BigEndian.PutUint32(pkt[i:i+4], lsa_data[i].adv_router_id)
	}
	return pkt
}
func decodeLSAReq(data []byte) (lsa_req ospfLSAReq) {
	lsa_req.ls_type = binary.BigEndian.Uint32(data[0:4])
	lsa_req.link_state_id = binary.BigEndian.Uint32(data[4:8])
	lsa_req.adv_router_id = binary.BigEndian.Uint32(data[8:12])
	return lsa_req
}

func decodeLSAReqPkt(data []byte, pktlen uint16) []ospfLSAReq {
	no_of_lsa := int(pktlen / 3)
	lsa_req_pkt := []ospfLSAReq{}
	for i := 0; i < no_of_lsa; i++ {
		lsa_req := decodeLSAReq(data[i : i+3])
		lsa_req_pkt = append(lsa_req_pkt, lsa_req)
	}
	return lsa_req_pkt
}

func (server *OSPFServer) BuildLSAReqPkt(intfKey IntfConfKey, ent IntfConf,
	nbrConf OspfNeighborEntry, lsa_req_pkt []ospfLSAReq) (data []byte) {
	ospfHdr := OSPFHeader{
		ver:      OSPF_VERSION_2,
		pktType:  uint8(DBDescriptionType),
		pktlen:   0,
		routerId: server.ospfGlobalConf.RouterId,
		areaId:   ent.IfAreaId,
		chksum:   0,
		authType: ent.IfAuthType,
		//authKey:        ent.IfAuthKey,
	}

	ospfPktlen := OSPF_HEADER_SIZE
	ospfPktlen = ospfPktlen + len(lsa_req_pkt)

	ospfHdr.pktlen = uint16(ospfPktlen)

	ospfEncHdr := encodeOspfHdr(ospfHdr)
	server.logger.Info(fmt.Sprintln("ospfEncHdr:", ospfEncHdr))
	lsaDataEnc := encodeLSAReqPkt(lsa_req_pkt)
	server.logger.Info(fmt.Sprintln("lsa Pkt:", lsaDataEnc))

	ospf := append(ospfEncHdr, lsaDataEnc...)
	server.logger.Info(fmt.Sprintln("OSPF LSA REQ:", ospf))
	csum := computeCheckSum(ospf)
	binary.BigEndian.PutUint16(ospf[12:14], csum)
	copy(ospf[16:24], ent.IfAuthKey)

	ipPktlen := IP_HEADER_MIN_LEN + ospfHdr.pktlen
	ipLayer := layers.IPv4{
		Version:  uint8(4),
		IHL:      uint8(IP_HEADER_MIN_LEN),
		TOS:      uint8(0xc0),
		Length:   uint16(ipPktlen),
		TTL:      uint8(1),
		Protocol: layers.IPProtocol(OSPF_PROTO_ID),
		SrcIP:    ent.IfIpAddr,
		DstIP:    net.IP{30, 0, 1, 3},
	}

	ethLayer := layers.Ethernet{
		SrcMAC:       ent.IfMacAddr,
		DstMAC:       net.HardwareAddr{0x62, 0x01, 0x0b, 0xca, 0x68, 0x90},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	gopacket.SerializeLayers(buffer, options, &ethLayer, &ipLayer, gopacket.Payload(ospf))
	lsaPkt := buffer.Bytes()
	server.logger.Info(fmt.Sprintln("lsaPkt: ", lsaPkt))

	return lsaPkt

}

func (server *OSPFServer) BuildAndSendLSAReq(nbrConf OspfNeighborEntry) (curr_index uint8) {
	/* calculate max no of requests that can be added
	for req packet */

	var add_items uint8
	var lsa_req []ospfLSAReq
	lsa_req = []ospfLSAReq{}
	var req ospfLSAReq
	var i uint8
	req_list_items := uint8(len(nbrConf.ospf_db_req_list)) - nbrConf.ospfNbrLsaReqIndex
	max_req := calculateMaxLsaReq()
	if max_req > req_list_items {
		add_items = req_list_items
		nbrConf.ospfNbrLsaReqIndex = uint8(len(nbrConf.ospf_db_req_list))

	} else {
		add_items = uint8(max_req)
		nbrConf.ospfNbrLsaReqIndex += max_req
	}
	index := nbrConf.ospfNbrLsaReqIndex
	for i = 0; i < add_items; i++ {
		req.ls_type = uint32(nbrConf.ospf_db_req_list[uint8(i)+nbrConf.ospfNbrLsaReqIndex].ls_type)
		req.link_state_id = nbrConf.ospf_db_req_list[uint8(i)+nbrConf.ospfNbrLsaReqIndex].link_state_id
		req.adv_router_id = nbrConf.ospf_db_req_list[uint8(i)+nbrConf.ospfNbrLsaReqIndex].adv_router_id
		lsa_req = append(lsa_req, req)
	}
	server.logger.Info(fmt.Sprintln("LSA request: total requests out, req_list_len, current req_list_index ", add_items, len(nbrConf.ospf_db_req_list), nbrConf.ospfNbrLsaReqIndex))
	server.logger.Info(fmt.Sprintln("LSA request: lsa_req", lsa_req))

	nbrConf.ospfNbrLsaSendCh <- lsa_req
	index += add_items
	return index
}

func (server *OSPFServer) ProcessLsaUpdPkt(data []byte, ospfHdrMd *OspfHdrMetadata, ipHdrMd *IpHdrMetadata, key IntfConfKey) error {

	routerId := convertIPv4ToUint32(ospfHdrMd.routerId)
	pktlen := ospfHdrMd.pktlen / 8
	/*  call lsdb API */
	server.logger.Info(fmt.Sprintln("LSA update: Received LSA update with router_id , lentgh ", routerId, pktlen))
	server.logger.Info(fmt.Sprintln("LSA update: pkt byte[]: ", data))
	return nil
}

/* link state ACK packet
0                   1                   2                   3
       0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |   Version #   |       5       |         Packet length         |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                          Router ID                            |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                           Area ID                             |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |           Checksum            |             AuType            |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                       Authentication                          |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                       Authentication                          |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                                                               |
      +-                                                             -+
      |                             A                                 |
      +-                 Link State Advertisement                    -+
      |                           Header                              |
      +-                                                             -+
      |                                                               |
      +-                                                             -+
      |                                                               |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                              ...                              |
*/

/*
LSA update packet
   0                   1                   2                   3
        0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |   Version #   |       4       | d        Packet length         |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                          Router ID                            |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                           Area ID                             |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |           Checksum            |             AuType            |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                       Authentication                          |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                       Authentication                          |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                            # LSAs                             |
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                                                               |
       +-                                                            +-+
       |                             LSAs                              |
       +-                                                            +-+
       |                              ...                              |

*/
func (server *OSPFServer) processRxLsaUpdPkt(data []byte, ospfHdrMd *OspfHdrMetadata, ipHdrMd *IpHdrMetadata, key IntfConfKey) error {
	server.logger.Info("Receievd LSA update packet. ")
	return nil
}
