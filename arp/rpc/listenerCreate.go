package rpc

import (
        "fmt"
        "arpd"
        "l3/arp/server"
)

func (h *ARPHandler) SendResolveArpIPv4(targetIp string, ifType arpd.Int, ifId arpd.Int) (arpd.Int) {
        rConf := server.ResolveIPv4 {
                TargetIP:       targetIp,
                IfType:         int(ifType),
                IfId:           int(ifId),
        }
        h.server.ResolveIPv4Ch <-rConf
        return arpd.Int(0)
}

func (h *ARPHandler) SendSetArpConfig(refTimeout arpd.Int) arpd.Int {
        arpConf := server.ArpConf {
                RefTimeout:     int(refTimeout),
        }
        h.server.ArpConfCh <- arpConf
        return arpd.Int(0)
}

func (h *ARPHandler) ResolveArpIPV4(targetIp string, ifType arpd.Int, ifId arpd.Int) (arpd.Int, error) {
        h.logger.Info(fmt.Sprintln("Received ResolveArpIPV4 call with targetIp:", targetIp, "ifType:", ifType, "ifId:", ifId))
        return h.SendResolveArpIPv4(targetIp, ifType, ifId), nil
}

func (h *ARPHandler) SetArpConfig(refTimeout arpd.Int) (arpd.Int, error) {
        h.logger.Info(fmt.Sprintln("Received SetArpConfig call with refTimeout:", refTimeout))
        return h.SendSetArpConfig(refTimeout), nil
}
