// ribdUtils.go
package main

import (
	"ribd"
	"encoding/json"
	"github.com/op/go-nanomsg"
	"net"
	"errors"
	"utils/patriciaDB"
	"l3/rib/ribdCommonDefs"
)

var RouteProtocolTypeMapDB = make(map[string]int)
var ReverseRouteProtoTypeMapDB = make(map[int]string)

func BuildRouteProtocolTypeMapDB() {
	RouteProtocolTypeMapDB["Connected"] = ribdCommonDefs.CONNECTED
	RouteProtocolTypeMapDB["BGP"]       = ribdCommonDefs.BGP
	RouteProtocolTypeMapDB["Static"]       = ribdCommonDefs.STATIC
	
	//reverse
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.CONNECTED] = "Connected"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.BGP] = "BGP"
	ReverseRouteProtoTypeMapDB[ribdCommonDefs.STATIC] = "Static"
}

func RouteNotificationSend(PUB *nanomsg.PubSocket, route ribd.Routes) {
	logger.Println("RouteNotificationSend") 
	msgBuf := ribdCommonDefs.RoutelistInfo{RouteInfo : route}
	msgbufbytes, err := json.Marshal( msgBuf)
    msg := ribdCommonDefs.RibdNotifyMsg {MsgType:ribdCommonDefs.NOTIFY_ROUTE_CREATED, MsgBuf: msgbufbytes}
	buf, err := json.Marshal( msg)
	if err != nil {
	   logger.Println("Error in marshalling Json")
	   return
	}
   	logger.Println("buf", buf)
   	PUB.Send(buf, nanomsg.DontWait)
}

func getIPInt(ip net.IP) (ipInt int, err error) {
	if ip == nil {
		logger.Printf("ip address %v invalid\n", ip)
		return ipInt, errors.New("Invalid destination network IP Address")
	}
	ip = ip.To4()
	parsedPrefixIP := int(ip[3]) | int(ip[2])<<8 | int(ip[1])<<16 | int(ip[0])<<24
	ipInt = parsedPrefixIP
	return ipInt, nil
}

func getIP(ipAddr string) (ip net.IP, err error) {
	ip = net.ParseIP(ipAddr)
	if ip == nil {
		return ip, errors.New("Invalid destination network IP Address")
	}
	ip = ip.To4()
	return ip, nil
}

func getPrefixLen(networkMask net.IP) (prefixLen int, err error) {
	ipInt, err := getIPInt(networkMask)
	if err != nil {
		return -1, err
	}
	for prefixLen = 0; ipInt != 0; ipInt >>= 1 {
		prefixLen += ipInt & 1
	}
	return prefixLen, nil
}

func getNetworkPrefix(destNetIp net.IP, networkMask net.IP) (destNet patriciaDB.Prefix, err error) {
	prefixLen, err := getPrefixLen(networkMask)
	if err != nil {
		return destNet, err
	}
	/*   ip, err := getIP(destNetIp)
	    if err != nil {
	        logger.Println("Invalid destination network IP Address")
			return destNet, err
	    }
	    vdestMaskIp,err := getIP(networkMask)
	    if err != nil {
	        logger.Println("Invalid network mask")
			return destNet, err
	    }*/
	vdestMask := net.IPv4Mask(networkMask[0], networkMask[1], networkMask[2], networkMask[3])
	netIp := destNetIp.Mask(vdestMask)
	numbytes := prefixLen / 8
	if (prefixLen % 8) != 0 {
		numbytes++
	}
	destNet = make([]byte, numbytes)
	for i := 0; i < numbytes; i++ {
		destNet[i] = netIp[i]
	}
	return destNet, nil
}
