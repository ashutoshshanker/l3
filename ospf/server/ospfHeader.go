package server

import (
    "encoding/binary"
)

type OSPFHeader struct {
    ver         uint8
    pktType     uint8
    pktlen      uint16
    routerId    []byte
    areaId      []byte
    chksum      uint16
    authType    uint16
    authKey     []uint8
}

func encodeOspfHdr(ospfHdr OSPFHeader) ([]byte) {
    pkt := make([]byte, OSPF_HEADER_SIZE)
    pkt[0] = ospfHdr.ver
    pkt[1] = ospfHdr.pktType
    binary.BigEndian.PutUint16(pkt[2:4], ospfHdr.pktlen)
    copy(pkt[4:8], ospfHdr.routerId)
    copy(pkt[8:12], ospfHdr.areaId)
   //binary.BigEndian.PutUint16(pkt[12:14], ospfHdr.chksum)
    binary.BigEndian.PutUint16(pkt[14:16], ospfHdr.authType)
    //copy(pkt[16:24], ospfHdr.authKey)

    return pkt
}
