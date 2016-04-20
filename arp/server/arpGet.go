package server

import (
	"errors"
	"fmt"
	"time"
)

func (server *ARPServer) GetArpEntry(ipAddr string) (arpState ArpState, err error) {
	arpEnt, exist := server.arpCache[ipAddr]
	if !exist {
		err = errors.New(fmt.Sprintln("Unable to find Arp entry for given IP:", ipAddr))
		return arpState, err
	}

	arpState.IpAddr = ipAddr
	if arpEnt.MacAddr != "incomplete" {
		arpState.MacAddr = arpEnt.MacAddr
		arpState.Intf = arpEnt.IfName
		arpState.VlanId = arpEnt.VlanId
		curTime := time.Now()
		expiryTime := time.Duration(server.timerGranularity*server.timeoutCounter) * time.Second
		timeElapsed := curTime.Sub(arpEnt.TimeStamp)
		timeLeft := expiryTime - timeElapsed
		arpState.ExpiryTimeLeft = timeLeft.String()
	} else {
		arpState.MacAddr = arpEnt.MacAddr
		arpState.Intf = "N/A"
		arpState.VlanId = -1
		arpState.ExpiryTimeLeft = "N/A"
	}
	return arpState, nil
}
