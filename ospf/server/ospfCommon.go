package server

import (
    "net"
    "time"
)

var ALLSPFROUTER   string = "224.0.0.5"
var ALLDROUTER      string = "224.0.0.6"

type OspfHdrMetadata struct {
    pktType     OspfType
    pktlen      uint16
    backbone    bool
}

func NewOspfHdrMetadata() *OspfHdrMetadata {
    return &OspfHdrMetadata{}
}

type DstIPType uint8
const (
    Normal          DstIPType = 1
    AllSPFRouter    DstIPType = 2
    AllDRouter      DstIPType = 3
)

type IpHdrMetadata struct {
    srcIP       net.IP
    dstIP       net.IP
    dstIPType   DstIPType
}

func NewIpHdrMetadata() *IpHdrMetadata {
    return &IpHdrMetadata{}
}

var (
    snapshot_len            int32 = 65549  //packet capture length
    promiscuous             bool = false  //mode
    timeout_pcap            time.Duration = 5 * time.Second
)

const (
    OSPF_HELLO_MIN_SIZE = 20
    OSPF_HEADER_SIZE = 24
    IP_HEADER_MIN_LEN = 20
    OSPF_PROTO_ID = 89
    OSPF_VERSION_2 = 2
)

type OspfType uint8
const (
    HelloType           OspfType = 1
    DBDescriptionType   OspfType = 2
    LSRequestType       OspfType = 3
    LSUpdateType        OspfType = 4
    LSAckType           OspfType = 5
)
