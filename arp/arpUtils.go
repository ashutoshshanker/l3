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
    if_name, _ := getLinuxIfc(int(iftype), int(vlan_id))
    if if_name == "" {
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

func getInterfaceNameByIndex(index int) (ifName string, err error) {
    ifi, err := net.InterfaceByIndex(index)
    if err != nil {
        logWriter.Err(fmt.Sprintf("Unable to get interface name.", ifi, err))
        return "", err
    }
    return ifi.Name, nil
}

func getIfIndex(portNum int) int32 {
        ent, exist := portLagPropertyMap[int32(portNum)]
        if exist {
                return ent.IfIndex
        }

        return int32(portNum)
}
