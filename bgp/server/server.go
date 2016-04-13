// server.go
package server

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"bfdd"
	"bytes"
	"encoding/json"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"l3/bgp/config"
	"l3/bgp/fsm"
	"l3/bgp/packet"
	bgppolicy "l3/bgp/policy"
	bgprib "l3/bgp/rib"
	"l3/rib/ribdCommonDefs"
	"net"
	"ribd"
	"ribdInt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"utils/logging"
	utilspolicy "utils/policy"
	"utils/policy/policyCommonDefs"

	nanomsg "github.com/op/go-nanomsg"
)

type PeerUpdate struct {
	OldPeer config.NeighborConfig
	NewPeer config.NeighborConfig
	AttrSet []bool
}

type PeerGroupUpdate struct {
	OldGroup config.PeerGroupConfig
	NewGroup config.PeerGroupConfig
	AttrSet  []bool
}

type AggUpdate struct {
	OldAgg  config.BGPAggregate
	NewAgg  config.BGPAggregate
	AttrSet []bool
}

type IfState struct {
	idx    int32
	ipaddr string
	state  uint8
}

type PolicyParams struct {
	CreateType      int
	DeleteType      int
	route           *bgprib.Route
	dest            *bgprib.Destination
	updated         *(map[*bgprib.Path][]*bgprib.Destination)
	withdrawn       *([]*bgprib.Destination)
	updatedAddPaths *([]*bgprib.Destination)
}

type BGPServer struct {
	logger           *logging.Writer
	bgpPE            *bgppolicy.BGPPolicyEngine
	ribdClient       *ribd.RIBDServicesClient
	AsicdClient      *asicdServices.ASICDServicesClient
	bfddClient       *bfdd.BFDDServicesClient
	BgpConfig        config.Bgp
	GlobalConfigCh   chan config.GlobalConfig
	AddPeerCh        chan PeerUpdate
	RemPeerCh        chan string
	AddPeerGroupCh   chan PeerGroupUpdate
	RemPeerGroupCh   chan string
	AddAggCh         chan AggUpdate
	RemAggCh         chan string
	PeerFSMConnCh    chan fsm.PeerFSMConn
	PeerConnEstCh    chan string
	PeerConnBrokenCh chan string
	PeerCommandCh    chan config.PeerCommand
	ReachabilityCh   chan config.ReachabilityInfo
	BGPPktSrcCh      chan *packet.BGPPktSrc

	NeighborMutex  sync.RWMutex
	PeerMap        map[string]*Peer
	Neighbors      []*Peer
	AdjRib         *bgprib.AdjRib
	connRoutesPath *bgprib.Path
	ifacePeerMap   map[int32][]string
	ifaceIP        net.IP
	actionFuncMap  map[int]bgppolicy.PolicyActionFunc
	addPathCount   int
}

func NewBGPServer(logger *logging.Writer, policyEngine *bgppolicy.BGPPolicyEngine, ribdClient *ribd.RIBDServicesClient,
	bfddClient *bfdd.BFDDServicesClient, asicdClient *asicdServices.ASICDServicesClient) *BGPServer {
	bgpServer := &BGPServer{}
	bgpServer.logger = logger
	bgpServer.bgpPE = policyEngine
	bgpServer.ribdClient = ribdClient
	bgpServer.bfddClient = bfddClient
	bgpServer.AsicdClient = asicdClient
	bgpServer.BgpConfig = config.Bgp{}
	bgpServer.GlobalConfigCh = make(chan config.GlobalConfig)
	bgpServer.AddPeerCh = make(chan PeerUpdate)
	bgpServer.RemPeerCh = make(chan string)
	bgpServer.AddPeerGroupCh = make(chan PeerGroupUpdate)
	bgpServer.RemPeerGroupCh = make(chan string)
	bgpServer.AddAggCh = make(chan AggUpdate)
	bgpServer.RemAggCh = make(chan string)
	bgpServer.PeerFSMConnCh = make(chan fsm.PeerFSMConn, 50)
	bgpServer.PeerConnEstCh = make(chan string)
	bgpServer.PeerConnBrokenCh = make(chan string)
	bgpServer.PeerCommandCh = make(chan config.PeerCommand)
	bgpServer.ReachabilityCh = make(chan config.ReachabilityInfo)
	bgpServer.BGPPktSrcCh = make(chan *packet.BGPPktSrc)
	bgpServer.NeighborMutex = sync.RWMutex{}
	bgpServer.PeerMap = make(map[string]*Peer)
	bgpServer.Neighbors = make([]*Peer, 0)
	bgpServer.AdjRib = bgprib.NewAdjRib(logger, ribdClient, &bgpServer.BgpConfig.Global.Config)
	bgpServer.ifacePeerMap = make(map[int32][]string)
	bgpServer.ifaceIP = nil
	bgpServer.actionFuncMap = make(map[int]bgppolicy.PolicyActionFunc)
	bgpServer.addPathCount = 0
	//bgpServer.actionFuncMap[ribdCommonDefs.PolicyActionTypeAggregate] = make([2]policy.ApplyActionFunc)

	var aggrActionFunc bgppolicy.PolicyActionFunc
	aggrActionFunc.ApplyFunc = bgpServer.ApplyAggregateAction
	aggrActionFunc.UndoFunc = bgpServer.UndoAggregateAction

	bgpServer.actionFuncMap[policyCommonDefs.PolicyActionTypeAggregate] = aggrActionFunc

	bgpServer.logger.Info(fmt.Sprintf("BGPServer: actionfuncmap=%v", bgpServer.actionFuncMap))
	bgpServer.bgpPE.SetEntityUpdateFunc(bgpServer.UpdateRouteAndPolicyDB)
	bgpServer.bgpPE.SetIsEntityPresentFunc(bgpServer.DoesRouteExist)
	bgpServer.bgpPE.SetActionFuncs(bgpServer.actionFuncMap)
	bgpServer.bgpPE.SetTraverseFuncs(bgpServer.TraverseAndApplyBGPRib, bgpServer.TraverseAndReverseBGPRib)

	return bgpServer
}

func (server *BGPServer) listenForPeers(acceptCh chan *net.TCPConn) {
	addr := ":" + config.BGPPort
	server.logger.Info(fmt.Sprintf("Listening for incomig connections on %s\n", addr))
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("ResolveTCPAddr failed with", err))
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		server.logger.Info(fmt.Sprintln("ListenTCP failed with", err))
	}

	for {
		server.logger.Info(fmt.Sprintln("Waiting for peer connections..."))
		tcpConn, err := listener.AcceptTCP()
		if err != nil {
			server.logger.Info(fmt.Sprintln("AcceptTCP failed with", err))
			continue
		}
		server.logger.Info(fmt.Sprintln("Got a peer connection from %s", tcpConn.RemoteAddr()))
		acceptCh <- tcpConn
	}
}

func (server *BGPServer) setupSubSocket(address string) (*nanomsg.SubSocket, error) {
	var err error
	var socket *nanomsg.SubSocket
	if socket, err = nanomsg.NewSubSocket(); err != nil {
		server.logger.Err(fmt.Sprintf("Failed to create subscribe socket %s, error:%s", address, err))
		return nil, err
	}

	if err = socket.Subscribe(""); err != nil {
		server.logger.Err(fmt.Sprintf("Failed to subscribe to \"\" on subscribe socket %s, error:%s", address, err))
		return nil, err
	}

	if _, err = socket.Connect(address); err != nil {
		server.logger.Err(fmt.Sprintf("Failed to connect to publisher socket %s, error:%s", address, err))
		return nil, err
	}

	server.logger.Info(fmt.Sprintf("Connected to publisher socker %s", address))
	if err = socket.SetRecvBuffer(1024 * 1024); err != nil {
		server.logger.Err(fmt.Sprintln("Failed to set the buffer size for subsriber socket %s, error:", address, err))
		return nil, err
	}
	return socket, nil
}

func (server *BGPServer) listenForRIBUpdates(socket *nanomsg.SubSocket, socketCh chan []byte, socketErrCh chan error) {
	for {
		server.logger.Info("Read on RIB subscriber socket...")
		rxBuf, err := socket.Recv(0)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Recv on RIB subscriber socket failed with error:", err))
			socketErrCh <- err
			continue
		}
		server.logger.Info(fmt.Sprintln("RIB subscriber recv returned:", rxBuf))
		socketCh <- rxBuf
	}
}

func (server *BGPServer) listenForBFDNotifications(socket *nanomsg.SubSocket, socketCh chan []byte, socketErrCh chan error) {
	for {
		server.logger.Info("Read on BFD subscriber socket...")
		rxBuf, err := socket.Recv(0)
		if err != nil {
			server.logger.Err(fmt.Sprintln("Recv on BFD subscriber socket failed with error:", err))
			socketErrCh <- err
			continue
		}
		server.logger.Info(fmt.Sprintln("BFD subscriber recv returned:", rxBuf))
		socketCh <- rxBuf
	}
}

func (server *BGPServer) handleRibUpdates(rxBuf []byte) {
	var routeListInfo ribdCommonDefs.RoutelistInfo
	routes := make([]*ribdInt.Routes, 0)
	reader := bytes.NewReader(rxBuf)
	decoder := json.NewDecoder(reader)
	msg := ribdCommonDefs.RibdNotifyMsg{}
	for err := decoder.Decode(&msg); err == nil; err = decoder.Decode(&msg) {
		err = json.Unmarshal(msg.MsgBuf, &routeListInfo)
		if err != nil {
			server.logger.Err(fmt.Sprintf("Unmarshal RIB route update failed with err %s", err))
		}
		server.logger.Info(fmt.Sprintln("Remove connected route, dest:", routeListInfo.RouteInfo.Ipaddr, "netmask:", routeListInfo.RouteInfo.Mask, "nexthop:", routeListInfo.RouteInfo.NextHopIp))
		routes = append(routes, &routeListInfo.RouteInfo)
	}

	if len(routes) > 0 {
		if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_CREATED {
			server.ProcessConnectedRoutes(routes, make([]*ribdInt.Routes, 0))
		} else if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
			server.ProcessConnectedRoutes(make([]*ribdInt.Routes, 0), routes)
		} else {
			server.logger.Err(fmt.Sprintf("**** Received RIB update with unknown type %d ****", msg.MsgType))
		}
	} else {
		server.logger.Err(fmt.Sprintf("**** Received RIB update type %d with no routes ****", msg.MsgType))
	}
}

func (server *BGPServer) handleBfdNotifications(rxBuf []byte) {
	bfd := bfddCommonDefs.BfddNotifyMsg{}
	err := json.Unmarshal(rxBuf, &bfd)
	if err != nil {
		server.logger.Err(fmt.Sprintf("Unmarshal BFD notification failed with err %s", err))
	}
	if peer, ok := server.PeerMap[bfd.DestIp]; ok {
		if !bfd.State && peer.NeighborConf.Neighbor.State.BfdNeighborState == "up" {
			//peer.StopFSM("Peer BFD Down")
			peer.Command(int(fsm.BGPEventManualStop))
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "down"
		}
		if bfd.State && peer.NeighborConf.Neighbor.State.BfdNeighborState == "down" {
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "up"
			peer.Command(int(fsm.BGPEventManualStart))
		}
		server.logger.Info(fmt.Sprintln("Bfd state of peer ", peer.NeighborConf.Neighbor.NeighborAddress, " is ", peer.NeighborConf.Neighbor.State.BfdNeighborState))
	}
}

func (server *BGPServer) listenForAsicdEvents(socket *nanomsg.SubSocket, ifStateCh chan IfState) {
	for {
		server.logger.Info("Read on Asicd subscriber socket...")
		rxBuf, err := socket.Recv(0)
		if err != nil {
			server.logger.Info(fmt.Sprintln("Error in receiving Asicd events", err))
			return
		}

		server.logger.Info(fmt.Sprintln("Asicd subscriber recv returned", rxBuf))
		event := asicdConstDefs.AsicdNotification{}
		err = json.Unmarshal(rxBuf, &event)
		if err != nil {
			server.logger.Err(fmt.Sprintf("Unmarshal Asicd event failed with err %s", err))
			return
		}

		switch event.MsgType {
		case asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE:
			var msg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(event.Msg, &msg)
			if err != nil {
				server.logger.Err(fmt.Sprintf("Unmarshal Asicd L3INTF event failed with err %s", err))
				return
			}

			server.logger.Info(fmt.Sprintf("Asicd L3INTF event idx %d ip %s state %d\n", msg.IfIndex, msg.IpAddr,
				msg.IfState))
			ifStateCh <- IfState{msg.IfIndex, msg.IpAddr, msg.IfState}
		}
	}
}

func (server *BGPServer) IsPeerLocal(peerIp string) bool {
	return server.PeerMap[peerIp].NeighborConf.RunningConf.PeerAS == server.BgpConfig.Global.Config.AS
}

func (server *BGPServer) SendUpdate(updated map[*bgprib.Path][]*bgprib.Destination, withdrawn []*bgprib.Destination, withdrawPath *bgprib.Path,
	updatedAddPaths []*bgprib.Destination) {
	for _, peer := range server.PeerMap {
		peer.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
	}
}

type ActionCbInfo struct {
	dest      *bgprib.Destination
	updated   *(map[*bgprib.Path][]*bgprib.Destination)
	withdrawn *([]*bgprib.Destination)
}

func (server *BGPServer) DoesRouteExist(params interface{}) bool {
	policyParams := params.(PolicyParams)
	dest := policyParams.dest
	if dest == nil {
		server.logger.Info(fmt.Sprintln("BGPServer:DoesRouteExist - dest not found for ip",
			policyParams.route.BGPRoute.Network, "prefix length", policyParams.route.BGPRoute.CIDRLen))
		return false
	}

	locRibRoute := dest.GetLocRibPathRoute()
	if policyParams.route == locRibRoute {
		return true
	}

	return false
}

func (server *BGPServer) getAggPrefix(conditionsList []interface{}) *packet.IPPrefix {
	server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix"))
	var ipPrefix *packet.IPPrefix
	var err error
	for _, condition := range conditionsList {
		/*
			server.logger.Info(fmt.Sprintf("BGPServer:getAggPrefix - Find policy condition name %s in the condition database\n", condition))
			conditionItem := server.bgpPE.PolicyEngine.PolicyConditionsDB.Get(patriciaDB.Prefix(condition))
			if conditionItem == nil {
				server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - Did not find condition ", condition, " in the condition database"))
				continue
			}
			conditionInfo := conditionItem.(utilspolicy.PolicyCondition)
			server.logger.Info(fmt.Sprintf("BGPServer:getAggPrefix - policy condition type %d\n", conditionInfo.ConditionType))
		*/
		switch condition.(type) {
		case utilspolicy.MatchPrefixConditionInfo:
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - PolicyConditionTypeDstIpPrefixMatch case"))
			matchPrefix := condition.(utilspolicy.MatchPrefixConditionInfo)
			//condition := conditionInfo.ConditionInfo.(utilspolicy.MatchPrefixConditionInfo)
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - exact prefix match conditiontype"))
			ipPrefix, err = packet.ConstructIPPrefixFromCIDR(matchPrefix.Prefix.IpPrefix)
			if err != nil {
				server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - ipPrefix invalid "))
				return nil
			}
			break
		default:
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - Not a known condition type"))
			break
		}
	}
	return ipPrefix
}

func (server *BGPServer) setUpdatedAddPaths(policyParams *PolicyParams, updatedAddPaths []*bgprib.Destination) {
	if len(updatedAddPaths) > 0 {
		addPathsMap := make(map[*bgprib.Destination]bool)
		for _, dest := range *(policyParams.updatedAddPaths) {
			addPathsMap[dest] = true
		}

		for _, dest := range updatedAddPaths {
			if !addPathsMap[dest] {
				(*policyParams.updatedAddPaths) = append((*policyParams.updatedAddPaths), dest)
			}
		}
	}
}

func (server *BGPServer) setWithdrawnWithAggPaths(policyParams *PolicyParams, withdrawn []*bgprib.Destination,
	sendSummaryOnly bool, updatedAddPaths []*bgprib.Destination) {
	destMap := make(map[*bgprib.Destination]bool)
	for _, dest := range *policyParams.withdrawn {
		destMap[dest] = true
	}

	aggDestMap := make(map[*bgprib.Destination]bool)
	for _, aggDestination := range withdrawn {
		aggDestMap[aggDestination] = true
		if !destMap[aggDestination] {
			server.logger.Info(fmt.Sprintf("setWithdrawnWithAggPaths: add agg dest %+v to withdrawn\n", aggDestination.IPPrefix.Prefix))
			(*policyParams.withdrawn) = append((*policyParams.withdrawn), aggDestination)
		}
	}

	/*
		for _, dest := range withdrawn {
			if !destMap[dest] {
				(*policyParams.withdrawn) = append((*policyParams.withdrawn), dest)
			}
		}
	*/

	// There will be only one destination per aggregated path.
	// So, break out of the loop as soon as we find it.
	for path, destinations := range *policyParams.updated {
		//dirty := false
		for idx, dest := range destinations {
			if aggDestMap[dest] {
				//(*actionCbInfo.updated)[path][idx] = (*actionCbInfo.updated)[path][len(destinations)-1]
				//(*actionCbInfo.updated)[path][len(destinations)-1] = nil
				//(*actionCbInfo.updated)[path] = (*actionCbInfo.updated)[path][:len(destinations)-1]
				(*policyParams.updated)[path][idx] = nil
				server.logger.Info(fmt.Sprintf("setWithdrawnWithAggPaths: remove dest %+v from withdrawn\n", dest.IPPrefix.Prefix))
				//dirty = true
			}
		}
		/*
			if dirty {
				lastIdx := len(destinations) - 1
				var movIdx int
				for idx := 0; idx <= lastIdx; idx++ {
					if destinations[idx] == nil {
						for movIdx = lastIdx; movIdx > idx && destinations[movIdx] == nil; movIdx-- {
						}
						if movIdx <= idx {
							lastIdx = movIdx - 1
							break
						}
						(*policyParams.updated)[path][idx] = (*policyParams.updated)[path][movIdx]
						(*policyParams.updated)[path][movIdx] = nil
						lastIdx = movIdx - 1
					}
				}
				(*policyParams.updated)[path] = (*policyParams.updated)[path][:lastIdx+1]
			}
		*/
	}

	if sendSummaryOnly {
		if policyParams.DeleteType == utilspolicy.Valid {
			for idx, dest := range *policyParams.withdrawn {
				if dest == policyParams.dest {
					server.logger.Info(fmt.Sprintf("setWithdrawnWithAggPaths: remove dest %+v from withdrawn\n", dest.IPPrefix.Prefix))
					(*policyParams.withdrawn)[idx] = nil
				}
			}
		} else if policyParams.CreateType == utilspolicy.Invalid {
			if policyParams.dest != nil && policyParams.dest.LocRibPath != nil {
				found := false
				if destinations, ok := (*policyParams.updated)[policyParams.dest.LocRibPath]; ok {
					for _, dest := range destinations {
						if dest == policyParams.dest {
							found = true
						}
					}
				} else {
					(*policyParams.updated)[policyParams.dest.LocRibPath] = make([]*bgprib.Destination, 0)
				}
				if !found {
					server.logger.Info(fmt.Sprintf("setWithdrawnWithAggPaths: add dest %+v to update\n", policyParams.dest.IPPrefix.Prefix))
					(*policyParams.updated)[policyParams.dest.LocRibPath] = append((*policyParams.updated)[policyParams.dest.LocRibPath], policyParams.dest)
				}
			}
		}
	}
	//(*policyParams.withdrawn) = append((*policyParams.withdrawn), withdrawn...)

	server.setUpdatedAddPaths(policyParams, updatedAddPaths)
}

func (server *BGPServer) setUpdatedWithAggPaths(policyParams *PolicyParams, updated map[*bgprib.Path][]*bgprib.Destination,
	sendSummaryOnly bool, ipPrefix *packet.IPPrefix, updatedAddPaths []*bgprib.Destination) {
	var routeDest *bgprib.Destination
	var ok bool
	if routeDest, ok = server.AdjRib.GetDest(ipPrefix, false); !ok {
		server.logger.Err(fmt.Sprintln("setUpdatedWithAggPaths: Did not find destination for ip", ipPrefix))
		if policyParams.dest != nil {
			routeDest = policyParams.dest
		} else {
			sendSummaryOnly = false
		}
	}

	withdrawMap := make(map[*bgprib.Destination]bool, len(*policyParams.withdrawn))
	if sendSummaryOnly {
		for _, dest := range *policyParams.withdrawn {
			withdrawMap[dest] = true
		}
	}

	//foundRouteDest := false
	for aggPath, aggDestinations := range updated {
		/*
			foundAggDest := false
			aggDestMap := make(map[*bgprib.Destination]bool)
			for _, dest := range aggDestinations {
				aggDestMap[dest] = true
			}
		*/

		destMap := make(map[*bgprib.Destination]bool)
		if _, ok := (*policyParams.updated)[aggPath]; !ok {
			(*policyParams.updated)[aggPath] = make([]*bgprib.Destination, 0)
		} else {
			for _, dest := range (*policyParams.updated)[aggPath] {
				destMap[dest] = true
			}
		}

		for _, dest := range aggDestinations {
			if !destMap[dest] {
				server.logger.Info(fmt.Sprintf("setUpdatedWithAggPaths: add agg dest %+v to updated\n", dest.IPPrefix.Prefix))
				(*policyParams.updated)[aggPath] = append((*policyParams.updated)[aggPath], dest)
			}
		}

		if sendSummaryOnly {
			if policyParams.CreateType == utilspolicy.Valid {
				for path, destinations := range *policyParams.updated {
					for idx, dest := range destinations {
						if routeDest == dest {
							(*policyParams.updated)[path][idx] = nil
							server.logger.Info(fmt.Sprintf("setUpdatedWithAggPaths: summaryOnly, remove dest %+v from updated\n", dest.IPPrefix.Prefix))
						}
					}
				}
			} else if policyParams.DeleteType == utilspolicy.Invalid {
				if !withdrawMap[routeDest] {
					server.logger.Info(fmt.Sprintf("setUpdatedWithAggPaths: summaryOnly, add dest %+v to withdrawn\n", routeDest.IPPrefix.Prefix))
					(*policyParams.withdrawn) = append((*policyParams.withdrawn), routeDest)
				}
			}
		}

		/*
			// There will be only one destination per aggregated path.
			// So, break out of the loop as soon as we find it.
			for path, destinations := range *policyParams.updated {
				for idx, dest := range destinations {
					if sendSummaryOnly && routeDest == dest {
						//(*policyParams.updated)[path][idx] = (*policyParams.updated)[path][len(destinations)-1]
						//(*policyParams.updated)[path][len(destinations)-1] = nil
						//(*policyParams.updated)[path] = (*policyParams.updated)[path][:len(destinations)-1]
						(*policyParams.updated)[path][idx] = nil
						foundRouteDest = true
						continue
					}
					if _, ok = aggDestMap[dest]; ok {
						(*policyParams.updated)[path][idx] = (*policyParams.updated)[path][len(destinations)-1]
						(*policyParams.updated)[path][len(destinations)-1] = nil
						(*policyParams.updated)[path] = (*policyParams.updated)[path][:len(destinations)-1]
						foundAggDest = true
						break
					}
				}
				if foundAggDest && foundRouteDest {
					break
				}
			}

			(*policyParams.updated)[aggPath] = make([]*bgprib.Destination, 0)
			(*policyParams.updated)[aggPath] = append((*policyParams.updated)[aggPath], aggDestinations...)

			if sendSummaryOnly {
				aggDestMap = make(map[*bgprib.Destination]bool)
				for _, dest := range *policyParams.withdrawn {
					aggDestMap[dest] = true
				}

				for _, dest := range aggDestinations {
					for _, singleDest := range dest.aggregatedDestMap {
						if !aggDestMap[singleDest] {
							(*policyParams.withdrawn) = append((*policyParams.withdrawn), singleDest)
						}
					}
				}
			}
		*/
	}

	server.setUpdatedAddPaths(policyParams, updatedAddPaths)
}

//func (server *BGPServer) UndoAggregateAction(route *bgpd.BGPRoute, conditionList []string, action interface{}, params interface{}, ctx interface{}) {
func (server *BGPServer) UndoAggregateAction(actionInfo interface{}, conditionList []interface{}, params interface{},
	policyStmt utilspolicy.PolicyStmt) {
	policyParams := params.(PolicyParams)
	ipPrefix := packet.NewIPPrefix(net.ParseIP(policyParams.route.BGPRoute.Network),
		uint8(policyParams.route.BGPRoute.CIDRLen))
	aggPrefix := server.getAggPrefix(conditionList)
	//actions := actionInfo.(utilspolicy.PolicyAggregateActionInfo)
	aggActions := actionInfo.(utilspolicy.PolicyAggregateActionInfo)
	bgpAgg := config.BGPAggregate{
		GenerateASSet:   aggActions.GenerateASSet,
		SendSummaryOnly: aggActions.SendSummaryOnly,
	}
	//allUpdated := make(map[*bgprib.Path][]*bgprib.Destination, 10)
	//allWithdrawn := make([]*bgprib.Destination, 0)

	server.logger.Info(fmt.Sprintf("UndoAggregateAction: ipPrefix=%+v, aggPrefix=%+v\n", ipPrefix.Prefix, aggPrefix.Prefix))
	var updated map[*bgprib.Path][]*bgprib.Destination
	var withdrawn []*bgprib.Destination
	var updatedAddPaths []*bgprib.Destination
	var origDest *bgprib.Destination
	//var actionCbInfo ActionCbInfo
	//var ctxOk bool
	if policyParams.dest != nil {
		origDest = policyParams.dest
	}
	updated, withdrawn, _, updatedAddPaths = server.AdjRib.RemoveRouteFromAggregate(ipPrefix, aggPrefix,
		server.BgpConfig.Global.Config.RouterId.String(), &bgpAgg, origDest, server.addPathCount)

	/*
		if !ctxOk {
			actionCbInfo = ActionCbInfo{
				updated:   &allUpdated,
				withdrawn: &allWithdrawn,
			}
		}
	*/

	server.logger.Info(fmt.Sprintf("UndoAggregateAction: aggregate result update=%+v, withdrawn=%+v\n", updated, withdrawn))
	//server.setUpdatedWithAggPaths(&policyParams, updated, aggActions.SendSummaryOnly, ipPrefix)
	server.setWithdrawnWithAggPaths(&policyParams, withdrawn, aggActions.SendSummaryOnly, updatedAddPaths)
	server.logger.Info(fmt.Sprintf("UndoAggregateAction: after updating withdraw agg paths, update=%+v, withdrawn=%+v, policyparams.update=%+v, policyparams.withdrawn=%+v\n",
		updated, withdrawn, *policyParams.updated, *policyParams.withdrawn))
	//server.SendUpdate(allUpdated, allWithdrawn, nil)
	return
}

//func (server *BGPServer) ApplyAggregateAction(route *bgpd.BGPRoute, conditionList []string, action interface{}, params interface{}, ctx interface{}) {
func (server *BGPServer) ApplyAggregateAction(actionInfo interface{}, conditionInfo []interface{}, params interface{}) {
	policyParams := params.(PolicyParams)
	ipPrefix := packet.NewIPPrefix(net.ParseIP(policyParams.route.BGPRoute.Network),
		uint8(policyParams.route.BGPRoute.CIDRLen))
	//conditionList := conditionInfo.([]string)
	aggPrefix := server.getAggPrefix(conditionInfo)
	//routeParams := params.(policy.RouteParams)
	//actions := actionInfo.(utilspolicy.PolicyAction.ActionInfo)
	aggActions := actionInfo.(utilspolicy.PolicyAggregateActionInfo)
	bgpAgg := config.BGPAggregate{
		GenerateASSet:   aggActions.GenerateASSet,
		SendSummaryOnly: aggActions.SendSummaryOnly,
	}

	server.logger.Info(fmt.Sprintf("ApplyAggregateAction: ipPrefix=%+v, aggPrefix=%+v\n", ipPrefix.Prefix, aggPrefix.Prefix))
	var updated map[*bgprib.Path][]*bgprib.Destination
	var withdrawn []*bgprib.Destination
	var updatedAddPaths []*bgprib.Destination
	if (policyParams.CreateType == utilspolicy.Valid) || (policyParams.DeleteType == utilspolicy.Invalid) {
		server.logger.Info(fmt.Sprintf("ApplyAggregateAction: CreateType = Valid or DeleteType = Invalid\n"))
		updated, withdrawn, _, updatedAddPaths = server.AdjRib.AddRouteToAggregate(ipPrefix, aggPrefix,
			server.BgpConfig.Global.Config.RouterId.String(), server.ifaceIP, &bgpAgg, server.addPathCount)
	} else if policyParams.DeleteType == utilspolicy.Valid {
		server.logger.Info(fmt.Sprintf("ApplyAggregateAction: DeleteType = Valid\n"))
		origDest := policyParams.dest
		updated, withdrawn, _, updatedAddPaths = server.AdjRib.RemoveRouteFromAggregate(ipPrefix, aggPrefix,
			server.BgpConfig.Global.Config.RouterId.String(), &bgpAgg, origDest, server.addPathCount)
	}

	server.logger.Info(fmt.Sprintf("ApplyAggregateAction: aggregate result update=%+v, withdrawn=%+v\n", updated, withdrawn))
	server.setUpdatedWithAggPaths(&policyParams, updated, aggActions.SendSummaryOnly, ipPrefix, updatedAddPaths)
	server.logger.Info(fmt.Sprintf("ApplyAggregateAction: after updating agg paths, update=%+v, withdrawn=%+v, policyparams.update=%+v, policyparams.withdrawn=%+v\n",
		updated, withdrawn, *policyParams.updated, *policyParams.withdrawn))
	//server.setWithdrawnWithAggPaths(&policyParams, withdrawn, aggActions.SendSummaryOnly)
	return
}

func (server *BGPServer) checkForAggregation(updated map[*bgprib.Path][]*bgprib.Destination, withdrawn []*bgprib.Destination,
	withdrawPath *bgprib.Path, updatedAddPaths []*bgprib.Destination) (map[*bgprib.Path][]*bgprib.Destination, []*bgprib.Destination, *bgprib.Path,
	[]*bgprib.Destination) {
	server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - start, updated %v withdrawn %v\n", updated, withdrawn))

	for _, dest := range withdrawn {
		if dest == nil || dest.LocRibPath == nil || dest.LocRibPath.IsAggregate() {
			continue
		}

		route := dest.GetLocRibPathRoute()
		if route == nil {
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - route not found withdraw dest %s\n",
				dest.IPPrefix.Prefix.String()))
			continue
		}
		peEntity := utilspolicy.PolicyEngineFilterEntityParams{
			DestNetIp:  route.BGPRoute.Network + "/" + strconv.Itoa(int(route.BGPRoute.CIDRLen)),
			NextHopIp:  route.BGPRoute.NextHop,
			DeletePath: true,
		}
		server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - withdraw dest %s policylist %v hit %v before applying delete policy\n",
			dest.IPPrefix.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
		/*
			routeParams := policy.RouteParams{
				CreateType:    policy.Invalid,
				DeleteType:    policy.Valid,
				ActionFuncMap: server.actionFuncMap,
			}
			callbackInfo := ActionCbInfo{
				dest:      dest,
				updated:   &updated,
				withdrawn: &withdrawn,
			}
		*/
		callbackInfo := PolicyParams{
			CreateType:      utilspolicy.Invalid,
			DeleteType:      utilspolicy.Valid,
			route:           route,
			dest:            dest,
			updated:         &updated,
			withdrawn:       &withdrawn,
			updatedAddPaths: &updatedAddPaths,
		}
		server.bgpPE.PolicyEngine.PolicyEngineFilter(peEntity, policyCommonDefs.PolicyPath_Export, callbackInfo)
	}

	for _, destinations := range updated {
		server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update destinations %+v\n", destinations))
		for _, dest := range destinations {
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update dest %+v\n", dest.IPPrefix.Prefix))
			if dest == nil || dest.LocRibPath == nil || dest.LocRibPath.IsAggregate() {
				continue
			}
			route := dest.GetLocRibPathRoute()
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update dest %s policylist %v hit %v before applying create policy\n",
				dest.IPPrefix.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
			if route != nil {
				peEntity := utilspolicy.PolicyEngineFilterEntityParams{
					DestNetIp:  route.BGPRoute.Network + "/" + strconv.Itoa(int(route.BGPRoute.CIDRLen)),
					NextHopIp:  route.BGPRoute.NextHop,
					CreatePath: true,
				}
				/*
					routeParams := policy.RouteParams{
						CreateType:    policy.Valid,
						DeleteType:    policy.Invalid,
						ActionFuncMap: server.actionFuncMap,
					}
					callbackInfo := ActionCbInfo{
						updated:   &updated,
						withdrawn: &withdrawn,
					}
				*/
				callbackInfo := PolicyParams{
					CreateType:      utilspolicy.Valid,
					DeleteType:      utilspolicy.Invalid,
					route:           route,
					dest:            dest,
					updated:         &updated,
					withdrawn:       &withdrawn,
					updatedAddPaths: &updatedAddPaths,
				}
				server.bgpPE.PolicyEngine.PolicyEngineFilter(peEntity, policyCommonDefs.PolicyPath_Export, callbackInfo)
				server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update dest %s policylist %v hit %v after applying create policy\n",
					dest.IPPrefix.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
			}
		}
	}

	server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - complete, updated %v withdrawn %v\n", updated, withdrawn))
	return updated, withdrawn, withdrawPath, updatedAddPaths
}

func (server *BGPServer) UpdateRouteAndPolicyDB(policyDetails utilspolicy.PolicyDetails, params interface{}) {
	policyParams := params.(PolicyParams)
	/*
		route := ribd.Routes{Ipaddr: routeInfo.destNetIp, Mask: routeInfo.networkMask, NextHopIp: routeInfo.nextHopIp,
			NextHopIfType: ribd.Int(routeInfo.nextHopIfType), IfIndex: routeInfo.nextHopIfIndex, Metric: routeInfo.metric,
			Prototype: ribd.Int(routeInfo.routeType)}
	*/
	var op int
	if policyParams.DeleteType != bgppolicy.Invalid {
		op = bgppolicy.Del
	} else {
		if policyDetails.EntityDeleted == false {
			server.logger.Info(fmt.Sprintln("Reject action was not applied, so add this policy to the route"))
			op = bgppolicy.Add
			bgppolicy.UpdateRoutePolicyState(policyParams.route, op, policyDetails.Policy, policyDetails.PolicyStmt)
		}
		policyParams.route.PolicyHitCounter++
	}
	server.bgpPE.UpdatePolicyRouteMap(policyParams.route, policyDetails.Policy, op)
}

func (server *BGPServer) TraverseAndApplyBGPRib(data interface{}, updateFunc utilspolicy.PolicyApplyfunc) {
	server.logger.Info(fmt.Sprintf("BGPServer:TraverseRibForPolicies - start"))
	policy := data.(utilspolicy.Policy)
	updated := make(map[*bgprib.Path][]*bgprib.Destination, 10)
	withdrawn := make([]*bgprib.Destination, 0, 10)
	updatedAddPaths := make([]*bgprib.Destination, 0)
	locRib := server.AdjRib.GetLocRib()
	for path, destinations := range locRib {
		for _, dest := range destinations {
			if !path.IsAggregatePath() {
				/*
					callbackInfo := ActionCbInfo{
						dest:      dest,
						updated:   &updated,
						withdrawn: &withdrawn,
					}
				*/

				route := dest.GetLocRibPathRoute()
				if route == nil {
					continue
				}
				peEntity := utilspolicy.PolicyEngineFilterEntityParams{
					DestNetIp:  route.BGPRoute.Network + "/" + strconv.Itoa(int(route.BGPRoute.CIDRLen)),
					NextHopIp:  route.BGPRoute.NextHop,
					PolicyList: route.PolicyList,
				}
				callbackInfo := PolicyParams{
					route:           route,
					dest:            dest,
					updated:         &updated,
					withdrawn:       &withdrawn,
					updatedAddPaths: &updatedAddPaths,
				}

				updateFunc(peEntity, policy, callbackInfo)
			}
		}
	}
	server.logger.Info(fmt.Sprintf("BGPServer:TraverseRibForPolicies - updated %v withdrawn %v", updated, withdrawn))
	server.SendUpdate(updated, withdrawn, nil, updatedAddPaths)
}

func (server *BGPServer) TraverseAndReverseBGPRib(policyData interface{}) {
	policy := policyData.(utilspolicy.Policy)
	server.logger.Info(fmt.Sprintln("BGPServer:TraverseAndReverseBGPRib - policy", policy.Name))
	policyExtensions := policy.Extensions.(bgppolicy.PolicyExtensions)
	if len(policyExtensions.RouteList) == 0 {
		fmt.Println("No route affected by this policy, so nothing to do")
		return
	}

	updated := make(map[*bgprib.Path][]*bgprib.Destination, 10)
	withdrawn := make([]*bgprib.Destination, 0, 10)
	updatedAddPaths := make([]*bgprib.Destination, 0)
	var route *bgprib.Route
	for idx := 0; idx < len(policyExtensions.RouteInfoList); idx++ {
		route = policyExtensions.RouteInfoList[idx]
		dest := server.AdjRib.GetDestFromIPAndLen(route.BGPRoute.Network, uint32(route.BGPRoute.CIDRLen))
		callbackInfo := PolicyParams{
			route:           route,
			dest:            dest,
			updated:         &updated,
			withdrawn:       &withdrawn,
			updatedAddPaths: &updatedAddPaths,
		}
		peEntity := utilspolicy.PolicyEngineFilterEntityParams{
			DestNetIp: route.BGPRoute.Network + "/" + strconv.Itoa(int(route.BGPRoute.CIDRLen)),
			NextHopIp: route.BGPRoute.NextHop,
		}

		ipPrefix, err := bgppolicy.GetNetworkPrefixFromCIDR(route.BGPRoute.Network + "/" +
			strconv.Itoa(int(route.BGPRoute.CIDRLen)))
		if err != nil {
			server.logger.Info(fmt.Sprintln("Invalid route ", ipPrefix))
			continue
		}
		server.bgpPE.PolicyEngine.PolicyEngineUndoPolicyForEntity(peEntity, policy, callbackInfo)
		server.bgpPE.DeleteRoutePolicyState(route, policy.Name)
		server.bgpPE.PolicyEngine.DeletePolicyEntityMapEntry(peEntity, policy.Name)
	}
}

func (server *BGPServer) ProcessUpdate(pktInfo *packet.BGPPktSrc) {
	peer, ok := server.PeerMap[pktInfo.Src]
	if !ok {
		server.logger.Err(fmt.Sprintln("BgpServer:ProcessUpdate - Peer not found, address:", pktInfo.Src))
		return
	}

	atomic.AddUint32(&peer.NeighborConf.Neighbor.State.Queues.Input, ^uint32(0))
	peer.NeighborConf.Neighbor.State.Messages.Received.Update++
	updated, withdrawn, withdrawPath, updatedAddPaths := server.AdjRib.ProcessUpdate(peer.NeighborConf, pktInfo, server.addPathCount)
	updated, withdrawn, withdrawPath, updatedAddPaths = server.checkForAggregation(updated, withdrawn, withdrawPath,
		updatedAddPaths)
	server.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
}

func (server *BGPServer) convertDestIPToIPPrefix(routes []*ribdInt.Routes) []packet.NLRI {
	dest := make([]packet.NLRI, 0, len(routes))
	for _, r := range routes {
		server.logger.Info(fmt.Sprintln("Route NS : ", r.NetworkStatement, " Route Origin ", r.RouteOrigin))
		ipPrefix := packet.ConstructIPPrefix(r.Ipaddr, r.Mask)
		dest = append(dest, ipPrefix)
	}
	return dest
}

func (server *BGPServer) ProcessConnectedRoutes(installedRoutes []*ribdInt.Routes, withdrawnRoutes []*ribdInt.Routes) {
	server.logger.Info(fmt.Sprintln("valid routes:", installedRoutes, "invalid routes:", withdrawnRoutes))
	valid := server.convertDestIPToIPPrefix(installedRoutes)
	invalid := server.convertDestIPToIPPrefix(withdrawnRoutes)
	updated, withdrawn, withdrawPath, updatedAddPaths := server.AdjRib.ProcessConnectedRoutes(
		server.BgpConfig.Global.Config.RouterId.String(), server.connRoutesPath, valid, invalid, server.addPathCount)
	updated, withdrawn, withdrawPath, updatedAddPaths = server.checkForAggregation(updated, withdrawn, withdrawPath,
		updatedAddPaths)
	server.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
}

func (server *BGPServer) ProcessRoutesFromRIB() {
	var currMarker ribdInt.Int
	var count ribdInt.Int
	count = 100
	for {
		server.logger.Info(fmt.Sprintln("Getting ", count, " objects from currMarker", currMarker))
		getBulkInfo, err := server.ribdClient.GetBulkRoutesForProtocol("BGP", currMarker, count)
		if err != nil {
			server.logger.Info(fmt.Sprintln("GetBulkRoutesForProtocol with err ", err))
			return
		}
		if getBulkInfo.Count == 0 {
			server.logger.Info("0 objects returned from GetBulkRoutesForProtocol")
			return
		}
		server.logger.Info(fmt.Sprintln("len(getBulkInfo.RouteList)  = ", len(getBulkInfo.RouteList), " num objects returned = ", getBulkInfo.Count))
		server.ProcessConnectedRoutes(getBulkInfo.RouteList, make([]*ribdInt.Routes, 0))
		if getBulkInfo.More == false {
			server.logger.Info("more returned as false, so no more get bulks")
			return
		}
		currMarker = ribdInt.Int(getBulkInfo.EndIdx)
	}
}

func (server *BGPServer) ProcessRemoveNeighbor(peerIp string, peer *Peer) {
	updated, withdrawn, withdrawPath, updatedAddPaths := server.AdjRib.RemoveUpdatesFromNeighbor(peerIp,
		peer.NeighborConf, server.addPathCount)
	server.logger.Info(fmt.Sprintf("ProcessRemoveNeighbor - Neighbor %s, send updated paths %v, withdrawn paths %v\n", peerIp, updated, withdrawn))
	updated, withdrawn, withdrawPath, updatedAddPaths = server.checkForAggregation(updated, withdrawn, withdrawPath,
		updatedAddPaths)
	server.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
}

func (server *BGPServer) SendAllRoutesToPeer(peer *Peer) {
	withdrawn := make([]*bgprib.Destination, 0)
	updatedAddPaths := make([]*bgprib.Destination, 0)
	updated := server.AdjRib.GetLocRib()
	server.SendUpdate(updated, withdrawn, nil, updatedAddPaths)
}

func (server *BGPServer) RemoveRoutesFromAllNeighbor() {
	server.AdjRib.RemoveUpdatesFromAllNeighbors(server.addPathCount)
}

func (server *BGPServer) addPeerToList(peer *Peer) {
	server.Neighbors = append(server.Neighbors, peer)
}

func (server *BGPServer) removePeerFromList(peer *Peer) {
	for idx, item := range server.Neighbors {
		if item == peer {
			server.Neighbors[idx] = server.Neighbors[len(server.Neighbors)-1]
			server.Neighbors[len(server.Neighbors)-1] = nil
			server.Neighbors = server.Neighbors[:len(server.Neighbors)-1]
			break
		}
	}
}

func (server *BGPServer) StopPeersByGroup(groupName string) []*Peer {
	peers := make([]*Peer, 0)
	for peerIP, peer := range server.PeerMap {
		if peer.NeighborConf.Group.Name == groupName {
			server.logger.Info(fmt.Sprintln("Clean up peer", peerIP))
			peer.Cleanup()
			server.ProcessRemoveNeighbor(peerIP, peer)
			peers = append(peers, peer)

			runtime.Gosched()
		}
	}

	return peers
}

func (server *BGPServer) UpdatePeerGroupInPeers(groupName string, peerGroup *config.PeerGroupConfig) {
	peers := server.StopPeersByGroup(groupName)
	for _, peer := range peers {
		peer.UpdatePeerGroup(peerGroup)
		peer.Init()
	}
}

func (server *BGPServer) copyGlobalConf(gConf config.GlobalConfig) {
	server.BgpConfig.Global.Config.AS = gConf.AS
	server.BgpConfig.Global.Config.RouterId = gConf.RouterId
	server.BgpConfig.Global.Config.UseMultiplePaths = gConf.UseMultiplePaths
	server.BgpConfig.Global.Config.EBGPMaxPaths = gConf.EBGPMaxPaths
	server.BgpConfig.Global.Config.EBGPAllowMultipleAS = gConf.EBGPAllowMultipleAS
	server.BgpConfig.Global.Config.IBGPMaxPaths = gConf.IBGPMaxPaths
}

func (server *BGPServer) ProcessBfd(peer *Peer) {
	bfdSession := bfdd.NewBfdSession()
	bfdSession.IpAddr = peer.NeighborConf.Neighbor.NeighborAddress.String()
	bfdSession.Owner = "bgp"
	if peer.NeighborConf.RunningConf.BfdEnable {
		server.logger.Info(fmt.Sprintln("Bfd enabled on :", peer.NeighborConf.Neighbor.NeighborAddress))
		server.logger.Info(fmt.Sprintln("Creating BFD Session: ", bfdSession))
		ret, err := server.bfddClient.CreateBfdSession(bfdSession)
		if !ret {
			server.logger.Info(fmt.Sprintln("BfdSessionConfig FAILED, ret:", ret, "err:", err))
		} else {
			server.logger.Info("Bfd session configured")
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "up"
		}
	} else {
		if peer.NeighborConf.Neighbor.State.BfdNeighborState != "" {
			server.logger.Info(fmt.Sprintln("Bfd disabled on :", peer.NeighborConf.Neighbor.NeighborAddress))
			server.logger.Info(fmt.Sprintln("Deleting BFD Session: ", bfdSession))
			ret, err := server.bfddClient.DeleteBfdSession(bfdSession)
			if !ret {
				server.logger.Info(fmt.Sprintln("BfdSessionConfig FAILED, ret:", ret, "err:", err))
			} else {
				server.logger.Info(fmt.Sprintln("Bfd session removed for ", peer.NeighborConf.Neighbor.NeighborAddress))
				peer.NeighborConf.Neighbor.State.BfdNeighborState = ""
			}
		}
	}
}

func (server *BGPServer) getIfaceIP(ip string) {
	if server.ifaceIP != nil {
		return
	}
	reachInfo, err := server.ribdClient.GetRouteReachabilityInfo(ip)
	if err != nil {
		server.logger.Info(fmt.Sprintf("Server: Peer %s is not reachable", ip))
		return
	}
	netIP := net.ParseIP(reachInfo.Ipaddr)
	if netIP != nil {
		server.ifaceIP = netIP
	}
}

func (server *BGPServer) setInterfaceMapForPeer(peerIP string, peer *Peer) {
	server.logger.Info(fmt.Sprintln("Server: setInterfaceMapForPeer Peer", peer, "calling GetRouteReachabilityInfo"))
	reachInfo, err := server.ribdClient.GetRouteReachabilityInfo(peerIP)
	server.logger.Info(fmt.Sprintln("Server: setInterfaceMapForPeer Peer", peer, "GetRouteReachabilityInfo returned", reachInfo))
	if err != nil {
		server.logger.Info(fmt.Sprintf("Server: Peer %s is not reachable", peerIP))
	} else {
		ifIdx := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(reachInfo.NextHopIfIndex), int(reachInfo.NextHopIfType))
		server.logger.Info(fmt.Sprintf("Server: Peer %s IfIdx %d", peerIP, ifIdx))
		if _, ok := server.ifacePeerMap[ifIdx]; !ok {
			server.ifacePeerMap[ifIdx] = make([]string, 0)
		}
		server.ifacePeerMap[ifIdx] = append(server.ifacePeerMap[ifIdx], peerIP)
		peer.setIfIdx(ifIdx)
	}
}

func (server *BGPServer) clearInterfaceMapForPeer(peerIP string, peer *Peer) {
	ifIdx := peer.getIfIdx()
	server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection broken ifIdx %v", peerIP, ifIdx))
	if peerList, ok := server.ifacePeerMap[ifIdx]; ok {
		for idx, ip := range peerList {
			if ip == peerIP {
				server.ifacePeerMap[ifIdx] = append(server.ifacePeerMap[ifIdx][:idx], server.ifacePeerMap[ifIdx][idx+1:]...)
				if len(server.ifacePeerMap[ifIdx]) == 0 {
					delete(server.ifacePeerMap, ifIdx)
				}
				break
			}
		}
	}
	peer.setIfIdx(-1)
}

func (server *BGPServer) constructBGPGlobalState(gConf *config.GlobalConfig) {
	server.BgpConfig.Global.State.AS = gConf.AS
	server.BgpConfig.Global.State.RouterId = gConf.RouterId
	server.BgpConfig.Global.State.UseMultiplePaths = gConf.UseMultiplePaths
	server.BgpConfig.Global.State.EBGPMaxPaths = gConf.EBGPMaxPaths
	server.BgpConfig.Global.State.EBGPAllowMultipleAS = gConf.EBGPAllowMultipleAS
	server.BgpConfig.Global.State.IBGPMaxPaths = gConf.IBGPMaxPaths
}

func (server *BGPServer) StartServer() {
	gConf := <-server.GlobalConfigCh
	server.logger.Info(fmt.Sprintln("Recieved global conf:", gConf))
	server.BgpConfig.Global.Config = gConf
	server.constructBGPGlobalState(&gConf)
	server.BgpConfig.PeerGroups = make(map[string]*config.PeerGroup)

	pathAttrs := packet.ConstructPathAttrForConnRoutes(gConf.RouterId, gConf.AS)
	server.connRoutesPath = bgprib.NewPath(server.AdjRib, nil, pathAttrs, false, false, bgprib.RouteTypeConnected)

	server.logger.Info("Listen for RIBd updates")
	ribSubSocket, _ := server.setupSubSocket(ribdCommonDefs.PUB_SOCKET_ADDR)
	ribSubBGPSocket, _ := server.setupSubSocket(ribdCommonDefs.PUB_SOCKET_BGPD_ADDR)
	asicdL3IntfSubSocket, _ := server.setupSubSocket(asicdConstDefs.PUB_SOCKET_ADDR)
	bfdSubSocket, _ := server.setupSubSocket(bfddCommonDefs.PUB_SOCKET_ADDR)

	ribSubSocketCh := make(chan []byte)
	ribSubSocketErrCh := make(chan error)
	ribSubBGPSocketCh := make(chan []byte)
	ribSubBGPSocketErrCh := make(chan error)
	asicdL3IntfStateCh := make(chan IfState)
	bfdSubSocketCh := make(chan []byte)
	bfdSubSocketErrCh := make(chan error)

	server.logger.Info("Setting up Peer connections")
	acceptCh := make(chan *net.TCPConn)
	go server.listenForPeers(acceptCh)

	//	routes, _ := server.ribdClient.GetConnectedRoutesInfo()
	server.ProcessRoutesFromRIB()
	//	server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))

	go server.listenForRIBUpdates(ribSubSocket, ribSubSocketCh, ribSubSocketErrCh)
	go server.listenForRIBUpdates(ribSubBGPSocket, ribSubBGPSocketCh, ribSubBGPSocketErrCh)
	go server.listenForAsicdEvents(asicdL3IntfSubSocket, asicdL3IntfStateCh)
	go server.listenForBFDNotifications(bfdSubSocket, bfdSubSocketCh, bfdSubSocketErrCh)
	//go server.AdjRib.ProcessRIBdRouteRequests()

	for {
		select {
		case gConf = <-server.GlobalConfigCh:
			for peerIP, peer := range server.PeerMap {
				server.logger.Info(fmt.Sprintf("Cleanup peer %s", peerIP))
				peer.Cleanup()
			}
			server.logger.Info(fmt.Sprintf("Giving up CPU so that all peer FSMs will get cleaned up"))
			runtime.Gosched()

			packet.SetNextHopPathAttrs(server.connRoutesPath.PathAttrs, gConf.RouterId)
			server.RemoveRoutesFromAllNeighbor()
			server.copyGlobalConf(gConf)
			server.constructBGPGlobalState(&gConf)
			for _, peer := range server.PeerMap {
				peer.Init()
			}

		case peerUpdate := <-server.AddPeerCh:
			server.logger.Info("message received on AddPeerCh")
			oldPeer := peerUpdate.OldPeer
			newPeer := peerUpdate.NewPeer
			var peer *Peer
			var ok bool
			if oldPeer.NeighborAddress != nil {
				if peer, ok = server.PeerMap[oldPeer.NeighborAddress.String()]; ok {
					server.logger.Info(fmt.Sprintln("Clean up peer", oldPeer.NeighborAddress.String()))
					peer.Cleanup()
					server.ProcessRemoveNeighbor(oldPeer.NeighborAddress.String(), peer)
					peer.UpdateNeighborConf(newPeer, &server.BgpConfig)

					runtime.Gosched()
				} else {
					server.logger.Info(fmt.Sprintln("Can't find neighbor with old address",
						oldPeer.NeighborAddress.String()))
				}
			}

			if !ok {
				_, ok = server.PeerMap[newPeer.NeighborAddress.String()]
				if ok {
					server.logger.Info(fmt.Sprintln("Failed to add neighbor. Neighbor at that address already exists,",
						newPeer.NeighborAddress.String()))
					break
				}

				var groupConfig *config.PeerGroupConfig
				if newPeer.PeerGroup != "" {
					if group, ok := server.BgpConfig.PeerGroups[newPeer.PeerGroup]; !ok {
						server.logger.Info(fmt.Sprintln("Peer group", newPeer.PeerGroup, "not created yet, creating peer",
							newPeer.NeighborAddress.String(), "without the group"))
					} else {
						groupConfig = &group.Config
					}
				}
				server.logger.Info(fmt.Sprintln("Add neighbor, ip:", newPeer.NeighborAddress.String()))
				peer = NewPeer(server, &server.BgpConfig.Global.Config, groupConfig, newPeer)
				server.PeerMap[newPeer.NeighborAddress.String()] = peer
				server.NeighborMutex.Lock()
				server.addPeerToList(peer)
				server.NeighborMutex.Unlock()
			}
			server.ProcessBfd(peer)
			peer.Init()

		case remPeer := <-server.RemPeerCh:
			server.logger.Info(fmt.Sprintln("Remove Peer:", remPeer))
			peer, ok := server.PeerMap[remPeer]
			if !ok {
				server.logger.Info(fmt.Sprintln("Failed to remove peer. Peer at that address does not exist,", remPeer))
				break
			}
			server.NeighborMutex.Lock()
			server.removePeerFromList(peer)
			server.NeighborMutex.Unlock()
			delete(server.PeerMap, remPeer)
			peer.Cleanup()
			server.ProcessRemoveNeighbor(remPeer, peer)

		case groupUpdate := <-server.AddPeerGroupCh:
			oldGroupConf := groupUpdate.OldGroup
			newGroupConf := groupUpdate.NewGroup
			server.logger.Info(fmt.Sprintln("Peer group update old:", oldGroupConf, "new:", newGroupConf))
			var ok bool

			if oldGroupConf.Name != "" {
				if _, ok = server.BgpConfig.PeerGroups[oldGroupConf.Name]; !ok {
					server.logger.Err(fmt.Sprintln("Could not find peer group", oldGroupConf.Name))
					break
				}
			}

			if _, ok = server.BgpConfig.PeerGroups[newGroupConf.Name]; !ok {
				server.logger.Info(fmt.Sprintln("Add new peer group with name", newGroupConf.Name))
				peerGroup := config.PeerGroup{
					Config: newGroupConf,
				}
				server.BgpConfig.PeerGroups[newGroupConf.Name] = &peerGroup
			}
			server.UpdatePeerGroupInPeers(newGroupConf.Name, &newGroupConf)

		case groupName := <-server.RemPeerGroupCh:
			server.logger.Info(fmt.Sprintln("Remove Peer group:", groupName))
			if _, ok := server.BgpConfig.PeerGroups[groupName]; !ok {
				server.logger.Info(fmt.Sprintln("Peer group", groupName, "not found"))
				break
			}
			delete(server.BgpConfig.PeerGroups, groupName)
			server.UpdatePeerGroupInPeers(groupName, nil)

		case tcpConn := <-acceptCh:
			server.logger.Info(fmt.Sprintln("Connected to", tcpConn.RemoteAddr().String()))
			host, _, _ := net.SplitHostPort(tcpConn.RemoteAddr().String())
			peer, ok := server.PeerMap[host]
			if !ok {
				server.logger.Info(fmt.Sprintln("Can't accept connection. Peer is not configured yet", host))
				tcpConn.Close()
				server.logger.Info(fmt.Sprintln("Closed connection from", host))
				break
			}
			peer.AcceptConn(tcpConn)

		case peerCommand := <-server.PeerCommandCh:
			server.logger.Info(fmt.Sprintln("Peer Command received", peerCommand))
			peer, ok := server.PeerMap[peerCommand.IP.String()]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to apply command %s. Peer at that address does not exist, %v\n",
					peerCommand.Command, peerCommand.IP))
			}
			peer.Command(peerCommand.Command)

		case peerFSMConn := <-server.PeerFSMConnCh:
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM established/broken channel\n", peerFSMConn.PeerIP))
			peer, ok := server.PeerMap[peerFSMConn.PeerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM connection success, Peer %s does not exist\n", peerFSMConn.PeerIP))
				break
			}

			if peerFSMConn.Established {
				peer.PeerConnEstablished(peerFSMConn.Conn)
				addPathsMaxTx := peer.getAddPathsMaxTx()
				if addPathsMaxTx > server.addPathCount {
					server.addPathCount = addPathsMaxTx
				}
				server.setInterfaceMapForPeer(peerFSMConn.PeerIP, peer)
				server.SendAllRoutesToPeer(peer)
			} else {
				peer.PeerConnBroken(true)
				addPathsMaxTx := peer.getAddPathsMaxTx()
				if addPathsMaxTx < server.addPathCount {
					server.addPathCount = 0
					for _, otherPeer := range server.PeerMap {
						addPathsMaxTx = otherPeer.getAddPathsMaxTx()
						if addPathsMaxTx > server.addPathCount {
							server.addPathCount = addPathsMaxTx
						}
					}
				}
				server.clearInterfaceMapForPeer(peerFSMConn.PeerIP, peer)
				server.ProcessRemoveNeighbor(peerFSMConn.PeerIP, peer)
			}

		case peerIP := <-server.PeerConnEstCh:
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection established", peerIP))
			peer, ok := server.PeerMap[peerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM connection success, Peer %s does not exist", peerIP))
				break
			}

			reachInfo, err := server.ribdClient.GetRouteReachabilityInfo(peerIP)
			if err != nil {
				server.logger.Info(fmt.Sprintf("Server: Peer %s is not reachable", peerIP))
			} else {
				ifIdx := asicdConstDefs.GetIfIndexFromIntfIdAndIntfType(int(reachInfo.NextHopIfIndex), int(reachInfo.NextHopIfType))
				server.logger.Info(fmt.Sprintf("Server: Peer %s IfIdx %d", peerIP, ifIdx))
				if _, ok := server.ifacePeerMap[ifIdx]; !ok {
					server.ifacePeerMap[ifIdx] = make([]string, 0)
				}
				server.ifacePeerMap[ifIdx] = append(server.ifacePeerMap[ifIdx], peerIP)
				peer.setIfIdx(ifIdx)
			}

			server.SendAllRoutesToPeer(peer)

		case peerIP := <-server.PeerConnBrokenCh:
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection broken", peerIP))
			peer, ok := server.PeerMap[peerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM connection failure, Peer %s does not exist", peerIP))
				break
			}
			ifIdx := peer.getIfIdx()
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection broken ifIdx %v", peerIP, ifIdx))
			if peerList, ok := server.ifacePeerMap[ifIdx]; ok {
				for idx, ip := range peerList {
					if ip == peerIP {
						server.ifacePeerMap[ifIdx] = append(server.ifacePeerMap[ifIdx][:idx], server.ifacePeerMap[ifIdx][idx+1:]...)
						if len(server.ifacePeerMap[ifIdx]) == 0 {
							delete(server.ifacePeerMap, ifIdx)
						}
						break
					}
				}
			}
			peer.setIfIdx(-1)
			server.ProcessRemoveNeighbor(peerIP, peer)

		case pktInfo := <-server.BGPPktSrcCh:
			server.logger.Info(fmt.Sprintln("Received BGP message from peer %s", pktInfo.Src))
			server.ProcessUpdate(pktInfo)

		case rxBuf := <-ribSubSocketCh:
			server.logger.Info(fmt.Sprintf("Server: Received update on RIB sub socket"))
			server.handleRibUpdates(rxBuf)

		case err := <-ribSubSocketErrCh:
			server.logger.Info(fmt.Sprintf("Server: RIB subscriber socket returned err:%s", err))

		case rxBuf := <-ribSubBGPSocketCh:
			server.logger.Info(fmt.Sprintf("Server: Received update on RIB BGP sub socket"))
			server.handleRibUpdates(rxBuf)

		case err := <-ribSubBGPSocketErrCh:
			server.logger.Info(fmt.Sprintf("Server: RIB BGP subscriber socket returned err:%s", err))

		case ifState := <-asicdL3IntfStateCh:
			server.logger.Info(fmt.Sprintf("Server: Received update on Asicd sub socket %+v, ifacePeerMap %+v",
				ifState, server.ifacePeerMap))
			/*
				if ifState.state == asicdConstDefs.INTF_STATE_UP {
					if peer, ok := server.PeerMap[strconv.Itoa(int(ifState.idx))]; ok {
						ip, _, err := net.ParseCIDR(ifState.ipaddr)
						if err == nil {
							server.logger.Info(fmt.Sprintln("Updating neighbor address with peer idx ", ifState.idx, " to ", ip.String()))
							peer.NeighborConf.Neighbor.NeighborAddress = ip
						}
					}
				}
			*/
			if peerList, ok := server.ifacePeerMap[ifState.idx]; ok && ifState.state == asicdConstDefs.INTF_STATE_DOWN {
				for _, peerIP := range peerList {
					if peer, ok := server.PeerMap[peerIP]; ok {
						peer.StopFSM("Interface Down")
					}
				}
			}

		case rxBuf := <-bfdSubSocketCh:
			server.logger.Info(fmt.Sprintf("Server: Received notification on BFD sub socket"))
			server.handleBfdNotifications(rxBuf)

		case err := <-bfdSubSocketErrCh:
			server.logger.Info(fmt.Sprintf("Server: BFD subscriber socket returned err:%s", err))

		case reachabilityInfo := <-server.ReachabilityCh:
			server.logger.Info(fmt.Sprintln("Server: Reachability info for ip", reachabilityInfo.IP))
			_, err := server.ribdClient.GetRouteReachabilityInfo(reachabilityInfo.IP)
			if err != nil {
				reachabilityInfo.ReachableCh <- false
			} else {
				reachabilityInfo.ReachableCh <- true
			}
		}
	}
}

func (s *BGPServer) GetBGPGlobalState() config.GlobalState {
	return s.BgpConfig.Global.State
}

func (s *BGPServer) GetBGPNeighborState(neighborIP string) *config.NeighborState {
	peer, ok := s.PeerMap[neighborIP]
	if !ok {
		s.logger.Err(fmt.Sprintf("GetBGPNeighborState - Neighbor not found for address:%s", neighborIP))
		return nil
	}
	return &peer.NeighborConf.Neighbor.State
}

func (s *BGPServer) BulkGetBGPNeighbors(index int, count int) (int, int, []*config.NeighborState) {
	defer s.NeighborMutex.RUnlock()

	s.NeighborMutex.RLock()
	if index+count > len(s.Neighbors) {
		count = len(s.Neighbors) - index
	}

	result := make([]*config.NeighborState, count)
	for i := 0; i < count; i++ {
		result[i] = &s.Neighbors[i+index].NeighborConf.Neighbor.State
	}

	index += count
	if index >= len(s.Neighbors) {
		index = 0
	}
	return index, count, result
}
