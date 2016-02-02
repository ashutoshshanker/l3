package server

import (
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

/*
This file decodes database description packets.as per below format
 0                   1                   2                   3
        0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |   Version #   |       2       |         Packet length         |
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
       |       0       |       0       |    Options    |0|0|0|0|0|I|M|MS
       +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
       |                     DD sequence number                        |
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
*/

/* TODO
remote hardcoding and get it while config.
*/
const INTF_MTU_MIN = 1500

type ospfDatabaseDescriptionData struct {
	options            uint8
	interface_mtu      uint16
	dd_sequence_number uint32
	ibit               bool
	mbit               bool
	msbit              bool
}

func newOspfDatabaseDescriptionData() *ospfDatabaseDescriptionData {
	return &ospfDatabaseDescriptionData{}
}

func decodeDatabaseDescriptionData(data []byte, dbd_data *ospfDatabaseDescriptionData) {
	dbd_data.interface_mtu = binary.BigEndian.Uint16(data[0:2])
	dbd_data.options = data[2]
	dbd_data.dd_sequence_number = binary.BigEndian.Uint32(data[4:8])
	imms_options := data[3]
	dbd_data.ibit = imms_options&0x04 != 0
	dbd_data.mbit = imms_options&0x02 != 0
	dbd_data.msbit = imms_options&0x01 != 0

	fmt.Println("Decoded packet options:", dbd_data.options,
		"IMMS:", dbd_data.ibit, dbd_data.mbit, dbd_data.msbit,
		"seq num:", dbd_data.dd_sequence_number)
}

func encodeDatabaseDescriptionData(dd_data ospfDatabaseDescriptionData) []byte {
	pkt := make([]byte, OSPF_DBD_MIN_SIZE)
	binary.BigEndian.PutUint16(pkt[0:2], dd_data.interface_mtu)
	//pkt[3] = dd_data.options
	pkt[2] = 0x2
	imms := 0x07
	if !dd_data.ibit {
		imms = imms & 0x04
	}
	if !dd_data.mbit {
		imms = imms & 0x02
	}
	if !dd_data.msbit {
		imms = imms & 0x01
	}
	pkt[3] = byte(imms)
	//	pkt[3] = dd_data.ibit | dd_data.mbit | dd_data.msbit
	binary.BigEndian.PutUint32(pkt[4:8], dd_data.dd_sequence_number)
	fmt.Println("data consrtructed  ", pkt)
	return pkt
}

/*
func constructDatabaseDescriptionPaket(intf IntfConf, nbr OspfNeighborEntry) {

}
*/
func (server *OSPFServer) BuildAndSendDBDPkt(intfKey IntfConfKey, ent IntfConf, nbrConf OspfNeighborEntry, dbdData ospfDatabaseDescriptionData) {
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
	ospfPktlen = ospfPktlen + OSPF_DBD_MIN_SIZE

	ospfHdr.pktlen = uint16(ospfPktlen)

	ospfEncHdr := encodeOspfHdr(ospfHdr)
	server.logger.Info(fmt.Sprintln("ospfEncHdr:", ospfEncHdr))
	dbdDataEnc := encodeDatabaseDescriptionData(dbdData)
	server.logger.Info(fmt.Sprintln("DBD Pkt:", dbdDataEnc))

	ospf := append(ospfEncHdr, dbdDataEnc...)
	server.logger.Info(fmt.Sprintln("OSPF DBD:", ospf))
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
	server.logger.Info(fmt.Sprintln("buffer: ", buffer))
	dbdPkt := buffer.Bytes()
	server.logger.Info(fmt.Sprintln("dbdPkt: ", dbdPkt))
	/*
		TODO get retransmit value from INtfconf
	*/
	go func() {
		for t := range nbrConf.ospfNbrDBDTicker.C {
			server.SendDBDPkt(intfKey, dbdPkt)
			server.logger.Info(fmt.Sprintln("Sent DBD at ", t))

		}
	}()

	server.SendDBDPkt(intfKey, dbdPkt)
	//return ospfPkt
}

func (server *OSPFServer) SendDBDPkt(intfKey IntfConfKey, data []byte) {
	server.SendOspfPkt(intfKey, data)

}

func (server *OSPFServer) processRxDbdPkt(data []byte, ospfHdrMd *OspfHdrMetadata, ipHdrMd *IpHdrMetadata, key IntfConfKey) error {
	//ent, _ := server.IntfConfMap[key]
	ospfdbd_data := newOspfDatabaseDescriptionData()
	routerId := convertIPv4ToUint32(ospfHdrMd.routerId)
	/*  TODO check min length
	 */
	decodeDatabaseDescriptionData(data, ospfdbd_data)
	dbdNbrMsg := ospfNeighborDBDMsg{
		ospfNbrConfKey: NeighborConfKey{
			OspfNbrRtrId: routerId,
		},
		ospfNbrDBDData: *ospfdbd_data,
	}
	server.neighborDBDEventCh <- dbdNbrMsg
	return nil
}

/*
func (server *OSPFServer) sendDBDPkt(dbd_pkt ospfDatabaseDescriptionData, nbrKey NeighborConfKey, intfKey IntfConfKey) {

}*/
