package server

import (
    "fmt"
//    "bytes"
    "net"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "encoding/binary"
//    "time"
    "errors"
)

type OSPFHelloData struct {
    netmask             []byte
    helloInterval       uint16
    options             uint8
    rtrPrio             uint8
    rtrDeadInterval     uint32
    designatedRtr        []byte
    backupDesignatedRtr []byte
    neighbor            []byte
}

func NewOSPFHelloData() *OSPFHelloData {
    return &OSPFHelloData{}
}

func encodeOspfHelloData(helloData OSPFHelloData) ([]byte) {
    pkt := make([]byte, OSPF_HELLO_MIN_SIZE)
    copy(pkt[0:4], helloData.netmask)
    binary.BigEndian.PutUint16(pkt[4:6], helloData.helloInterval)
    pkt[6] = helloData.options
    pkt[7] = helloData.rtrPrio
    binary.BigEndian.PutUint32(pkt[8:12], helloData.rtrDeadInterval)
    copy(pkt[12:16], helloData.designatedRtr)
    copy(pkt[16:20], helloData.backupDesignatedRtr)
    //copy(pkt[20:24], helloData.neighbor)

    return pkt
}

func decodeOspfHelloData(data []byte, ospfHelloData *OSPFHelloData) {
    ospfHelloData.netmask = data[0:4]
    ospfHelloData.helloInterval = binary.BigEndian.Uint16(data[4:6])
    ospfHelloData.options = data[6]
    ospfHelloData.rtrPrio = data[7]
    ospfHelloData.rtrDeadInterval = binary.BigEndian.Uint32(data[8:12])
    ospfHelloData.designatedRtr = data[12:16]
    ospfHelloData.backupDesignatedRtr = data[16:20]
}

func (server *OSPFServer)BuildHelloPkt(ent IntfConf) ([]byte) {
    ospfHdr := OSPFHeader {
        ver:            OSPF_VERSION_2,
        pktType:        uint8(HelloType),
        pktlen:         0,
        routerId:       server.ospfGlobalConf.RouterId,
        areaId:         ent.IfAreaId,
        chksum:         0,
        authType:       ent.IfAuthType,
        //authKey:        ent.IfAuthKey,
    }
    helloData := OSPFHelloData {
        netmask:                ent.IfNetmask,
        helloInterval:          ent.IfHelloInterval,
        options:                uint8(2),
        rtrPrio:                ent.IfRtrPriority,
        rtrDeadInterval:        ent.IfRtrDeadInterval,
        designatedRtr:          []byte {0, 0, 0, 0},
        backupDesignatedRtr:    []byte {0, 0, 0, 0},
        //neighbor:               []byte {1, 1, 1, 1},
    }

    ospfPktlen := OSPF_HEADER_SIZE
    ospfPktlen = ospfPktlen + OSPF_HELLO_MIN_SIZE

    ospfHdr.pktlen = uint16(ospfPktlen)

    ospfEncHdr := encodeOspfHdr(ospfHdr)
    server.logger.Info(fmt.Sprintln("ospfEncHdr:", ospfEncHdr))
    helloDataEnc := encodeOspfHelloData(helloData)
    server.logger.Info(fmt.Sprintln("HelloPkt:", helloDataEnc))

    ospf := append(ospfEncHdr, helloDataEnc...)
    server.logger.Info(fmt.Sprintln("ospf:", ospf))
    csum := computeCheckSum(ospf)
    binary.BigEndian.PutUint16(ospf[12:14], csum)
    copy(ospf[16:24], ent.IfAuthKey)

    ipPktlen := IP_HEADER_MIN_LEN + ospfHdr.pktlen
    ipLayer := layers.IPv4 {
        Version:            uint8(4),
        IHL:                uint8(IP_HEADER_MIN_LEN),
        TOS:                uint8(0xc0),
        Length:             uint16(ipPktlen),
        TTL:                uint8(1),
        Protocol:           layers.IPProtocol(OSPF_PROTO_ID),
        SrcIP:              ent.IfIpAddr,
        DstIP:              net.IP{224, 0, 0, 5},
    }

    ethLayer := layers.Ethernet {
        SrcMAC:         ent.IfMacAddr,
        DstMAC:         net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x05},
        EthernetType:   layers.EthernetTypeIPv4,
    }

    buffer := gopacket.NewSerializeBuffer()
    options := gopacket.SerializeOptions {
        FixLengths:         true,
        ComputeChecksums:   true,
    }
    gopacket.SerializeLayers(buffer, options, &ethLayer, &ipLayer, gopacket.Payload(ospf))
    server.logger.Info(fmt.Sprintln("buffer: ", buffer))
    ospfPkt := buffer.Bytes()
    server.logger.Info(fmt.Sprintln("ospfPkt: ", ospfPkt))
    return ospfPkt
}

func (server *OSPFServer)processRxHelloPkt(data []byte, ospfHdrMd *OspfHdrMetadata, ipHdrMd *IpHdrMetadata, key IntfConfKey) error {
    ent, _ := server.IntfConfMap[key]
    ospfHelloData := NewOSPFHelloData()
    if len(data) < OSPF_HELLO_MIN_SIZE {
        err := errors.New("Invalid Hello Pkt data length")
        return err
    }
    decodeOspfHelloData(data, ospfHelloData)
    if bytesEqual(ent.IfNetmask, ospfHelloData.netmask) == false {
        err := errors.New("Netmask mismatch")
        return err
    }

    if ent.IfHelloInterval != ospfHelloData.helloInterval {
        err := errors.New("Hello Interval mismatch")
        return err
    }

    if ent.IfRtrDeadInterval != ospfHelloData.rtrDeadInterval {
        err := errors.New("Router Dead Interval mismatch")
        return err
    }
    return nil
}

