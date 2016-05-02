package rpc

import (
	"arpd"
	"arpdInt"
	"fmt"
	"l3/arp/server"
)

func (h *ARPHandler) SendResolveArpIPv4(targetIp string, ifType arpdInt.Int, ifId arpdInt.Int) {
	rConf := server.ResolveIPv4{
		TargetIP: targetIp,
		IfType:   int(ifType),
		IfId:     int(ifId),
	}
	h.server.ResolveIPv4Ch <- rConf
	return
}

func (h *ARPHandler) SendDeleteResolveArpIPv4(NextHopIp string) {
	return
}

func (h *ARPHandler) SendSetArpConfig(refTimeout int) bool {
	arpConf := server.ArpConf{
		RefTimeout: refTimeout,
	}
	h.server.ArpConfCh <- arpConf
	return true
}

//func (h *ARPHandler) ResolveArpIPV4(targetIp string, ifType arpdInt.Int, ifId arpdInt.Int) (arpdInt.Int, error) {
func (h *ARPHandler) ResolveArpIPV4(targetIp string, ifType arpdInt.Int, ifId arpdInt.Int) error {
	h.logger.Info(fmt.Sprintln("Received ResolveArpIPV4 call with targetIp:", targetIp, "ifType:", ifType, "ifId:", ifId))
	h.SendResolveArpIPv4(targetIp, ifType, ifId)
	return nil
}

//func (h *ARPHandler) CreateArpConfig(refTimeout arpd.Int) (arpd.Int, error) {
func (h *ARPHandler) CreateArpConfig(conf *arpd.ArpConfig) (bool, error) {
	h.logger.Info(fmt.Sprintln("Received CreateArpConfig call with Timeout:", conf.Timeout))
	return h.SendSetArpConfig(int(conf.Timeout)), nil
}

func (h *ARPHandler) DeleteResolveArpIPv4(NextHopIp string) error {
	h.logger.Info(fmt.Sprintln("Received DeleteResolveArpIPv4 call with NextHopIp:", NextHopIp))
	h.SendDeleteResolveArpIPv4(NextHopIp)
	return nil
}
