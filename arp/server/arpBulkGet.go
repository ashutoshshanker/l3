package server

import (
	"time"
)

func (server *ARPServer) GetBulkArpEntry(idx int, cnt int) (int, int, []ArpState) {
	var nextIdx int
	var count int

	ret := server.arpSliceRefreshTimer.Stop()
	if ret == false {
		server.logger.Err("Arp is busy refreshing the Arp Entry Cache")
		return nextIdx, count, nil
	}

	length := len(server.arpSlice)
	result := make([]ArpState, cnt)
	var i int
	var j int
	for i, j = 0, idx; i < cnt && j < length; j++ {
		arpSliceEnt := server.arpSlice[j]
		arpEnt, exist := server.arpCache[arpSliceEnt]
		if !exist {
			continue
		}
		result[i].IpAddr = arpSliceEnt
		if arpEnt.MacAddr != "incomplete" {
			result[i].MacAddr = arpEnt.MacAddr
			result[i].Intf = arpEnt.IfName
			result[i].VlanId = arpEnt.VlanId
			curTime := time.Now()
			expiryTime := time.Duration(server.timerGranularity*server.timeoutCounter) * time.Second
			timeElapsed := curTime.Sub(arpEnt.TimeStamp)
			timeLeft := expiryTime - timeElapsed
			result[i].ExpiryTimeLeft = timeLeft.String()
		} else {
			result[i].MacAddr = arpEnt.MacAddr
			result[i].Intf = "N/A"
			result[i].VlanId = -1
			result[i].ExpiryTimeLeft = "N/A"
			i++
		}
		i++
	}
	if j == length {
		nextIdx = 0
	}
	count = i
	server.arpSliceRefreshTimer.Reset(server.arpSliceRefreshDuration)
	server.printArpEntries()
	return nextIdx, count, result
}
