// server.go
package server

import (
	"fmt"
	"l3/bgp/api"
	"l3/bgp/config"
	"l3/bgp/fsm"
	"l3/bgp/packet"
	bgppolicy "l3/bgp/policy"
	bgprib "l3/bgp/rib"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"utils/logging"
	utilspolicy "utils/policy"
	"utils/policy/policyCommonDefs"
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
	bfdCh            chan config.BfdInfo
	intfCh           chan config.IntfStateInfo
	routesCh         chan *config.RouteCh
	acceptCh         chan *net.TCPConn
	GlobalCfgDone    bool

	NeighborMutex  sync.RWMutex
	PeerMap        map[string]*Peer
	Neighbors      []*Peer
	AdjRib         *bgprib.AdjRib
	ConnRoutesPath *bgprib.Path
	IfacePeerMap   map[int32][]string
	ifaceIP        net.IP
	actionFuncMap  map[int]bgppolicy.PolicyActionFunc
	AddPathCount   int
	// all managers
	IntfMgr   config.IntfStateMgrIntf
	policyMgr config.PolicyMgrIntf
	routeMgr  config.RouteMgrIntf
	bfdMgr    config.BfdMgrIntf
}

func NewBGPServer(logger *logging.Writer, policyEngine *bgppolicy.BGPPolicyEngine,
	iMgr config.IntfStateMgrIntf, pMgr config.PolicyMgrIntf, rMgr config.RouteMgrIntf,
	bMgr config.BfdMgrIntf) *BGPServer {
	bgpServer := &BGPServer{}
	bgpServer.logger = logger
	bgpServer.bgpPE = policyEngine
	bgpServer.BgpConfig = config.Bgp{}
	bgpServer.GlobalCfgDone = false
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
	bgpServer.IntfMgr = iMgr
	bgpServer.routeMgr = rMgr
	bgpServer.policyMgr = pMgr
	bgpServer.bfdMgr = bMgr
	bgpServer.AdjRib = bgprib.NewAdjRib(logger, rMgr, &bgpServer.BgpConfig.Global.Config)
	bgpServer.IfacePeerMap = make(map[int32][]string)
	bgpServer.ifaceIP = nil
	bgpServer.actionFuncMap = make(map[int]bgppolicy.PolicyActionFunc)
	bgpServer.AddPathCount = 0

	var aggrActionFunc bgppolicy.PolicyActionFunc
	aggrActionFunc.ApplyFunc = bgpServer.ApplyAggregateAction
	aggrActionFunc.UndoFunc = bgpServer.UndoAggregateAction

	bgpServer.actionFuncMap[policyCommonDefs.PolicyActionTypeAggregate] = aggrActionFunc

	bgpServer.logger.Info(fmt.Sprintf("BGPServer: actionfuncmap=%v", bgpServer.actionFuncMap))
	bgpServer.bgpPE.SetEntityUpdateFunc(bgpServer.UpdateRouteAndPolicyDB)
	bgpServer.bgpPE.SetIsEntityPresentFunc(bgpServer.DoesRouteExist)
	bgpServer.bgpPE.SetActionFuncs(bgpServer.actionFuncMap)
	bgpServer.bgpPE.SetTraverseFuncs(bgpServer.TraverseAndApplyBGPRib,
		bgpServer.TraverseAndReverseBGPRib)

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
		server.acceptCh <- tcpConn
	}
}

func (server *BGPServer) IsPeerLocal(peerIp string) bool {
	return server.PeerMap[peerIp].NeighborConf.RunningConf.PeerAS == server.BgpConfig.Global.Config.AS
}

func (server *BGPServer) SendUpdate(updated map[*bgprib.Path][]*bgprib.Destination,
	withdrawn []*bgprib.Destination, withdrawPath *bgprib.Path,
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
			policyParams.route.BGPRouteState.Network, "prefix length",
			policyParams.route.BGPRouteState.CIDRLen))
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
		switch condition.(type) {
		case utilspolicy.MatchPrefixConditionInfo:
			server.logger.Info(fmt.Sprintln("BGPServer:getAggPrefix -",
				"PolicyConditionTypeDstIpPrefixMatch case"))
			matchPrefix := condition.(utilspolicy.MatchPrefixConditionInfo)
			server.logger.Info(fmt.Sprintln(
				"BGPServer:getAggPrefix - exact prefix match conditiontype"))
			ipPrefix, err = packet.ConstructIPPrefixFromCIDR(matchPrefix.Prefix.IpPrefix)
			if err != nil {
				server.logger.Info(fmt.Sprintln(
					"BGPServer:getAggPrefix - ipPrefix invalid "))
				return nil
			}
			break
		default:
			server.logger.Info(fmt.Sprintln(
				"BGPServer:getAggPrefix - Not a known condition type"))
			break
		}
	}
	return ipPrefix
}

func (server *BGPServer) setUpdatedAddPaths(policyParams *PolicyParams,
	updatedAddPaths []*bgprib.Destination) {
	if len(updatedAddPaths) > 0 {
		addPathsMap := make(map[*bgprib.Destination]bool)
		for _, dest := range *(policyParams.updatedAddPaths) {
			addPathsMap[dest] = true
		}

		for _, dest := range updatedAddPaths {
			if !addPathsMap[dest] {
				(*policyParams.updatedAddPaths) =
					append((*policyParams.updatedAddPaths), dest)
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
			server.logger.Info(fmt.Sprintf(
				"setWithdrawnWithAggPaths: add agg dest %+v to",
				"withdrawn\n", aggDestination.IPPrefix.Prefix))
			(*policyParams.withdrawn) = append((*policyParams.withdrawn), aggDestination)
		}
	}

	// There will be only one destination per aggregated path.
	// So, break out of the loop as soon as we find it.
	for path, destinations := range *policyParams.updated {
		for idx, dest := range destinations {
			if aggDestMap[dest] {
				(*policyParams.updated)[path][idx] = nil
				server.logger.Info(fmt.Sprintf(
					"setWithdrawnWithAggPaths: remove dest",
					"%+v from withdrawn\n", dest.IPPrefix.Prefix))
			}
		}
	}

	if sendSummaryOnly {
		if policyParams.DeleteType == utilspolicy.Valid {
			for idx, dest := range *policyParams.withdrawn {
				if dest == policyParams.dest {
					server.logger.Info(fmt.Sprintf(
						"setWithdrawnWithAggPaths: remove dest",
						"%+v from withdrawn\n", dest.IPPrefix.Prefix))
					(*policyParams.withdrawn)[idx] = nil
				}
			}
		} else if policyParams.CreateType == utilspolicy.Invalid {
			if policyParams.dest != nil && policyParams.dest.LocRibPath != nil {
				found := false
				if destinations, ok :=
					(*policyParams.updated)[policyParams.dest.LocRibPath]; ok {
					for _, dest := range destinations {
						if dest == policyParams.dest {
							found = true
						}
					}
				} else {
					(*policyParams.updated)[policyParams.dest.LocRibPath] =
						make([]*bgprib.Destination, 0)
				}
				if !found {
					server.logger.Info(fmt.Sprintf(
						"setWithdrawnWithAggPaths: add dest %+v",
						"to update\n", policyParams.dest.IPPrefix.Prefix))
					(*policyParams.updated)[policyParams.dest.LocRibPath] =
						append((*policyParams.updated)[policyParams.dest.LocRibPath],
							policyParams.dest)
				}
			}
		}
	}

	server.setUpdatedAddPaths(policyParams, updatedAddPaths)
}

func (server *BGPServer) setUpdatedWithAggPaths(policyParams *PolicyParams,
	updated map[*bgprib.Path][]*bgprib.Destination,
	sendSummaryOnly bool, ipPrefix *packet.IPPrefix, updatedAddPaths []*bgprib.Destination) {
	var routeDest *bgprib.Destination
	var ok bool
	if routeDest, ok = server.AdjRib.GetDest(ipPrefix, false); !ok {
		server.logger.Err(fmt.Sprintln("setUpdatedWithAggPaths: Did not",
			"find destination for ip", ipPrefix))
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

	for aggPath, aggDestinations := range updated {
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
				server.logger.Info(fmt.Sprintf(
					"setUpdatedWithAggPaths: add agg dest %+v to",
					"updated\n", dest.IPPrefix.Prefix))
				(*policyParams.updated)[aggPath] =
					append((*policyParams.updated)[aggPath], dest)
			}
		}

		if sendSummaryOnly {
			if policyParams.CreateType == utilspolicy.Valid {
				for path, destinations := range *policyParams.updated {
					for idx, dest := range destinations {
						if routeDest == dest {
							(*policyParams.updated)[path][idx] = nil
							server.logger.Info(
								fmt.Sprintf("setUpdatedWithAggPaths:",
									"summaryOnly, remove dest %+v from",
									"updated\n",
									dest.IPPrefix.Prefix))
						}
					}
				}
			} else if policyParams.DeleteType == utilspolicy.Invalid {
				if !withdrawMap[routeDest] {
					server.logger.Info(fmt.Sprintf(
						"setUpdatedWithAggPaths: summaryOnly,",
						"add dest %+v to withdrawn\n",
						routeDest.IPPrefix.Prefix))
					(*policyParams.withdrawn) =
						append((*policyParams.withdrawn), routeDest)
				}
			}
		}

	}

	server.setUpdatedAddPaths(policyParams, updatedAddPaths)
}

func (server *BGPServer) UndoAggregateAction(actionInfo interface{},
	conditionList []interface{}, params interface{}, policyStmt utilspolicy.PolicyStmt) {
	policyParams := params.(PolicyParams)
	ipPrefix := packet.NewIPPrefix(net.ParseIP(policyParams.route.BGPRouteState.Network),
		uint8(policyParams.route.BGPRouteState.CIDRLen))
	aggPrefix := server.getAggPrefix(conditionList)
	aggActions := actionInfo.(utilspolicy.PolicyAggregateActionInfo)
	bgpAgg := config.BGPAggregate{
		GenerateASSet:   aggActions.GenerateASSet,
		SendSummaryOnly: aggActions.SendSummaryOnly,
	}

	server.logger.Info(fmt.Sprintf("UndoAggregateAction: ipPrefix=%+v, aggPrefix=%+v\n",
		ipPrefix.Prefix, aggPrefix.Prefix))
	var updated map[*bgprib.Path][]*bgprib.Destination
	var withdrawn []*bgprib.Destination
	var updatedAddPaths []*bgprib.Destination
	var origDest *bgprib.Destination
	if policyParams.dest != nil {
		origDest = policyParams.dest
	}
	updated, withdrawn, _, updatedAddPaths = server.AdjRib.RemoveRouteFromAggregate(ipPrefix, aggPrefix,
		server.BgpConfig.Global.Config.RouterId.String(), &bgpAgg, origDest, server.AddPathCount)

	server.logger.Info(fmt.Sprintf("UndoAggregateAction: aggregate result",
		"update=%+v, withdrawn=%+v\n", updated, withdrawn))
	server.setWithdrawnWithAggPaths(&policyParams, withdrawn, aggActions.SendSummaryOnly, updatedAddPaths)
	server.logger.Info(fmt.Sprintf("UndoAggregateAction: after updating",
		"withdraw agg paths, update=%+v, withdrawn=%+v, policyparams.update=%+v,",
		"policyparams.withdrawn=%+v\n",
		updated, withdrawn, *policyParams.updated, *policyParams.withdrawn))
	return
}

func (server *BGPServer) ApplyAggregateAction(actionInfo interface{},
	conditionInfo []interface{}, params interface{}) {
	policyParams := params.(PolicyParams)
	ipPrefix := packet.NewIPPrefix(net.ParseIP(policyParams.route.BGPRouteState.Network),
		uint8(policyParams.route.BGPRouteState.CIDRLen))
	aggPrefix := server.getAggPrefix(conditionInfo)
	aggActions := actionInfo.(utilspolicy.PolicyAggregateActionInfo)
	bgpAgg := config.BGPAggregate{
		GenerateASSet:   aggActions.GenerateASSet,
		SendSummaryOnly: aggActions.SendSummaryOnly,
	}

	server.logger.Info(fmt.Sprintf("ApplyAggregateAction: ipPrefix=%+v, aggPrefix=%+v\n",
		ipPrefix.Prefix, aggPrefix.Prefix))
	var updated map[*bgprib.Path][]*bgprib.Destination
	var withdrawn []*bgprib.Destination
	var updatedAddPaths []*bgprib.Destination
	if (policyParams.CreateType == utilspolicy.Valid) ||
		(policyParams.DeleteType == utilspolicy.Invalid) {
		server.logger.Info(fmt.Sprintf("ApplyAggregateAction: CreateType",
			"= Valid or DeleteType = Invalid\n"))
		updated, withdrawn, _, updatedAddPaths =
			server.AdjRib.AddRouteToAggregate(ipPrefix, aggPrefix,
				server.BgpConfig.Global.Config.RouterId.String(),
				server.ifaceIP, &bgpAgg, server.AddPathCount)
	} else if policyParams.DeleteType == utilspolicy.Valid {
		server.logger.Info(fmt.Sprintf("ApplyAggregateAction: DeleteType = Valid\n"))
		origDest := policyParams.dest
		updated, withdrawn, _, updatedAddPaths =
			server.AdjRib.RemoveRouteFromAggregate(ipPrefix, aggPrefix,
				server.BgpConfig.Global.Config.RouterId.String(), &bgpAgg,
				origDest, server.AddPathCount)
	}

	server.logger.Info(fmt.Sprintf("ApplyAggregateAction: aggregate result update=%+v,",
		"withdrawn=%+v\n", updated, withdrawn))
	server.setUpdatedWithAggPaths(&policyParams, updated, aggActions.SendSummaryOnly,
		ipPrefix, updatedAddPaths)
	server.logger.Info(fmt.Sprintf("ApplyAggregateAction: after updating agg paths, update=%+v,",
		"withdrawn=%+v, policyparams.update=%+v, policyparams.withdrawn=%+v\n",
		updated, withdrawn, *policyParams.updated, *policyParams.withdrawn))
	return
}

func (server *BGPServer) CheckForAggregation(updated map[*bgprib.Path][]*bgprib.Destination,
	withdrawn []*bgprib.Destination, withdrawPath *bgprib.Path,
	updatedAddPaths []*bgprib.Destination) (map[*bgprib.Path][]*bgprib.Destination,
	[]*bgprib.Destination, *bgprib.Path, []*bgprib.Destination) {
	server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - start, updated %v withdrawn %v\n",
		updated, withdrawn))

	for _, dest := range withdrawn {
		if dest == nil || dest.LocRibPath == nil || dest.LocRibPath.IsAggregate() {
			continue
		}

		route := dest.GetLocRibPathRoute()
		if route == nil {
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - route",
				"not found withdraw dest %s\n",
				dest.IPPrefix.Prefix.String()))
			continue
		}
		peEntity := utilspolicy.PolicyEngineFilterEntityParams{
			DestNetIp: route.BGPRouteState.Network + "/" +
				strconv.Itoa(int(route.BGPRouteState.CIDRLen)),
			NextHopIp:  route.BGPRouteState.NextHop,
			DeletePath: true,
		}
		server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - withdraw dest %s policylist",
			"%v hit %v before applying delete policy\n",
			dest.IPPrefix.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
		callbackInfo := PolicyParams{
			CreateType:      utilspolicy.Invalid,
			DeleteType:      utilspolicy.Valid,
			route:           route,
			dest:            dest,
			updated:         &updated,
			withdrawn:       &withdrawn,
			updatedAddPaths: &updatedAddPaths,
		}
		server.bgpPE.PolicyEngine.PolicyEngineFilter(peEntity,
			policyCommonDefs.PolicyPath_Export, callbackInfo)
	}

	for _, destinations := range updated {
		server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update destinations %+v\n",
			destinations))
		for _, dest := range destinations {
			server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update dest %+v\n",
				dest.IPPrefix.Prefix))
			if dest == nil || dest.LocRibPath == nil || dest.LocRibPath.IsAggregate() {
				continue
			}
			route := dest.GetLocRibPathRoute()
			server.logger.Info(fmt.Sprintf(
				"BGPServer:checkForAggregate - update dest %s policylist %v",
				"hit %v before applying create policy\n",
				dest.IPPrefix.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
			if route != nil {
				peEntity := utilspolicy.PolicyEngineFilterEntityParams{
					DestNetIp: route.BGPRouteState.Network + "/" +
						strconv.Itoa(int(route.BGPRouteState.CIDRLen)),
					NextHopIp:  route.BGPRouteState.NextHop,
					CreatePath: true,
				}
				callbackInfo := PolicyParams{
					CreateType:      utilspolicy.Valid,
					DeleteType:      utilspolicy.Invalid,
					route:           route,
					dest:            dest,
					updated:         &updated,
					withdrawn:       &withdrawn,
					updatedAddPaths: &updatedAddPaths,
				}
				server.bgpPE.PolicyEngine.PolicyEngineFilter(peEntity,
					policyCommonDefs.PolicyPath_Export, callbackInfo)
				server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - update dest %s",
					"policylist %v hit %v after applying create policy\n",
					dest.IPPrefix.Prefix.String(), route.PolicyList, route.PolicyHitCounter))
			}
		}
	}

	server.logger.Info(fmt.Sprintf("BGPServer:checkForAggregate - complete, updated %v withdrawn %v\n",
		updated, withdrawn))
	return updated, withdrawn, withdrawPath, updatedAddPaths
}

func (server *BGPServer) UpdateRouteAndPolicyDB(policyDetails utilspolicy.PolicyDetails, params interface{}) {
	policyParams := params.(PolicyParams)
	var op int
	if policyParams.DeleteType != bgppolicy.Invalid {
		op = bgppolicy.Del
	} else {
		if policyDetails.EntityDeleted == false {
			server.logger.Info(fmt.Sprintln("Reject action was not applied,",
				"so add this policy to the route"))
			op = bgppolicy.Add
			bgppolicy.UpdateRoutePolicyState(policyParams.route, op,
				policyDetails.Policy, policyDetails.PolicyStmt)
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
				route := dest.GetLocRibPathRoute()
				if route == nil {
					continue
				}
				peEntity := utilspolicy.PolicyEngineFilterEntityParams{
					DestNetIp: route.BGPRouteState.Network + "/" +
						strconv.Itoa(int(route.BGPRouteState.CIDRLen)),
					NextHopIp:  route.BGPRouteState.NextHop,
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
	server.logger.Info(fmt.Sprintf("BGPServer:TraverseRibForPolicies - updated %v withdrawn %v",
		updated, withdrawn))
	server.SendUpdate(updated, withdrawn, nil, updatedAddPaths)
}

func (server *BGPServer) TraverseAndReverseBGPRib(policyData interface{}) {
	policy := policyData.(utilspolicy.Policy)
	server.logger.Info(fmt.Sprintln("BGPServer:TraverseAndReverseBGPRib - policy",
		policy.Name))
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
		dest := server.AdjRib.GetDestFromIPAndLen(route.BGPRouteState.Network,
			uint32(route.BGPRouteState.CIDRLen))
		callbackInfo := PolicyParams{
			route:           route,
			dest:            dest,
			updated:         &updated,
			withdrawn:       &withdrawn,
			updatedAddPaths: &updatedAddPaths,
		}
		peEntity := utilspolicy.PolicyEngineFilterEntityParams{
			DestNetIp: route.BGPRouteState.Network + "/" +
				strconv.Itoa(int(route.BGPRouteState.CIDRLen)),
			NextHopIp: route.BGPRouteState.NextHop,
		}

		ipPrefix, err := bgppolicy.GetNetworkPrefixFromCIDR(route.BGPRouteState.Network + "/" +
			strconv.Itoa(int(route.BGPRouteState.CIDRLen)))
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
		server.logger.Err(fmt.Sprintln("BgpServer:ProcessUpdate - Peer not found,",
			"address:", pktInfo.Src))
		return
	}

	atomic.AddUint32(&peer.NeighborConf.Neighbor.State.Queues.Input, ^uint32(0))
	peer.NeighborConf.Neighbor.State.Messages.Received.Update++
	updated, withdrawn, withdrawPath, updatedAddPaths, addedAllPrefixes :=
		server.AdjRib.ProcessUpdate(peer.NeighborConf, pktInfo, server.AddPathCount)
	if !addedAllPrefixes {
		peer.MaxPrefixesExceeded()
	}
	updated, withdrawn, withdrawPath, updatedAddPaths =
		server.CheckForAggregation(updated, withdrawn, withdrawPath,
			updatedAddPaths)
	server.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
}

func (server *BGPServer) convertDestIPToIPPrefix(routes []*config.RouteInfo) []packet.NLRI {
	dest := make([]packet.NLRI, 0, len(routes))
	for _, r := range routes {
		server.logger.Info(fmt.Sprintln("Route NS : ",
			r.NetworkStatement, " Route Origin ", r.RouteOrigin))
		ipPrefix := packet.ConstructIPPrefix(r.Ipaddr, r.Mask)
		dest = append(dest, ipPrefix)
	}
	return dest
}

func (server *BGPServer) ProcessConnectedRoutes(installedRoutes []*config.RouteInfo,
	withdrawnRoutes []*config.RouteInfo) {
	server.logger.Info(fmt.Sprintln("valid routes:", installedRoutes,
		"invalid routes:", withdrawnRoutes))
	valid := server.convertDestIPToIPPrefix(installedRoutes)
	invalid := server.convertDestIPToIPPrefix(withdrawnRoutes)
	updated, withdrawn, withdrawPath, updatedAddPaths := server.AdjRib.ProcessConnectedRoutes(
		server.BgpConfig.Global.Config.RouterId.String(),
		server.ConnRoutesPath, valid, invalid, server.AddPathCount)
	updated, withdrawn, withdrawPath, updatedAddPaths =
		server.CheckForAggregation(updated, withdrawn, withdrawPath,
			updatedAddPaths)
	server.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
}

func (server *BGPServer) ProcessRemoveNeighbor(peerIp string, peer *Peer) {
	updated, withdrawn, withdrawPath, updatedAddPaths :=
		server.AdjRib.RemoveUpdatesFromNeighbor(peerIp,
			peer.NeighborConf, server.AddPathCount)
	server.logger.Info(fmt.Sprintf("ProcessRemoveNeighbor - Neighbor %s,",
		"send updated paths %v, withdrawn paths %v\n",
		peerIp, updated, withdrawn))
	updated, withdrawn, withdrawPath, updatedAddPaths =
		server.CheckForAggregation(updated, withdrawn, withdrawPath,
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
	server.AdjRib.RemoveUpdatesFromAllNeighbors(server.AddPathCount)
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

func (server *BGPServer) handleBfdNotifications(oper config.Operation, DestIp string,
	State bool) {
	if peer, ok := server.PeerMap[DestIp]; ok {
		if !State && peer.NeighborConf.Neighbor.State.BfdNeighborState == "up" {
			peer.Command(int(fsm.BGPEventManualStop), fsm.BGPCmdReasonNone)
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "down"
		}
		if State && peer.NeighborConf.Neighbor.State.BfdNeighborState == "down" {
			peer.NeighborConf.Neighbor.State.BfdNeighborState = "up"
			peer.Command(int(fsm.BGPEventManualStart), fsm.BGPCmdReasonNone)
		}
		server.logger.Info(fmt.Sprintln("Bfd state of peer ",
			peer.NeighborConf.Neighbor.NeighborAddress, " is ",
			peer.NeighborConf.Neighbor.State.BfdNeighborState))
	}
}

func (server *BGPServer) setInterfaceMapForPeer(peerIP string, peer *Peer) {
	server.logger.Info(fmt.Sprintln("Server: setInterfaceMapForPeer Peer", peer,
		"calling GetRouteReachabilityInfo"))
	reachInfo, err := server.routeMgr.GetNextHopInfo(peerIP)
	server.logger.Info(fmt.Sprintln("Server: setInterfaceMapForPeer Peer",
		peer, "GetRouteReachabilityInfo returned", reachInfo))
	if err != nil {
		server.logger.Info(fmt.Sprintf("Server: Peer %s is not reachable", peerIP))
	} else {
		// @TODO: jgheewala think of something better for ovsdb....
		ifIdx := server.IntfMgr.GetIfIndex(int(reachInfo.NextHopIfIndex),
			int(reachInfo.NextHopIfType))
		///		ifIdx := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(reachInfo.NextHopIfIndex), int(reachInfo.NextHopIfType))
		server.logger.Info(fmt.Sprintf("Server: Peer %s IfIdx %d", peerIP, ifIdx))
		if _, ok := server.IfacePeerMap[ifIdx]; !ok {
			server.IfacePeerMap[ifIdx] = make([]string, 0)
		}
		server.IfacePeerMap[ifIdx] = append(server.IfacePeerMap[ifIdx], peerIP)
		peer.setIfIdx(ifIdx)
	}
}

func (server *BGPServer) clearInterfaceMapForPeer(peerIP string, peer *Peer) {
	ifIdx := peer.getIfIdx()
	server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection broken ifIdx %v", peerIP, ifIdx))
	if peerList, ok := server.IfacePeerMap[ifIdx]; ok {
		for idx, ip := range peerList {
			if ip == peerIP {
				server.IfacePeerMap[ifIdx] = append(server.IfacePeerMap[ifIdx][:idx],
					server.IfacePeerMap[ifIdx][idx+1:]...)
				if len(server.IfacePeerMap[ifIdx]) == 0 {
					delete(server.IfacePeerMap, ifIdx)
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

func (server *BGPServer) listenChannelUpdates() {
	for {
		select {
		case gConf := <-server.GlobalConfigCh:
			for peerIP, peer := range server.PeerMap {
				server.logger.Info(fmt.Sprintf("Cleanup peer %s", peerIP))
				peer.Cleanup()
			}
			server.logger.Info(fmt.Sprintf("Giving up CPU so that all peer FSMs",
				"will get cleaned up"))
			runtime.Gosched()

			packet.SetNextHopPathAttrs(server.ConnRoutesPath.PathAttrs, gConf.RouterId)
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
					server.logger.Info(fmt.Sprintln("Clean up peer",
						oldPeer.NeighborAddress.String()))
					peer.Cleanup()
					server.ProcessRemoveNeighbor(oldPeer.NeighborAddress.String(),
						peer)
					peer.UpdateNeighborConf(newPeer, &server.BgpConfig)

					runtime.Gosched()
				} else {
					server.logger.Info(fmt.Sprintln(
						"Can't find neighbor with old address",
						oldPeer.NeighborAddress.String()))
				}
			}

			if !ok {
				_, ok = server.PeerMap[newPeer.NeighborAddress.String()]
				if ok {
					server.logger.Info(fmt.Sprintln("Failed to add neighbor.",
						"Neighbor at that address already exists,",
						newPeer.NeighborAddress.String()))
					break
				}

				var groupConfig *config.PeerGroupConfig
				if newPeer.PeerGroup != "" {
					if group, ok :=
						server.BgpConfig.PeerGroups[newPeer.PeerGroup]; !ok {
						server.logger.Info(fmt.Sprintln("Peer group",
							newPeer.PeerGroup,
							"not created yet, creating peer",
							newPeer.NeighborAddress.String(),
							"without the group"))
					} else {
						groupConfig = &group.Config
					}
				}
				server.logger.Info(fmt.Sprintln("Add neighbor, ip:",
					newPeer.NeighborAddress.String()))
				peer = NewPeer(server, &server.BgpConfig.Global.Config,
					groupConfig, newPeer)
				server.PeerMap[newPeer.NeighborAddress.String()] = peer
				server.NeighborMutex.Lock()
				server.addPeerToList(peer)
				server.NeighborMutex.Unlock()
			}
			peer.Init()

		case remPeer := <-server.RemPeerCh:
			server.logger.Info(fmt.Sprintln("Remove Peer:", remPeer))
			peer, ok := server.PeerMap[remPeer]
			if !ok {
				server.logger.Info(fmt.Sprintln("Failed to remove peer.",
					"Peer at that address does not exist,", remPeer))
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
			server.logger.Info(fmt.Sprintln("Peer group update old:",
				oldGroupConf, "new:", newGroupConf))
			var ok bool

			if oldGroupConf.Name != "" {
				if _, ok = server.BgpConfig.PeerGroups[oldGroupConf.Name]; !ok {
					server.logger.Err(fmt.Sprintln("Could not find peer group",
						oldGroupConf.Name))
					break
				}
			}

			if _, ok = server.BgpConfig.PeerGroups[newGroupConf.Name]; !ok {
				server.logger.Info(fmt.Sprintln("Add new peer group with name",
					newGroupConf.Name))
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

		case tcpConn := <-server.acceptCh:
			server.logger.Info(fmt.Sprintln("Connected to", tcpConn.RemoteAddr().String()))
			host, _, _ := net.SplitHostPort(tcpConn.RemoteAddr().String())
			peer, ok := server.PeerMap[host]
			if !ok {
				server.logger.Info(fmt.Sprintln("Can't accept connection.",
					"Peer is not configured yet", host))
				tcpConn.Close()
				server.logger.Info(fmt.Sprintln("Closed connection from", host))
				break
			}
			peer.AcceptConn(tcpConn)

		case peerCommand := <-server.PeerCommandCh:
			server.logger.Info(fmt.Sprintln("Peer Command received", peerCommand))
			peer, ok := server.PeerMap[peerCommand.IP.String()]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to apply command %s.",
					"Peer at that address does not exist, %v\n",
					peerCommand.Command, peerCommand.IP))
			}
			peer.Command(peerCommand.Command, fsm.BGPCmdReasonNone)

		case peerFSMConn := <-server.PeerFSMConnCh:
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM established/broken",
				"channel\n", peerFSMConn.PeerIP))
			peer, ok := server.PeerMap[peerFSMConn.PeerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM connection",
					"success, Peer %s does not exist\n", peerFSMConn.PeerIP))
				break
			}

			if peerFSMConn.Established {
				peer.PeerConnEstablished(peerFSMConn.Conn)
				addPathsMaxTx := peer.getAddPathsMaxTx()
				if addPathsMaxTx > server.AddPathCount {
					server.AddPathCount = addPathsMaxTx
				}
				server.setInterfaceMapForPeer(peerFSMConn.PeerIP, peer)
				server.SendAllRoutesToPeer(peer)
			} else {
				peer.PeerConnBroken(true)
				addPathsMaxTx := peer.getAddPathsMaxTx()
				if addPathsMaxTx < server.AddPathCount {
					server.AddPathCount = 0
					for _, otherPeer := range server.PeerMap {
						addPathsMaxTx = otherPeer.getAddPathsMaxTx()
						if addPathsMaxTx > server.AddPathCount {
							server.AddPathCount = addPathsMaxTx
						}
					}
				}
				server.clearInterfaceMapForPeer(peerFSMConn.PeerIP, peer)
				server.ProcessRemoveNeighbor(peerFSMConn.PeerIP, peer)
			}

		case peerIP := <-server.PeerConnEstCh:
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection",
				"established", peerIP))
			peer, ok := server.PeerMap[peerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM",
					"connection success,",
					"Peer %s does not exist", peerIP))
				break
			}
			reachInfo, err := server.routeMgr.GetNextHopInfo(peerIP)
			if err != nil {
				server.logger.Info(fmt.Sprintf(
					"Server: Peer %s is not reachable", peerIP))
			} else {
				// @TODO: jgheewala think of something better for ovsdb....
				ifIdx := server.IntfMgr.GetIfIndex(int(reachInfo.NextHopIfIndex),
					int(reachInfo.NextHopIfType))
				server.logger.Info(fmt.Sprintf("Server: Peer %s IfIdx %d",
					peerIP, ifIdx))
				if _, ok := server.IfacePeerMap[ifIdx]; !ok {
					server.IfacePeerMap[ifIdx] = make([]string, 0)
					//ifIdx := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(reachInfo.NextHopIfIndex), int(reachInfo.NextHopIfType))
				}
				server.IfacePeerMap[ifIdx] = append(server.IfacePeerMap[ifIdx],
					peerIP)
				peer.setIfIdx(ifIdx)
			}

			server.SendAllRoutesToPeer(peer)

		case peerIP := <-server.PeerConnBrokenCh:
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection broken",
				peerIP))
			peer, ok := server.PeerMap[peerIP]
			if !ok {
				server.logger.Info(fmt.Sprintf("Failed to process FSM",
					"connection failure,",
					"Peer %s does not exist", peerIP))
				break
			}
			ifIdx := peer.getIfIdx()
			server.logger.Info(fmt.Sprintf("Server: Peer %s FSM connection broken ifIdx %v",
				peerIP, ifIdx))
			if peerList, ok := server.IfacePeerMap[ifIdx]; ok {
				for idx, ip := range peerList {
					if ip == peerIP {
						server.IfacePeerMap[ifIdx] =
							append(server.IfacePeerMap[ifIdx][:idx],
								server.IfacePeerMap[ifIdx][idx+1:]...)
						if len(server.IfacePeerMap[ifIdx]) == 0 {
							delete(server.IfacePeerMap, ifIdx)
						}
						break
					}
				}
			}
			peer.setIfIdx(-1)
			server.ProcessRemoveNeighbor(peerIP, peer)

		case pktInfo := <-server.BGPPktSrcCh:
			server.logger.Info(fmt.Sprintln("Received BGP message from peer %s",
				pktInfo.Src))
			server.ProcessUpdate(pktInfo)

		case reachabilityInfo := <-server.ReachabilityCh:
			server.logger.Info(fmt.Sprintln("Server: Reachability info for ip",
				reachabilityInfo.IP))

			_, err := server.routeMgr.GetNextHopInfo(reachabilityInfo.IP)
			if err != nil {
				reachabilityInfo.ReachableCh <- false
			} else {
				reachabilityInfo.ReachableCh <- true
			}
		case bfdNotify := <-server.bfdCh:
			server.handleBfdNotifications(bfdNotify.Oper,
				bfdNotify.DestIp, bfdNotify.State)
		case ifState := <-server.intfCh:
			if peerList, ok := server.IfacePeerMap[ifState.Idx]; ok &&
				ifState.State == config.INTF_STATE_DOWN {
				for _, peerIP := range peerList {
					if peer, ok := server.PeerMap[peerIP]; ok {
						peer.StopFSM("Interface Down")
					}
				}
			}
		case routeInfo := <-server.routesCh:
			server.ProcessConnectedRoutes(routeInfo.Add, routeInfo.Remove)
		}
	}

}

func (server *BGPServer) StartServer() {
	gConf := <-server.GlobalConfigCh
	server.GlobalCfgDone = true
	server.logger.Info(fmt.Sprintln("Recieved global conf:", gConf))
	server.BgpConfig.Global.Config = gConf
	server.constructBGPGlobalState(&gConf)
	server.BgpConfig.PeerGroups = make(map[string]*config.PeerGroup)

	pathAttrs := packet.ConstructPathAttrForConnRoutes(gConf.RouterId, gConf.AS)
	server.ConnRoutesPath = bgprib.NewPath(server.AdjRib, nil, pathAttrs,
		false, false, bgprib.RouteTypeConnected)

	server.logger.Info("Setting up Peer connections")
	// channel for accepting connections
	server.acceptCh = make(chan *net.TCPConn)
	// Channel for handling BFD notifications
	server.bfdCh = make(chan config.BfdInfo)
	// Channel for handling Interface notifications
	server.intfCh = make(chan config.IntfStateInfo)
	// Channel for handling route notifications
	server.routesCh = make(chan *config.RouteCh)

	go server.listenForPeers(server.acceptCh)
	go server.listenChannelUpdates()

	server.logger.Info("Start all managers and initialize API Layer")
	api.Init(server.bfdCh, server.intfCh, server.routesCh)
	server.IntfMgr.Start()
	server.routeMgr.Start()
	server.bfdMgr.Start()
}

func (s *BGPServer) GetBGPGlobalState() config.GlobalState {
	return s.BgpConfig.Global.State
}

func (s *BGPServer) GetBGPNeighborState(neighborIP string) *config.NeighborState {
	peer, ok := s.PeerMap[neighborIP]
	if !ok {
		s.logger.Err(fmt.Sprintf("GetBGPNeighborState - Neighbor not found for address:%s",
			neighborIP))
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

func (svr *BGPServer) VerifyBgpGlobalConfig() bool {
	return svr.GlobalCfgDone
}
