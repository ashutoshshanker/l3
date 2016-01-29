package server

import (
	"encoding/binary"
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
const INTF_MTU_MIN = 1600

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

	/*fmt.Println("Decoded packet options:", dbd_data.options,
	"IMMS:", dbd_data.ibit, dbd_data.mbit, dbd_data.msbit,
	"seq num:", dbd_data.dd_sequence_number) */
}

func encodeDatabaseDescriptionData(dd_data ospfDatabaseDescriptionData) []byte {
	pkt := make([]byte, INTF_MTU_MIN)
	binary.BigEndian.PutUint16(pkt[0:2], dd_data.interface_mtu)
	pkt[2] = dd_data.options
	//	pkt[3] = dd_data.ibit | dd_data.mbit | dd_data.msbit
	binary.BigEndian.PutUint32(pkt[4:8], dd_data.dd_sequence_number)
	//fmt.Println("data consrtructed  ", pkt)
	return pkt
}

/*
func constructDatabaseDescriptionPaket(intf IntfConf, nbr OspfNeighborEntry) {

}
*/

func (server *OSPFServer) processRxDbdPkt(data []byte, ospfHdrMd *OspfHdrMetadata, ipHdrMd *IpHdrMetadata, key IntfConfKey) error {
	//ent, _ := server.IntfConfMap[key]
	ospfdbd_data := newOspfDatabaseDescriptionData()
	/*  TODO check min length
	 */
	decodeDatabaseDescriptionData(data, ospfdbd_data)

	return nil
}
