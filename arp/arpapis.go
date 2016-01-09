package main

import (
        "os"
        "syscall"
	"arpd"
	"asicdServices"
	"bytes"
        "reflect"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
        "github.com/vishvananda/netlink"
	"github.com/google/gopacket/pcap"
        _ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log/syslog"
	"net"
	"portdServices"
	"strconv"
	"time"
        "os/signal"
        "database/sql"
        "utils/dbutils"
        "utils/ipcutils"
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
	macAddr     net.HardwareAddr
	vlanid      arpd.Int
        valid       bool
        counter     int
        port        int
        ifName      string
        ifType      arpd.Int
        localIP     string
        sliceIdx    int
        timestamp   time.Time
}

type arpCache struct {
	cacheTimeout time.Duration
	arpMap          map[string]arpEntry
	//dev_handle      *pcap.Handle
	//hostTO          time.Duration
	//routerTO        time.Duration
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

type arpMsgSlice struct {
        ipAddr      string
        macAddr     string
        vlan        int
        intf        string
        valid       bool
        timestamp   time.Time
}

type pcapHandle struct {
        pcap_handle     *pcap.Handle
        ifName          string
}

type portProperty struct {
    untagged_vlanid     arpd.Int
}

type arpEntryResponseMsg struct {
    arp_msg     arpMsgSlice
}

type arpEntryRequestMsg struct {
    idx         int
}

/*
 * connection params.
 */
var (
	snapshot_len            int32 = 65549  //packet capture length
	promiscuous             bool = false  //mode
	err                     error
	timeout_pcap            time.Duration = 5 * time.Second
        config_refresh_timeout  int = 600       // 600 Seconds
        min_refresh_timeout     int = 300       // 300 Seconds
        timer_granularity       int = 10        // 10 Seconds
	timeout                 time.Duration = time.Duration(timer_granularity) * time.Second
        timeout_counter         int = 60
        retry_cnt               int = 2         // Number of retries before entry in deleted
        min_cnt                 int = 1         // Counter value at which entry will be deleted
	handle                  *pcap.Handle    // handle for pcap connection
	logWriter               *syslog.Writer
	log_err                 error
        dbHdl                   *sql.DB
        UsrConfDbName           string = "/UsrConfDb.db"
        dump_arp_table          bool = false
        arp_entry_timer         *time.Timer
        arp_entry_duration      time.Duration = 10 * time.Minute
)
var arp_cache *arpCache
var asicdClient AsicdClient //Thrift client to connect to asicd
var portdClient PortdClient //portd services client

var pcap_handle_map map[int]pcapHandle
var port_property_map map[int]portProperty

var arpSlice []string

//var portCfgList []PortConfigJson

var arp_cache_update_chl chan arpUpdateMsg = make(chan arpUpdateMsg, 100)
var arp_entry_req_chl chan arpEntryRequestMsg = make(chan arpEntryRequestMsg, 100)
var arp_entry_res_chl chan arpEntryResponseMsg = make(chan arpEntryResponseMsg, 100)
var arp_entry_refresh_start_chl chan bool = make(chan bool, 100)
var arp_entry_refresh_done_chl chan bool = make(chan bool, 100)

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
			//logger.Printf("found asicd at port %d", client.Port)
			logWriter.Info(fmt.Sprintln("found asicd at port %d", client.Port))
			asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdClient.Transport, asicdClient.PtrProtocolFactory = ipcutils.CreateIPCHandles(asicdClient.Address)
			if asicdClient.Transport != nil && asicdClient.PtrProtocolFactory != nil {
				logWriter.Info("connecting to asicd")
				asicdClient.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdClient.Transport, asicdClient.PtrProtocolFactory)
				asicdClient.IsConnected = true
			}

		}
		if client.Name == "portd" {
			//logger.Printf("found portd at port %d", client.Port)
			logWriter.Info(fmt.Sprintln("found portd at port %d", client.Port))
			portdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			portdClient.Transport, portdClient.PtrProtocolFactory = ipcutils.CreateIPCHandles(portdClient.Address)
			if portdClient.Transport != nil && portdClient.PtrProtocolFactory != nil {
				logWriter.Info("connecting to asicd")
				portdClient.ClientHdl = portdServices.NewPortServiceClientFactory(portdClient.Transport, portdClient.PtrProtocolFactory)
				portdClient.IsConnected = true
			}

		}
	}
}

func sigHandler(sigChan <-chan os.Signal) {
    signal := <-sigChan
    switch signal {
    case syscall.SIGHUP:
        //Cache the existing ARP entries
        //logger.Println("Received SIGHUP signal")
        logWriter.Info("Received SIGHUP signal")
        printArpEntries()
        //logger.Println("Closing DB handler")
        logWriter.Info("Closing DB handler")
        if dbHdl != nil {
            dbHdl.Close()
        }
        os.Exit(0)
    default:
        //logger.Println("Unhandled signal : ", signal)
        logWriter.Info(fmt.Sprintln("Unhandled signal : ", signal))
    }
}

func initARPhandlerParams() {
	//init syslog
	logWriter, log_err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "ARPD_LOG")
	defer logWriter.Close()

	// Initialise arp cache.
	success := initArpCache()
        port_property_map = make(map[int]portProperty)
	if success != true {
		logWriter.Err("server: Failed to initialise ARP cache")
		//logger.Println("Failed to initialise ARP cache")
		return
	}

        // init DB
        err := intantiateDB()
        if err != nil {
            //logger.Println("DB intantiate failure: ", err)
            logWriter.Err(fmt.Sprintln("DB intantiate failure: ", err))
        } else {
            //logger.Println("ArpCache DB has been Initiated")
            logWriter.Info(fmt.Sprintln("ArpCache DB has been Initiated"))
            updateARPCacheFromDB()
            refreshARPDB()
        }
	//connect to asicd and portd
	configFile := params_dir + "/clients.json"
	ConnectToClients(configFile)
        go updateArpCache()
        go timeout_thread()
        //List of signals to handle
        sigChan := make(chan os.Signal, 1)
        signalList := []os.Signal{syscall.SIGHUP}
        signal.Notify(sigChan, signalList...)
        go sigHandler(sigChan)
        go arp_entry_refresh()
	initPortParams()
        /* Open Response thread */
        processResponse()

}

func arp_entry_refresh() {
    var arp_entry_ref_func func()
    arp_entry_ref_func = func() {
        arp_entry_refresh_start_chl<-true
        arp_entry_refresh_done_msg := <-arp_entry_refresh_done_chl
        if arp_entry_refresh_done_msg == true {
            logWriter.Info("ARP Entry refresh done")
        } else {
            logWriter.Err("ARP Entry refresh not done")
            //logger.Println("ARP Entry refresh not done")
        }

        arp_entry_timer.Reset(arp_entry_duration)
    }

    arp_entry_timer = time.AfterFunc(arp_entry_duration, arp_entry_ref_func)
}

/*
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
		local_handle, err := pcap.OpenLive(v.Ifname, snapshot_len, promiscuous, timeout_pcap)
		if local_handle == nil {
			logWriter.Err(fmt.Sprintln("Server: No device found.: ", v.Ifname, err))
		}
                ent := pcap_handle_map[v.Port]
                ent.pcap_handle = local_handle
                ent.ifName = v.Ifname
                pcap_handle_map[v.Port] = ent
	}
}
*/
func BuildAsicToLinuxMap() {
        pcap_handle_map = make(map[int]pcapHandle)
        var filter string = "not ether proto 0x8809"
        var ifName string
	for i := 1; i < 73; i++ {
                ifName = fmt.Sprintf("fpPort-%d", i)
                //logger.Println("BuildAsicToLinuxMap : iface = ", ifName)
                //logger.Println("BuildAsicToLinuxMap : port = ", i)
		local_handle, err := pcap.OpenLive(ifName, snapshot_len, promiscuous, timeout_pcap)
		if local_handle == nil {
			logWriter.Err(fmt.Sprintln("Server: No device found.: ", ifName, err))
		} else {
                    err = local_handle.SetBPFFilter(filter)
                    if err != nil {
                        logWriter.Err(fmt.Sprintln("Unable to set filter on", ifName, err))
                    }
                }
                ent := pcap_handle_map[i]
                ent.pcap_handle = local_handle
                ent.ifName = ifName
                pcap_handle_map[i] = ent
	}
}
func initPortParams() {
	//configFile := params_dir + "/clients.json"
	//ConnectToClients(configFile)
/*
	portCfgFile := params_dir + "/portd.json"
	BuildAsicToLinuxMap(portCfgFile)
*/
	BuildAsicToLinuxMap()
}

func processPacket(targetIp string, iftype arpd.Int, vlanid arpd.Int, handle *pcap.Handle, mac_addr string, localIp string) {
        //logger.Println("processPacket() : Arp request for ", targetIp, "from", localIp)
        logWriter.Info(fmt.Sprintln("processPacket() : Arp request for ", targetIp, "from", localIp))
/*
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
*/
	//_, exist := arp_cache.arpMap[targetIp]
	//if !exist {
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
/*
	} else {
            // get MAC from cache.
            logger.Println("ARP entry already existed")
            printArpEntries()
            return
        }
*/

	// get MAC from cache.
        //logger.Println("ARP entry got created")
	//printArpEntries()
	return
}

func processResponse() {
        for port_id, p_hdl := range pcap_handle_map {
                //logger.Println("ifName = ", p_hdl.ifName, " Port = ", port_id)
                if p_hdl.pcap_handle == nil {
                    //logger.Println("pcap handle is nil");
                    logWriter.Err("pcap handle is nil")
                    continue
                }
                mac_addr, err := getMacAddrInterfaceName(p_hdl.ifName)
                if err != nil {
                    logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", p_hdl.ifName))
                    continue
                }
                //logger.Println("MAC addr of ", p_hdl.ifName, ": ", mac_addr)
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
        //logger.Println("sendArpReq(): sending arp requeust for targetIp ", targetIp,
        logWriter.Info(fmt.Sprintln("sendArpReq(): sending arp requeust for targetIp ", targetIp,
                        "local IP ", localIp))

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
    var src_Mac net.HardwareAddr

    src := gopacket.NewPacketSource(rec_handle, layers.LayerTypeEthernet)
    in := src.Packets()
    for {
        packet, ok := <-in
        if ok {
            //logger.Println("Receive some packet on arp response thread")

            //vlan_layer := packet.Layer(layers.LayerTypeEthernet)
            //vlan_tag := vlan_layer.(*layers.Ethernet)
            //vlan_id := vlan_layer.LayerContents()
            //logWriter.Err(vlan_tag.)
            arpLayer := packet.Layer(layers.LayerTypeARP)
            if arpLayer != nil {
                arp := arpLayer.(*layers.ARP)
                if arp == nil {
                    continue
                }
                if bytes.Equal([]byte(myMac), arp.SourceHwAddress) {
                    continue
                }

                if arp.Operation == layers.ARPReply {
                    src_Mac = net.HardwareAddr(arp.SourceHwAddress)
                    src_ip_addr := (net.IP(arp.SourceProtAddress)).String()
                    dest_Mac := net.HardwareAddr(arp.DstHwAddress)
                    dest_ip_addr := (net.IP(arp.DstProtAddress)).String()
                    logger.Println("Received Arp response SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr, "DST_MAC:", dest_Mac)
                    logWriter.Info(fmt.Sprintln("Received Arp response SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr, "DST_MAC:", dest_Mac))
                    ent, exist := arp_cache.arpMap[src_ip_addr]
                    if exist {
                        if ent.port == -2 {
                            port_map_ent, exists := port_property_map[port_id]
                            var vlan_id arpd.Int
                            if exists {
                                vlan_id = port_map_ent.untagged_vlanid
                            } else {
                                // vlan_id = 1
                                continue
                            }
                            arp_cache_update_chl <- arpUpdateMsg {
                                                        ip: src_ip_addr,
                                                        ent: arpEntry {
                                                            macAddr: src_Mac,
                                                            vlanid: vlan_id,
                                                            valid: true,
                                                            port: port_id,
                                                            ifName: if_Name,
                                                            ifType: ent.ifType,
                                                            localIP: ent.localIP,
                                                            counter: timeout_counter,
                                                        },
                                                        msg_type: 6,
                                                    }
                        } else {
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
                        }
                    } else {
                        port_map_ent, exists := port_property_map[port_id]
                        var vlan_id arpd.Int
                        if exists {
                            vlan_id = port_map_ent.untagged_vlanid
                        } else {
                            // vlan_id = 1
                            continue
                        }
                        arp_cache_update_chl <- arpUpdateMsg {
                                                    ip: src_ip_addr,
                                                    ent: arpEntry {
                                                        macAddr: src_Mac,
                                                        vlanid: vlan_id, // Need to be re-visited
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
                } else if arp.Operation == layers.ARPRequest {
                    src_Mac = net.HardwareAddr(arp.SourceHwAddress)
                    src_ip_addr := (net.IP(arp.SourceProtAddress)).String()
                    dest_ip_addr := (net.IP(arp.DstProtAddress)).String()
                    dstip := net.ParseIP(dest_ip_addr)
                    //logger.Println("Received Arp request SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr)
                    logWriter.Info(fmt.Sprintln("Received Arp Request SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr))
                    _, exist := arp_cache.arpMap[src_ip_addr]
                    if !exist {

                        route, err := netlink.RouteGet(dstip)
                        var ifName string
                        for _, rt := range route {
                            if rt.LinkIndex > 0 {
                                ifName, err = getInterfaceNameByIndex(rt.LinkIndex)
                                if err != nil || ifName == "" {
                                    //logger.Println("Unable to get the outgoing interface", err)
                                    logWriter.Err(fmt.Sprintf("Unable to get the outgoing interface", err))
                                    continue
                                }
                            }
                        }
                        //logger.Println("Outgoing interface:", ifName)
                        if ifName != "lo" {
                            continue
                        }
                        port_map_ent, exists := port_property_map[port_id]
                        var vlan_id arpd.Int
                        if exists {
                            vlan_id = port_map_ent.untagged_vlanid
                        } else {
                            // vlan_id = 1
                            continue
                        }
                        arp_cache_update_chl <- arpUpdateMsg {
                                                    ip: src_ip_addr,
                                                    ent: arpEntry {
                                                        macAddr: src_Mac,
                                                        vlanid: vlan_id, // Need to be re-visited
                                                        valid: true,
                                                        port: port_id,
                                                        ifName: if_Name,
                                                        ifType: 1,
                                                        localIP: dest_ip_addr,
                                                        counter: timeout_counter,
                                                    },
                                                    msg_type: 4,
                                                }
                    }
                }
            } else {
                //logger.Println("Not an ARP Packet")
                if nw := packet.NetworkLayer(); nw != nil {
                    src_ip, dst_ip := nw.NetworkFlow().Endpoints()
                    dst_ip_addr := dst_ip.String()
                    dstip := net.ParseIP(dst_ip_addr)
                    src_ip_addr := src_ip.String()
                    /*
                    if src_ip_addr == localIP || dst_ip_addr == localIP {
                        continue
                    }
                    */
                    _, exist := arp_cache.arpMap[dst_ip_addr]
                    if !exist {
                        //dst_ip_addr := src_ip.String()
                        route, err := netlink.RouteGet(dstip)
                        var ifName string
                        for _, rt := range route {
                            if rt.LinkIndex > 1 {
                                ifName, err = getInterfaceNameByIndex(rt.LinkIndex)
                                if err != nil || ifName == "" {
                                    logWriter.Err(fmt.Sprintf("Unable to get the outgoing interface", err))
                                    continue
                                }
                            }
                        }
                        if ifName == "" {
                            continue
                        }
                        //logger.Println("Receive Some packet from src_ip:", src_ip_addr, "dst_ip:", dst_ip_addr, "Outgoing Interface:", ifName)
                        logWriter.Info(fmt.Sprintln("Receive Some packet from src_ip:", src_ip_addr, "dst_ip:", dst_ip_addr, "Outgoing Interface:", ifName))
                        go createAndSendArpReuqest(dst_ip_addr, ifName)
                    }
                }
            }
        }
    }
}

func createAndSendArpReuqest(targetIP string, outgoingIfName string) {
    localIp, err := getIPv4ForInterfaceName(outgoingIfName)
    if err != nil || localIp == "" {
        logWriter.Err(fmt.Sprintf("Unable to get the ip address of ", outgoingIfName))
        return
    }
    handle, err = pcap.OpenLive(outgoingIfName, snapshot_len, promiscuous, timeout_pcap)
    if handle == nil {
            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", outgoingIfName, err))
            return
    }

    mac_addr, err := getMacAddrInterfaceName(outgoingIfName)
    if err != nil {
        logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", outgoingIfName))
    }
    //logger.Println("MAC addr of ", outgoingIfName, ": ", mac_addr)
    logWriter.Info(fmt.Sprintln("MAC addr of ", outgoingIfName, ": ", mac_addr))
    sendArpReq(targetIP, handle, mac_addr, localIp)
    arp_cache_update_chl <- arpUpdateMsg {
                                ip: targetIP,
                                ent: arpEntry {
                                        macAddr: []byte{0,0,0,0,0,0},
                                        vlanid: -1,
                                        valid: false,
                                        port: -2,
                                        ifName: outgoingIfName,
                                        ifType: 1,
                                        localIP: localIp,
                                        counter: timeout_counter,
                                     },
                                msg_type: 0,
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
    var dbCmd string
    var arpIPtoSliceIdxMap map[string]int = make(map[string]int)

        for {
            select {
                case msg := <-arp_cache_update_chl:
                    if msg.msg_type == 1 {
                    //if msg.ent.vlanid == 0 {
                        ent, exist := arp_cache.arpMap[msg.ip]
                        if reflect.DeepEqual(ent.macAddr, msg.ent.macAddr) &&
                           ent.valid == msg.ent.valid && ent.port == msg.ent.port &&
                           ent.ifName == msg.ent.ifName && ent.vlanid == msg.ent.vlanid &&
                           ent.ifType == msg.ent.ifType && exist {
                           //logger.Println("Updating counter after retry after expiry")
                           logWriter.Info(fmt.Sprintln("Updating counter after retry after expiry"))
                           ent.counter = msg.ent.counter
                           ent.timestamp = time.Now()
                           arp_cache.arpMap[msg.ip] = ent
                           continue
                        }
                        sliceIdx, exist := arpIPtoSliceIdxMap[msg.ip]
                        if !exist {
                            ent.sliceIdx = len(arpSlice)
                            arpIPtoSliceIdxMap[msg.ip] = len(arpSlice)
                            arpSlice = append(arpSlice, msg.ip)
                        } else {
                            ent.sliceIdx =  sliceIdx
                        }
                        ent.macAddr = msg.ent.macAddr
                        ent.valid = msg.ent.valid
                        ent.vlanid = msg.ent.vlanid
                        // Every entry will be expired after 10 mins
                        ent.counter = msg.ent.counter
                        ent.timestamp = time.Now()
                        ent.port    = msg.ent.port
                        ent.ifName  = msg.ent.ifName
                        ent.ifType  = msg.ent.ifType
                        ent.localIP = msg.ent.localIP
                        arp_cache.arpMap[msg.ip] = ent
                        //logger.Println("1 updateArpCache(): ", arp_cache.arpMap[msg.ip])
                        logWriter.Info(fmt.Sprintln("1 updateArpCache(): ", arp_cache.arpMap[msg.ip]))
                        err := updateArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, (net.HardwareAddr(ent.macAddr).String()))
                        if err != nil {
                            logWriter.Err("Unable to cache ARP Table in DB")
                        }
                        //3) Update asicd.
                        if asicdClient.IsConnected {
        /*
                                logger.Println("1. Deleting an entry in asic for ", msg.ip)
                                rv, error := asicdClient.ClientHdl.DeleteIPv4Neighbor(msg.ip,
                                                     "00:00:00:00:00:00", 0, 0)
                                logWriter.Err(fmt.Sprintf("Asicd Del rv: ", rv, " error : ", error))
        */
                                //logger.Println("1. Updating an entry in asic for ", msg.ip)
                                logWriter.Info(fmt.Sprintln("1. Updating an entry in asic for ", msg.ip))
                                rv, error := asicdClient.ClientHdl.UpdateIPv4Neighbor(msg.ip,
                                                     (msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), (int32)(msg.ent.port))
                                if rv < 0 {
                                    ent, _ = arp_cache.arpMap[msg.ip]
                                    ent.valid = false
                                    arp_cache.arpMap[msg.ip] = ent
                                }
                                logWriter.Err(fmt.Sprintf("Asicd Update rv: ", rv, " error : ", error))
                        } else {
                                logWriter.Err("1. Asicd client is not connected.")
                        }
                    } else if msg.msg_type == 0 {
                        ent, exist := arp_cache.arpMap[msg.ip]
                        sliceIdx, exist := arpIPtoSliceIdxMap[msg.ip]
                        if !exist {
                            ent.sliceIdx = len(arpSlice)
                            arpIPtoSliceIdxMap[msg.ip] = len(arpSlice)
                            arpSlice = append(arpSlice, msg.ip)
                        } else {
                            ent.sliceIdx =  sliceIdx
                        }
                        ent.vlanid = msg.ent.vlanid
                        ent.valid = msg.ent.valid
                        ent.counter = msg.ent.counter
                        ent.port    = msg.ent.port
                        ent.ifName  = msg.ent.ifName
                        ent.ifType  = msg.ent.ifType
                        ent.localIP = msg.ent.localIP
                        arp_cache.arpMap[msg.ip] = ent
                        err := storeArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, "incomplete")
                        if err != nil {
                            logWriter.Err("Unable to cache ARP Table in DB")
                        }
                    } else if msg.msg_type == 2 {
                        for ip, arp := range arp_cache.arpMap {
                            if arp.counter == min_cnt && arp.valid == true {
                                dbCmd = fmt.Sprintf(`DELETE FROM ARPCache WHERE key='%s' ;`, ip)
                                //logger.Println(dbCmd)
                                logWriter.Info(dbCmd)
                                if dbHdl != nil {
                                    //logger.Println("Executing DB Command:", dbCmd)
                                    logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
                                    _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
                                    if err != nil {
                                        logWriter.Err(fmt.Sprintln("Failed to Delete ARP entries from DB for %s %s", ip, err))
                                    }
                                } else {
                                    //logger.Println("DB handler is nil");
                                    logWriter.Err("DB handler is nil");
                                }
                                //logger.Println("1. Deleting entry ", ip, " from Arp cache")
                                logWriter.Info(fmt.Sprintln("1. Deleting entry ", ip, " from Arp cache"))
                                delete(arp_cache.arpMap, ip)
                                //logger.Println("Deleting an entry in asic for ", ip)
                                logWriter.Info(fmt.Sprintln("Deleting an entry in asic for ", ip))
                                rv, error := asicdClient.ClientHdl.DeleteIPv4Neighbor(ip,
                                                     "00:00:00:00:00:00", 0, 0)
                                logWriter.Err(fmt.Sprintf("Asicd Del rv: ", rv, " error : ", error))
                            } else if (arp.counter <= (min_cnt + retry_cnt + 1) &&
                                         arp.counter >= (min_cnt + 1)) &&
                                         arp.valid == true {
                                ent := arp_cache.arpMap[ip]
                                cnt = arp.counter
                                cnt--
                                ent.counter = cnt
                                //logger.Println("1. Decrementing counter for ", ip);
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
                                //logger.Println("2. Decrementing counter for ", ip);
                                arp_cache.arpMap[ip] = ent
                                retry_arp_req(ip, ent.vlanid, ent.ifType, ent.localIP)
                            } else if (arp.counter == (timeout_counter - retry_cnt)) &&
                                       arp.valid == false {
                                //logger.Println("2. Deleting entry ", ip, " from Arp cache")
                                logWriter.Info(fmt.Sprintln("2. Deleting entry ", ip, " from Arp cache"))
                                delete(arp_cache.arpMap, ip)
                                dbCmd = fmt.Sprintf(`DELETE FROM ARPCache WHERE key='%s' ;`, ip)
                                //logger.Println(dbCmd)
                                logWriter.Info(dbCmd)
                                if dbHdl != nil {
                                    //logger.Println("Executing DB Command:", dbCmd)
                                    logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
                                    _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
                                    if err != nil {
                                        logWriter.Err(fmt.Sprintln("Failed to Delete ARP entries from DB for %s %s", ip, err))
                                    }
                                } else {
                                    //logger.Println("DB handler is nil");
                                    logWriter.Err("DB handler is nil");
                                }
                            } else if (arp.counter > (min_cnt + retry_cnt + 1)) {
                                ent := arp_cache.arpMap[ip]
                                cnt = arp.counter
                                cnt--
                                ent.counter = cnt
                                //logger.Println("3. Decrementing counter for ", ip);
                                arp_cache.arpMap[ip] = ent
                            } else {
                                dbCmd = fmt.Sprintf(`DELETE FROM ARPCache WHERE key='%s' ;`, ip)
                                //logger.Println(dbCmd)
                                logWriter.Info(dbCmd)
                                if dbHdl != nil {
                                    //logger.Println("Executing DB Command:", dbCmd)
                                    logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
                                    _, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
                                    if err != nil {
                                        logWriter.Err(fmt.Sprintln("Failed to Delete ARP entries from DB for %s %s", ip, err))
                                    }
                                } else {
                                    //logger.Println("DB handler is nil");
                                    logWriter.Err("DB handler is nil");
                                }
                                //logger.Println("3. Deleting entry ", ip, " from Arp cache")
                                logWriter.Info(fmt.Sprintln("3. Deleting entry ", ip, " from Arp cache"))
                                delete(arp_cache.arpMap, ip)
                            }
                        }
                    } else if msg.msg_type == 3 {
                        logger.Println("Received ARP response from neighbor...", msg.ip)
                        //logWriter.Info(fmt.Sprintln("Received ARP response from neighbor...", msg.ip))
                        ent, exist := arp_cache.arpMap[msg.ip]
                        sliceIdx, exist := arpIPtoSliceIdxMap[msg.ip]
                        if !exist {
                            ent.sliceIdx = len(arpSlice)
                            arpIPtoSliceIdxMap[msg.ip] = len(arpSlice)
                            arpSlice = append(arpSlice, msg.ip)
                        } else {
                            ent.sliceIdx =  sliceIdx
                        }
                        ent.macAddr = msg.ent.macAddr
                        ent.vlanid = msg.ent.vlanid
                        ent.valid = msg.ent.valid
                        ent.counter = msg.ent.counter
                        ent.port    = msg.ent.port
                        ent.ifName  = msg.ent.ifName
                        ent.ifType  = msg.ent.ifType
                        ent.localIP = msg.ent.localIP
                        ent.timestamp = time.Now()
                        arp_cache.arpMap[msg.ip] = ent
                        //logger.Println("2. updateArpCache(): ", arp_cache.arpMap[msg.ip])
                        logWriter.Info(fmt.Sprintln("2. updateArpCache(): ", arp_cache.arpMap[msg.ip]))
                        err := storeArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, (net.HardwareAddr(ent.macAddr).String()))
                        if err != nil {
                            logWriter.Err("Unable to cache ARP Table in DB")
                        }
                        //3) Update asicd.
                        if asicdClient.IsConnected {
                                //logger.Println("2. Creating an entry in asic for IP:", msg.ip, "MAC:",
                                logWriter.Info(fmt.Sprintln("2. Creating an entry in asic for IP:", msg.ip, "MAC:",
                                                (msg.ent.macAddr).String(), "VLAN:",
                                                (int32)(arp_cache.arpMap[msg.ip].vlanid)))
                                rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
                                                     (msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid),
                                                     (int32)(msg.ent.port))
                                if rv < 0 {
                                    ent, _ = arp_cache.arpMap[msg.ip]
                                    ent.valid = false
                                    arp_cache.arpMap[msg.ip] = ent
                                }
                                logWriter.Err(fmt.Sprintf("Asicd Create rv: ", rv, " error : ", error))
                        } else {
                                logWriter.Err("2. Asicd client is not connected.")
                        }
                    } else if msg.msg_type == 4 {
                        logger.Println("Received ARP Request from neighbor...", msg.ip)
                        //logWriter.Info(fmt.Sprintln("Received ARP Request from neighbor...", msg.ip))
                        ent, exist := arp_cache.arpMap[msg.ip]
                        sliceIdx, exist := arpIPtoSliceIdxMap[msg.ip]
                        if !exist {
                            ent.sliceIdx = len(arpSlice)
                            arpIPtoSliceIdxMap[msg.ip] = len(arpSlice)
                            arpSlice = append(arpSlice, msg.ip)
                        } else {
                            ent.sliceIdx =  sliceIdx
                        }
                        ent.macAddr = msg.ent.macAddr
                        ent.vlanid = msg.ent.vlanid
                        ent.valid = msg.ent.valid
                        ent.counter = msg.ent.counter
                        ent.port    = msg.ent.port
                        ent.ifName  = msg.ent.ifName
                        ent.ifType  = msg.ent.ifType
                        ent.localIP = msg.ent.localIP
                        ent.timestamp = time.Now()
                        arp_cache.arpMap[msg.ip] = ent
                        //logger.Println("3. updateArpCache(): ", arp_cache.arpMap[msg.ip])
                        logWriter.Info(fmt.Sprintln("3. updateArpCache(): ", arp_cache.arpMap[msg.ip]))
                        err := storeArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, (net.HardwareAddr(ent.macAddr).String()))
                        if err != nil {
                            logWriter.Err("Unable to cache ARP Table in DB")
                        }
                        //3) Update asicd.
                        if asicdClient.IsConnected {
                                //logger.Println("3. Creating an entry in asic for IP:", msg.ip, "MAC:",
                                  //              (msg.ent.macAddr).String(), "VLAN:",
                                    //            (int32)(arp_cache.arpMap[msg.ip].vlanid))
                                logWriter.Info(fmt.Sprintln("3. Creating an entry in asic for IP:", msg.ip, "MAC:",
                                                (msg.ent.macAddr).String(), "VLAN:",
                                                (int32)(arp_cache.arpMap[msg.ip].vlanid)))
                                rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
                                                     (msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid),
                                                     (int32)(msg.ent.port))
                                if rv < 0 {
                                    ent, _ = arp_cache.arpMap[msg.ip]
                                    ent.valid = false
                                    arp_cache.arpMap[msg.ip] = ent
                                }
                                logWriter.Err(fmt.Sprintf("Asicd Create rv: ", rv, " error : ", error))
                        } else {
                                logWriter.Err("2. Asicd client is not connected.")
                        }
                    } else if msg.msg_type == 5 { //Update Refresh Timer
                        for ip, arp := range arp_cache.arpMap {
                            ent := arp_cache.arpMap[ip]
                            cnt = arp.counter
                            ent.counter = timeout_counter
                            arp_cache.arpMap[ip] = ent
                        }
                    } else if msg.msg_type == 6 {
                        ent, exist := arp_cache.arpMap[msg.ip]
                        sliceIdx, exist := arpIPtoSliceIdxMap[msg.ip]
                        if !exist {
                            ent.sliceIdx = len(arpSlice)
                            arpIPtoSliceIdxMap[msg.ip] = len(arpSlice)
                            arpSlice = append(arpSlice, msg.ip)
                        } else {
                            ent.sliceIdx =  sliceIdx
                        }
                        ent.macAddr = msg.ent.macAddr
                        ent.valid = msg.ent.valid
                        ent.vlanid = msg.ent.vlanid
                        // Every entry will be expired after 10 mins
                        ent.counter = msg.ent.counter
                        ent.timestamp = time.Now()
                        ent.port    = msg.ent.port
                        ent.ifName  = msg.ent.ifName
                        ent.ifType  = msg.ent.ifType
                        ent.localIP = msg.ent.localIP
                        arp_cache.arpMap[msg.ip] = ent
                        //logger.Println("1 updateArpCache(): ", arp_cache.arpMap[msg.ip])
                        logWriter.Info(fmt.Sprintln("6 updateArpCache(): ", arp_cache.arpMap[msg.ip]))
                        err := updateArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, (net.HardwareAddr(ent.macAddr).String()))
                        if err != nil {
                            logWriter.Err("Unable to cache ARP Table in DB")
                        }
                        //3) Update asicd.
                        if asicdClient.IsConnected {
                                logger.Println("6. Creating an entry in asic for ", msg.ip)
                                logWriter.Info(fmt.Sprintln("6. Creating an entry in asic for ", msg.ip))
                                rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
                                                     (msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), (int32)(msg.ent.port))
                                if rv < 0 {
                                    ent, _ = arp_cache.arpMap[msg.ip]
                                    ent.valid = false
                                    arp_cache.arpMap[msg.ip] = ent
                                }
                                logWriter.Err(fmt.Sprintf("Asicd Update rv: ", rv, " error : ", error))
                        } else {
                                logWriter.Err("6. Asicd client is not connected.")
                        }
                    } else {
                        //logger.Println("Invalid Msg type.")
                        logWriter.Err("Invalid Msg type.")
                        continue
                    }
                case arp_req_msg := <-arp_entry_req_chl:
                    ip := arpSlice[arp_req_msg.idx]
                    var arp_entry_msg arpMsgSlice
                    ent, exist := arp_cache.arpMap[ip]
                    if !exist {
                        arp_entry_msg.ipAddr = ip
                        arp_entry_msg.macAddr = "invalid"
                        arp_entry_msg.vlan = -1
                        arp_entry_msg.intf = "invalid"
                        arp_entry_msg.valid = false
                    } else {
                        arp_entry_msg.ipAddr = ip
                        if (ent.macAddr).String() == "" {
                            arp_entry_msg.macAddr = "incomplete"
                        } else {
                            arp_entry_msg.macAddr = (ent.macAddr).String()
                        }
                        arp_entry_msg.vlan = int(ent.vlanid)
                        arp_entry_msg.intf = ent.ifName
                        arp_entry_msg.valid = ent.valid
                        arp_entry_msg.timestamp = ent.timestamp
                    }
                    arp_entry_res_chl<-arpEntryResponseMsg {
                                            arp_msg: arp_entry_msg,
                                        }
                case arp_entry_refresh_start_msg := <-arp_entry_refresh_start_chl:
                    if arp_entry_refresh_start_msg == true {
                        arpIPtoSliceIdxMap = make(map[string]int)
                        arpSlice = []string{}
                        for ip, arp := range arp_cache.arpMap {
                            logWriter.Info("Refreshing ARP entry")
                            arpIPtoSliceIdxMap[ip] = len(arpSlice)
                            arp.sliceIdx = len(arpSlice)
                            arp_cache.arpMap[ip] = arp
                            arpSlice = append(arpSlice, ip)
                        }
                        arp_entry_refresh_done_chl<-true
                    } else {
                        logWriter.Err("Invalid arp_entry_refresh_start_msg")
                        arp_entry_refresh_done_chl<-false
                    }
                //default:
            }
        }
}

func refresh_arp_entry(ip string, ifName string, localIP string) {
        logWriter.Err(fmt.Sprintln("Refresh ARP entry ", ifName))
        handle, err = pcap.OpenLive(ifName, snapshot_len, promiscuous, timeout_pcap)
        if handle == nil {
            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", ifName, err))
            return
        }
        mac_addr, err := getMacAddrInterfaceName(ifName)
        if err != nil {
            logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", ifName))
            return
        }
        //logger.Println("MAC addr of ", ifName, ": ", mac_addr)
        logWriter.Info(fmt.Sprintln("MAC addr of ", ifName, ": ", mac_addr))
        sendArpReq(ip, handle, mac_addr, localIP)
        return
}

func retry_arp_req(ip string, vlanid arpd.Int, ifType arpd.Int, localIP string) {
        //logger.Println("Calling ResolveArpIPv4...", ip, " ", int32(ifType), " ", int32(vlanid))
//        var linux_device string
        if portdClient.IsConnected {
		linux_device, err := portdClient.ClientHdl.GetLinuxIfc(int32(ifType), int32(vlanid))
/*
                for _, port_cfg := range portCfgList {
                    linux_device = port_cfg.Ifname
*/
                    //logger.Println("linux_device ", linux_device)
                    logWriter.Info(fmt.Sprintln("linux_device ", linux_device))
                    if err != nil {
                            logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", vlanid, "type : ", ifType))
                            return
                    }
                    logWriter.Err(fmt.Sprintln("Server:Connecting to device ", linux_device))
                    handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout_pcap)
                    if handle == nil {
                            logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", linux_device, err))
                            return
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

                    sendArpReq(ip, handle, mac_addr, localIP)
/*
                }
*/

	} else {
		logWriter.Err("portd client is not connected.")
		//logger.Println("Portd is not connected.")
	}
}

func printArpEntries() {
	//logger.Println("************")
	logWriter.Info("************")
	for ip, arp := range arp_cache.arpMap {
		//logger.Println("IP:", ip, " VLAN:", arp.vlanid, " MAC:", arp.macAddr, "CNT:", arp.counter, "PORT:", arp.port, "IfName:", arp.ifName, "IfType:", arp.ifType, "LocalIP:", arp.localIP, "Valid:", arp.valid)
		logWriter.Info(fmt.Sprintln("IP:", ip, " VLAN:", arp.vlanid, " MAC:", arp.macAddr, "CNT:", arp.counter, "PORT:", arp.port, "IfName:", arp.ifName, "IfType:", arp.ifType, "LocalIP:", arp.localIP, "Valid:", arp.valid, "SliceIdx:", arp.sliceIdx, "Timestamp:", arp.timestamp))
	}
	logWriter.Info("************")
	//logger.Println("************")
}

func timeout_thread() {
    for {
        time.Sleep(timeout)
        if dump_arp_table == true {
            //logger.Println("===============Message from ARP Timeout Thread==============")
            logWriter.Info("===============Message from ARP Timeout Thread==============")
            printArpEntries()
            //logger.Println("========================================================")
            logWriter.Info("========================================================")
            logger.Println(arpSlice)
        }
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

func updateCounterInArpCache() {
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
                                msg_type: 5,
                             }

}
