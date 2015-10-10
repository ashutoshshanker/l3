// packet.go
package server

import (
    "encoding/binary"
	"fmt"
    "net"
)

const BGP_HEADER_MARKER int = 16

const (
    _ uint8 = iota
    BGP_OPEN
    BGP_UPDATE
    BGP_NOTIFICATION
    BGP_KEEPALIVE
)

const (
    BGP_MSG_HEADER_LEN = 19
    BGP_MSG_MAX_LEN = 4096
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
	Marker [BGP_HEADER_MARKER]byte
	Length uint16
	Type uint8
}

func (header *BGPHeader) Encode() ([]byte, error) {
	pkt := make([]byte, 19)
	for i := 0; i < BGP_HEADER_MARKER; i++ {
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
		Header: BGPHeader{Type: BGP_OPEN},
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
		Header: BGPHeader{Length: 19, Type: BGP_KEEPALIVE},
		Body:   &BGPKeepAlive{},
	}
}

type BGPNotificationMessage struct {
    ErrorCode uint8
    ErrorSubcode uint8
    Data uint16
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
		if BGP_MSG_HEADER_LEN + len(body) > BGP_MSG_MAX_LEN {
			return nil, BGPMessageError{0, 0, nil, fmt.Sprintf("BGP message is %d bytes long", BGP_MSG_HEADER_LEN + len(body))}
		}
		msg.Header.Length = BGP_MSG_HEADER_LEN + uint16(len(body))
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
        case BGP_OPEN:
            msg.Body = &BGPOpen{}

        case BGP_KEEPALIVE:
            msg.Body = &BGPKeepAlive{}

        default:
            return nil
    }
    msg.Body.Decode(pkt)
    return nil
}
