package server

import (
	"fmt"
	//    "bytes"
	"encoding/binary"
	"errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"l3/ospf/config"
	"net"
	"time"
)

type OSPFHelloData struct {
	netmask             []byte
	helloInterval       uint16
	options             uint8
	rtrPrio             uint8
	rtrDeadInterval     uint32
	designatedRtr       []byte
	backupDesignatedRtr []byte
	neighbor            []byte
}

func NewOSPFHelloData() *OSPFHelloData {
	return &OSPFHelloData{}
}

func encodeOspfHelloData(helloData OSPFHelloData) []byte {
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

func (server *OSPFServer) BuildHelloPkt(ent IntfConf) []byte {
	ospfHdr := OSPFHeader{
		ver:      OSPF_VERSION_2,
		pktType:  uint8(HelloType),
		pktlen:   0,
		routerId: server.ospfGlobalConf.RouterId,
		areaId:   ent.IfAreaId,
		chksum:   0,
		authType: ent.IfAuthType,
		//authKey:        ent.IfAuthKey,
	}
	helloData := OSPFHelloData{
		netmask:             ent.IfNetmask,
		helloInterval:       ent.IfHelloInterval,
		options:             uint8(2), // Need to revisit
		rtrPrio:             ent.IfRtrPriority,
		rtrDeadInterval:     ent.IfRtrDeadInterval,
		designatedRtr:       ent.IfDRIp,
		backupDesignatedRtr: ent.IfBDRIp,
		//designatedRtr:          []byte {0, 0, 0, 0},
		//backupDesignatedRtr:    []byte {0, 0, 0, 0},
		//neighbor:               []byte {1, 1, 1, 1},
	}

	var neighbor []byte
	var nbrlen = 0
	nbr := make([]byte, 4)
	for key, _ := range ent.NeighborMap {
		binary.BigEndian.PutUint32(nbr, key.RouterId)
		nbrlen = nbrlen + 4
		neighbor = append(neighbor, nbr...)
	}

	ospfPktlen := OSPF_HEADER_SIZE
	ospfPktlen = ospfPktlen + OSPF_HELLO_MIN_SIZE + nbrlen

	ospfHdr.pktlen = uint16(ospfPktlen)

	ospfEncHdr := encodeOspfHdr(ospfHdr)
	//server.logger.Info(fmt.Sprintln("ospfEncHdr:", ospfEncHdr))
	helloDataEnc := encodeOspfHelloData(helloData)
	//server.logger.Info(fmt.Sprintln("HelloPkt:", helloDataEnc))
	helloDataNbrEnc := append(helloDataEnc, neighbor...)
	//server.logger.Info(fmt.Sprintln("HelloPkt with Neighbor:", helloDataNbrEnc))

	ospf := append(ospfEncHdr, helloDataNbrEnc...)
	//server.logger.Info(fmt.Sprintln("ospf:", ospf))
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
		DstIP:    net.IP{224, 0, 0, 5},
	}

	ethLayer := layers.Ethernet{
		SrcMAC:       ent.IfMacAddr,
		DstMAC:       net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0x05},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	gopacket.SerializeLayers(buffer, options, &ethLayer, &ipLayer, gopacket.Payload(ospf))
	//server.logger.Info(fmt.Sprintln("buffer: ", buffer))
	ospfPkt := buffer.Bytes()
	//server.logger.Info(fmt.Sprintln("ospfPkt: ", ospfPkt))
	return ospfPkt
}

func (server *OSPFServer) processRxHelloPkt(data []byte, ospfHdrMd *OspfHdrMetadata,
	ipHdrMd *IpHdrMetadata, ethHdrMd *EthHdrMetadata, key IntfConfKey) error {
	ent, _ := server.IntfConfMap[key]
	ospfHelloData := NewOSPFHelloData()
	if len(data) < OSPF_HELLO_MIN_SIZE {
		err := errors.New("Invalid Hello Pkt data length")
		return err
	}
	decodeOspfHelloData(data, ospfHelloData)

	//  Todo: Sec 10.5 RFC2328 Need to add check for Virtual links
	if ent.IfType != config.PointToPoint {
		if bytesEqual(ent.IfNetmask, ospfHelloData.netmask) == false {
			server.logger.Info(fmt.Sprintln("HELLO: Netmask mismatch. Int mask", ent.IfNetmask, " Hello mask ", ospfHelloData.netmask, " ip ", ipHdrMd.srcIP))
			err := errors.New("Netmask mismatch")
			return err
		}
	}

	if ent.IfHelloInterval != ospfHelloData.helloInterval {
		err := errors.New("Hello Interval mismatch")
		return err
	}

	if ent.IfRtrDeadInterval != ospfHelloData.rtrDeadInterval {
		err := errors.New("Router Dead Interval mismatch")
		return err
	}

	if ospfHdrMd.backbone == true {
		//server.logger.Info(fmt.Sprintln("Options:", ospfHelloData.options, "EOPTIONS:", EOption))
		if (ospfHelloData.options & EOption) == 0 {
			err := errors.New("External Routing Capability mismatch")
			return err
		}
	}

	//Todo: Find whether one way or two way
	TwoWayStatus := false
	/*
	   ospfHdrMetadata := OspfHdrMetadata {
	       pktType:    ospfHdrMd.pktType,
	       pktlen:     ospfHdrMd.pktlen,
	       backbone:   ospfHdrMd.backbone,
	   }
	   ospfHdrMetadata.routerId = ospfHdrMd.routerId

	   ipHdrMetadata := IpHdrMetadata {
	       dstIPType:      ipHdrMd.dstIPType,
	   }
	   ipHdrMetadata.srcIP = ipHdrMd.srcIP
	   copy(ipHdrMetadata.dstIP, ipHdrMd.dstIP)
	*/
	//server.logger.Info(fmt.Sprintln("ospfHelloData", ospfHelloData))
	//server.logger.Info(fmt.Sprintln("ipHdrMd", ipHdrMd))
	//server.logger.Info(fmt.Sprintln("ospfHdrMd", ospfHdrMd))
	nbrlen := ospfHdrMd.pktlen - (OSPF_HELLO_MIN_SIZE + OSPF_HEADER_SIZE)
	if nbrlen > 0 {
		j := uint16(OSPF_HELLO_MIN_SIZE)
		i := OSPF_HELLO_MIN_SIZE + 4
		k := 0
		for ; k < int(nbrlen); i, j, k = i+4, j+4, k+4 {
			if bytesEqual(data[j:i], server.ospfGlobalConf.RouterId) == true {
				TwoWayStatus = true
				break
			}
		}
	}

	routerId := convertIPv4ToUint32(ospfHdrMd.routerId)
	ospfNeighborIPToMAC[routerId] = ethHdrMd.srcMAC

	server.processOspfHelloNeighbor(TwoWayStatus, ospfHelloData, ipHdrMd, ospfHdrMd, key)

	return nil
}

func (server *OSPFServer) processOspfHelloNeighbor(TwoWayStatus bool, ospfHelloData *OSPFHelloData, ipHdrMd *IpHdrMetadata, ospfHdrMd *OspfHdrMetadata, key IntfConfKey) {

	//server.logger.Info(fmt.Sprintln("ospfHelloData", ospfHelloData))
	//server.logger.Info(fmt.Sprintln("ipHdrMd", ipHdrMd))
	//server.logger.Info(fmt.Sprintln("ospfHdrMd", ospfHdrMd))
	routerId := convertIPv4ToUint32(ospfHdrMd.routerId)
	NbrIP := convertIPv4ToUint32(ipHdrMd.srcIP)
	neighborKey := NeighborKey{
		RouterId: routerId,
		//NbrIP:          NbrIP,
	}

	//Todo: Find whether one way or two way
	/*
	   TwoWayStatus := false

	   j := uint16(OSPF_HELLO_MIN_SIZE)
	   i := OSPF_HELLO_MIN_SIZE + 4
	   for ; j < ospfHdrMd.pktlen; i, j = i+4, j+4 {
	       if bytesEqual(data[i:j], server.ospfGlobalConf.RouterId) == true {
	           TwoWayStatus = true
	           break
	       }
	   }
	*/
	ent, _ := server.IntfConfMap[key]

	neighborEntry, exist := ent.NeighborMap[neighborKey]
	if !exist {
		var neighCreateMsg NeighCreateMsg
		neighCreateMsg.RouterId = routerId
		neighCreateMsg.NbrIP = NbrIP
		neighCreateMsg.RtrPrio = ospfHelloData.rtrPrio
		neighCreateMsg.TwoWayStatus = TwoWayStatus
		neighCreateMsg.DRtr = append(neighCreateMsg.DRtr, ospfHelloData.designatedRtr...)
		neighCreateMsg.BDRtr = append(neighCreateMsg.BDRtr, ospfHelloData.backupDesignatedRtr...)
		ent.NeighCreateCh <- neighCreateMsg
		server.logger.Info(fmt.Sprintln("Neighbor Entry Created", neighborEntry))
	} else {
		if neighborEntry.TwoWayStatus != TwoWayStatus ||
			bytesEqual(neighborEntry.DRtr, ospfHelloData.designatedRtr) == false ||
			bytesEqual(neighborEntry.BDRtr, ospfHelloData.backupDesignatedRtr) == false ||
			neighborEntry.RtrPrio != ospfHelloData.rtrPrio {
			var neighChangeMsg NeighChangeMsg
			neighChangeMsg.RouterId = routerId
			neighChangeMsg.NbrIP = NbrIP
			neighChangeMsg.TwoWayStatus = TwoWayStatus
			neighChangeMsg.RtrPrio = ospfHelloData.rtrPrio
			neighChangeMsg.DRtr = append(neighChangeMsg.DRtr, ospfHelloData.designatedRtr...)
			neighChangeMsg.BDRtr = append(neighChangeMsg.BDRtr, ospfHelloData.backupDesignatedRtr...)
			ent.NeighChangeCh <- neighChangeMsg
		}
		server.logger.Info(fmt.Sprintln("Neighbor Entry already exist", neighborEntry))
	}

	nbrDeadInterval := time.Duration(ent.IfRtrDeadInterval) * time.Second
	server.CreateAndSendHelloRecvdMsg(routerId, ipHdrMd, ospfHdrMd, nbrDeadInterval,
		ent.IfType, TwoWayStatus, ospfHelloData.rtrPrio, key)

	var backupSeenMsg BackupSeenMsg
	if TwoWayStatus == true && ent.IfFSMState == config.Waiting {
		if bytesEqual(ipHdrMd.srcIP, ospfHelloData.designatedRtr) == true {
			if bytesEqual(ospfHelloData.backupDesignatedRtr, []byte{0, 0, 0, 0}) == false {
				ret := ent.WaitTimer.Stop()
				server.logger.Info("DIE")

				if ret == true {
					backupSeenMsg.RouterId = routerId
					backupSeenMsg.DRId = append(backupSeenMsg.DRId, ipHdrMd.srcIP...)
					backupSeenMsg.BDRId = append(backupSeenMsg.BDRId, ospfHelloData.backupDesignatedRtr...)
					server.logger.Info("Neigbor choose itself as Designated Router")
					server.logger.Info("Backup Designated Router also exist")
					ent.BackupSeenCh <- backupSeenMsg
				}
			}
		} else if bytesEqual(ipHdrMd.srcIP, ospfHelloData.backupDesignatedRtr) == true {
			ret := ent.WaitTimer.Stop()
			if ret == true {
				server.logger.Info("Neigbor choose itself as Backup Designated Router")
				backupSeenMsg.RouterId = routerId
				backupSeenMsg.DRId = append(backupSeenMsg.DRId, ospfHelloData.designatedRtr...)
				backupSeenMsg.BDRId = append(backupSeenMsg.BDRId, ipHdrMd.srcIP...)
				ent.BackupSeenCh <- backupSeenMsg
			}
		}
	}
}

func (server *OSPFServer) CreateAndSendHelloRecvdMsg(routerId uint32,
	ipHdrMd *IpHdrMetadata,
	ospfHdrMd *OspfHdrMetadata,
	nbrDeadInterval time.Duration, ifType config.IfType,
	TwoWayStatus bool, rtrPrio uint8, key IntfConfKey) {
	var msg IntfToNeighMsg

	if ifType == config.Broadcast ||
		ifType == config.Nbma ||
		ifType == config.PointToMultipoint {
		msg.NeighborIP = net.IPv4(ipHdrMd.srcIP[0], ipHdrMd.srcIP[1], ipHdrMd.srcIP[2], ipHdrMd.srcIP[3])
		//copy(msg.NeighborIP, ipHdrMd.srcIP)
	} else { //Check for Virtual Links and p2p
		msg.NeighborIP = net.IPv4(ospfHdrMd.routerId[0], ospfHdrMd.routerId[1], ospfHdrMd.routerId[2], ospfHdrMd.routerId[3])
		//copy(msg.NeighborIP, ospfHdrMd.routerId)
	}
	msg.RouterId = routerId
	msg.RtrPrio = rtrPrio
	msg.nbrDeadTimer = nbrDeadInterval
	msg.IntfConfKey.IPAddr = key.IPAddr
	msg.IntfConfKey.IntfIdx = key.IntfIdx
	msg.TwoWayStatus = TwoWayStatus

	server.logger.Info(fmt.Sprintln("Sending msg to Neighbor State Machine", msg))
	server.neighborHelloEventCh <- msg
}
