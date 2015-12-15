package main

import (
	"arpd"
	"asicdServices"
	"bytes"
        "reflect"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"io/ioutil"
	"log/syslog"
	"net"
	"portdServices"
	"strconv"
	"time"
)

const (
	ARP_ERR_NOT_FOUND = iota
	ARP_PARSE_ADDR_FAIL
	ARP_ERR_REQ_FAIL
	ARP_ERR_RESP_FAIL
	ARP_ERR_ADD_FAIL
	ARP_REQ_SUCCESS
	ARP_ERR_LAST
)

const (
	ARP_ADD_ENTRY = iota
	ARP_DEL_ENTRY
	ARP_UPDATE_ENTRY
)

type ARPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	ARPClientBase
	ClientHdl *asicdServices.AsicdServiceClient
}

type PortdClient struct {
	ARPClientBase
	ClientHdl *portdServices.PortServiceClient
}

type arpEntry struct {
	macAddr net.HardwareAddr
	vlanid  arpd.Int
        valid   bool
        counter int
        port    int
        ifName  string
        ifType  arpd.Int
        localIP string
}

type arpCache struct {
	cacheTimeout time.Duration
	arpMap       map[string]arpEntry
	//dev_handle   *pcap.Handle
	hostTO       time.Duration
	routerTO     time.Duration
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}
type PortConfigJson struct {
	Port   int    `json:Port`
	Ifname string `json:Ifname`
}

type arpUpdateMsg struct {
        ip string
        ent arpEntry
        msg_type int
}

type pcapHandle struct {
        pcap_handle     *pcap.Handle
        ifName          string
}

/*
 * connection params.
 */
var (
	//device          string = "fpPort2"
	//device       string = "eth0"
	snapshot_len int32  = 1024 //packet capture length
	//promiscuous  bool   = true //mode
	promiscuous  bool   = false //mode
	err          error
	timeout      time.Duration = 60 * time.Second
        timeout_counter int = 10
        retry_cnt    int    = 2
	handle       *pcap.Handle  // handle for pcap connection
	//device_ip    string        = "40.1.1.1"
	//device_ip       string = "10.0.2.15"
	//filter_string   string = "arp host 10.1.10.1"
	//filter_optimize int    = 0
	logWriter       *syslog.Writer
	log_err         error
	//rec_handle      []*pcap.Handle
)
var arp_cache *arpCache
var asicdClient AsicdClient //Thrift client to connect to asicd
var portdClient PortdClient //portd services client

var pcap_handle_map map[int]pcapHandle

var portCfgList []PortConfigJson

var arp_cache_update_chl chan arpUpdateMsg = make(chan arpUpdateMsg, 100)


/*** TEMP DEFINES **/
//var myMac = "00:11:22:33:44:55"
//var myMac = "08:00:27:75:bc:4d"
//var myMac = "fa:15:f5:69:a4:c9"

/****** Utility APIs.****/
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

    logger.Println("Local Interface name =", if_name)
    return getIPv4ForInterfaceName(if_name)
}

/***** Thrift APIs ******/
func (m ARPServiceHandler) RestolveArpIPV4(targetIp string,
	iftype arpd.Int, vlan_id arpd.Int) (rc arpd.Int, err error) {

        logger.Println("Calling ResotolveArpIPv4...", targetIp, " ", int32(iftype), " ", int32(vlan_id))
        ip_addr, err := getIPv4ForInterface(iftype, vlan_id)
        if len(ip_addr) == 0 || err != nil {
            logWriter.Err(fmt.Sprintf("Failed to get the ip address of ifType:", iftype, "VLAN:", vlan_id))
            return ARP_ERR_REQ_FAIL, err
        }
        logger.Println("Local IP address of is:", ip_addr)
        var linux_device string
//        if portdClient.IsConnected {
//		linux_device, err := portdClient.ClientHdl.GetLinuxIfc(int32(iftype), int32(vlan_id))
                for _, port_cfg := range portCfgList {
                    linux_device = port_cfg.Ifname
                    logger.Println("linux_device ", linux_device)
/*
                    if err != nil {
                            logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", vlan_id, "type : ", iftype))
                            return ARP_ERR_REQ_FAIL, err
                    }
*/
                    logWriter.Err(fmt.Sprintln("Server:Connecting to device ", linux_device))
                    handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout)
                    if handle == nil {
                            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", linux_device, err))
                            return 0, err
                    }
                    mac_addr, err := getMacAddrInterfaceName(port_cfg.Ifname)
                    if err != nil {
                        logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", port_cfg.Ifname))
                        continue
                    }
                    logger.Println("MAC addr of ", port_cfg.Ifname, ": ", mac_addr)
                    go processPacket(targetIp, iftype, vlan_id, handle, mac_addr, ip_addr)
                }

//	} else {
//		logWriter.Err("portd client is not connected.")
//		logger.Println("Portd is not connected.")
//	}

	return ARP_REQ_SUCCESS, err

}

/*
 * @fn SetArpTimeout
 *     This API sets arp cache timeout.
 *     current defauls -
 *     hostTimeout = 10 sec
 *     routerTimeout = 10sec
 */
func (m ARPServiceHandler) SetArpTimeout(ifName string,
	hostTimeout int,
	routerTimeout int) (rc arpd.Int, err error) {
	cp := arp_cache
	if time.Duration(hostTimeout) > cp.hostTO {
		cp.hostTO = time.Duration(hostTimeout)
	}
	if time.Duration(routerTimeout) > cp.routerTO {
		cp.routerTO = time.Duration(routerTimeout)
	}
	return 0, nil

}

/*****Local API calls. *****/

/*
 * @fn ConnectToClients
 *     connect to other deamons such as asicd.
 */
func ConnectToClients(paramsFile string) {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		logWriter.Err("Error in reading configuration file")
		return
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		logWriter.Err("Error in Unmarshalling Json")
		return
	}

	for _, client := range clientsList {
		logWriter.Err("#### Client name is ")
		logWriter.Err(client.Name)
		if client.Name == "asicd" {
			logger.Printf("found asicd at port %d", client.Port)
			asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdClient.Transport, asicdClient.PtrProtocolFactory = CreateIPCHandles(asicdClient.Address)
			if asicdClient.Transport != nil && asicdClient.PtrProtocolFactory != nil {
				logWriter.Info("connecting to asicd")
				asicdClient.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdClient.Transport, asicdClient.PtrProtocolFactory)
				asicdClient.IsConnected = true
			}

		}
		if client.Name == "portd" {
			logger.Printf("found portd at port %d", client.Port)
			portdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			portdClient.Transport, portdClient.PtrProtocolFactory = CreateIPCHandles(portdClient.Address)
			if portdClient.Transport != nil && portdClient.PtrProtocolFactory != nil {
				logWriter.Info("connecting to asicd")
				portdClient.ClientHdl = portdServices.NewPortServiceClientFactory(portdClient.Transport, portdClient.PtrProtocolFactory)
				portdClient.IsConnected = true
			}

		}
	}
}

func initARPhandlerParams() {
	//init syslog
	logWriter, log_err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "ARPD_LOG")
	defer logWriter.Close()

	// Initialise arp cache.
	success := initArpCache()
	if success != true {
		logWriter.Err("server: Failed to initialise ARP cache")
		logger.Println("Failed to initialise ARP cache")
		return
	}

	//connect to asicd and portd
	configFile := params_dir + "/clients.json"
	ConnectToClients(configFile)
        go updateArpCache()
        go timeout_thread()
	initPortParams()
        /* Open Response thread */
        processResponse()

}

func BuildAsicToLinuxMap(cfgFile string) {
	bytes, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		logger.Println("Error in reading port configuration file")
		logWriter.Err(fmt.Sprintln("Error in reading port configuration file: ", err))
		return
	}
	err = json.Unmarshal(bytes, &portCfgList)
	if err != nil {
		logWriter.Err(fmt.Sprintln("Error in Unmarshalling Json, err=", err))
		return
	}
        pcap_handle_map = make(map[int]pcapHandle)
	for _, v := range portCfgList {
                logger.Println("BuildAsicToLinuxMap : iface = ", v.Ifname)
                logger.Println("BuildAsicToLinuxMap : port = ", v.Port)
		local_handle, err := pcap.OpenLive(v.Ifname, snapshot_len, promiscuous, timeout)
		if local_handle == nil {
			logWriter.Err(fmt.Sprintln("Server: No device found.: ", v.Ifname, err))
		}
                ent := pcap_handle_map[v.Port]
                ent.pcap_handle = local_handle
                ent.ifName = v.Ifname
                pcap_handle_map[v.Port] = ent
	}

}
func initPortParams() {
	//configFile := params_dir + "/clients.json"
	//ConnectToClients(configFile)
	portCfgFile := params_dir + "/portd.json"
	BuildAsicToLinuxMap(portCfgFile)
}

func processPacket(targetIp string, iftype arpd.Int, vlanid arpd.Int, handle *pcap.Handle, mac_addr string, localIp string) {
        logger.Println("processPacket() : Arp request for ", targetIp, "from", localIp)
	_, exist := arp_cache.arpMap[targetIp]
	if !exist {
                sendArpReq(targetIp, handle, mac_addr, localIp)
                arp_cache_update_chl <- arpUpdateMsg {
                                            ip: targetIp,
                                            ent: arpEntry {
                                                    macAddr: []byte{0,0,0,0,0,0},
                                                    vlanid: vlanid,
                                                    valid: false,
                                                    port: -1,
                                                    ifName: "",
                                                    ifType: iftype,
                                                    localIP: localIp,
                                                    counter: timeout_counter,
                                                 },
                                            msg_type: 0,
                                         }
	} else {
            // get MAC from cache.
            logger.Println("ARP entry already existed")
            printArpEntries()
            return
        }

	// get MAC from cache.
        logger.Println("ARP entry got created")
	printArpEntries()
	return
}

func processResponse() {
        for port_id, p_hdl := range pcap_handle_map {
                logger.Println("ifName = ", p_hdl.ifName, " Port = ", port_id)
                if p_hdl.pcap_handle == nil {
                    logger.Println("Hello handle is nil");
                    continue
                }
                mac_addr, err := getMacAddrInterfaceName(p_hdl.ifName)
                if err != nil {
                    logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", p_hdl.ifName))
                    continue
                }
                logger.Println("MAC addr of ", p_hdl.ifName, ": ", mac_addr)
                myMac_addr, fail := getHWAddr(mac_addr)
                if fail != nil {
                        logWriter.Err(fmt.Sprintf("corrupted my mac : ", mac_addr))
                        continue
                }
                go receiveArpResponse(p_hdl.pcap_handle, myMac_addr,
                                      port_id, p_hdl.ifName)
        }
	return
}

/*
 *@fn sendArpReq
 *  Send the ARP request for ip targetIP
 */
func sendArpReq(targetIp string, handle *pcap.Handle, myMac string, localIp string) int {
        logger.Println("sendArpReq(): sending arp requeust for targetIp ", targetIp,
                        "local IP ", localIp)

	source_ip, err := getIP(localIp)
	if err != ARP_REQ_SUCCESS {
		logWriter.Err(fmt.Sprintf("Corrupted source ip :  ", localIp))
		return ARP_ERR_REQ_FAIL
	}
	dest_ip, err := getIP(targetIp)
	if err != ARP_REQ_SUCCESS {
		logWriter.Err(fmt.Sprintf("Corrupted dest ip :  ", targetIp))
		return ARP_ERR_REQ_FAIL
	}
	myMac_addr, fail := getHWAddr(myMac)
	if fail != nil {
		logWriter.Err(fmt.Sprintf("corrupted my mac : ", myMac))
		return ARP_ERR_REQ_FAIL
	}
	arp_layer := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   myMac_addr,
		SourceProtAddress: source_ip,
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
	}
	eth_layer := layers.Ethernet{
		SrcMAC:       myMac_addr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	arp_layer.DstProtAddress = dest_ip
	gopacket.SerializeLayers(buffer, options, &eth_layer, &arp_layer)

        //logger.Println("Buffer : ", buffer)
        // send arp request and retry after timeout if arp cache is not updated
        if err := handle.WritePacketData(buffer.Bytes()); err != nil {
            return ARP_ERR_REQ_FAIL
        }
	return ARP_REQ_SUCCESS
}

/*
 *@fn receiveArpResponse
 * Process ARP response from the interface for ARP
 * req sent for targetIp
 */
func receiveArpResponse(rec_handle *pcap.Handle,
	myMac net.HardwareAddr, port_id int, if_Name string) {
        logger.Println("Starting Arp response recv thread on ", port_id, if_Name)
        var src_Mac net.HardwareAddr
	src := gopacket.NewPacketSource(rec_handle, layers.LayerTypeEthernet)
	in := src.Packets()
	for {
		packet, ok := <-in
		if ok {
                        logger.Println("Receive some packet on arp response thread")

			//vlan_layer := packet.Layer(layers.LayerTypeEthernet)
			//vlan_tag := vlan_layer.(*layers.Ethernet)
			//vlan_id := vlan_layer.LayerContents()
			//logWriter.Err(vlan_tag.)
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)
			if arp == nil {
				continue
			}
			if arp.Operation != layers.ARPReply || bytes.Equal([]byte(myMac), arp.SourceHwAddress) {
				continue
			}

			logger.Println("Received Arp response from: ", (net.IP(arp.SourceProtAddress)).String(), " ", (net.HardwareAddr(arp.SourceHwAddress)).String())

                        src_Mac = net.HardwareAddr(arp.SourceHwAddress)
                        src_ip_addr := (net.IP(arp.SourceProtAddress)).String()
                        //dest_Mac := net.HardwareAddr(arp.DstHwAddress)
                        dest_ip_addr := (net.IP(arp.DstProtAddress)).String()
                        ent, exist := arp_cache.arpMap[src_ip_addr]
                        if exist {
                            arp_cache_update_chl <- arpUpdateMsg {
                                                        ip: src_ip_addr,
                                                        ent: arpEntry {
                                                                macAddr: src_Mac,
                                                                vlanid: ent.vlanid,
                                                                valid: true,
                                                                port: port_id,
                                                                ifName: if_Name,
                                                                ifType: ent.ifType,
                                                                localIP: ent.localIP,
                                                                counter: timeout_counter,
                                                             },
                                                        msg_type: 1,
                                                     }
                        } else {
                            arp_cache_update_chl <- arpUpdateMsg {
                                                        ip: src_ip_addr,
                                                        ent: arpEntry {
                                                                macAddr: src_Mac,
                                                                vlanid: 1, // Need to be re-visited
                                                                valid: true,
                                                                port: port_id,
                                                                ifName: if_Name,
                                                                ifType: 1,
                                                                localIP: dest_ip_addr,
                                                                counter: timeout_counter,
                                                             },
                                                        msg_type: 3,
                                                     }
                        }
		}

	}
}

/*
 *@fn InitArpCache
 * Initiliase s/w cache. It also acts a reset API for timeout.
 */
func initArpCache() bool {
	arp_cache = &arpCache{arpMap: make(map[string]arpEntry)}
	//arp_cache.arpMap = make(map[string]arpEntry)
	logWriter.Err("InitArpCache done.")
	return true
}

/*
 * @fn UpdateArpCache
 *  Update IP to the ARP mapping for the hash table.
 */
func updateArpCache() {
    var cnt int
        for {
            msg := <-arp_cache_update_chl
            if msg.msg_type == 1 {
            //if msg.ent.vlanid == 0 {
                ent := arp_cache.arpMap[msg.ip]
                if reflect.DeepEqual(ent.macAddr, msg.ent.macAddr) &&
                   ent.valid == msg.ent.valid && ent.port == msg.ent.port &&
                   ent.ifName == msg.ent.ifName && ent.vlanid == msg.ent.vlanid &&
                   ent.ifType == msg.ent.ifType {
                   logger.Println("Updating counter after retry after expiry")
                   ent.counter = msg.ent.counter
                   arp_cache.arpMap[msg.ip] = ent
                   continue
                }
                ent.macAddr = msg.ent.macAddr
                ent.valid = msg.ent.valid
                ent.vlanid = msg.ent.vlanid
                // Every entry will be expired after 10 mins
                ent.counter = msg.ent.counter
                ent.port    = msg.ent.port
                ent.ifName  = msg.ent.ifName
                ent.ifType  = msg.ent.ifType
                ent.localIP = msg.ent.localIP
                arp_cache.arpMap[msg.ip] = ent
                logger.Println("1 updateArpCache(): ", arp_cache.arpMap[msg.ip])
                //3) Update asicd.
                if asicdClient.IsConnected {
                        logger.Println("1. Deleting an entry in asic for ", msg.ip)
                        rv, error := asicdClient.ClientHdl.DeleteIPv4Neighbor(msg.ip,
                                             "00:00:00:00:00:00", 0, 0)
                        logWriter.Err(fmt.Sprintf("Asicd Del rv: ", rv, " error : ", error))
                        logger.Println("1. Creating an entry in asic for ", msg.ip)
                        rv, error = asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
                                             (msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), (int32)(msg.ent.port))
                        logWriter.Err(fmt.Sprintf("Asicd Create rv: ", rv, " error : ", error))
                } else {
                        logWriter.Err("1. Asicd client is not connected.")
                }
            } else if msg.msg_type == 0 {
                ent := arp_cache.arpMap[msg.ip]
                ent.vlanid = msg.ent.vlanid
                ent.valid = msg.ent.valid
                ent.counter = msg.ent.counter
                ent.port    = msg.ent.port
                ent.ifName  = msg.ent.ifName
                ent.ifType  = msg.ent.ifType
                ent.localIP = msg.ent.localIP
                arp_cache.arpMap[msg.ip] = ent
            } else if msg.msg_type == 2 {
                for ip, arp := range arp_cache.arpMap {
                    if arp.counter == -2 && arp.valid == true {
                        logger.Println("1. Deleting entry ", ip, " from Arp cache")
                        delete(arp_cache.arpMap, ip)
                        logger.Println("Deleting an entry in asic for ", ip)
                        rv, error := asicdClient.ClientHdl.DeleteIPv4Neighbor(ip,
                                             "00:00:00:00:00:00", 0, 0)
                        logWriter.Err(fmt.Sprintf("Asicd Del rv: ", rv, " error : ", error))
                    } else if (arp.counter == 0 || arp.counter == -1) && arp.valid == true {
                        ent := arp_cache.arpMap[ip]
                        cnt = arp.counter
                        cnt--
                        ent.counter = cnt
                        logger.Println("1. Decrementing counter for ", ip);
                        arp_cache.arpMap[ip] = ent
                        //Send arp request after entry expires
                        refresh_arp_entry(ip, ent.ifName, ent.localIP)
                    } else if ((arp.counter <= (timeout_counter)) &&
                               (arp.counter > (timeout_counter - retry_cnt))) &&
                              arp.valid == false {
                        ent := arp_cache.arpMap[ip]
                        cnt = arp.counter
                        cnt--
                        ent.counter = cnt
                        logger.Println("2. Decrementing counter for ", ip);
                        arp_cache.arpMap[ip] = ent
                        retry_arp_req(ip, ent.vlanid, ent.ifType, ent.localIP)
                    } else if (arp.counter == (timeout_counter - retry_cnt)) &&
                               arp.valid == false {
                        logger.Println("2. Deleting entry ", ip, " from Arp cache")
                        delete(arp_cache.arpMap, ip)
                    } else if arp.counter != 0 {
                        ent := arp_cache.arpMap[ip]
                        cnt = arp.counter
                        cnt--
                        ent.counter = cnt
                        logger.Println("3. Decrementing counter for ", ip);
                        arp_cache.arpMap[ip] = ent
                    } else {
                        logger.Println("2. Deleting entry ", ip, " from Arp cache")
                        delete(arp_cache.arpMap, ip)
                    }
                }
            } else if msg.msg_type == 3 {
                logger.Println("Received ARP response from neighbor...", msg.ip)
                ent := arp_cache.arpMap[msg.ip]
                ent.macAddr = msg.ent.macAddr
                ent.vlanid = msg.ent.vlanid
                ent.valid = msg.ent.valid
                ent.counter = msg.ent.counter
                ent.port    = msg.ent.port
                ent.ifName  = msg.ent.ifName
                ent.ifType  = msg.ent.ifType
                ent.localIP = msg.ent.localIP
                arp_cache.arpMap[msg.ip] = ent
                logger.Println("2. updateArpCache(): ", arp_cache.arpMap[msg.ip])
                //3) Update asicd.
                if asicdClient.IsConnected {
                        logger.Println("2. Creating an entry in asic for IP:", msg.ip, "MAC:",
                                        (msg.ent.macAddr).String(), "VLAN:",
                                        (int32)(arp_cache.arpMap[msg.ip].vlanid))
                        rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
                                             (msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid),
                                             (int32)(msg.ent.port))
                        logWriter.Err(fmt.Sprintf("Asicd Create rv: ", rv, " error : ", error))
                } else {
                        logWriter.Err("2. Asicd client is not connected.")
                }
            } else {
                logger.Println("Invalid Msg type.")
                continue
            }
        }
}

func refresh_arp_entry(ip string, ifName string, localIP string) {
        logWriter.Err(fmt.Sprintln("Refresh ARP entry ", ifName))
        handle, err = pcap.OpenLive(ifName, snapshot_len, promiscuous, timeout)
        if handle == nil {
            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", ifName, err))
            return
        }
        mac_addr, err := getMacAddrInterfaceName(ifName)
        if err != nil {
            logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", ifName))
            return
        }
        logger.Println("MAC addr of ", ifName, ": ", mac_addr)
        sendArpReq(ip, handle, mac_addr, localIP)
        return
}

func retry_arp_req(ip string, vlanid arpd.Int, ifType arpd.Int, localIP string) {
        logger.Println("Calling ResotolveArpIPv4...", ip, " ", int32(ifType), " ", int32(vlanid))
        var linux_device string
//        if portdClient.IsConnected {
//		linux_device, err := portdClient.ClientHdl.GetLinuxIfc(int32(ifType), int32(vlanid))
                for _, port_cfg := range portCfgList {
                    linux_device = port_cfg.Ifname
                    logger.Println("linux_device ", linux_device)
/*
                    if err != nil {
                            logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", vlan_id, "type : ", iftype))
                            return ARP_ERR_REQ_FAIL, err
                    }
*/
                    logWriter.Err(fmt.Sprintln("Server:Connecting to device ", linux_device))
                    handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout)
                    if handle == nil {
                            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", linux_device, err))
                            return
                    }
                    mac_addr, err := getMacAddrInterfaceName(port_cfg.Ifname)
                    if err != nil {
                        logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", port_cfg.Ifname))
                        continue
                    }
                    logger.Println("MAC addr of ", port_cfg.Ifname, ": ", mac_addr)
                    sendArpReq(ip, handle, mac_addr, localIP)
                }

//	} else {
//		logWriter.Err("portd client is not connected.")
//		logger.Println("Portd is not connected.")
//	}
}

func printArpEntries() {
	logger.Println("************")
	for ip, arp := range arp_cache.arpMap {
		logger.Println("IP:", ip, " VLAN:", arp.vlanid, " MAC:", arp.macAddr, "CNT:", arp.counter, "PORT:", arp.port, "IfName:", arp.ifName, "IfType:", arp.ifType, "LocalIP:", arp.localIP, "Valid:", arp.valid)
	}
	logger.Println("************")
}

func timeout_thread() {
    for {
        time.Sleep(timeout)
        logger.Println("===============Message from Timeout Thread==============")
        printArpEntries()
        logger.Println("========================================================")
        arp_cache_update_chl <- arpUpdateMsg {
                                    ip: "0",
                                    ent: arpEntry {
                                            macAddr: []byte{0, 0, 0, 0, 0, 0},
                                            vlanid: 0,
                                            valid: false,
                                            port: -1,
                                            ifName: "",
                                            ifType: -1,
                                            localIP: "",
                                            counter: timeout_counter,
                                         },
                                    msg_type: 2,
                                 }
    }
}
