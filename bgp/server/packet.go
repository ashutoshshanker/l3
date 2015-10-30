// packet.go
package server

import (
    "encoding/binary"
	"fmt"
    "net"
)

const BGPHeaderMarkerLen int = 16

const (
    _ uint8 = iota
    BGPMsgTypeOpen
    BGPMsgTypeUpdate
    BGPMsgTypeNotification
    BGPMsgTypeKeepAlive
)

const (
    BGPMsgHeaderLen = 19
    BGPMsgMaxLen = 4096
)

const (
	_ uint8 = iota
	BGPMsgHeaderError
	BGPOpenMsgError
	BGPUpdateMsgError
	BGPHoldTimerExpired
	BGPFSMError
	BGPCease
)

const (
	_ uint8 = iota
	BGPConnNotSychd
	BGPBadMessageLen
	BGPBadMessageType
)

const (
	_ uint8 = iota
	BGPUnsupportedVersionNumber
	BGPBadPeerAS
	BGPBadBGPIdentifier
	BGPUnsupportedOptionalParam
	_
	BGPUnacceptableHoldTime
)

const (
	_ uint8 = iota
	BGPMalformedAttrList
	BGPUnrecognizedWellKnownAttr
	BGPMissingWellKnownAttr
	BGPAttrFlagsError
	BGPAttrLenError
	BGPInvalidOriginAttr
	_
	BGPInvalidNextHopAttr
	BGPOptionalAttrError
	BGPInvalidNetworkField
	BGPMalformedASPath
)

type BGPPathAttrFlag uint8

const (
	_ BGPPathAttrFlag = 1 << (iota + 3)
	BGPPathAttrFlagExtendedLen
	BGPPathAttrFlagPartial
	BGPPathAttrFlagTransitive
	BGPPathAttrFlagOptional
)

type BGPPathAttrType uint8

const (
	_ BGPPathAttrType = iota
	BGPPathAttrTypeOrigin
	BGPPathAttrTypeASPath
	BGPPathAttrTypeNextHop
	BGPPathAttrTypeMultiExitDisc
	BGPPathAttrTypeLocalPref
	BGPPathAttrTypeAtomicAggregate
	BGPPathAttrTypeAggregator
	BGPPathAttrTypeUnknown
)

type BGPPathAttrOriginType uint8

const (
	BGPPathAttrOriginIGP BGPPathAttrOriginType = iota
	BGPPathAttrOriginEGP
	BGPPathAttrOriginIncomplete
)

type BGPASPathSegmentType uint8

const (
	BGPASPathSet BGPASPathSegmentType = iota
	BGPASPathSequence
)

type BGPMessageError struct {
    TypeCode uint8
    SubTypeCode uint8
    Data []byte
    Message string
}

func (e BGPMessageError) Error() string {
    return fmt.Sprintf("%v:%v - %v", e.TypeCode, e.SubTypeCode, e.Message)
}

type BGPHeader struct {
	Marker [BGPHeaderMarkerLen]byte
	Length uint16
	Type uint8
}

func (header *BGPHeader) Encode() ([]byte, error) {
	pkt := make([]byte, 19)
	for i := 0; i < BGPHeaderMarkerLen; i++ {
		pkt[i] = 0xff
	}
	binary.BigEndian.PutUint16(pkt[16:18], header.Length)
	pkt[18] = header.Type
	return pkt, nil
}

func (header *BGPHeader) Decode(pkt []byte) error {
    header.Length = binary.BigEndian.Uint16(pkt[16:18])
    header.Type = pkt[18]
    return nil
}

func (header *BGPHeader) Len() uint32 {
	return uint32(header.Length)
}

type BGPBody interface {
	Encode() ([]byte, error)
    Decode(*BGPHeader, []byte) error
}

type OptionParameterInterface struct {
    bytes []byte
}

type BGPOpen struct {
	Version uint8
	MyAS uint16
	HoldTime uint16
	BGPId net.IP
	OptParamLen uint8
	//OptParams []OptionParameterInterface
}

func (msg *BGPOpen) Encode() ([]byte, error) {
	pkt := make([]byte, 10)
	pkt[0] = msg.Version
	binary.BigEndian.PutUint16(pkt[1:3], msg.MyAS)
	binary.BigEndian.PutUint16(pkt[3:5], msg.HoldTime)
	copy(pkt[5:9], msg.BGPId.To4())
	pkt[9] = 0
	return pkt, nil
}

func (msg *BGPOpen) Decode(header *BGPHeader, pkt []byte) error {
    msg.Version = pkt[0]
    msg.MyAS = binary.BigEndian.Uint16(pkt[1:3])
    msg.HoldTime = binary.BigEndian.Uint16(pkt[3:5])
    msg.BGPId = net.IP(pkt[5:9]).To4()
    msg.OptParamLen = pkt[9]
    return nil
}

func NewBGPOpenMessage(myAS uint16, holdTime uint16, bgpId string) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Type: BGPMsgTypeOpen},
		Body: &BGPOpen{4, myAS, holdTime, net.ParseIP(bgpId), 0},
	}
}

type BGPKeepAlive struct {
}

func (msg *BGPKeepAlive) Encode() ([]byte, error) {
	return nil, nil
}

func (msg *BGPKeepAlive) Decode(*BGPHeader, []byte) error {
    return nil
}

func NewBGPKeepAliveMessage() *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Length: 19, Type: BGPMsgTypeKeepAlive},
		Body:   &BGPKeepAlive{},
	}
}

type BGPNotification struct {
    ErrorCode uint8
    ErrorSubcode uint8
    Data []byte
}

func (msg *BGPNotification) Encode() ([]byte, error) {
	pkt := make([]byte, 2)
	pkt[0] = msg.ErrorCode
	pkt[1] = msg.ErrorSubcode
	pkt = append(pkt, msg.Data...)
	return pkt, nil
}

func (msg *BGPNotification) Decode(header *BGPHeader, pkt []byte) error {
	msg.ErrorCode = pkt[0]
	msg.ErrorSubcode = pkt[1]
	if len(pkt) > 2 {
		msg.Data = pkt[2:]
	}
    return nil
}

func NewBGPNotificationMessage(errorCode uint8, errorSubCode uint8, data []byte) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Length: 21 + uint16(len(data)), Type: BGPMsgTypeNotification},
		Body:   &BGPNotification{errorCode, errorSubCode, data},
	}
}

type IPPrefix struct	 {
	Length uint8
	Prefix net.IP
}

func (ip *IPPrefix) Decode(pkt []byte) error {
	ip.Length = pkt[0]
	bytes := (ip.Length + 7) / 8
	if len(pkt) < int(bytes) {
		return BGPMessageError{BGPUpdateMsgError, BGPMalformedAttrList, nil, "Prefix length invalid"}
	}
	ip.Prefix = make(net.IP, 4)
	copy(ip.Prefix, pkt[1:bytes + 1])
	return nil
}

func	 (ip *IPPrefix) Len() uint32	{
	return uint32(((ip.Length + 7) / 8) + 1)
}

type BGPPathAttr interface {
	Decode(pkt []byte) error
	TotalLen() uint32
}

type BGPPathAttrBase struct {
	Flags BGPPathAttrFlag
	Code BGPPathAttrType
	Length uint16
	BGPPathAttrLen uint16
}

func (pa *BGPPathAttrBase) Decode (pkt []byte) error {
	if len(pkt) < 3 {
		return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}

	pa.Flags = BGPPathAttrFlag(pkt[0])
	pa.Code = BGPPathAttrType(pkt[1])

	if pa.Flags & BGPPathAttrFlagExtendedLen != 0 {
		if len(pkt) < 4 {
			return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
		}
		pa.Length = binary.BigEndian.Uint16(pkt[2:4])
		pa.BGPPathAttrLen = 4
	} else {
		pa.Length = uint16(pkt[2])
		pa.BGPPathAttrLen = 3
	}
	if len(pkt) < int(pa.Length + pa.BGPPathAttrLen) {
		return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}
	return nil
}

func (pa *BGPPathAttrBase) TotalLen() uint32 {
	return uint32(pa.Length) + uint32(pa.BGPPathAttrLen)
}

type BGPPathAttrOrigin struct {
	BGPPathAttrBase
	Value uint8
}

func (o *BGPPathAttrOrigin) Decode(pkt []byte) error {
	err := o.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	if o.Length != 1 {
		return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}

	o.Value = pkt[o.BGPPathAttrLen]
	return nil
}

type BGPASPathSegment struct {
	Type BGPASPathSegmentType
	Length uint8
	AS []uint16
	BGPASPathSegmentLen uint16
}

func (ps *BGPASPathSegment) Decode(pkt []byte) error {
	if len(pkt) <= 2 {
		return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}

	ps.Type = BGPASPathSegmentType(pkt[0])
	ps.Length = pkt[1]

	if len(pkt) < int(ps.Length) - 2 {
		return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}

	ps.AS = make([]uint16, ps.Length)
	for i := 0; i < int(ps.Length); i++ {
		ps.AS[i] = binary.BigEndian.Uint16(pkt[(i * 2) + 2:])
	}
	ps.BGPASPathSegmentLen = uint16(ps.Length * 2 + 2)
	return nil
}

type BGPPathAttrASPath struct {
	BGPPathAttrBase
	Value []BGPASPathSegment
}

func (as *BGPPathAttrASPath) Decode(pkt []byte) error {
	err := as.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	//as.Value = make([]BGPASPathSegment, 1)
	ptr := uint32(as.BGPPathAttrLen)
	for ptr < (uint32(as.Length) + uint32(as.BGPPathAttrLen)) {
		asPathSegment := BGPASPathSegment{}
		err = asPathSegment.Decode(pkt[ptr:])
		if err != nil {
			return nil
		}
		ptr += uint32(asPathSegment.BGPASPathSegmentLen)
		if ptr > (uint32(as.Length) + uint32(as.BGPPathAttrLen)) {
			return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
		}
		as.Value = append(as.Value, asPathSegment)
	}
	if ptr != (uint32(as.Length) + uint32(as.BGPPathAttrLen)) {
		return BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}
	return nil
}

type BGPPathAttrNextHop struct {
	BGPPathAttrBase
	Value net.IP
}

func (n *BGPPathAttrNextHop) Decode(pkt []byte) error {
	err := n.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	n.Value = make(net.IP, n.Length)
	copy(n.Value, pkt[n.BGPPathAttrLen:n.BGPPathAttrLen + n.Length])
	return nil
}

type BGPPathAttrMultiExitDisc struct {
	BGPPathAttrBase
	Value uint32
}

func (m *BGPPathAttrMultiExitDisc) Decode(pkt []byte) error {
	err := m.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	m.Value = binary.BigEndian.Uint32(pkt[m.BGPPathAttrLen:m.BGPPathAttrLen + m.Length])
	return nil
}

type BGPPathAttrLocalPref struct {
	BGPPathAttrBase
	Value uint32
}

func (l *BGPPathAttrLocalPref) Decode(pkt []byte) error {
	err := l.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	l.Value = binary.BigEndian.Uint32(pkt[l.BGPPathAttrLen:l.BGPPathAttrLen + l.Length])
	return nil
}

type BGPPathAttrAtomicAggregate struct {
	BGPPathAttrBase
}

type BGPPathAttrAggregator struct {
	BGPPathAttrBase
	AS uint16
	IP net.IP
}

func (a *BGPPathAttrAggregator) Decode(pkt []byte) error {
	err := a.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	a.AS = binary.BigEndian.Uint16(pkt[a.BGPPathAttrLen:a.BGPPathAttrLen + 2])
	a.IP = make(net.IP, 4)
	copy(a.IP, pkt[a.BGPPathAttrLen + 2:a.BGPPathAttrLen + 6])
	return nil
}

type BGPPathAttrUnknown struct {
	BGPPathAttrBase
	Value []byte
}

func (u *BGPPathAttrUnknown) Decode(pkt []byte) error {
	err := u.BGPPathAttrBase.Decode(pkt)
	if err != nil {
		return err
	}

	u.Value = make([]byte, u.Length)
	copy(u.Value, pkt[u.BGPPathAttrLen:u.BGPPathAttrLen + u.Length])
	return nil
}

var BGPPathAttrTypeToStructMap map[BGPPathAttrType]BGPPathAttr = map[BGPPathAttrType]BGPPathAttr {
	BGPPathAttrTypeOrigin: &BGPPathAttrOrigin{},
	BGPPathAttrTypeASPath: &BGPPathAttrASPath{},
	BGPPathAttrTypeNextHop: &BGPPathAttrNextHop{},
	BGPPathAttrTypeMultiExitDisc: &BGPPathAttrMultiExitDisc{},
	BGPPathAttrTypeLocalPref: &BGPPathAttrLocalPref{},
	BGPPathAttrTypeAtomicAggregate: &BGPPathAttrAtomicAggregate{},
	BGPPathAttrTypeAggregator: &BGPPathAttrAggregator{},
}

func BGPGetPathAttr(pkt []byte) (BGPPathAttr) {
	typeCode := (pkt[0] & 0x80) >> 7
	var pathAttr BGPPathAttr

	pathAttr, ok := BGPPathAttrTypeToStructMap[BGPPathAttrType(typeCode)]
	if !ok {
		return &BGPPathAttrUnknown{}
	} else {
		return pathAttr
	}
}

type BGPUpdate struct {
	WithdrawnRoutesLen uint16
	WithdrawnRoutes []IPPrefix
	TotalPathAttrLen uint16
	PathAttributes []BGPPathAttr
	NLRI []IPPrefix
}

func (msg *BGPUpdate) Encode() ([]byte, error) {
	pkt := make([]byte, 10)
	return pkt, nil
}

func (msg *BGPUpdate) decodeIPPrefix(pkt []byte, ipPrefix *[]IPPrefix, length uint32) (uint32, error) {
	ptr := uint32(0)

	for ptr < length {
		ip := IPPrefix{}
		err := ip.Decode(pkt[ptr:])
		if err != nil {
			return ptr, err
		}

		*ipPrefix = append(*ipPrefix, ip)
		ptr += ip.Len()
	}

	if ptr != length {
		return ptr, BGPMessageError{BGPUpdateMsgError, BGPAttrLenError, pkt, "Bad Attribute Length"}
	}
	return ptr, nil
}

func (msg *BGPUpdate) Decode(header *BGPHeader, pkt []byte) error {
	msg.WithdrawnRoutesLen = binary.BigEndian.Uint16(pkt[0:2])
	//msg.WithdrawnRoutes = make([]IPPrefix, 1)
	ptr := uint32(2)
	length := uint32(msg.WithdrawnRoutesLen)
	ipLen := uint32(0)
	var err error

	ipLen, err = msg.decodeIPPrefix(pkt[ptr:], &msg.WithdrawnRoutes, length)
	if err != nil {
		return nil
	}
	ptr += ipLen

	msg.TotalPathAttrLen = binary.BigEndian.Uint16(pkt[ptr:ptr + 2])
	ptr += 2

	length = uint32(msg.TotalPathAttrLen)

	//msg.PathAttributes = make([]BGPPathAttr, 1)
	for length > 0 {
		pa := BGPGetPathAttr(pkt[ptr:])
		pa.Decode(pkt)
		ptr += pa.TotalLen()
		length += pa.TotalLen()
	}

	length = header.Len() - 23 - uint32(msg.WithdrawnRoutesLen) - uint32(msg.TotalPathAttrLen)
	ipLen, err = msg.decodeIPPrefix(pkt[ptr:], &msg.NLRI, length)
	if err != nil {
		return nil
	}
	return nil
}

type BGPMessage struct {
	Header BGPHeader
	Body   BGPBody
}

func (msg *BGPMessage) Encode() ([]byte, error) {
	body, err := msg.Body.Encode()
	if err != nil {
		return nil, err
	}
	if msg.Header.Length == 0 {
		if BGPMsgHeaderLen + len(body) > BGPMsgMaxLen {
			return nil, BGPMessageError{0, 0, nil, fmt.Sprintf("BGP message is %d bytes long", BGPMsgHeaderLen + len(body))}
		}
		msg.Header.Length = BGPMsgHeaderLen + uint16(len(body))
	}
	header, err := msg.Header.Encode()
	if err != nil {
		return nil, err
	}
	return append(header, body...), nil
}

func (msg *BGPMessage) Decode(header *BGPHeader, pkt []byte) error {
    msg.Header = *header
    switch header.Type {
        case BGPMsgTypeOpen:
            msg.Body = &BGPOpen{}

        case BGPMsgTypeKeepAlive:
            msg.Body = &BGPKeepAlive{}

        default:
            return nil
    }
    msg.Body.Decode(header, pkt)
    return nil
}
