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

type BGPBody interface {
	Encode() ([]byte, error)
    Decode([]byte) error
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

func (msg *BGPOpen) Decode(pkt []byte) error {
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

func (msg *BGPKeepAlive) Decode([]byte) error {
    return nil
}

func NewBGPKeepAliveMessage() *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Length: 19, Type: BGPMsgTypeKeepAlive},
		Body:   &BGPKeepAlive{},
	}
}

type BGPNotificationMessage struct {
    ErrorCode uint8
    ErrorSubcode uint8
    Data []byte
}

func (msg *BGPNotificationMessage) Encode() ([]byte, error) {
	pkt := make([]byte, 2)
	pkt[0] = msg.ErrorCode
	pkt[1] = msg.ErrorSubcode
	pkt = append(pkt, msg.Data...)
	return pkt, nil
}

func (msg *BGPNotificationMessage) Decode(pkt []byte) error {
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
		Body:   &BGPNotificationMessage{errorCode, errorSubCode, data},
	}
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
    msg.Body.Decode(pkt)
    return nil
}
