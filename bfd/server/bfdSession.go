package server

import (
	"l3/bfd/protocol"
)

//BFD session state machine

func CreateBfdSession(IfIndex int32, DestIp string) error {
	return nil
}

func DeleteBfdSession(IfIndex int32, DestIp string) error {
	return nil
}

func (session *BfdSession) EventHandler(event protocol.BfdSessionEvent) error {
	return nil
}
