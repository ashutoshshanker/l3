package FSMgr

import (
	"bfdd"
	"encoding/json"
	"errors"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bfd/bfddCommonDefs"
	"l3/bgp/fsm"
	"l3/bgp/rpc"
	"l3/bgp/server"
	"utils/logging"
)

/*  Init bfd manager with bfd client as its core
 */
func NewFSBfdMgr(logger *logging.Writer, fileName string) (*FSBfdMgr, error) {
	var bfddClient *bfdd.BFDDServicesClient = nil
	bfddClientChan := make(chan *bfdd.BFDDServicesClient)

	logger.Info("Connecting to BFDd")
	go rpc.StartBfddClient(logger, fileName, bfddClientChan)
	bfddClient = <-bfddClientChan
	if bfddClient == nil {
		logger.Err("Failed to connect to BFDd\n")
		return nil, errors.New("Failed to connect to BFDd")
	} else {
		logger.Info("Connected to BFDd")
	}
	mgr := &FSBfdMgr{
		plugin:     "ovsdb",
		logger:     logger,
		bfddClient: bfddClient,
	}

	return mgr, nil
}

/*  Do any necessary init. Called from server..
 */
func (mgr *FSBfdMgr) Init(server *server.BGPServer) {
	// create bfd sub socket listener
	mgr.bfdSubSocket, _ = mgr.SetupSubSocket(bfddCommonDefs.PUB_SOCKET_ADDR)
	mgr.Server = server
	go mgr.listenForBFDNotifications()
}

/*  Listen for any BFD notifications
 */
func (mgr *FSBfdMgr) listenForBFDNotifications() {
	for {
		mgr.logger.Info("Read on BFD subscriber socket...")
		rxBuf, err := mgr.bfdSubSocket.Recv(0)
		if err != nil {
			mgr.logger.Err(fmt.Sprintln("Recv on BFD subscriber socket failed with error:", err))
			continue
		}
		mgr.logger.Info(fmt.Sprintln("BFD subscriber recv returned:", rxBuf))
		mgr.handleBfdNotifications(rxBuf)
	}
}

func (mgr *FSBfdMgr) handleBfdNotifications(rxBuf []byte) {
	bfd := bfddCommonDefs.BfddNotifyMsg{}
	err := json.Unmarshal(rxBuf, &bfd)
	if err != nil {
		mgr.logger.Err(fmt.Sprintf("Unmarshal BFD notification failed with err %s", err))
	}
	if peer, ok := mgr.Server.PeerMap[bfd.DestIp]; ok {
		if !bfd.State && peer.NeighborConf.Neighbor.State.BfdNeighborState == "up" {
			peer.Command(int(fsm.BGPEventManualStop), fsm.BGPCmdReasonNone)
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "down"
		}
		if bfd.State && peer.NeighborConf.Neighbor.State.BfdNeighborState == "down" {
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "up"
			peer.Command(int(fsm.BGPEventManualStart), fsm.BGPCmdReasonNone)
		}
		mgr.logger.Info(fmt.Sprintln("Bfd state of peer ",
			peer.NeighborConf.Neighbor.NeighborAddress, " is ",
			peer.NeighborConf.Neighbor.State.BfdNeighborState))
	}
}

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
