package FSMgr

import (
	"bfdd"
	"fmt"
	"l3/bgp/server"
)

func (mgr *FSBfdMgr) ProcessBfd(peer *server.Peer) {
	bfdSession := bfdd.NewBfdSession()
	bfdSession.IpAddr = peer.NeighborConf.Neighbor.NeighborAddress.String()
	bfdSession.Owner = "bgp"
	if peer.NeighborConf.RunningConf.BfdEnable {
		mgr.logger.Info(fmt.Sprintln("Bfd enabled on :", peer.NeighborConf.Neighbor.NeighborAddress))
		mgr.logger.Info(fmt.Sprintln("Creating BFD Session: ", bfdSession))
		ret, err := mgr.bfddClient.CreateBfdSession(bfdSession)
		if !ret {
			mgr.logger.Info(fmt.Sprintln("BfdSessionConfig FAILED, ret:", ret, "err:", err))
		} else {
			mgr.logger.Info("Bfd session configured")
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "up"
		}
	} else {
		if peer.NeighborConf.Neighbor.State.BfdNeighborState != "" {
			mgr.logger.Info(fmt.Sprintln("Bfd disabled on :",
				peer.NeighborConf.Neighbor.NeighborAddress))
			mgr.logger.Info(fmt.Sprintln("Deleting BFD Session: ", bfdSession))
			ret, err := mgr.bfddClient.DeleteBfdSession(bfdSession)
			if !ret {
				mgr.logger.Info(fmt.Sprintln("BfdSessionConfig FAILED, ret:",
					ret, "err:", err))
			} else {
				mgr.logger.Info(fmt.Sprintln("Bfd session removed for ",
					peer.NeighborConf.Neighbor.NeighborAddress))
				peer.NeighborConf.Neighbor.State.BfdNeighborState = ""
			}
		}
	}

}
