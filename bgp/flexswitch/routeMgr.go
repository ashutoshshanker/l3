package FSMgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bgp/api"
	"l3/bgp/config"
	_ "l3/bgp/packet"
	"l3/rib/ribdCommonDefs"
	_ "ribd"
	"ribdInt"
)

func (mgr *FSRouteMgr) Init() {
	mgr.ribSubSocket, _ = mgr.setupSubSocket(ribdCommonDefs.PUB_SOCKET_ADDR)
	mgr.ribSubBGPSocket, _ = mgr.setupSubSocket(ribdCommonDefs.PUB_SOCKET_BGPD_ADDR)
	go mgr.listenForRIBUpdates(mgr.ribSubSocket)
	go mgr.listenForRIBUpdates(mgr.ribSubBGPSocket)
	//mgr.processRoutesFromRIB()
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
			//			socketErrCh <- err
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
			mgr.logger.Err(fmt.Sprintf("Unmarshal RIB route update failed with err %s", err))
		}
		mgr.logger.Info(fmt.Sprintln(updateMsg, "connected route, dest:",
			routeListInfo.RouteInfo.Ipaddr, "netmask:",
			routeListInfo.RouteInfo.Mask, "nexthop:", routeListInfo.RouteInfo.NextHopIp))
		routes = append(routes, mgr.populateConfigRoute(&routeListInfo.RouteInfo))
	}

	if len(routes) > 0 {
		if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_CREATED {
			api.SendRouteNotification(routes, nil)
			//	mgr.ProcessConnectedRoutes(routes, nil) //make([]*ribdInt.Routes, 0))
		} else if msg.MsgType == ribdCommonDefs.NOTIFY_ROUTE_DELETED {
			api.SendRouteNotification(nil, routes)
			//	mgr.ProcessConnectedRoutes(nil, routes) //make([]*ribdInt.Routes, 0), routes)
		} else {
			mgr.logger.Err(fmt.Sprintf("**** Received RIB update with ",
				"unknown type %d ****", msg.MsgType))
		}
	} else {
		mgr.logger.Err(fmt.Sprintf("**** Received RIB update type %d with no routes ****",
			msg.MsgType))
	}
}

/*
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
		mgr.ProcessConnectedRoutes(getBulkInfo.RouteList, make([]*ribdInt.Routes, 0))
		if getBulkInfo.More == false {
			mgr.logger.Info("more returned as false, so no more get bulks")
			return
		}
		currMarker = ribdInt.Int(getBulkInfo.EndIdx)
	}
}

func (mgr *FSRouteMgr) ProcessConnectedRoutes(installedRoutes []*ribdInt.Routes,
	withdrawnRoutes []*ribdInt.Routes) {
	mgr.logger.Info(fmt.Sprintln("valid routes:", installedRoutes,
		"invalid routes:", withdrawnRoutes))
	valid := mgr.convertDestIPToIPPrefix(installedRoutes)
	invalid := mgr.convertDestIPToIPPrefix(withdrawnRoutes)
	updated, withdrawn, withdrawPath, updatedAddPaths :=
		mgr.Server.AdjRib.ProcessConnectedRoutes(
			mgr.Server.BgpConfig.Global.Config.RouterId.String(),
			mgr.Server.ConnRoutesPath, valid,
			invalid, mgr.Server.AddPathCount)
	updated, withdrawn, withdrawPath, updatedAddPaths =
		mgr.Server.CheckForAggregation(updated, withdrawn, withdrawPath,
			updatedAddPaths)
	mgr.Server.SendUpdate(updated, withdrawn, withdrawPath, updatedAddPaths)
}

func (mgr *FSRouteMgr) convertDestIPToIPPrefix(routes []*ribdInt.Routes) []packet.NLRI {
	dest := make([]packet.NLRI, 0, len(routes))
	for _, r := range routes {
		mgr.logger.Info(fmt.Sprintln("Route NS : ", r.NetworkStatement,
			" Route Origin ", r.RouteOrigin))
		ipPrefix := packet.ConstructIPPrefix(r.Ipaddr, r.Mask)
		dest = append(dest, ipPrefix)
	}
	return dest
}
*/
