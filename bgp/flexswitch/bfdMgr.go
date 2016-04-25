package FSMgr

import (
	"bfdd"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bfd/bfddCommonDefs"
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

func (mgr *FSBfdMgr) SetupSubSocket(address string) (*nanomsg.SubSocket, error) {
	var err error
	var socket *nanomsg.SubSocket
	if socket, err = nanomsg.NewSubSocket(); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to create subscribe socket %s, error:%s", address, err))
		return nil, err
	}

	if err = socket.Subscribe(""); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to subscribe to \"\" on subscribe socket %s, error:%s",
			address, err))
		return nil, err
	}

	if _, err = socket.Connect(address); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to connect to publisher socket %s, error:%s", address, err))
		return nil, err
	}

	mgr.logger.Info(fmt.Sprintf("Connected to publisher socker %s", address))
	if err = socket.SetRecvBuffer(1024 * 1024); err != nil {
		mgr.logger.Err(fmt.Sprintln("Failed to set the buffer size for subsriber socket %s, error:",
			address, err))
		return nil, err
	}
	return socket, nil
}

func (mgr *FSBfdMgr) Init() {
	// create bfd sub socket listener
	bfdSubSocketCh := make(chan []byte)
	bfdSubSocketErrCh := make(chan error)
	bfdSubSocket, _ := mgr.SetupSubSocket(bfddCommonDefs.PUB_SOCKET_ADDR)
	for {
		mgr.logger.Info("Read on BFD subscriber socket...")
		rxBuf, err := bfdSubSocket.Recv(0)
		if err != nil {
			mgr.logger.Err(fmt.Sprintln("Recv on BFD subscriber socket failed with error:", err))
			bfdSubSocketErrCh <- err
			continue
		}
		mgr.logger.Info(fmt.Sprintln("BFD subscriber recv returned:", rxBuf))
		bfdSubSocketCh <- rxBuf
	}
}
