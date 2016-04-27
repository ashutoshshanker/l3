package FSMgr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bgp/api"
	"l3/bgp/config"
	"l3/bgp/rpc"
	"l3/rib/ribdCommonDefs"
	"ribd"
	"ribdInt"
	"utils/logging"
)

/*  Init route manager with ribd client as its core
 */
func NewFSRouteMgr(logger *logging.Writer, fileName string) (*FSRouteMgr, error) {
	var ribdClient *ribd.RIBDServicesClient = nil
	ribdClientChan := make(chan *ribd.RIBDServicesClient)

	logger.Info("Connecting to RIBd")
	go rpc.StartRibdClient(logger, fileName, ribdClientChan)
	ribdClient = <-ribdClientChan
	if ribdClient == nil {
		logger.Err("Failed to connect to RIBd\n")
		return nil, errors.New("Failed to connect to RIBd")
	} else {
		logger.Info("Connected to RIBd")
	}

	mgr := &FSRouteMgr{
		plugin:     "ovsdb",
		ribdClient: ribdClient,
		logger:     logger,
	}

	return mgr, nil
}

/*  Start nano msg socket with ribd
 */
func (mgr *FSRouteMgr) Start() {
	mgr.ribSubSocket, _ = mgr.setupSubSocket(ribdCommonDefs.PUB_SOCKET_ADDR)
	mgr.ribSubBGPSocket, _ = mgr.setupSubSocket(ribdCommonDefs.PUB_SOCKET_BGPD_ADDR)
	go mgr.listenForRIBUpdates(mgr.ribSubSocket)
	go mgr.listenForRIBUpdates(mgr.ribSubBGPSocket)
	mgr.processRoutesFromRIB()
}

func (mgr *FSRouteMgr) setupSubSocket(address string) (*nanomsg.SubSocket, error) {
	var err error
	var socket *nanomsg.SubSocket
	if socket, err = nanomsg.NewSubSocket(); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to create subscribe socket %s",
			"error:%s", address, err))
		return nil, err
	}

	if err = socket.Subscribe(""); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to subscribe to \"\" on ",
			"subscribe socket %s, error:%s", address, err))
		return nil, err
	}

	if _, err = socket.Connect(address); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to connect to publisher socket %s,",
			"error:%s", address, err))
		return nil, err
	}

	mgr.logger.Info(fmt.Sprintf("Connected to publisher socker %s", address))
	if err = socket.SetRecvBuffer(1024 * 1024); err != nil {
		mgr.logger.Err(fmt.Sprintln("Failed to set the buffer size for",
			"subsriber socket %s, error:", address, err))
		return nil, err
	}
	return socket, nil
}

func (mgr *FSRouteMgr) listenForRIBUpdates(socket *nanomsg.SubSocket) {
	for {
		mgr.logger.Info("Read on RIB subscriber socket...")
		rxBuf, err := socket.Recv(0)
		if err != nil {
			mgr.logger.Err(fmt.Sprintln("Recv on RIB subscriber socket",
				"failed with error:", err))
			continue
		}
		mgr.logger.Info(fmt.Sprintln("RIB subscriber recv returned:", rxBuf))
		mgr.handleRibUpdates(rxBuf)
	}
}

func (mgr *FSRouteMgr) populateConfigRoute(route *ribdInt.Routes) *config.RouteInfo {
	rv := &config.RouteInfo{
		Ipaddr:           route.Ipaddr,
		Mask:             route.Mask,
		NextHopIp:        route.NextHopIp,
		Prototype:        int(route.Prototype),
		NetworkStatement: route.NetworkStatement,
		RouteOrigin:      route.RouteOrigin,
	}
	return rv
}

func (mgr *FSRouteMgr) handleRibUpdates(rxBuf []byte) {
	var routeListInfo ribdCommonDefs.RoutelistInfo
	routes := make([]*config.RouteInfo, 0)
	reader := bytes.NewReader(rxBuf)
	decoder := json.NewDecoder(reader)
	msg := ribdCommonDefs.RibdNotifyMsg{}
	updateMsg := "Add"
	if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
		updateMsg = "Remove"
	}

	for err := decoder.Decode(&msg); err == nil; err = decoder.Decode(&msg) {
		err = json.Unmarshal(msg.MsgBuf, &routeListInfo)
		if err != nil {
			mgr.logger.Err(fmt.Sprintf(
				"Unmarshal RIB route update failed with err %s", err))
		}
		mgr.logger.Info(fmt.Sprintln(updateMsg, "connected route, dest:",
			routeListInfo.RouteInfo.Ipaddr, "netmask:",
			routeListInfo.RouteInfo.Mask, "nexthop:",
			routeListInfo.RouteInfo.NextHopIp))
		route := mgr.populateConfigRoute(&routeListInfo.RouteInfo)
		routes = append(routes, route)
	}

	if len(routes) > 0 {
		if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_CREATED {
			api.SendRouteNotification(routes, make([]*config.RouteInfo, 0))
		} else if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
			api.SendRouteNotification(make([]*config.RouteInfo, 0), routes)
		} else {
			mgr.logger.Err(fmt.Sprintf("**** Received RIB update with ",
				"unknown type %d ****", msg.MsgType))
		}
	} else {
		mgr.logger.Err(fmt.Sprintf("**** Received RIB update type %d with no routes ****",
			msg.MsgType))
	}
}

func (mgr *FSRouteMgr) processRoutesFromRIB() {
	var currMarker ribdInt.Int
	var count ribdInt.Int
	count = 100
	for {
		mgr.logger.Info(fmt.Sprintln("Getting ", count,
			"objects from currMarker", currMarker))
		getBulkInfo, err := mgr.ribdClient.GetBulkRoutesForProtocol("BGP",
			currMarker, count)
		if err != nil {
			mgr.logger.Info(fmt.Sprintln("GetBulkRoutesForProtocol with err ", err))
			return
		}
		if getBulkInfo.Count == 0 {
			mgr.logger.Info("0 objects returned from GetBulkRoutesForProtocol")
			return
		}
		mgr.logger.Info(fmt.Sprintln("len(getBulkInfo.RouteList)  = ",
			len(getBulkInfo.RouteList), " num objects returned = ",
			getBulkInfo.Count))
		routes := make([]*config.RouteInfo, 0, len(getBulkInfo.RouteList))
		for idx, _ := range getBulkInfo.RouteList {
			route := mgr.populateConfigRoute(getBulkInfo.RouteList[idx])
			routes = append(routes, route)
		}
		api.SendRouteNotification(routes, make([]*config.RouteInfo, 0))
		if getBulkInfo.More == false {
			mgr.logger.Info("more returned as false, so no more get bulks")
			return
		}
		currMarker = ribdInt.Int(getBulkInfo.EndIdx)
	}
}

func (mgr *FSRouteMgr) GetNextHopInfo(ipAddr string) (*config.NextHopInfo, error) {
	info, err := mgr.ribdClient.GetRouteReachabilityInfo(ipAddr)
	if err != nil {
		mgr.logger.Err(fmt.Sprintln("Getting route reachability for ",
			ipAddr, "failed, error:", err))
		return nil, err
	}
	reachInfo := &config.NextHopInfo{
		Ipaddr:         info.Ipaddr,
		Mask:           info.Mask,
		Metric:         int32(info.Metric),
		NextHopIp:      info.NextHopIp,
		IsReachable:    info.IsReachable,
		NextHopIfType:  int32(info.NextHopIfType),
		NextHopIfIndex: int32(info.NextHopIfIndex),
	}
	return reachInfo, err
}

func (mgr *FSRouteMgr) createRibdIPv4RouteCfg(cfg *config.RouteConfig,
	create bool) *ribd.IPv4Route {
	nextHopIfTypeStr := ""
	if create {
		nextHopIfTypeStr, _ = ribdCommonDefs.GetNextHopIfTypeStr(
			ribdInt.Int(cfg.IntfType))
	}
	rCfg := ribd.IPv4Route{
		Cost:              cfg.Cost,
		Protocol:          cfg.Protocol,
		NextHopIp:         cfg.NextHopIp,
		NetworkMask:       cfg.NetworkMask,
		DestinationNw:     cfg.DestinationNw,
		OutgoingIntfType:  nextHopIfTypeStr,
		OutgoingInterface: cfg.OutgoingInterface,
	}
	return &rCfg
}

func (mgr *FSRouteMgr) CreateRoute(cfg *config.RouteConfig) {
	mgr.ribdClient.OnewayCreateIPv4Route(mgr.createRibdIPv4RouteCfg(cfg,
		true /*create*/))
}

func (mgr *FSRouteMgr) DeleteRoute(cfg *config.RouteConfig) {
	mgr.ribdClient.OnewayDeleteIPv4Route(mgr.createRibdIPv4RouteCfg(cfg,
		false /*delete*/))
}
