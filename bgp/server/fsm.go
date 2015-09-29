// fsm.go
package server

import (
	_ "fmt"
)

type FSM struct {
    Global *GlobalConfig
    Peer *PeerConfig
    State int
}

func NewFSM(gConf *GlobalConfig, pConf *PeerConfig) *FSM {
    fsm := FSM{
        Global: gConf,
        Peer: pConf,
    }
    return &fsm
}