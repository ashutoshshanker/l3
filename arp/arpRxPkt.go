package main

import (
        "arpd"
        "fmt"
        "github.com/google/gopacket"
        "github.com/google/gopacket/layers"
        "github.com/google/gopacket/pcap"
        _ "github.com/mattn/go-sqlite3"
        "github.com/vishvananda/netlink"
        "net"
        "bytes"
        "utils/commonDefs"
)


func processArpReply(arp *layers.ARP, port_id int, myMac net.HardwareAddr, if_Name string) {
        src_Mac := net.HardwareAddr(arp.SourceHwAddress)
        src_ip_addr := (net.IP(arp.SourceProtAddress)).String()
        dest_Mac := net.HardwareAddr(arp.DstHwAddress)
        dest_ip_addr := (net.IP(arp.DstProtAddress)).String()
        logWriter.Info(fmt.Sprintln("Received Arp response SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr, "DST_MAC:", dest_Mac))
        if dest_ip_addr == "0.0.0.0" {
                logWriter.Err(fmt.Sprintln("Recevied reply for ARP Probe and there is a conflicting IP Address", src_ip_addr))
                return
        }
        ent, exist := arp_cache.arpMap[src_ip_addr]
        if exist {
                if ent.port == -2 {
                        port_map_ent, exists := port_property_map[port_id]
                        var vlan_id arpd.Int
                        if exists {
                                vlan_id = arpd.Int(port_map_ent.untagged_vlanid)
                        } else {
                                // vlan_id = 1
                                return
                        }
                        arp_cache_update_chl <- arpUpdateMsg{
                                ip: src_ip_addr,
                                ent: arpEntry{
                                        macAddr: src_Mac,
                                        vlanid:  vlan_id,
                                        valid:   true,
                                        port:    port_id,
                                        ifName:  if_Name,
                                        ifType:  ent.ifType,
                                        localIP: ent.localIP,
                                        counter: timeout_counter,
                                },
                                msg_type: 6,
                        }
                } else {
                        arp_cache_update_chl <- arpUpdateMsg{
                                ip: src_ip_addr,
                                ent: arpEntry{
                                        macAddr: src_Mac,
                                        vlanid:  ent.vlanid,
                                        valid:   true,
                                        port:    port_id,
                                        ifName:  if_Name,
                                        ifType:  ent.ifType,
                                        localIP: ent.localIP,
                                        counter: timeout_counter,
                                },
                                msg_type: 1,
                        }
                }
        } else {
                port_map_ent, exists := port_property_map[port_id]
                var vlan_id arpd.Int
                var ifType arpd.Int
                if exists {
                        vlan_id = arpd.Int(port_map_ent.untagged_vlanid)
                        ifType = arpd.Int(commonDefs.L2RefTypeVlan)
                } else {
                        // vlan_id = 1
                        return
                }
                arp_cache_update_chl <- arpUpdateMsg{
                        ip: src_ip_addr,
                        ent: arpEntry{
                                macAddr: src_Mac,
                                vlanid:  vlan_id, // Need to be re-visited
                                valid:   true,
                                port:    port_id,
                                ifName:  if_Name,
                                ifType:  ifType,
                                localIP: dest_ip_addr,
                                counter: timeout_counter,
                        },
                        msg_type: 3,
                }
        }

}

func processArpRequest(arp *layers.ARP, port_id int, myMac net.HardwareAddr, if_Name string) {
        src_Mac := net.HardwareAddr(arp.SourceHwAddress)
        src_ip_addr := (net.IP(arp.SourceProtAddress)).String()
        dest_ip_addr := (net.IP(arp.DstProtAddress)).String()
        dstip := net.ParseIP(dest_ip_addr)
        //logger.Println("Received Arp request SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr)
        logWriter.Info(fmt.Sprintln("Received Arp Request SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr))
        _, exist := arp_cache.arpMap[src_ip_addr]
        if !exist {
                port_map_ent, exists := port_property_map[port_id]
                var vlan_id arpd.Int
                var ifType arpd.Int
                if exists {
                        vlan_id = arpd.Int(port_map_ent.untagged_vlanid)
                        ifType = arpd.Int(commonDefs.L2RefTypeVlan)
                } else {
                        // vlan_id = 1
                        return
                }
                if src_ip_addr == "0.0.0.0" { // ARP Probe Request
                        local_ip_addr, _ := getIPv4ForInterface(arpd.Int(0), arpd.Int(vlan_id))
                        if local_ip_addr == dest_ip_addr {
                                // Send Arp Reply for ARP Probe
                                logger.Println("Linux will Send Arp Reply for recevied ARP Probe because of conflicting address")
                                return
                        }
                }

                if src_ip_addr == dest_ip_addr { // Gratuitous ARP Request
                        logger.Println("Received a Gratuitous ARP from ", src_ip_addr)
                } else { // Any other ARP request which are not locally originated
                        route, err := netlink.RouteGet(dstip)
                        var ifName string
                        for _, rt := range route {
                                if rt.LinkIndex > 0 {
                                        ifName, err = getInterfaceNameByIndex(rt.LinkIndex)
                                        if err != nil || ifName == "" {
                                                //logger.Println("Unable to get the outgoing interface", err)
                                                logWriter.Err(fmt.Sprintf("Unable to get the outgoing interface", err))
                                                return
                                        }
                                }
                        }
                        logger.Println("Outgoing interface:", ifName)
                        if ifName != "lo" {
                                return
                        }
                }
                arp_cache_update_chl <- arpUpdateMsg{
                        ip: src_ip_addr,
                        ent: arpEntry{
                                macAddr: src_Mac,
                                vlanid:  vlan_id, // Need to be re-visited
                                valid:   true,
                                port:    port_id,
                                ifName:  if_Name,
                                ifType:  ifType,
                                localIP: dest_ip_addr,
                                counter: timeout_counter,
                        },
                        msg_type: 4,
                }
        }
}

func processArpPackets(arpLayer gopacket.Layer, port_id int, myMac net.HardwareAddr, if_Name string) {
        arp := arpLayer.(*layers.ARP)
        if arp == nil {
                logWriter.Err("Arp layer returns nil")
                return
        }
        if bytes.Equal([]byte(myMac), arp.SourceHwAddress) {
                logWriter.Err("Received ARP Packet with our own MAC Address, hence not processing it")
                return
        }

        if arp.Operation == layers.ARPReply {
                processArpReply(arp, port_id, myMac, if_Name)
        } else if arp.Operation == layers.ARPRequest {
                processArpRequest(arp, port_id, myMac, if_Name)
        }
}


func processIpPackets(packet gopacket.Packet, port_id int, myMac net.HardwareAddr, if_Name string) {
        //logger.Println("Not an ARP Packet")
        if nw := packet.NetworkLayer(); nw != nil {
                src_ip, dst_ip := nw.NetworkFlow().Endpoints()
                dst_ip_addr := dst_ip.String()
                //dstip := net.ParseIP(dst_ip_addr)
                src_ip_addr := src_ip.String()

                _, exist := arp_cache.arpMap[dst_ip_addr]
                if !exist {
                        ifName, vlan_id, ifType, ret := isInLocalSubnet(dst_ip_addr)
                        if ret == false {
                                return
                        }
                        logWriter.Info(fmt.Sprintln("Sending ARP for dst_ip:", dst_ip_addr, "Outgoing Interface:", ifName, "vlanId:", vlan_id, "ifType:", ifType))
                        go createAndSendArpReuqest(dst_ip_addr, ifName, arpd.Int(vlan_id), arpd.Int(ifType))
                }
                _, exist = arp_cache.arpMap[src_ip_addr]
                if !exist {
                        ifName, vlan_id, ifType, ret := isInLocalSubnet(src_ip_addr)
                        if ret == false {
                                return
                        }
                        logWriter.Info(fmt.Sprintln("Sending ARP for src_ip:", src_ip_addr, "Outgoing Interface:", ifName, "vlanId:", vlan_id, "ifType:", ifType))
                        go createAndSendArpReuqest(src_ip_addr, ifName, arpd.Int(vlan_id), arpd.Int(ifType))
                }
        }
}


//ToDo: This function need to cleaned up
/*
 *@fn receiveArpResponse
 * Process ARP response from the interface for ARP
 * req sent for targetIp
 */
func receiveArpResponse(rec_handle *pcap.Handle,
        myMac net.HardwareAddr, port_id int, if_Name string) {

        src := gopacket.NewPacketSource(rec_handle, layers.LayerTypeEthernet)
        in := src.Packets()
        for {
                packet, ok := <-in
                if ok {
                        //logger.Println("Receive some packet on arp response thread")
                        arpLayer := packet.Layer(layers.LayerTypeARP)
                        if arpLayer != nil {
                                processArpPackets(arpLayer, port_id, myMac, if_Name)
                        } else {
                                processIpPackets(packet, port_id, myMac, if_Name)
                        }
                }
        }
}


