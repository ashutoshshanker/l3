package server

import (
    "fmt"
    "l3/ospf/config"
    "github.com/google/gopacket"
    "net"
    "errors"
    "encoding/binary"
    "github.com/google/gopacket/layers"
)

func (server *OSPFServer)processIPv4Layer(ipLayer gopacket.Layer, ipAddr net.IP, ipHdrMd *IpHdrMetadata) error {
    ipLayerContents := ipLayer.LayerContents()
    ipChkSum := binary.BigEndian.Uint16(ipLayerContents[10:12])
    binary.BigEndian.PutUint16(ipLayerContents[10:], 0)
    allSPFRouter := net.ParseIP(ALLSPFROUTER)
    allDRouter := net.ParseIP(ALLDROUTER)

    csum := computeCheckSum(ipLayerContents)
    if csum != ipChkSum {
        err := errors.New("Incorrect IPv4 checksum, hence dicarding the packet")
        return err
    }

    ipPkt := ipLayer.(*layers.IPv4)
    if ipAddr.Equal(ipPkt.SrcIP) == true {
        err := errors.New(fmt.Sprintln("locally generated pkt", ipPkt.SrcIP, "hence dicarding the packet"))
        return err
    }

    if ipAddr.Equal(ipPkt.DstIP) == false &&
        allSPFRouter.Equal(ipPkt.DstIP) == false &&
        allDRouter.Equal(ipPkt.DstIP) == false {
        err := errors.New(fmt.Sprintln("Incorrect DstIP", ipPkt.DstIP, "hence dicarding the packet"))
        return err
    }

    if ipPkt.Protocol != layers.IPProtocol(OSPF_PROTO_ID) {
        err := errors.New(fmt.Sprintln("Incorrect ProtocolID", ipPkt.Protocol, "hence dicarding the packet"))
        return err
    }

    ipHdrMd.srcIP = ipPkt.SrcIP.To4()
    ipHdrMd.dstIP = ipPkt.DstIP.To4()
    if allSPFRouter.Equal(ipPkt.DstIP) {
        ipHdrMd.dstIPType = AllSPFRouter
    } else if allDRouter.Equal(ipPkt.DstIP) {
        ipHdrMd.dstIPType = AllDRouter
    } else {
        ipHdrMd.dstIPType = Normal
    }
/*
    ToDo:
    RFC 2328 Section 8.2
    1. Destination IP (AllDRouters)
*/
    return nil
}

func (server *OSPFServer)processOspfHeader(ospfPkt []byte, key IntfConfKey, md *OspfHdrMetadata) error {
    if len(ospfPkt) < OSPF_HEADER_SIZE {
        err := errors.New("Invalid length of Ospf Header");
        return err
    }

    ent, exist := server.IntfConfMap[key]
    if !exist {
        err := errors.New("Dropped because of interface no more valid")
        return err
    }

    ospfHdr := NewOSPFHeader()

    decodeOspfHdr(ospfPkt, ospfHdr)

    if server.ospfGlobalConf.Version != ospfHdr.ver {
        err := errors.New("Dropped because of Ospf Version not matching")
        return err
    }

    if ent.IfType != config.PointToPoint {
        if bytesEqual(ent.IfAreaId, ospfHdr.areaId) == false &&
            isInSubnet(net.IP(ent.IfAreaId), net.IP(ospfHdr.areaId), net.IPMask(ent.IfNetmask)) == false {
            err := errors.New("Dropped because of Src IP is not in subnet or Area ID not matching")
            return err
        }
    }

    // Todo: when areaId is of backbone
    if bytesEqual(ospfHdr.areaId, []byte{0, 0, 0, 0}) == true {
        md.backbone = true
    } else {
        md.backbone = false
    }

    //OSPF Auth Type
    if ent.IfAuthType != ospfHdr.authType {
        err := errors.New("Dropped because of Router Id not matching")
        return err
    }

    //OSPF Header CheckSum
    binary.BigEndian.PutUint16(ospfPkt[12:14], 0)
    copy(ospfPkt[16:OSPF_HEADER_SIZE], []byte{0, 0, 0, 0, 0, 0, 0, 0})
    csum := computeCheckSum(ospfPkt)
    if csum != ospfHdr.chksum {
        err := errors.New("Dropped because of invalid checksum")
        return err
    }

/*
    ToDo:
    RFC 2328 Section 8.2
    1. Complete AreaID check
    2. Authentication
*/
    md.pktType = OspfType(ospfHdr.pktType)
    md.pktlen = ospfHdr.pktlen
    md.routerId = ospfHdr.routerId
    return nil
}

func (server *OSPFServer)ProcessOspfRecvPkt(key IntfConfKey, pkt gopacket.Packet) {
    server.logger.Info(fmt.Sprintf("Recevied Ospf Packet"))
    ipLayer := pkt.Layer(layers.LayerTypeIPv4)
    if ipLayer == nil {
        server.logger.Err("Not an IP packet")
        return
    }

    ent, exist := server.IntfConfMap[key]
    if !exist {
        server.logger.Err("Dropped because of interface no more valid")
        return
    }

    ipHdrMd := NewIpHdrMetadata()
    err := server.processIPv4Layer(ipLayer, ent.IfIpAddr, ipHdrMd)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Dropped because of IPv4 layer processing", err))
        return
    } else {
        server.logger.Info("IPv4 Header is processed succesfully")
    }

    ospfHdrMd := NewOspfHdrMetadata()
    ospfPkt := ipLayer.LayerPayload()
    err = server.processOspfHeader(ospfPkt, key, ospfHdrMd)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Dropped because of Ospf Header processing", err))
        return
    } else {
        server.logger.Info("Ospfv2 Header is processed successfully")
    }

    ospfData := ospfPkt[OSPF_HEADER_SIZE:]
    err = server.processOspfData(ospfData, ipHdrMd, ospfHdrMd, key)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Dropped because of Ospf Header processing", err))
        return
    } else {
        server.logger.Info("Ospfv2 Header is processed successfully")
    }
    return
}

func (server *OSPFServer)processOspfData(data []byte, ipHdrMd *IpHdrMetadata, ospfHdrMd *OspfHdrMetadata, key IntfConfKey) error {
    var err error = nil
    switch ospfHdrMd.pktType {
    case HelloType:
       err = server.processRxHelloPkt(data, ospfHdrMd, ipHdrMd, key)
    case DBDescriptionType:
            ;
    case LSRequestType:
        ;
    case LSUpdateType:
        ;
    case LSAckType:
        ;
    default:
        err = errors.New("Invalid Ospf packet type")
    }
    return err
}
