// ribNetlink.go
package server

import (
	"asicd/asicdCommonDefs"
	"fmt"
	"github.com/vishvananda/netlink"
	"l3/rib/ribdCommonDefs"
	"net"
)

func delLinuxRoute(route RouteInfoRecord) {
	logger.Info("delLinuxRoute")
	if route.protocol == ribdCommonDefs.CONNECTED {
		logger.Info("This is a connected route, do nothing")
		return
	}
	mask := net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3])
	maskedIP := route.destNetIp.Mask(mask)
	logger.Info(fmt.Sprintln("mask = ", mask, " destip:= ", route.destNetIp, " maskedIP ", maskedIP))
	dst := &net.IPNet{
		IP:   maskedIP, //route.destNetIp,
		Mask: mask,     //net.CIDRMask(prefixLen, 32),//net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3]),
	}
	ifId := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(route.nextHopIfIndex), int(route.nextHopIfType))
	logger.Info(fmt.Sprintln("IfId = ", ifId))
	intfEntry, ok := IntfIdNameMap[ifId]
	if !ok {
		logger.Err(fmt.Sprintln("IfName not updated for ifId ", ifId))
		return
	}
	ifName := intfEntry.name
	logger.Info(fmt.Sprintln("ifName = ", ifName, " for ifId ", ifId))
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		logger.Err(fmt.Sprintln("LinkByIndex call failed with error ", err, "for linkName ", ifName))
		return
	}

	lxroute := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Gw: route.nextHopIp}
	err = netlink.RouteDel(&lxroute)
	if err != nil {
		logger.Err(fmt.Sprintln("Route delete call failed with error ", err))
	}
	return
}

func addLinuxRoute(route RouteInfoRecord) {
	logger.Info("addLinuxRoute")
	if route.protocol == ribdCommonDefs.CONNECTED {
		logger.Info("This is a connected route, do nothing")
		return
	}
	mask := net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3])
	maskedIP := route.destNetIp.Mask(mask)
	logger.Info(fmt.Sprintln("mask = ", mask, " destip:= ", route.destNetIp, " maskedIP ", maskedIP))
	dst := &net.IPNet{
		IP:   maskedIP, //route.destNetIp,
		Mask: mask,     //net.CIDRMask(prefixLen, 32),//net.IPv4Mask(route.networkMask[0], route.networkMask[1], route.networkMask[2], route.networkMask[3]),
	}
	ifId := asicdCommonDefs.GetIfIndexFromIntfIdAndIntfType(int(route.nextHopIfIndex), int(route.nextHopIfType))
	logger.Info(fmt.Sprintln("IfId = ", ifId))
	intfEntry, ok := IntfIdNameMap[ifId]
	if !ok {
		logger.Err(fmt.Sprintln("IfName not updated for ifId ", ifId))
		return
	}
	ifName := intfEntry.name
	logger.Info(fmt.Sprintln("ifName = ", ifName, " for ifId ", ifId))
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		logger.Err(fmt.Sprintln("LinkByIndex call failed with error ", err, "for linkName ", ifName))
		return
	}

	logger.Info(fmt.Sprintln("adding linux route for dst.ip= ", dst.IP.String(), " mask: ", dst.Mask.String(), "Gw: ", route.nextHopIp, " maskedIP: ", maskedIP))
	lxroute := netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Gw: route.nextHopIp}
	routeList, err := netlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{Dst: dst}, netlink.RT_FILTER_DST)
	logger.Info(fmt.Sprintln("After RouteListFiltered call  err :", err, " len(routeList) :", len(routeList)))
	if routeList != nil && len(routeList) > 0 {
		logger.Info(fmt.Sprintln("Appending to existing route"))
		err = netlink.RouteAppend(&lxroute)
	} else {
		logger.Info(fmt.Sprintln("Adding new route"))
		err = netlink.RouteAdd(&lxroute)
	}
	if err != nil {
		logger.Err(fmt.Sprintln("Route add call failed with error ", err))
	}
	return
}
func (ribdServiceHandler *RIBDServer) StartNetlinkServer() {
	logger.Info("Starting the netlinkserver loop")
	for {
		select {
		case route := <-ribdServiceHandler.NetlinkAddRouteCh:
			logger.Info(" received message on NetlinkAddRouteCh")
			addLinuxRoute(route)
		case route := <-ribdServiceHandler.NetlinkDelRouteCh:
			logger.Info(" received message on NetlinkDelRouteCh")
			delLinuxRoute(route)
		}
	}
}
