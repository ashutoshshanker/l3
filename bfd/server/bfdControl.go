package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	//"fmt"
	"time"
)

type BfdSessionState int

const (
	STATE_ADMIN_DOWN BfdSessionState = 0
	STATE_DOWN       BfdSessionState = 1
	STATE_INIT       BfdSessionState = 2
	STATE_UP         BfdSessionState = 3
)

type BfdSessionEvent int

const (
	REMOTE_DOWN BfdSessionEvent = 1
	REMOTE_INIT BfdSessionEvent = 2
	REMOTE_UP   BfdSessionEvent = 3
	TIMEOUT     BfdSessionEvent = 4
	ADMIN_DOWN  BfdSessionEvent = 5
	ADMIN_UP    BfdSessionEvent = 6
)

type BfdDiagnostic int

const (
	DIAG_NONE                 BfdDiagnostic = 0 // No Diagnostic
	DIAG_TIME_EXPIRED         BfdDiagnostic = 1 // Control Detection Time Expired
	DIAG_ECHO_FAILED          BfdDiagnostic = 2 // Echo Function Failed
	DIAG_NEIGHBOR_SIGNAL_DOWN BfdDiagnostic = 3 // Neighbor Signaled Session Down
	DIAG_FORWARD_PLANE_RESET  BfdDiagnostic = 4 // Forwarding Plane Reset
	DIAG_PATH_DOWN            BfdDiagnostic = 5 // Path Down
	DIAG_CONCAT_PATH_DOWN     BfdDiagnostic = 6 // Concatenated Path Down
	DIAG_ADMIN_DOWN           BfdDiagnostic = 7 // Administratively Down
	DIAG_REV_CONCAT_PATH_DOWN BfdDiagnostic = 8 // Reverse Concatenated Path Down
)

type BfdControlPacket struct {
	Version                   uint8
	Diagnostic                BfdDiagnostic
	State                     BfdSessionState
	Poll                      bool
	Final                     bool
	ControlPlaneIndependent   bool
	AuthPresent               bool
	Demand                    bool
	Multipoint                bool // Must always be false
	DetectMult                uint8
	MyDiscriminator           uint32
	YourDiscriminator         uint32
	DesiredMinTxInterval      time.Duration
	RequiredMinRxInterval     time.Duration
	RequiredMinEchoRxInterval time.Duration
	AuthHeader                *BfdAuthHeader
}

// Constants
const (
	DEFAULT_BFD_VERSION                   = 1
	DEFAULT_DETECT_MULTI                  = 3
	DEFAULT_DESIRED_MIN_TX_INTERVAL       = 1000000
	DEFAULT_REQUIRED_MIN_RX_INTERVAL      = 1000000
	DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL = 0
	DEFAULT_CONTROL_PACKET_LEN            = 24
	MAX_NUM_SESSIONS                      = 1024
	DEST_PORT                             = 3784
	SRC_PORT                              = 49152
)

// Flags in BFD Control packet
const (
	BFD_MP             = 0x01 // Multipoint
	BFD_DEMAND         = 0x02 // Demand mode
	BFD_AUTH_PRESENT   = 0x04 // Authentication present
	BFD_CP_INDEPENDENT = 0x08 // Control plane independent
	BFD_FINAL          = 0x10 // Final message, response to Poll
	BFD_POLL           = 0x20 // Poll message
)

var BfdControlPacketDefaults = BfdControlPacket{
	Version:    DEFAULT_BFD_VERSION,
	Diagnostic: DIAG_NONE,
	State:      STATE_DOWN,
	Poll:       false,
	Final:      false,
	ControlPlaneIndependent:   false,
	AuthPresent:               false,
	Demand:                    false,
	Multipoint:                false,
	DetectMult:                DEFAULT_DETECT_MULTI,
	MyDiscriminator:           0,
	YourDiscriminator:         0,
	DesiredMinTxInterval:      DEFAULT_DESIRED_MIN_TX_INTERVAL,
	RequiredMinRxInterval:     DEFAULT_REQUIRED_MIN_RX_INTERVAL,
	RequiredMinEchoRxInterval: DEFAULT_REQUIRED_MIN_ECHO_RX_INTERVAL,
	AuthHeader:                nil,
}

/*
 * Create a control packet
 */
func (p *BfdControlPacket) CreateBfdControlPacket() ([]byte, error) {
	var auth []byte
	var err error
	buf := bytes.NewBuffer([]uint8{})
	flags := uint8(0)
	length := uint8(DEFAULT_CONTROL_PACKET_LEN)

	binary.Write(buf, binary.BigEndian, (p.Version<<5 | (uint8(p.Diagnostic) & 0x1f)))

	if p.Poll {
		flags |= BFD_POLL
	}
	if p.Final {
		flags |= BFD_FINAL
	}
	if p.ControlPlaneIndependent {
		flags |= BFD_CP_INDEPENDENT
	}
	if p.AuthPresent && (p.AuthHeader != nil) {
		flags |= BFD_AUTH_PRESENT
		auth, err = p.AuthHeader.createBfdAuthHeader()
		if err != nil {
			return nil, err
		}
		length += uint8(len(auth))
	}
	if p.Demand {
		flags |= BFD_DEMAND
	}
	if p.Multipoint {
		flags |= BFD_MP
	}

	binary.Write(buf, binary.BigEndian, (uint8(p.State)<<6 | flags))
	binary.Write(buf, binary.BigEndian, p.DetectMult)
	binary.Write(buf, binary.BigEndian, length)

	binary.Write(buf, binary.BigEndian, p.MyDiscriminator)
	binary.Write(buf, binary.BigEndian, p.YourDiscriminator)
	binary.Write(buf, binary.BigEndian, uint32(p.DesiredMinTxInterval))
	binary.Write(buf, binary.BigEndian, uint32(p.RequiredMinRxInterval))
	binary.Write(buf, binary.BigEndian, uint32(p.RequiredMinEchoRxInterval))

	if len(auth) > 0 {
		binary.Write(buf, binary.BigEndian, auth)
	}

	return buf.Bytes(), nil
}

/*
 * Decode the control packet
 */
func DecodeBfdControlPacket(data []byte) (*BfdControlPacket, error) {
	var err error
	packet := &BfdControlPacket{}

	packet.Version = uint8((data[0] & 0xE0) >> 5)
	packet.Diagnostic = BfdDiagnostic(data[0] & 0x1F)

	packet.State = BfdSessionState((data[1] & 0xD0) >> 6)

	// bit flags
	packet.Poll = (data[1]&0x20 != 0)
	packet.Final = (data[1]&0x10 != 0)
	packet.ControlPlaneIndependent = (data[1]&0x08 != 0)
	packet.AuthPresent = (data[1]&0x04 != 0)
	packet.Demand = (data[1]&0x02 != 0)
	packet.Multipoint = (data[1]&0x01 != 0)
	packet.DetectMult = uint8(data[2])

	length := uint8(data[3]) // No need to store this
	if uint8(len(data)) != length {
		err = errors.New("Packet length mis-match!")
		return nil, err
	}

	packet.MyDiscriminator = binary.BigEndian.Uint32(data[4:8])
	packet.YourDiscriminator = binary.BigEndian.Uint32(data[8:12])
	packet.DesiredMinTxInterval = time.Duration(binary.BigEndian.Uint32(data[12:16]))
	packet.RequiredMinRxInterval = time.Duration(binary.BigEndian.Uint32(data[16:20]))
	packet.RequiredMinEchoRxInterval = time.Duration(binary.BigEndian.Uint32(data[20:24]))

	if packet.AuthPresent {
		if len(data) > 24 {
			packet.AuthHeader, err = decodeBfdAuthHeader(data[24:])
		} else {
			err = errors.New("Header flag set, but packet too short!")
		}
	}

	return packet, err
}
