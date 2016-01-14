package main

import (
    "arpd"
    "fmt"
    "github.com/google/gopacket/pcap"
    "errors"
    "time"
)

/***** Thrift APIs ******/
func (m ARPServiceHandler) UpdateUntaggedPortToVlanMap(vlanid arpd.Int,
        untaggedPorts string) (rval arpd.Int, err error) {

    //logger.Println("Received UpdateUntaggedPortToVlanMap(): vlanid:", vlanid, "ports:", untaggedPorts)
    logWriter.Info(fmt.Sprintln("Received UpdateUntaggedPortToVlanMap(): vlanid:", vlanid, "ports:", untaggedPorts))

    portTagStr, err := parseUsrPortStrToPbm(untaggedPorts)
    if err != nil {
        return 0, err
    }

    for i := 0; i < len(portTagStr); i++ {
        if (portTagStr[i] - '0') == 1 {
            ent := port_property_map[i]
            ent.untagged_vlanid = int(vlanid)
            port_property_map[i] = ent
        }
    }

    return rval, nil
}

func (m ARPServiceHandler) ResolveArpIPV4(targetIp string,
        iftype arpd.Int, vlan_id arpd.Int) (rc arpd.Int, err error) {

        //logger.Println("Calling ResolveArpIPv4...", targetIp, " ", int32(iftype), " ", int32(vlan_id))
        logWriter.Info(fmt.Sprintln("ResolveArpIPv4...", targetIp, " ", int32(iftype), " ", int32(vlan_id)))
        if targetIp == "0.0.0.0" {
            logWriter.Err(fmt.Sprintln("Not resolving Mac address for ", targetIp))
            err = errors.New(fmt.Sprintln("Not resolving ARP for", targetIp))
            return 0, err
        }
        ip_addr, err := getIPv4ForInterface(iftype, vlan_id)
        if len(ip_addr) == 0 || err != nil {
            logWriter.Err(fmt.Sprintf("Failed to get the ip address of ifType:", iftype, "VLAN:", vlan_id))
            return ARP_ERR_REQ_FAIL, err
        }
        //logger.Println("Local IP address of is:", ip_addr)
        logWriter.Info(fmt.Sprintln("Local IP address of is:", ip_addr))
        //var linux_device string
        if portdClient.IsConnected {
                //linux_device, err := portdClient.ClientHdl.GetLinuxIfc(int32(iftype), int32(vlan_id))
                linux_device, err := getLinuxIfc(int(iftype), int(vlan_id))
/*
                for _, port_cfg := range portCfgList {
                    linux_device = port_cfg.Ifname
*/
                    //logger.Println("linux_device ", linux_device)
                    logWriter.Info(fmt.Sprintln("linux_device ", linux_device))
                    if err != nil {
                            logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", vlan_id, "type : ", iftype))
                            return ARP_ERR_REQ_FAIL, err
                    }
                    logWriter.Info(fmt.Sprintln("Server:Connecting to device ", linux_device))
                    handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout_pcap)
                    if handle == nil {
                            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", linux_device, err))
                            return 0, err
                    }
/*
                    mac_addr, err := getMacAddrInterfaceName(port_cfg.Ifname)
                    if err != nil {
                        logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", port_cfg.Ifname))
                        continue
                    }
                    logger.Println("MAC addr of ", port_cfg.Ifname, ": ", mac_addr)
*/
                    mac_addr, err := getMacAddrInterfaceName(linux_device)
                    if err != nil {
                        logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", linux_device))
                    }
                    //logger.Println("MAC addr of ", linux_device, ": ", mac_addr)
                    logWriter.Info(fmt.Sprintln("MAC addr of ", linux_device, ": ", mac_addr))

                    go processPacket(targetIp, iftype, vlan_id, handle, mac_addr, ip_addr)
/*
                }
*/

        } else {
                logWriter.Err("portd client is not connected.")
                //logger.Println("Portd is not connected.")
        }

        return ARP_REQ_SUCCESS, err

}


/*
 * Function: SetArpConfig
 */
func (m ARPServiceHandler) SetArpConfig(refresh_timeout arpd.Int) (rc arpd.Int, err error) {
        ref_timeout := int(refresh_timeout)
        logger.Println("Received ARP timeout value:", refresh_timeout)
        if ref_timeout < min_refresh_timeout {
            logger.Println("Refresh Timeout is below minimum allowed refresh timeout")
            return 0, errors.New(fmt.Sprintln("Timeout value too low. Minimum timeout value is %s seconds", min_refresh_timeout))
        } else if ref_timeout == config_refresh_timeout {
            logger.Println("Refresh Timeout is same as already configured refresh timeout")
            return 0, nil
        }

        timeout_counter = ref_timeout / timer_granularity
        go updateCounterInArpCache()
        return 0, nil

}

func (m ARPServiceHandler) GetBulkArpEntry(fromIndex arpd.Int, count arpd.Int) (arpEntry *arpd.ArpEntryBulk, err error) {
    logger.Println("Inside GetBulkArpEntry...")
    var tempArpEntry []arpd.ArpEntry = make([]arpd.ArpEntry, count)
    var nextArpEntry *arpd.ArpEntry
    var returnArpEntry []*arpd.ArpEntry
    var returnArpEntryBulk arpd.ArpEntryBulk
    var more bool
    var index, cnt, endIdx int
    arpEntry = &returnArpEntryBulk
    var ret bool

    if arpSlice == nil {
        logger.Println("Arp Entry Slice is not initialized")
        return arpEntry, err
    }

    ret = arp_entry_timer.Stop()
    if ret == false {
        logWriter.Err("Arpd is busy refreshing arp cache")
        err = errors.New("Arpd is busy refreshing arp cache")
        return arpEntry, err
    }

    index = int(fromIndex)
    cnt = int(count)
    length := len(arpSlice)

    if index + cnt >= length {
        cnt = length - index
        endIdx = 0
        more = false
    } else {
        endIdx = index + cnt
        more = true
    }

    for i := 0; i < cnt; i++ {
        arp_entry_req_chl<- arpEntryRequestMsg {
                                idx:    (index + i),
                            }

        arp_res_msg := <-arp_entry_res_chl
        nextArpEntry = &tempArpEntry[i]
        if arp_res_msg.arp_msg.valid == true {
            nextArpEntry.IpAddr = arp_res_msg.arp_msg.ipAddr
            nextArpEntry.MacAddr = arp_res_msg.arp_msg.macAddr
            nextArpEntry.Vlan = arpd.Int(arp_res_msg.arp_msg.vlan)
            nextArpEntry.Intf = arp_res_msg.arp_msg.intf
            curTime := time.Now()
            expiryTime := time.Duration(timer_granularity * timeout_counter) * time.Second
            timeElapsed := curTime.Sub(arp_res_msg.arp_msg.timestamp)
            timeLeft := expiryTime - timeElapsed
            nextArpEntry.ExpiryTimeLeft = timeLeft.String()
        } else {
            nextArpEntry.IpAddr = arp_res_msg.arp_msg.ipAddr
            nextArpEntry.MacAddr = arp_res_msg.arp_msg.macAddr
            nextArpEntry.Vlan = -1
            nextArpEntry.Intf = "none"
            nextArpEntry.ExpiryTimeLeft = "N/A"
        }
        if len(returnArpEntry) == 0 {
            returnArpEntry = make([]*arpd.ArpEntry, 0)
        }
        returnArpEntry = append(returnArpEntry, nextArpEntry)
    }
    arpEntry.ArpList = returnArpEntry
    arpEntry.StartIdx = fromIndex
    arpEntry.EndIdx = arpd.Int(endIdx)
    arpEntry.More = more
    arpEntry.Count = arpd.Int(cnt)

    arp_entry_timer.Reset(arp_entry_duration)

    return arpEntry, err
}

func (m ARPServiceHandler) ArpProbeV4Intf(ipAddr string, vlan_id arpd.Int, iftype arpd.Int) (rc arpd.Int, err error) {
    logger.Println("ArpProbeV4Intf() : ipAddr:", ipAddr, "vlan_id:", vlan_id, "iftype:", iftype)
    if portdClient.IsConnected {
        //linux_device, err := portdClient.ClientHdl.GetLinuxIfc(int32(iftype), int32(vlan_id))
        linux_device, err := getLinuxIfc(int(iftype), int(vlan_id))
        //logger.Println("linux_device ", linux_device)
        logWriter.Info(fmt.Sprintln("linux_device ", linux_device))
        if err != nil {
                logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", vlan_id, "type : ", iftype))
                return ARP_ERR_REQ_FAIL, err
        }
        logWriter.Info(fmt.Sprintln("Server:Connecting to device ", linux_device))
        handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout_pcap)
        if handle == nil {
                logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", linux_device, err))
                return 0, err
        }

        mac_addr, err := getMacAddrInterfaceName(linux_device)
        if err != nil {
            logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", linux_device))
        }
        //logger.Println("MAC addr of ", linux_device, ": ", mac_addr)
        logWriter.Info(fmt.Sprintln("MAC addr of ", linux_device, ": ", mac_addr))

        go sendArpProbe(ipAddr, handle, mac_addr)
    } else {
            logWriter.Err("portd client is not connected.")
            //logger.Println("Portd is not connected.")
    }

    return ARP_REQ_SUCCESS, err
}
