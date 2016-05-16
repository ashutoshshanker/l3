package rpc

import (
	"net"
	"strings"
)

func convertIPStrToUint32(ipStr string) (uint32, bool) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, false
	}

	IP := ip.To4()
	ipAddr := uint32(IP[0])<<24 | uint32(IP[1])<<16 | uint32(IP[2])<<8 | uint32(IP[3])
	return ipAddr, true
}

func parseIPRangeStr(ipAddrRangeStr string) (uint32, uint32, bool) {
	ret := strings.Contains(ipAddrRangeStr, "-")
	if !ret {
		return 0, 0, false
	}
	ipStr := strings.Split(ipAddrRangeStr, "-")
	if len(ipStr) != 2 {
		return 0, 0, false
	}
	lIP := strings.TrimSpace(ipStr[0])
	hIP := strings.TrimSpace(ipStr[1])
	lowerIP, ret := convertIPStrToUint32(lIP)
	if !ret {
		return 0, 0, false
	}
	higherIP, ret := convertIPStrToUint32(hIP)
	if !ret {
		return 0, 0, false
	}
	return lowerIP, higherIP, true
}
