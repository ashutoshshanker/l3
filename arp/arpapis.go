package main

import (
	"arpd"
	"asicd/asicdConstDefs"
	"asicdServices"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	_ "github.com/mattn/go-sqlite3"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"log/syslog"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"utils/commonDefs"
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
	ClientHdl *asicdServices.ASICDServicesClient
}

type arpEntry struct {
	macAddr   net.HardwareAddr
	vlanid    arpd.Int
	valid     bool
	counter   int
	port      int
	ifName    string
	ifType    arpd.Int
	localIP   string
	sliceIdx  int
	timestamp time.Time
}

type arpCache struct {
	cacheTimeout time.Duration
	arpMap       map[string]arpEntry
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
	ip       string
	ent      arpEntry
	msg_type int
}

type arpMsgSlice struct {
	ipAddr    string
	macAddr   string
	vlan      int
	intf      string
	valid     bool
	timestamp time.Time
}

type pcapHandle struct {
	pcap_handle *pcap.Handle
	ifName      string
}

type portProperty struct {
	untagged_vlanid   int
	untagged_vlanName string
}

type portLagProperty struct {
	IfIndex int32
}

type vlanProperty struct {
	vlanName      string
	untaggedPorts []int32
}

type ipv4IntfProperty struct {
	ifIdx   int
	ifType  int
	ifState uint8
}

type ipv4Address struct {
	ipAddress net.IP
	ipNet     net.IPNet
}

type arpEntryResponseMsg struct {
	arp_msg arpMsgSlice
}

type arpEntryRequestMsg struct {
	idx int
}

type portConfig struct {
	Name string
}

type l3IntfDownMsg struct {
	ifType int
	ifIdx  int
}

/*
 * connection params.
 */
var (
	snapshot_len           int32 = 65549 //packet capture length
	promiscuous            bool  = false //mode
	err                    error
	timeout_pcap           time.Duration = 1 * time.Second
	config_refresh_timeout int           = 600 // 600 Seconds
	min_refresh_timeout    int           = 300 // 300 Seconds
	timer_granularity      int           = 1   // 1 Seconds
	timeout                time.Duration = time.Duration(timer_granularity) * time.Second
	timeout_counter        int           = 600 // The value of timeout_counter = (config_refresh_timeout/timer_granularity)
	retry_cnt              int           = 5   // Number of retries before entry in deleted
	min_cnt                int           = 1   // Counter value at which entry will be deleted
	one_minute_cnt         int           = (60 / timer_granularity)
	thirty_sec_cnt         int           = (30 / timer_granularity)
	handle                 *pcap.Handle  // handle for pcap connection
	logWriter              *syslog.Writer
	log_err                error
	dbHdl                  *sql.DB
	UsrConfDbName          string = "/UsrConfDb.db"
	dump_arp_table         bool   = false
	arp_entry_timer        *time.Timer
	arp_entry_duration     time.Duration = 10 * time.Minute
	probe_wait             int           = 5  // 5 Seconds
	probe_num              int           = 5  // Number of ARP Probe
	probe_max              int           = 20 // 20 Seconds
	probe_min              int           = 10 // 10 Seconds
)
var arp_cache *arpCache
var asicdClient AsicdClient //Thrift client to connect to asicd

var pcap_handle_map map[int]pcapHandle
var port_property_map map[int]portProperty
var vlanPropertyMap map[int]vlanProperty
var portConfigMap map[int]portConfig
var ipv4IntfPropertyMap map[string]ipv4IntfProperty
var portLagPropertyMap map[int32]portLagProperty

var asicdSubSocket *nanomsg.SubSocket

var arpSlice []string

//var portCfgList []PortConfigJson

var arp_cache_update_chl chan arpUpdateMsg = make(chan arpUpdateMsg, 100)
var arp_entry_req_chl chan arpEntryRequestMsg = make(chan arpEntryRequestMsg, 100)
var arp_entry_res_chl chan arpEntryResponseMsg = make(chan arpEntryResponseMsg, 100)
var arp_entry_refresh_start_chl chan bool = make(chan bool, 100)
var arp_entry_refresh_done_chl chan bool = make(chan bool, 100)
var arp_l3_down_chl chan l3IntfDownMsg = make(chan l3IntfDownMsg, 100)
var asicdSubSocketCh chan []byte = make(chan []byte)
var asicdSubSocketErrCh chan error = make(chan error)

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
			logWriter.Info(fmt.Sprintln("found asicd at port", client.Port))
			asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			asicdClient.Transport, asicdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdClient.Address)
			if asicdClient.Transport != nil && asicdClient.PtrProtocolFactory != nil {
				logWriter.Info("connecting to asicd")
				asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdClient.Transport, asicdClient.PtrProtocolFactory)
				asicdClient.IsConnected = true
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

func listenForASICUpdate(address string) error {
	var err error
	if asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		logger.Println(fmt.Sprintln("Failed to create ASIC subscribe socket, error:", err))
		return err
	}

	if err = asicdSubSocket.Subscribe(""); err != nil {
		logger.Println(fmt.Sprintln("Failed to subscribe to \"\" on ASIC subscribe socket, error:", err))
		return err
	}

	if _, err = asicdSubSocket.Connect(address); err != nil {
		logWriter.Err(fmt.Sprintln("Failed to connect to ASIC publisher socket, address:", address, "error:", err))
		return err
	}

	logger.Println(fmt.Sprintln("Connected to ASIC publisher at address:", address))
	if err = asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		logger.Println(fmt.Sprintln("Failed to set the buffer size for ASIC publisher socket, error:", err))
		return err
	}
	return nil

}

func asicdSubscriber() {
	for {
		logger.Println("Read on Asic subscriber socket...")
		rxBuf, err := asicdSubSocket.Recv(0)
		if err != nil {
			logWriter.Err(fmt.Sprintln("Recv on Asicd subscriber socket failed with error:", err))
			asicdSubSocketErrCh <- err
			continue
		}
		logWriter.Info(fmt.Sprintln("Asicd subscriber recv returned:", rxBuf))
		asicdSubSocketCh <- rxBuf
	}
}

func initARPhandlerParams() {
	//init syslog
	logWriter, log_err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "ARPD_LOG")
	defer logWriter.Close()

	// Initialise arp cache.
	success := initArpCache()
	port_property_map = make(map[int]portProperty)
	vlanPropertyMap = make(map[int]vlanProperty)
	portConfigMap = make(map[int]portConfig)
	ipv4IntfPropertyMap = make(map[string]ipv4IntfProperty)
	portLagPropertyMap = make(map[int32]portLagProperty)
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
	//connect to asicd
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
	logger.Println("Listen for ASICd for Vlan Delete and Create Messages")
	err = listenForASICUpdate(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go asicdSubscriber()
	}
	initPortParams()
	/* Open Response thread */
	processResponse()

}

func arp_entry_refresh() {
	var arp_entry_ref_func func()
	arp_entry_ref_func = func() {
		arp_entry_refresh_start_chl <- true
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

func BuildAsicToLinuxMap() {
	pcap_handle_map = make(map[int]pcapHandle, len(portConfigMap))
	var filter string = "not ether proto 0x8809"
	for ifNum, portConfig := range portConfigMap {
		ifName := portConfig.Name
		//logger.Println("ifNum: ", ifNum, "ifName:", ifName)
		local_handle, err := pcap.OpenLive(ifName, snapshot_len, promiscuous, timeout_pcap)
		if local_handle == nil {
			logWriter.Err(fmt.Sprintln("Server: No device found.: ", ifName, err))
		} else {
			err = local_handle.SetBPFFilter(filter)
			if err != nil {
				logWriter.Err(fmt.Sprintln("Unable to set filter on", ifName, err))
			}
		}
		ent := pcap_handle_map[ifNum]
		ent.pcap_handle = local_handle
		ent.ifName = ifName
		pcap_handle_map[ifNum] = ent
	}
}

func constructPortConfigMap() {
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	if asicdClient.IsConnected {
		logger.Println("Calling asicd for port config")
		count := 10
		for {
			bulkInfo, err := asicdClient.ClientHdl.GetBulkPortState(asicdServices.Int(currMarker), asicdServices.Int(count))
			if err != nil {
				logger.Println("Error: ", err)
				return
			}
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			currMarker = int64(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				portNum := int(bulkInfo.PortStateList[i].PortNum)
				ent := portConfigMap[portNum]
				ent.Name = bulkInfo.PortStateList[i].Name
				//logger.Println("Port Num:", portNum, "Name:", ent.Name)
				portConfigMap[portNum] = ent
			}
			if more == false {
				return
			}
		}
	}
}

func initPortParams() {
	constructPortConfigMap()
	//logger.Println("Port Config Map:", portConfigMap)
	BuildAsicToLinuxMap()
}

func processPacket(targetIp string, iftype arpd.Int, vlanid arpd.Int, handle *pcap.Handle, mac_addr string, localIp string) {
	//logger.Println("processPacket() : Arp request for ", targetIp, "from", localIp)
	logWriter.Info(fmt.Sprintln("processPacket() : Arp request for ", targetIp, "from", localIp))
	sendArpReq(targetIp, handle, mac_addr, localIp)
	arp_cache_update_chl <- arpUpdateMsg{
		ip: targetIp,
		ent: arpEntry{
			macAddr: []byte{0, 0, 0, 0, 0, 0},
			vlanid:  vlanid,
			valid:   false,
			port:    -1,
			ifName:  "",
			ifType:  iftype,
			localIP: localIp,
			counter: timeout_counter,
		},
		msg_type: 0,
	}
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

//ToDo: This function need to cleaned up
/*
 *@fn receiveArpResponse
 * Process ARP response from the interface for ARP
 * req sent for targetIp
 */
/*
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
					//logger.Println("Received Arp response SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr, "DST_MAC:", dest_Mac)
					logWriter.Info(fmt.Sprintln("Received Arp response SRC_IP:", src_ip_addr, "SRC_MAC: ", src_Mac, "DST_IP:", dest_ip_addr, "DST_MAC:", dest_Mac))
					if dest_ip_addr == "0.0.0.0" {
						//logger.Println("Recevied reply from ARP Probe and there is a conflicting IP Address", src_ip_addr)
						logWriter.Err(fmt.Sprintln("Recevied reply for ARP Probe and there is a conflicting IP Address", src_ip_addr))
						continue
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
								continue
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
							continue
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
				} else if arp.Operation == layers.ARPRequest {
					src_Mac = net.HardwareAddr(arp.SourceHwAddress)
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
							continue
						}
						if src_ip_addr == "0.0.0.0" { // ARP Probe Request
							local_ip_addr, _ := getIPv4ForInterface(arpd.Int(0), arpd.Int(vlan_id))
							if local_ip_addr == dest_ip_addr {
								// Send Arp Reply for ARP Probe
								logger.Println("Linux will Send Arp Reply for recevied ARP Probe because of conflicting address")
								continue
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
										continue
									}
								}
							}
							logger.Println("Outgoing interface:", ifName)
							if ifName != "lo" {
								continue
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
			} else {
				//logger.Println("Not an ARP Packet")
				if nw := packet.NetworkLayer(); nw != nil {
					src_ip, dst_ip := nw.NetworkFlow().Endpoints()
					dst_ip_addr := dst_ip.String()
					//dstip := net.ParseIP(dst_ip_addr)
					src_ip_addr := src_ip.String()
                                        port_map_ent, exists := port_property_map[port_id]
                                        var vlan_id arpd.Int
                                        var ifType arpd.Int
                                        if exists {
                                                vlan_id = arpd.Int(port_map_ent.untagged_vlanid)
                                                ifType = arpd.Int(commonDefs.L2RefTypeVlan)
                                        } else {
                                                // vlan_id = 1
                                                continue
                                        }
					_, exist := arp_cache.arpMap[dst_ip_addr]
					if !exist {
						ifName, ret := isInLocalSubnet(dst_ip_addr)
						if ret == false {
							continue
						}
						logWriter.Info(fmt.Sprintln("Sending ARP for dst_ip:", dst_ip_addr, "Outgoing Interface:", ifName))
						go createAndSendArpReuqest(dst_ip_addr, ifName, vlan_id, ifType)
					}
					_, exist = arp_cache.arpMap[src_ip_addr]
					if !exist {
						ifName, ret := isInLocalSubnet(src_ip_addr)
						if ret == false {
							continue
						}
						logWriter.Info(fmt.Sprintln("Sending ARP for src_ip:", src_ip_addr, "Outgoing Interface:", ifName))
						go createAndSendArpReuqest(src_ip_addr, ifName, vlan_id, ifType)
					}
				}
			}
		}
	}
}
*/

func isInLocalSubnet(ipaddr string) (ifName string, vlanId int, ifType int, ret bool) {
	var flag bool = false
	var ipv4IntfProp ipv4IntfProperty
	var ipAddr string
	ipIn := net.ParseIP(ipaddr)

	for ipAddr, ipv4IntfProp = range ipv4IntfPropertyMap {
		ip, ipNet, err := net.ParseCIDR(ipAddr)
		if err != nil {
			continue
		}
		if ip.Equal(ipIn) {
			// IP Address of local interface
			return "", 0, 0, false
		}
		net1 := ipIn.Mask(ipNet.Mask)
		net2 := ip.Mask(ipNet.Mask)
		if net1.Equal(net2) {
			flag = true
			break
		}
	}

	if flag == false {
		return "", 0, 0, false
	}

	if ipv4IntfProp.ifType == commonDefs.L2RefTypeVlan { // VLAN
		ent, exist := vlanPropertyMap[ipv4IntfProp.ifIdx]
		if !exist {
			return "", 0, 0, false
		}
		ifName = ent.vlanName
		vlanId = ipv4IntfProp.ifIdx
		ifType = commonDefs.L2RefTypeVlan
	} else if ipv4IntfProp.ifType == commonDefs.L2RefTypePort { //PHY
		ent, exist := portConfigMap[ipv4IntfProp.ifIdx]
		if !exist {
			return "", 0, 0, false
		}
		ifName = ent.Name
		vlanId = ipv4IntfProp.ifIdx
		ifType = commonDefs.L2RefTypePort
	} else {
		return "", 0, 0, false
	}
	return ifName, vlanId, ifType, true
}

func createAndSendArpReuqest(targetIP string, outgoingIfName string, vlan_id arpd.Int, ifType arpd.Int) {
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
	arp_cache_update_chl <- arpUpdateMsg{
		ip: targetIP,
		ent: arpEntry{
			macAddr: []byte{0, 0, 0, 0, 0, 0},
			vlanid:  vlan_id,
			valid:   false,
			port:    -2,
			ifName:  outgoingIfName,
			ifType:  ifType,
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

//ToDo: This function need to cleaned up
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
				ent, exist := arp_cache.arpMap[msg.ip]
				if ent.macAddr.String() == msg.ent.macAddr.String() &&
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
					ent.sliceIdx = sliceIdx
				}
				ent.macAddr = msg.ent.macAddr
				ent.valid = msg.ent.valid
				ent.vlanid = msg.ent.vlanid
				// Every entry will be expired after 10 mins
				ent.counter = msg.ent.counter
				ent.timestamp = time.Now()
				ent.port = msg.ent.port
				ent.ifName = msg.ent.ifName
				ent.ifType = msg.ent.ifType
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
					//logger.Println("1. Updating an entry in asic for ", msg.ip)
					logWriter.Info(fmt.Sprintln("1. Updating an entry in asic for ", msg.ip))
					ifIndex := getIfIndex(msg.ent.port)
					rv, error := asicdClient.ClientHdl.UpdateIPv4Neighbor(msg.ip,
						(msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), ifIndex)
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
					ent.sliceIdx = sliceIdx
				}
				ent.vlanid = msg.ent.vlanid
				ent.valid = msg.ent.valid
				ent.counter = msg.ent.counter
				ent.port = msg.ent.port
				ent.ifName = msg.ent.ifName
				ent.ifType = msg.ent.ifType
				ent.localIP = msg.ent.localIP
				arp_cache.arpMap[msg.ip] = ent
				if !exist {
					err := storeArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, "incomplete")
					if err != nil {
						logWriter.Err("Unable to cache ARP Table in DB")
					}
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
							logWriter.Err("DB handler is nil")
						}
						//logger.Println("1. Deleting entry ", ip, " from Arp cache")
						logWriter.Info(fmt.Sprintln("1. Deleting entry ", ip, " from Arp cache"))
						delete(arp_cache.arpMap, ip)
						//logger.Println("Deleting an entry in asic for ", ip)
						logWriter.Info(fmt.Sprintln("Deleting an entry in asic for ", ip))
						rv, error := asicdClient.ClientHdl.DeleteIPv4Neighbor(ip,
							"00:00:00:00:00:00", 0, 0)
						logWriter.Err(fmt.Sprintf("Asicd Del rv: ", rv, " error : ", error))
					} else if ((arp.counter <= (min_cnt+retry_cnt+1) &&
						arp.counter >= (min_cnt+1)) ||
						arp.counter == (timeout_counter/2) ||
						arp.counter == (timeout_counter/4) ||
						arp.counter == one_minute_cnt ||
						arp.counter == thirty_sec_cnt) &&
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
							logWriter.Err("DB handler is nil")
						}
					} else if arp.counter > (min_cnt + retry_cnt + 1) {
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
							logWriter.Err("DB handler is nil")
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
					ent.sliceIdx = sliceIdx
				}
				ent.macAddr = msg.ent.macAddr
				ent.vlanid = msg.ent.vlanid
				ent.valid = msg.ent.valid
				ent.counter = msg.ent.counter
				ent.port = msg.ent.port
				ent.ifName = msg.ent.ifName
				ent.ifType = msg.ent.ifType
				ent.localIP = msg.ent.localIP
				ent.timestamp = time.Now()
				arp_cache.arpMap[msg.ip] = ent
				//logger.Println("2. updateArpCache(): ", arp_cache.arpMap[msg.ip])
				logWriter.Info(fmt.Sprintln("2. updateArpCache(): ", arp_cache.arpMap[msg.ip]))
				if !exist {
					err := storeArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, (net.HardwareAddr(ent.macAddr).String()))
					if err != nil {
						logWriter.Err("Unable to cache ARP Table in DB")
					}
				}
				//3) Update asicd.
				if asicdClient.IsConnected {
					//logger.Println("2. Creating an entry in asic for IP:", msg.ip, "MAC:",
					logWriter.Info(fmt.Sprintln("2. Creating an entry in asic for IP:", msg.ip, "MAC:",
						(msg.ent.macAddr).String(), "VLAN:",
						(int32)(arp_cache.arpMap[msg.ip].vlanid)))
					ifIndex := getIfIndex(msg.ent.port)
					rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
						(msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), ifIndex)
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
					ent.sliceIdx = sliceIdx
				}
				ent.macAddr = msg.ent.macAddr
				ent.vlanid = msg.ent.vlanid
				ent.valid = msg.ent.valid
				ent.counter = msg.ent.counter
				ent.port = msg.ent.port
				ent.ifName = msg.ent.ifName
				ent.ifType = msg.ent.ifType
				ent.localIP = msg.ent.localIP
				ent.timestamp = time.Now()
				arp_cache.arpMap[msg.ip] = ent
				//logger.Println("3. updateArpCache(): ", arp_cache.arpMap[msg.ip])
				logWriter.Info(fmt.Sprintln("3. updateArpCache(): ", arp_cache.arpMap[msg.ip]))
				if !exist {
					err := storeArpTableInDB(int(ent.ifType), int(ent.vlanid), ent.ifName, int(ent.port), msg.ip, ent.localIP, (net.HardwareAddr(ent.macAddr).String()))
					if err != nil {
						logWriter.Err("Unable to cache ARP Table in DB")
					}
				}
				//3) Update asicd.
				if asicdClient.IsConnected {
					//logger.Println("3. Creating an entry in asic for IP:", msg.ip, "MAC:",
					//              (msg.ent.macAddr).String(), "VLAN:",
					//            (int32)(arp_cache.arpMap[msg.ip].vlanid))
					logWriter.Info(fmt.Sprintln("3. Creating an entry in asic for IP:", msg.ip, "MAC:",
						(msg.ent.macAddr).String(), "VLAN:",
						(int32)(arp_cache.arpMap[msg.ip].vlanid)))
					ifIndex := getIfIndex(msg.ent.port)
					rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
						(msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), ifIndex)
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
					ent.sliceIdx = sliceIdx
				}
				ent.macAddr = msg.ent.macAddr
				ent.valid = msg.ent.valid
				ent.vlanid = msg.ent.vlanid
				// Every entry will be expired after 10 mins
				ent.counter = msg.ent.counter
				ent.timestamp = time.Now()
				ent.port = msg.ent.port
				ent.ifName = msg.ent.ifName
				ent.ifType = msg.ent.ifType
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
					ifIndex := getIfIndex(msg.ent.port)
					rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(msg.ip,
						(msg.ent.macAddr).String(), (int32)(arp_cache.arpMap[msg.ip].vlanid), ifIndex)
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
			arp_entry_res_chl <- arpEntryResponseMsg{
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
				arp_entry_refresh_done_chl <- true
			} else {
				logWriter.Err("Invalid arp_entry_refresh_start_msg")
				arp_entry_refresh_done_chl <- false
			}
		case rxBuf := <-asicdSubSocketCh:
			processAsicdNotification(rxBuf)
		case <-asicdSubSocketErrCh:

		case msg := <-arp_l3_down_chl:
			deleteArpEntry(msg.ifType, msg.ifIdx)
			//default:
		}
	}
}

func processAsicdNotification(rxBuf []byte) {
	var msg asicdConstDefs.AsicdNotification
	err = json.Unmarshal(rxBuf, &msg)
	if err != nil {
		logWriter.Err(fmt.Sprintln("Unable to unmashal Asicd Msg:", msg.Msg))
		return
	}
	if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
		msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
		//Vlan Create Msg
		var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
		err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
		if err != nil {
			logWriter.Err(fmt.Sprintln("Unable to unmashal vlanNotifyMsg:", msg.Msg))
			return
		}
		updatePortPropertyMap(vlanNotifyMsg, msg.MsgType)
	} else if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
		msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
		//IPV4INTF_CREATE and IPV4INTF_DELETE
		// if create send ARPProbe
		// else delete
		var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
		err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
		if err != nil {
			logWriter.Err(fmt.Sprintln("Unable to unmashal ipv4IntfNotifyMsg:", msg.Msg))
			return
		}
		updateIpv4IntfPropertyMap(ipv4IntfNotifyMsg, msg.MsgType)
	} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
		//INTF_STATE_CHANGE
		var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
		err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
		if err != nil {
			logWriter.Err(fmt.Sprintln("Unable to unmashal l3IntfStateNotifyMsg:", msg.Msg))
			return
		}
		processL3StateChange(l3IntfStateNotifyMsg)
	} else if msg.MsgType == asicdConstDefs.NOTIFY_LAG_CREATE ||
		msg.MsgType == asicdConstDefs.NOTIFY_LAG_DELETE {
		var lagNotifyMsg asicdConstDefs.LagNotifyMsg
		err = json.Unmarshal(msg.Msg, &lagNotifyMsg)
		if err != nil {
			logWriter.Err(fmt.Sprintln("Unable to unmashal lagNotifyMsg:", msg.Msg))
			return
		}
		updatePortLagPropertyMap(lagNotifyMsg, msg.MsgType)
	}
}

func updatePortPropertyMap(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		entry := vlanPropertyMap[int(vlanNotifyMsg.VlanId)]
		entry.vlanName = vlanNotifyMsg.VlanName
		entry.untaggedPorts = vlanNotifyMsg.UntagPorts
		vlanPropertyMap[int(vlanNotifyMsg.VlanId)] = entry
		for _, portNum := range vlanNotifyMsg.UntagPorts {
			ent := port_property_map[int(portNum)]
			ent.untagged_vlanid = int(vlanNotifyMsg.VlanId)
			ent.untagged_vlanName = vlanNotifyMsg.VlanName
			port_property_map[int(portNum)] = ent
		}
	} else { // Delete Vlan
		delete(vlanPropertyMap, int(vlanNotifyMsg.VlanId))
		for _, portNum := range vlanNotifyMsg.UntagPorts {
			delete(port_property_map, int(portNum))
		}
	}
}

func updatePortLagPropertyMap(msg asicdConstDefs.LagNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_LAG_CREATE { // Create LAG
		for _, portNum := range msg.IfIndexList {
			ent := portLagPropertyMap[portNum]
			ent.IfIndex = msg.IfIndex
			portLagPropertyMap[portNum] = ent
		}
	} else { // Delete Lag
		for _, portNum := range msg.IfIndexList {
			delete(portLagPropertyMap, portNum)
		}
	}
}

func updateIpv4IntfPropertyMap(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE {
		entry := ipv4IntfPropertyMap[msg.IpAddr]
		ifIndex := msg.IfIndex
		entry.ifIdx = asicdConstDefs.GetIntfIdFromIfIndex(ifIndex)
		entry.ifType = asicdConstDefs.GetIntfTypeFromIfIndex(ifIndex)
		entry.ifState = asicdConstDefs.INTF_STATE_DOWN
		ipv4IntfPropertyMap[msg.IpAddr] = entry
	} else {
		delete(ipv4IntfPropertyMap, msg.IpAddr)
	}
}

func processL3StateChange(msg asicdConstDefs.L3IntfStateNotifyMsg) {
	if msg.IfState == asicdConstDefs.INTF_STATE_UP {
		logger.Println("Received L3 interface up notification for", msg.IpAddr)
		entry := ipv4IntfPropertyMap[msg.IpAddr]
		entry.ifState = asicdConstDefs.INTF_STATE_UP
		// Send ARP Probe
		ip, _, err := net.ParseCIDR(msg.IpAddr)
		if err != nil {
			logWriter.Err(fmt.Sprintln("Error parsing ip address:", err))
			return
		}
		ipv4IntfPropertyMap[msg.IpAddr] = entry
		arpProbe(ip.String(), entry.ifType, entry.ifIdx)
	} else if msg.IfState == asicdConstDefs.INTF_STATE_DOWN {
		logger.Println("Received L3 interface down notification for", msg.IpAddr)
		entry := ipv4IntfPropertyMap[msg.IpAddr]
		entry.ifState = asicdConstDefs.INTF_STATE_UP
		// Delete Arp Entry as L3 interface went down
		arp_l3_down_chl <- l3IntfDownMsg{
			ifType: entry.ifType,
			ifIdx:  entry.ifIdx,
		}
		ipv4IntfPropertyMap[msg.IpAddr] = entry
	}
}

func deleteArpIfnameEntry(ifName string) {
	for ip, arp := range arp_cache.arpMap {
		if arp.ifName == ifName {
			dbCmd := fmt.Sprintf(`DELETE FROM ARPCache WHERE key='%s' ;`, ip)
			logger.Println(dbCmd)
			if dbHdl != nil {
				//logger.Println("Executing DB Command:", dbCmd)
				logWriter.Info(fmt.Sprintln("Executing DB Command:", dbCmd))
				_, err = dbutils.ExecuteSQLStmt(dbCmd, dbHdl)
				if err != nil {
					logWriter.Err(fmt.Sprintln("Failed to Delete ARP entries from DB for %s %s", ip, err))
				}
			}
			_, err := asicdClient.ClientHdl.DeleteIPv4Neighbor(ip,
				"00:00:00:00:00:00", 0, 0)
			if err != nil {
				logWriter.Err(fmt.Sprintln("Failed to delete neighbor entry", err))
			} else {
				logWriter.Info(fmt.Sprintln("Deleted an entry in asic for ", ip))
			}
			delete(arp_cache.arpMap, ip)
		}
	}
}

func deleteArpEntry(ifType int, ifIdx int) {
	if ifType == commonDefs.L2RefTypeVlan { // Vlan
		vlanEntry, exist := vlanPropertyMap[ifIdx]
		if exist {
			for _, portNum := range vlanEntry.untaggedPorts {
				portEntry, exist := portConfigMap[int(portNum)]
				if exist {
					deleteArpIfnameEntry(portEntry.Name)
				}
			}
		}
	} else if ifType == commonDefs.L2RefTypePort { // PHY
		portEntry, exist := portConfigMap[ifIdx]
		if exist {
			deleteArpIfnameEntry(portEntry.Name)
		}
	}
}

func arpProbe(ipAddr string, ifType int, ifIdx int) {
	logWriter.Info(fmt.Sprintln("Sending arp probe for: ", ipAddr))
	linux_device, err := getLinuxIfc(ifType, ifIdx)
	logWriter.Info(fmt.Sprintln("linux_device ", linux_device))
	if err != nil {
		logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", ifIdx, "type : ", ifType))
		return
	}

	logWriter.Info(fmt.Sprintln("Server:Connecting to device ", linux_device))
	handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout_pcap)
	if handle == nil {
		logWriter.Err(fmt.Sprintln("Server: No device found.:device , err ", linux_device, err))
		return
	}

	mac_addr, err := getMacAddrInterfaceName(linux_device)
	if err != nil {
		logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", linux_device))
	}
	//logger.Println("MAC addr of ", linux_device, ": ", mac_addr)
	logWriter.Info(fmt.Sprintln("MAC addr of ", linux_device, ": ", mac_addr))
	go sendArpProbe(ipAddr, handle, mac_addr)
	return
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

func getLinuxIfc(ifType int, idx int) (ifName string, err error) {
	err = nil
	if ifType == commonDefs.L2RefTypeVlan { // Vlan
		ifName = vlanPropertyMap[idx].vlanName
	} else if ifType == commonDefs.L2RefTypePort { // PHY
		ifName = portConfigMap[idx].Name
	} else {
		ifName = ""
		err = errors.New("Invalid Interface Type")
	}
	return ifName, err
}

func retry_arp_req(ip string, vlanid arpd.Int, ifType arpd.Int, localIP string) {
	linux_device, err := getLinuxIfc(int(ifType), int(vlanid))
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
	mac_addr, err := getMacAddrInterfaceName(linux_device)
	if err != nil {
		logWriter.Err(fmt.Sprintln("Unable to get the MAC addr of ", linux_device))
	}
	//logger.Println("MAC addr of ", linux_device, ": ", mac_addr)
	logWriter.Info(fmt.Sprintln("MAC addr of ", linux_device, ": ", mac_addr))

	sendArpReq(ip, handle, mac_addr, localIP)
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
		arp_cache_update_chl <- arpUpdateMsg{
			ip: "0",
			ent: arpEntry{
				macAddr: []byte{0, 0, 0, 0, 0, 0},
				vlanid:  0,
				valid:   false,
				port:    -1,
				ifName:  "",
				ifType:  -1,
				localIP: "",
				counter: timeout_counter,
			},
			msg_type: 2,
		}
	}
}

func updateCounterInArpCache() {
	arp_cache_update_chl <- arpUpdateMsg{
		ip: "0",
		ent: arpEntry{
			macAddr: []byte{0, 0, 0, 0, 0, 0},
			vlanid:  0,
			valid:   false,
			port:    -1,
			ifName:  "",
			ifType:  -1,
			localIP: "",
			counter: timeout_counter,
		},
		msg_type: 5,
	}

}

func sendArpProbe(ipAddr string, handle *pcap.Handle, mac_addr string) int {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	s2 := rand.NewSource(time.Now().UnixNano())
	r2 := rand.New(s2)
	wait := r1.Intn(probe_wait)
	time.Sleep(time.Duration(wait) * time.Second)
	for i := 0; i < probe_num; i++ {
		sendArpReq(ipAddr, handle, mac_addr, "0.0.0.0")
		diff := r2.Intn(probe_max - probe_min)
		diff = diff + probe_min
		time.Sleep(time.Duration(diff) * time.Second)
	}
	return 0
}
