// interfaces.go
package utils

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"utils/logging"
)

type InterfaceMgr struct {
	logger      *logging.Writer
	rwMutex     *sync.RWMutex
	ifIndexToIP map[int32]string
	ipToIfIndex map[string]int32
}

var ifaceMgr *InterfaceMgr

func NewInterfaceMgr(logger *logging.Writer) *InterfaceMgr {
	if ifaceMgr != nil {
		logger.Info(fmt.Sprintln("NewInterfaceMgr: Return the existing interface manager", ifaceMgr))
		return ifaceMgr
	}

	ifaceMgr = &InterfaceMgr{
		logger:      logger,
		rwMutex:     &sync.RWMutex{},
		ifIndexToIP: make(map[int32]string),
		ipToIfIndex: make(map[string]int32),
	}
	logger.Info(fmt.Sprintln("NewInterfaceMgr: Creating new interface manager", ifaceMgr))
	return ifaceMgr
}

func (i *InterfaceMgr) IsIPConfigured(ip string) bool {
	i.rwMutex.RLock()
	defer i.rwMutex.RUnlock()
	i.logger.Info(fmt.Sprintln("IsIPConfigured: ip", ip, "ipToIfIndex", i.ipToIfIndex))
	_, ok := i.ipToIfIndex[ip]
	return ok
}

func (i *InterfaceMgr) GetIfaceIP(ifIndex int32) (ip string, err error) {
	var ok bool
	i.rwMutex.RLock()
	defer i.rwMutex.RUnlock()
	i.logger.Info(fmt.Sprintln("GetIfaceIP: ifIndex", ifIndex, "ifIndexToIP", i.ifIndexToIP))
	if ip, ok = i.ifIndexToIP[ifIndex]; ok {
		err = errors.New(fmt.Sprintf("Iface %d is not configured", ifIndex))
	}

	return ip, err
}

func (i *InterfaceMgr) AddIface(ifIndex int32, addr string) {
	i.rwMutex.Lock()
	defer i.rwMutex.Unlock()
	i.logger.Info(fmt.Sprintln("AddIface: ifIndex", ifIndex, "ip", addr, "ifIndexToIP", i.ifIndexToIP, "ipToIfIndex",
		i.ipToIfIndex))

	ip, _, err := net.ParseCIDR(addr)
	if err != nil {
		i.logger.Err(fmt.Sprintln("AddIface: ParseCIDR failed for addr", addr, "with error", err))
		return
	}

	if oldIP, ok := i.ifIndexToIP[ifIndex]; ok {
		delete(i.ifIndexToIP, ifIndex)
		delete(i.ipToIfIndex, oldIP)
	}

	i.ifIndexToIP[ifIndex] = ip.String()
	i.ipToIfIndex[ip.String()] = ifIndex
}

func (i *InterfaceMgr) RemoveIface(ifIndex int32, addr string) {
	i.rwMutex.Lock()
	defer i.rwMutex.Unlock()
	i.logger.Info(fmt.Sprintln("RemoveIface: ifIndex", ifIndex, "ip", addr, "ifIndexToIP", i.ifIndexToIP, "ipToIfIndex",
		i.ipToIfIndex))

	if oldIP, ok := i.ifIndexToIP[ifIndex]; ok {
		delete(i.ifIndexToIP, ifIndex)
		delete(i.ipToIfIndex, oldIP)
	}
}
