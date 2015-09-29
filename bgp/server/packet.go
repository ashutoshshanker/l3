// packet.go
package server

import (
    "encoding/binary"
	"fmt"
    "net"
)

const BGP_HEADER_MARKER int = 16

const (
    BGP_OPEN = 1 + iota
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

func (header *BGPHeader) Serialize() ([]byte, error) {
	pkt := make([]byte, 19)
	for i := 0; i < BGP_HEADER_MARKER; i++ {
		pkt[i] = 0xff
	}
	binary.BigEndian.PutUint16(pkt[16:18], header.Length)
	pkt[18] = header.Type
	return pkt, nil
}
            
type BGPBody interface {
	Serialize() ([]byte, error)
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

func (msg *BGPOpen) Serialize() ([]byte, error) {
	pkt := make([]byte, 10)
	pkt[0] = msg.Version
	binary.BigEndian.PutUint16(pkt[1:3], msg.MyAS)
	binary.BigEndian.PutUint16(pkt[3:5], msg.HoldTime)
	copy(pkt[5:9], msg.BGPId.To4())
	pkt[9] = 0
	return pkt, nil
}

func NewBGPOpenMessage(myAS uint16, holdTime uint16, bgpId string) *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Type: BGP_OPEN},
		Body: &BGPOpen{4, myAS, holdTime, net.ParseIP(bgpId), 0},
	}
}

type BGPKeepAlive struct {
}

func (msg *BGPKeepAlive) Serialize() ([]byte, error) {
	return nil, nil
}

func NewBGPKeepAliveMessage() *BGPMessage {
	return &BGPMessage{
		Header: BGPHeader{Length: 19, Type: BGP_KEEPALIVE},
		Body:   &BGPKeepAlive{},
	}
}

type BGPMessage struct {
	Header BGPHeader
	Body   BGPBody
}

func (msg *BGPMessage) Serialize() ([]byte, error) {
	body, err := msg.Body.Serialize()
	if err != nil {
		return nil, err
	}
	if msg.Header.Length == 0 {
		if BGP_MSG_HEADER_LEN + len(body) > BGP_MSG_MAX_LEN {
			return nil, BGPMessageError{0, 0, nil, fmt.Sprintf("BGP message is %d bytes long", BGP_MSG_HEADER_LEN + len(body))}
		}
		msg.Header.Length = BGP_MSG_HEADER_LEN + uint16(len(body))
	}
	header, err := msg.Header.Serialize()
	if err != nil {
		return nil, err
	}
	return append(header, body...), nil
}
