package server

import (
        "fmt"
        "errors"
        "asicd/asicdConstDefs"
        "utils/commonDefs"
)

func (server *ARPServer)processResolveIPv4(conf ResolveIPv4) {
        server.logger.Info(fmt.Sprintln("Received ResolveIPv4 call for TargetIP:", conf.TargetIP, "ifType:", conf.IfType, "ifId:", conf.IfId))
        if conf.TargetIP == "0.0.0.0" {
                return
        }
        IfIndex := int(asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(conf.IfId, conf.IfType))
        if conf.IfType == commonDefs.L2RefTypeVlan {
                vlanEnt := server.vlanPropMap[IfIndex]
                for port, _ := range vlanEnt.UntagPortMap {
                        server.sendArpReq(conf.TargetIP, port)
                        server.arpEntryUpdateCh <- UpdateArpEntryMsg {
                                PortNum: port,
                                IpAddr: conf.TargetIP,
                                MacAddr: "incomplete",
                        }
                }
        } else if conf.IfType == commonDefs.L2RefTypeLag {
                lagEnt := server.lagPropMap[IfIndex]
                for port, _ := range lagEnt.PortMap {
                        server.sendArpReq(conf.TargetIP, port)
                        server.arpEntryUpdateCh <- UpdateArpEntryMsg {
                                PortNum: port,
                                IpAddr: conf.TargetIP,
                                MacAddr: "incomplete",
                        }
                }
        } else if conf.IfType == commonDefs.L2RefTypePort {
                server.sendArpReq(conf.TargetIP, IfIndex)
                server.arpEntryUpdateCh <- UpdateArpEntryMsg {
                        PortNum: IfIndex,
                        IpAddr: conf.TargetIP,
                        MacAddr: "incomplete",
                }
        }
}

func (server *ARPServer)processArpConf(conf ArpConf) (int, error) {
        server.logger.Info(fmt.Sprintln("Received ARP Timeout Value via Configuration:", conf.RefTimeout))
        if conf.RefTimeout < server.minRefreshTimeout {
                server.logger.Info(fmt.Sprintln("Refresh Timeout is below minimum allowed refresh timeout value of:", server.minRefreshTimeout))
                err := errors.New("Invalid Timeout Value")
                return 0, err
        } else if conf.RefTimeout == server.confRefreshTimeout {
                server.logger.Info(fmt.Sprintln("Arp is already configured with Refresh Timeout Value of:", server.confRefreshTimeout, "(seconds)"))
                return 0, nil
        }

        server.timeoutCounter = conf.RefTimeout/server.timerGranularity
        server.arpEntryCntUpdateCh <- server.timeoutCounter
        return 0, nil
}
