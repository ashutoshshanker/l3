// server.go
package server

import (
	"asicd/asicdConstDefs"
	"bfdd"
	"bgpd"
	"bytes"
	"encoding/json"
	"fmt"
	"l3/bfd/bfddCommonDefs"
	"l3/bgp/config"
	"l3/bgp/packet"
	"l3/bgp/policy"
	"l3/rib/ribdCommonDefs"
	"log/syslog"
	"net"
	"ribd"
	"runtime"
	"sync"
	"sync/atomic"
	"utils/patriciaDB"
	"utils/policy/policyCommonDefs"

	nanomsg "github.com/op/go-nanomsg"
)

const IP string = "12.1.12.202" //"192.168.1.1"
const BGPPort string = "179"

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
	idx   int32
	state uint8
}

type PeerFSMConn struct {
	peerIP      string
	established bool
	conn        *net.Conn
}

type BGPServer struct {
	logger           *syslog.Writer
	policyEngine     *policy.BGPPolicyEngine
	ribdClient       *ribd.RouteServiceClient
	bfddClient       *bfdd.BFDDServicesClient
	BgpConfig        config.Bgp
	GlobalConfigCh   chan config.GlobalConfig
	AddPeerCh        chan PeerUpdate
	RemPeerCh        chan string
	AddPeerGroupCh   chan PeerGroupUpdate
	RemPeerGroupCh   chan string
	AddAggCh         chan AggUpdate
	RemAggCh         chan string
	PeerFSMConnCh    chan PeerFSMConn
	PeerConnEstCh    chan string
	PeerConnBrokenCh chan string
	PeerCommandCh    chan config.PeerCommand
	BGPPktSrc        chan *packet.BGPPktSrc

	NeighborMutex  sync.RWMutex
	PeerMap        map[string]*Peer
	Neighbors      []*Peer
	AdjRib         *AdjRib
	connRoutesPath *Path
	ifacePeerMap   map[int32][]string
	ifaceIP        net.IP
	actionFuncMap  map[int][2]policy.ApplyActionFunc
}

func NewBGPServer(logger *syslog.Writer, policyEngine *policy.BGPPolicyEngine, ribdClient *ribd.RouteServiceClient,
	bfddClient *bfdd.BFDDServicesClient) *BGPServer {
	bgpServer := &BGPServer{}
	bgpServer.logger = logger
	bgpServer.policyEngine = policyEngine
	bgpServer.ribdClient = ribdClient
	bgpServer.bfddClient = bfddClient
	bgpServer.GlobalConfigCh = make(chan config.GlobalConfig)
	bgpServer.AddPeerCh = make(chan PeerUpdate)
	bgpServer.RemPeerCh = make(chan string)
	bgpServer.AddPeerGroupCh = make(chan PeerGroupUpdate)
	bgpServer.RemPeerGroupCh = make(chan string)
	bgpServer.AddAggCh = make(chan AggUpdate)
	bgpServer.RemAggCh = make(chan string)
	bgpServer.PeerFSMConnCh = make(chan PeerFSMConn, 50)
	bgpServer.PeerConnEstCh = make(chan string)
	bgpServer.PeerConnBrokenCh = make(chan string)
	bgpServer.PeerCommandCh = make(chan config.PeerCommand)
	bgpServer.BGPPktSrc = make(chan *packet.BGPPktSrc)
	bgpServer.NeighborMutex = sync.RWMutex{}
	bgpServer.PeerMap = make(map[string]*Peer)
	bgpServer.Neighbors = make([]*Peer, 0)
	bgpServer.AdjRib = NewAdjRib(bgpServer)
	bgpServer.ifacePeerMap = make(map[int32][]string)
	bgpServer.ifaceIP = nil
	bgpServer.actionFuncMap = make(map[int][2]policy.ApplyActionFunc)
	//bgpServer.actionFuncMap[ribdCommonDefs.PolicyActionTypeAggregate] = make([2]policy.ApplyActionFunc)

	var aggrActionFunc [2]policy.ApplyActionFunc
	aggrActionFunc[0] = bgpServer.ApplyAggregateAction
	aggrActionFunc[1] = bgpServer.UndoAggregateAction

	bgpServer.actionFuncMap[policyCommonDefs.PolicyActionTypeAggregate] = aggrActionFunc

	bgpServer.logger.Info(fmt.Sprintf("BGPServer: actionfuncmap=%v", bgpServer.actionFuncMap))
	bgpServer.policyEngine.SetTraverseFunc(bgpServer.TraverseRibForPolicies)
	bgpServer.policyEngine.SetApplyActionFunc(bgpServer.actionFuncMap)
	return bgpServer
}

func (server *BGPServer) listenForPeers(acceptCh chan *net.TCPConn) {
	addr := ":" + BGPPort
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
	routes := make([]*ribd.Routes, 0)
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
			server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))
		} else if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
			server.ProcessConnectedRoutes(make([]*ribd.Routes, 0), routes)
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

	if peer, ok := server.PeerMap[bfd.DestIp]; ok && !bfd.State {
		peer.StopFSM("Peer BFD Down")
		peer.Neighbor.State.BfdNeighborState = "Down"
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
			ifStateCh <- IfState{msg.IfIndex, msg.IfState}
		}
	}
}

func (server *BGPServer) IsPeerLocal(peerIp string) bool {
	return server.PeerMap[peerIp].PeerConf.PeerAS == server.BgpConfig.Global.Config.AS
}

func (server *BGPServer) sendUpdateMsgToAllPeers(msg *packet.BGPMessage, path *Path) {
	for _, peer := range server.PeerMap {
		// If we recieve the route from IBGP peer, don't send it to other IBGP peers
		if path != nil && path.peer != nil {
			if path.peer.IsInternal() {

				if peer.IsInternal() && !path.peer.IsRouteReflectorClient() && !peer.IsRouteReflectorClient() {
					continue
				}
			}

			// Don't send the update to the peer that sent the update.
			if peer.PeerConf.NeighborAddress.String() == path.peer.PeerConf.NeighborAddress.String() {
				continue
			}
		}

		peer.SendUpdate(*msg.Clone(), path)
	}
}

func (server *BGPServer) SendUpdate(updated map[*Path][]*Destination, withdrawn []*Destination, withdrawPath *Path) {
	nlri := make([]packet.IPPrefix, 0)
	if len(withdrawn) > 0 {
		for _, dest := range withdrawn {
			nlri = append(nlri, dest.nlri)
		}
		updateMsg := packet.NewBGPUpdateMessage(nlri, nil, nil)
		server.sendUpdateMsgToAllPeers(updateMsg, withdrawPath)
		nlri = nlri[:0]
	}

	for path, destinations := range updated {
		for _, dest := range destinations {
			nlri = append(nlri, dest.nlri)
		}
		updateMsg := packet.NewBGPUpdateMessage(make([]packet.IPPrefix, 0), path.pathAttrs, nlri)
		server.sendUpdateMsgToAllPeers(updateMsg, path)
		nlri = nlri[:0]
	}
}

type ActionCbInfo struct {
	dest      *Destination
	updated   *(map[*Path][]*Destination)
	withdrawn *([]*Destination)
}

func (server *BGPServer) getAggPrefix(conditionsList []string) *packet.IPPrefix {
	server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix"))
	var ipPrefix *packet.IPPrefix
	var err error
	for _, condition := range conditionsList {
		server.logger.Info(fmt.Sprintf("BGPServer:getAggPrefix - Find policy condition name %s in the condition database\n", condition))
		conditionItem := policy.PolicyConditionsDB.Get(patriciaDB.Prefix(condition))
		if conditionItem == nil {
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - Did not find condition ", condition, " in the condition database"))
			continue
		}
		conditionInfo := conditionItem.(policy.PolicyCondition)
		server.logger.Info(fmt.Sprintf("BGPServer:getAggPrefix - policy condition type %d\n", conditionInfo.ConditionType))
		switch conditionInfo.ConditionType {
		case policyCommonDefs.PolicyConditionTypeDstIpPrefixMatch:
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - PolicyConditionTypeDstIpPrefixMatch case"))
			condition := conditionInfo.ConditionInfo.(policy.MatchPrefixConditionInfo)
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix - exact prefix match conditiontype"))
			ipPrefix, err = packet.ConstructIPPrefixFromCIDR(condition.Prefix.IpPrefix)
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

func (server *BGPServer) setWithdrawnWithAggPaths(actionCbInfo ActionCbInfo, withdrawn []*Destination,
	sendSummaryOnly bool) {
	aggDestMap := make(map[*Destination]bool)
	for _, aggDestination := range withdrawn {
		aggDestMap[aggDestination] = true
	}

	// There will be only one destination per aggregated path.
	// So, break out of the loop as soon as we find it.
	for path, destinations := range *actionCbInfo.updated {
		dirty := false
		for idx, dest := range destinations {
			if aggDestMap[dest] {
				//(*actionCbInfo.updated)[path][idx] = (*actionCbInfo.updated)[path][len(destinations)-1]
				//(*actionCbInfo.updated)[path][len(destinations)-1] = nil
				//(*actionCbInfo.updated)[path] = (*actionCbInfo.updated)[path][:len(destinations)-1]
				(*actionCbInfo.updated)[path][idx] = nil
				dirty = true
			}
		}
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
					(*actionCbInfo.updated)[path][idx] = (*actionCbInfo.updated)[path][movIdx]
					(*actionCbInfo.updated)[path][movIdx] = nil
					lastIdx = movIdx - 1
				}
			}
			(*actionCbInfo.updated)[path] = (*actionCbInfo.updated)[path][:lastIdx+1]
		}
	}

	if sendSummaryOnly {
		aggDestMap = make(map[*Destination]bool)
		for _, dest := range *actionCbInfo.withdrawn {
			aggDestMap[dest] = true
		}

		for _, dest := range withdrawn {
			for _, singleDest := range dest.aggregatedDestMap {
				if singleDest.locRibPath != nil {
					if _, ok := (*actionCbInfo.updated)[singleDest.locRibPath]; !ok {
						(*actionCbInfo.updated)[singleDest.locRibPath] = make([]*Destination, 0)
					}
					(*actionCbInfo.updated)[singleDest.locRibPath] = append((*actionCbInfo.updated)[singleDest.locRibPath], singleDest)
				}
			}
		}
	}

	(*actionCbInfo.withdrawn) = append((*actionCbInfo.withdrawn), withdrawn...)
}

func (server *BGPServer) setUpdatedWithAggPaths(actionCbInfo ActionCbInfo, updated map[*Path][]*Destination,
	sendSummaryOnly bool, ipPrefix packet.IPPrefix) {
	var routeDest *Destination
	var ok bool
	if routeDest, ok = server.AdjRib.getDest(ipPrefix, false); !ok {
		server.logger.Info(fmt.Sprintln("setUpdatedWithAggPaths: Did not find destination for ip", ipPrefix))
		sendSummaryOnly = false
	}

	foundRouteDest := false
	for aggPath, aggDestinations := range updated {
		foundAggDest := false
		aggDestMap := make(map[*Destination]bool)
		for _, dest := range aggDestinations {
			aggDestMap[dest] = true
		}
		// There will be only one destination per aggregated path.
		// So, break out of the loop as soon as we find it.
		for path, destinations := range *actionCbInfo.updated {
			for idx, dest := range destinations {
				if sendSummaryOnly && routeDest == dest {
					(*actionCbInfo.updated)[path][idx] = (*actionCbInfo.updated)[path][len(destinations)-1]
					(*actionCbInfo.updated)[path][len(destinations)-1] = nil
					(*actionCbInfo.updated)[path] = (*actionCbInfo.updated)[path][:len(destinations)-1]
					foundRouteDest = true
					continue
				}
				if _, ok = aggDestMap[dest]; ok {
					(*actionCbInfo.updated)[path][idx] = (*actionCbInfo.updated)[path][len(destinations)-1]
					(*actionCbInfo.updated)[path][len(destinations)-1] = nil
					(*actionCbInfo.updated)[path] = (*actionCbInfo.updated)[path][:len(destinations)-1]
					foundAggDest = true
					break
				}
			}
			if foundAggDest && foundRouteDest {
				break
			}
		}

		(*actionCbInfo.updated)[aggPath] = make([]*Destination, 0)
		(*actionCbInfo.updated)[aggPath] = append((*actionCbInfo.updated)[aggPath], aggDestinations...)

		if sendSummaryOnly {
			aggDestMap = make(map[*Destination]bool)
			for _, dest := range *actionCbInfo.withdrawn {
				aggDestMap[dest] = true
			}

			for _, dest := range aggDestinations {
				for _, singleDest := range dest.aggregatedDestMap {
					if !aggDestMap[singleDest] {
						(*actionCbInfo.withdrawn) = append((*actionCbInfo.withdrawn), singleDest)
					}
				}
			}
		}
	}
}

func (server *BGPServer) UndoAggregateAction(route *bgpd.BGPRoute, conditionList []string, action interface{}, params interface{}, ctx interface{}) {
	ipPrefix := packet.NewIPPrefix(net.ParseIP(route.Network), uint8(route.CIDRLen))
	aggPrefix := server.getAggPrefix(conditionList)
	aggActions := action.(policy.PolicyAggregateActionInfo)
	bgpAgg := config.BGPAggregate{
		GenerateASSet:   aggActions.GenerateASSet,
		SendSummaryOnly: aggActions.SendSummaryOnly,
	}
	allUpdated := make(map[*Path][]*Destination, 10)
	allWithdrawn := make([]*Destination, 0)

	var updated map[*Path][]*Destination
	var withdrawn []*Destination
	var origDest *Destination
	var actionCbInfo ActionCbInfo
	var ctxOk bool
	if actionCbInfo, ctxOk = ctx.(ActionCbInfo); ctxOk {
		origDest = actionCbInfo.dest
	}
	updated, withdrawn, _ = server.AdjRib.RemoveRouteFromAggregate(*ipPrefix, *aggPrefix,
		server.BgpConfig.Global.Config.RouterId.String(), &bgpAgg, origDest)

	if !ctxOk {
		actionCbInfo = ActionCbInfo{
			updated:   &allUpdated,
			withdrawn: &allWithdrawn,
		}
	}

	server.setUpdatedWithAggPaths(actionCbInfo, updated, aggActions.SendSummaryOnly, *ipPrefix)
	server.setWithdrawnWithAggPaths(actionCbInfo, withdrawn, aggActions.SendSummaryOnly)
	server.SendUpdate(allUpdated, allWithdrawn, nil)
	return
}

func (server *BGPServer) ApplyAggregateAction(route *bgpd.BGPRoute, conditionList []string, action interface{}, params interface{}, ctx interface{}) {
	ipPrefix := packet.NewIPPrefix(net.ParseIP(route.Network), uint8(route.CIDRLen))
	aggPrefix := server.getAggPrefix(conditionList)
	routeParams := params.(policy.RouteParams)
	aggActions := action.(policy.PolicyAggregateActionInfo)
	bgpAgg := config.BGPAggregate{
		GenerateASSet:   aggActions.GenerateASSet,
		SendSummaryOnly: aggActions.SendSummaryOnly,
	}
	actionCbInfo := ctx.(ActionCbInfo)

	var updated map[*Path][]*Destination
	var withdrawn []*Destination
	if (routeParams.CreateType == policy.Valid) || (routeParams.DeleteType == policy.Invalid) {
		updated, withdrawn, _ = server.AdjRib.AddRouteToAggregate(*ipPrefix, *aggPrefix,
			server.BgpConfig.Global.Config.RouterId.String(), server.ifaceIP, &bgpAgg)
	} else if routeParams.DeleteType == policy.Valid {
		origDest := actionCbInfo.dest
		updated, withdrawn, _ = server.AdjRib.RemoveRouteFromAggregate(*ipPrefix, *aggPrefix,
			server.BgpConfig.Global.Config.RouterId.String(), &bgpAgg, origDest)
	}
	server.setUpdatedWithAggPaths(actionCbInfo, updated, aggActions.SendSummaryOnly, *ipPrefix)
	server.setWithdrawnWithAggPaths(actionCbInfo, withdrawn, aggActions.SendSummaryOnly)
	return
}

func (server *BGPServer) checkForAggregation(updated map[*Path][]*Destination, withdrawn []*Destination,
	withdrawPath *Path) (map[*Path][]*Destination, []*Destination, *Path) {
	server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - start, updated %v withdrawn %v", updated, withdrawn))

	for _, dest := range withdrawn {
		route := dest.GetLocRibPathRoute()
		if route == nil {
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - route not found withdraw dest %s",
				dest.nlri.Prefix.String()))
			continue
		}
		server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - dest %s policylist %v hit %v before applying delete policy",
			dest.nlri.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
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
		policy.PolicyEngineFilter(route, policyCommonDefs.PolicyPath_Export, routeParams, callbackInfo)
	}

	for _, destinations := range updated {
		for _, dest := range destinations {
			route := dest.GetLocRibPathRoute()
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - dest %s policylist %v hit %v before applying create policy",
				dest.nlri.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
			if route != nil {
				routeParams := policy.RouteParams{
					CreateType:    policy.Valid,
					DeleteType:    policy.Invalid,
					ActionFuncMap: server.actionFuncMap,
				}
				callbackInfo := ActionCbInfo{
					updated:   &updated,
					withdrawn: &withdrawn,
				}
				policy.PolicyEngineFilter(route, policyCommonDefs.PolicyPath_Export, routeParams, callbackInfo)
				server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - dest %s policylist %v hit %v after applying create policy",
					dest.nlri.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
			}
		}
	}

	server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - complete, updated %v withdrawn %v", updated, withdrawn))
	return updated, withdrawn, withdrawPath
}

func (server *BGPServer) TraverseRibForPolicies(updateFunc policy.UpdateFunc, policy policy.Policy) {
	server.logger.Info(fmt.Sprintf("BGPServer:TraverseRibForPolicies - start"))
	updated := make(map[*Path][]*Destination, 10)
	withdrawn := make([]*Destination, 0, 10)
	locRib := server.AdjRib.GetLocRib()
	for path, destinations := range locRib {
		for _, dest := range destinations {
			if !path.isAggregatePath() {
				callbackInfo := ActionCbInfo{
					dest:      dest,
					updated:   &updated,
					withdrawn: &withdrawn,
				}
				updateFunc(dest.GetLocRibPathRoute(), policy, callbackInfo)
			}
		}
	}
	server.logger.Info(fmt.Sprintf("BGPServer:TraverseRibForPolicies - updated %v withdrawn %v", updated, withdrawn))
	server.SendUpdate(updated, withdrawn, nil)
}

func (server *BGPServer) ProcessUpdate(pktInfo *packet.BGPPktSrc) {
	peer, ok := server.PeerMap[pktInfo.Src]
	if !ok {
		server.logger.Err(fmt.Sprintln("BgpServer:ProcessUpdate - Peer not found, address:", pktInfo.Src))
		return
	}

	atomic.AddUint32(&peer.Neighbor.State.Queues.Input, ^uint32(0))
	peer.Neighbor.State.Messages.Received.Update++
	updated, withdrawn, withdrawPath := server.AdjRib.ProcessUpdate(peer, pktInfo)
	updated, withdrawn, withdrawPath = server.checkForAggregation(updated, withdrawn, withdrawPath)
	server.SendUpdate(updated, withdrawn, withdrawPath)
}

func (server *BGPServer) convertDestIPToIPPrefix(routes []*ribd.Routes) []packet.IPPrefix {
	dest := make([]packet.IPPrefix, 0, len(routes))
	for _, r := range routes {
		server.logger.Info(fmt.Sprintln("Route NS : ", r.NetworkStatement, " Route Origin ", r.RouteOrigin))
		ipPrefix := packet.ConstructIPPrefix(r.Ipaddr, r.Mask)
		dest = append(dest, *ipPrefix)
	}
	return dest
}

func (server *BGPServer) ProcessConnectedRoutes(installedRoutes []*ribd.Routes, withdrawnRoutes []*ribd.Routes) {
	server.logger.Info(fmt.Sprintln("valid routes:", installedRoutes, "invalid routes:", withdrawnRoutes))
	valid := server.convertDestIPToIPPrefix(installedRoutes)
	invalid := server.convertDestIPToIPPrefix(withdrawnRoutes)
	updated, withdrawn, withdrawPath := server.AdjRib.ProcessConnectedRoutes(server.BgpConfig.Global.Config.RouterId.String(),
		server.connRoutesPath, valid, invalid)
	updated, withdrawn, withdrawPath = server.checkForAggregation(updated, withdrawn, withdrawPath)
	server.SendUpdate(updated, withdrawn, withdrawPath)
}
func (server *BGPServer) ProcessRoutesFromRIB() {
	var currMarker ribd.Int
	var count ribd.Int
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
		server.ProcessConnectedRoutes(getBulkInfo.RouteList, make([]*ribd.Routes, 0))
		if getBulkInfo.More == false {
			server.logger.Info("more returned as false, so no more get bulks")
			return
		}
		currMarker = ribd.Int(getBulkInfo.EndIdx)
	}
}
func (server *BGPServer) ProcessRemoveNeighbor(peerIp string, peer *Peer) {
	updated, withdrawn, withdrawPath := server.AdjRib.RemoveUpdatesFromNeighbor(peerIp, peer)
	updated, withdrawn, withdrawPath = server.checkForAggregation(updated, withdrawn, withdrawPath)
	server.SendUpdate(updated, withdrawn, withdrawPath)
}

func (server *BGPServer) SendAllRoutesToPeer(peer *Peer) {
	withdrawn := make([]packet.IPPrefix, 0)
	nlri := make([]packet.IPPrefix, 0)
	updated := server.AdjRib.GetLocRib()
	for path, destinations := range updated {
		for _, dest := range destinations {
			nlri = append(nlri, dest.nlri)
		}

		updateMsg := packet.NewBGPUpdateMessage(withdrawn, path.pathAttrs, nlri)
		peer.SendUpdate(*updateMsg.Clone(), path)
		nlri = nlri[:0]
	}
}

func (server *BGPServer) RemoveRoutesFromAllNeighbor() {
	server.AdjRib.RemoveUpdatesFromAllNeighbors()
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
		if peer.PeerGroup.Name == groupName {
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
}

func (server *BGPServer) ProcessBfd(peer *Peer) {
	if peer.PeerConf.BfdEnable {
		server.logger.Info(fmt.Sprintln("Bfd enabled on :", peer.Neighbor.NeighborAddress))
		bfdSession := bfdd.NewBfdSessionConfig()
		bfdSession.IpAddr = peer.Neighbor.NeighborAddress.String()
		bfdSession.Owner = "bgp"
		bfdSession.Operation = "create"
		server.logger.Info(fmt.Sprintln("Creating BFD Session: ", bfdSession))
		ret, err := server.bfddClient.CreateBfdSessionConfig(bfdSession)
		if !ret {
			server.logger.Info(fmt.Sprintln("BfdSessionConfig FAILED, ret:", ret, "err:", err))
		} else {
			server.logger.Info("Bfd session configured")
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

func (server *BGPServer) StartServer() {
	gConf := <-server.GlobalConfigCh
	server.logger.Info(fmt.Sprintln("Recieved global conf:", gConf))
	server.BgpConfig.Global.Config = gConf
	server.BgpConfig.PeerGroups = make(map[string]*config.PeerGroup)

	pathAttrs := packet.ConstructPathAttrForConnRoutes(gConf.RouterId, gConf.AS)
	server.connRoutesPath = NewPath(server, nil, pathAttrs, false, false, RouteTypeConnected)

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

	for {
		select {
		case gConf = <-server.GlobalConfigCh:
			for peerIP, peer := range server.PeerMap {
				server.logger.Info(fmt.Sprintf("Cleanup peer %s", peerIP))
				peer.Cleanup()
			}
			server.logger.Info(fmt.Sprintf("Giving up CPU so that all peer FSMs will get cleaned up"))
			runtime.Gosched()

			packet.SetNextHopPathAttrs(server.connRoutesPath.pathAttrs, gConf.RouterId)
			server.RemoveRoutesFromAllNeighbor()
			server.copyGlobalConf(gConf)
			for _, peer := range server.PeerMap {
				peer.Init()
			}

		case peerUpdate := <-server.AddPeerCh:
			oldPeer := peerUpdate.OldPeer
			newPeer := peerUpdate.NewPeer
			var peer *Peer
			var ok bool
			if oldPeer.NeighborAddress != nil {
				if peer, ok = server.PeerMap[oldPeer.NeighborAddress.String()]; ok {
					server.logger.Info(fmt.Sprintln("Clean up peer", oldPeer.NeighborAddress.String()))
					peer.Cleanup()
					server.ProcessRemoveNeighbor(oldPeer.NeighborAddress.String(), peer)
					peer.UpdateNeighborConf(newPeer)

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
				server.ProcessBfd(peer)
				server.NeighborMutex.Lock()
				server.addPeerToList(peer)
				server.NeighborMutex.Unlock()
			}
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
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM established/broken channel", peerFSMConn.peerIP))
			peer, ok := server.PeerMap[peerFSMConn.peerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM connection success, Peer %s does not exist", peerFSMConn.peerIP))
				break
			}

			if peerFSMConn.established {
				peer.PeerConnEstablished(peerFSMConn.conn)
				server.setInterfaceMapForPeer(peerFSMConn.peerIP, peer)
				server.SendAllRoutesToPeer(peer)
			} else {
				peer.PeerConnBroken(true)
				server.clearInterfaceMapForPeer(peerFSMConn.peerIP, peer)
				server.ProcessRemoveNeighbor(peerFSMConn.peerIP, peer)
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

		case pktInfo := <-server.BGPPktSrc:
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
	return &peer.Neighbor.State
}

func (s *BGPServer) BulkGetBGPNeighbors(index int, count int) (int, int, []*config.NeighborState) {
	defer s.NeighborMutex.RUnlock()

	s.NeighborMutex.RLock()
	if index+count > len(s.Neighbors) {
		count = len(s.Neighbors) - index
	}

	result := make([]*config.NeighborState, count)
	for i := 0; i < count; i++ {
		result[i] = &s.Neighbors[i+index].Neighbor.State
	}

	index += count
	if index >= len(s.Neighbors) {
		index = 0
	}
	return index, count, result
}
