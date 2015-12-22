package main

import (
    "net"
    "fmt"
    "arpd"
    "strconv"
    "strings"
)

func getIP(ipAddr string) (ip net.IP, err int) {
        ip = net.ParseIP(ipAddr)
        if ip == nil {
                return ip, ARP_PARSE_ADDR_FAIL
        }
        ip = ip.To4()
        return ip, ARP_REQ_SUCCESS
}

func getHWAddr(macAddr string) (mac net.HardwareAddr, err error) {
        mac, err = net.ParseMAC(macAddr)
        if mac == nil {
                return mac, err
        }

        return mac, nil
}

func getMacAddrInterfaceName(ifName string) (macAddr string, err error) {

        ifi, err := net.InterfaceByName(ifName)
        if err != nil {
            logWriter.Err(fmt.Sprintf("Failed to get the mac address of ", ifName))
            return macAddr, err
        }
        macAddr = ifi.HardwareAddr.String()
        return macAddr, nil
}

func getIPv4ForInterfaceName(ifname string) (iface_ip string, err error) {
    interfaces, err := net.Interfaces()
    if err != nil {
        logWriter.Err(fmt.Sprintf("Failed to get the interface"))
        return "", err
    }
    for _, inter := range interfaces {
        if inter.Name == ifname {
            if addrs, err := inter.Addrs(); err == nil {
                for _, addr := range addrs {
                    switch ip := addr.(type) {
                        case *net.IPNet:
                            if ip.IP.DefaultMask() != nil {
                                return (ip.IP).String(), nil
                            }
                    }
                }
            } else {
                logWriter.Err(fmt.Sprintf("Failed to get the ip address of", ifname))
                return "", err
            }
        }
    }
    return "", err
}

func getIPv4ForInterface(iftype arpd.Int, vlan_id arpd.Int) (ip_addr string, err error) {
    var if_name string

    if iftype == 0 { //VLAN
        if_name = fmt.Sprintf("SVI%d", vlan_id)
    } else if iftype == 1 { //PHY
        if_name = fmt.Sprintf("fpPort-", vlan_id)
    } else {
        return "", err
    }

    //logger.Println("Local Interface name =", if_name)
    logWriter.Info(fmt.Sprintln("Local Interface name =", if_name))
    return getIPv4ForInterfaceName(if_name)
}

//Note: Caller validates that portStr is a valid port range string
func parsePortRange(portStr string) (int, int, error) {
        portNums := strings.Split(portStr, "-")
        startPort, err := strconv.Atoi(portNums[0])
        if err != nil {
                return 0, 0, err
        }
        endPort, err := strconv.Atoi(portNums[1])
        if err != nil {
                return 0, 0, err
        }
        return startPort, endPort, nil
}


/*
 * Utility function to parse from a user specified port string to a port bitmap.
 * Supported formats for port string shown below:
 * - 1,2,3,10 (comma separate list of ports)
 * - 1-10,24,30-31 (hypen separated port ranges)
 * - 00011 (direct port bitmap)
 */
func parseUsrPortStrToPbm(usrPortStr string) (string, error) {
        //FIXME: Assuming max of 256 ports, create common def (another instance in main.go)
        var portList [256]int
        var pbmStr string = ""
        //Handle ',' separated strings
        if strings.Contains(usrPortStr, ",") {
                commaSepList := strings.Split(usrPortStr, ",")
                for _, subStr := range commaSepList {
                        //Substr contains '-' separated range
                        if strings.Contains(subStr, "-") {
                                startPort, endPort, err := parsePortRange(subStr)
                                if err != nil {
                                        return pbmStr, err
                                }
                                for port := startPort; port <= endPort; port++ {
                                        portList[port] = 1
                                }
                        } else {
                                //Substr is a port number
                                port, err := strconv.Atoi(subStr)
                                if err != nil {
                                        return pbmStr, err
                                }
                                portList[port] = 1
                        }
                }
        } else if strings.Contains(usrPortStr, "-") {
                //Handle '-' separated range
                startPort, endPort, err := parsePortRange(usrPortStr)
                if err != nil {
                        return pbmStr, err
                }
                for port := startPort; port <= endPort; port++ {
                        portList[port] = 1
                }
        } else {
        if len(usrPortStr) > 1 {
            //Port bitmap directly specified
            return usrPortStr, nil
        } else {
            //Handle single port number
            port, err := strconv.Atoi(usrPortStr)
            if err != nil {
                return pbmStr, err
            }
            portList[port] = 1
        }
        }
        //Convert portList to port bitmap string
        var zeroStr string = ""
        for _, port := range portList {
                if port == 1 {
                        pbmStr += zeroStr
                        pbmStr += "1"
                        zeroStr = ""
                } else {
                        zeroStr += "0"
                }
        }
        return pbmStr, nil
}

func getInterfaceNameByIndex(index int) (ifName string, err error) {
    ifi, err := net.InterfaceByIndex(index)
    if err != nil {
        logWriter.Err(fmt.Sprintf("Unable to get interface name.", ifi, err))
        return "", err
    }
    return ifi.Name, nil
}


