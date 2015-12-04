package main

import (
	"arpd"
	"asicdServices"
	"bytes"
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
}

type arpCache struct {
	cacheTimeout time.Duration
	arpMap       map[string]arpEntry
	dev_handle   *pcap.Handle
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

/*
 * connection params.
 */
var (
	//device          string = "fpPort2"
	device       string = "eth0"
	snapshot_len int32  = 1024 //packet capture length
	promiscuous  bool   = true //mode
	err          error
	timeout      time.Duration = 1 * time.Second
	handle       *pcap.Handle  // handle for pcap connection
	device_ip    string        = "40.1.1.1"
	//device_ip       string = "10.0.2.15"
	filter_string   string = "arp host 10.1.10.1"
	filter_optimize int    = 0
	logWriter       *syslog.Writer
	log_err         error
	rec_handle      []*pcap.Handle
)
var arp_cache *arpCache
var asicdClient AsicdClient //Thrift client to connect to asicd
var portdClient PortdClient //portd services client
var rec_handle_map map[*pcap.Handle]int

/*** TEMP DEFINES **/
var myMac = "00:11:22:33:44:55"

//var myMac = "08:00:27:75:bc:4d"

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

/***** Thrift APIs ******/
func (m ARPServiceHandler) RestolveArpIPV4(targetIp string,
	iftype arpd.Int, vlan_id arpd.Int) (rc arpd.Int, err error) {

	if portdClient.IsConnected {
		linux_device, err := portdClient.ClientHdl.GetLinuxIfc(int32(iftype), int32(vlan_id))
		logger.Println(linux_device)
		if err != nil {
			logWriter.Err(fmt.Sprintf("Failed to get ifname for interface : ", vlan_id, "type : ", iftype))
			return ARP_ERR_REQ_FAIL, err
		}
		handle, err = pcap.OpenLive(linux_device, snapshot_len, promiscuous, timeout)
		//handle, err = pcap.OpenLive(device, snapshot_len, promiscuous, timeout)
		if handle == nil {
			logWriter.Err(fmt.Sprintln("Server: No device found.: ", device))
			return 0, nil
		}
		arp_cache.dev_handle = handle
		initPortParams()
		go processPacket(targetIp, vlan_id)

	} else {
		logWriter.Err("portd client is not connected.")
		logger.Println("Portd is not connected.")
	}

	logWriter.Err(fmt.Sprintln("Server: Created listener port on ", device))

	//logWriter.Err("ARP Request served")
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

	// connect to asicd and portd
	configFile := "/opt/flexswitch/params/clients.json"
	ConnectToClients(configFile)

	//initPortParams()

}

func BuildAsicToLinuxMap(cfgFile string) {
	var portCfgList []PortConfigJson
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
	rec_handle_map = make(map[*pcap.Handle]int)
	for _, v := range portCfgList {
		local_handle, err := pcap.OpenLive(v.Ifname, snapshot_len, promiscuous, timeout)
		//local_handle, err := pcap.OpenLive(device, snapshot_len, promiscuous, timeout)
		if local_handle == nil {
			logWriter.Err(fmt.Sprintln("Server: No device found.: ", v.Ifname, err))
			return
		}
		rec_handle_map[local_handle] = v.Port
	}

}
func initPortParams() {
	configFile := "/opt/flexswitch/params/clients.json"
	ConnectToClients(configFile)
	portCfgFile := "/opt/flexswitch/params/portd.json"
	BuildAsicToLinuxMap(portCfgFile)
}

func processPacket(targetIp string, vlanid arpd.Int) {
	logWriter.Err("Receive the ARP request")
	logger.Println("Receive arp req for ", targetIp)
	_, exist := arp_cache.arpMap[targetIp]
	if !exist {
		// 1) send arp request
		success := sendArpReq(targetIp, device_ip)
		if success != ARP_REQ_SUCCESS {
			logWriter.Err(fmt.Sprintf("Failed to send ARP request. for Ip : ", targetIp))
			return
		}
		processResponse(targetIp, vlanid)
		logWriter.Err("Receive arp response")

	}
	// get MAC from cache.
	arp_entry := arp_cache.arpMap[targetIp]
	logWriter.Err(fmt.Sprintf("Exists MAC entry as - ", arp_entry.macAddr))
	printArpEntries()

	return
}

func processResponse(targetIp string, vlanid arpd.Int) {
	myMac_addr, fail := getHWAddr(myMac)
	if fail != nil {
		logWriter.Err(fmt.Sprintf("corrupted my mac : ", myMac))
		return
	}
	for rec_handle, port_id := range rec_handle_map {
		go receiveArpResponse(targetIp, rec_handle,
			myMac_addr, vlanid, port_id)
	}
	return
}

/*
 *@fn sendArpReq
 *  Send the ARP request for ip targetIP
 */
func sendArpReq(targetIp string, myIp string) int {
	source_ip, err := getIP(myIp)
	if err != ARP_REQ_SUCCESS {
		logWriter.Err(fmt.Sprintf("Corrupted source ip :  ", myIp))
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

	if err := arp_cache.dev_handle.WritePacketData(buffer.Bytes()); err != nil {
		return ARP_ERR_REQ_FAIL
	}
	return ARP_REQ_SUCCESS
}

/*
 *@fn receiveArpResponse
 * Process ARP response from the interface for ARP
 * req sent for targetIp
 */
func receiveArpResponse(targetIp string, rec_handle *pcap.Handle,
	myMac net.HardwareAddr, vlanid arpd.Int, port_id int) (err int, destMac net.HardwareAddr) {
	logger.Println("Check arp response for #### ", targetIp)
	src := gopacket.NewPacketSource(rec_handle, layers.LayerTypeEthernet)
	in := src.Packets()
	for {
		packet, ok := <-in
		if ok {

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

			logWriter.Err(fmt.Sprintf("arp response### ", net.IP(arp.SourceProtAddress), " ", net.HardwareAddr(arp.SourceHwAddress)))
			logger.Println("Arp response ###", net.IP(arp.SourceProtAddress), " ", net.HardwareAddr(arp.SourceHwAddress))

			destMac = net.HardwareAddr(arp.SourceHwAddress)
			/* update cache
			 */
			arp_entry := arpEntry{
				vlanid:  vlanid,
				macAddr: destMac,
			}
			updateArpCache(net.IP(arp.SourceProtAddress).String(), arp_entry)
			logger.Println("ip ", net.IP(arp.SourceProtAddress), "mac", net.HardwareAddr(arp.SourceHwAddress))
			if err != ARP_REQ_SUCCESS {
				logWriter.Err("Failed to receive arp response")
			}
			logWriter.Err(fmt.Sprintf("MAC entry as - ", destMac))

			//3) Update asicd.
			if asicdClient.IsConnected {
				rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(targetIp, destMac.String(), (int32)(vlanid), (int32)(port_id))
				logWriter.Err(fmt.Sprintf("Asicd rv: ", rv, " error : ", error))
			} else {
				logWriter.Err("Asicd client is not connected.")
			}
			return
		}

	}
	return ARP_ERR_REQ_FAIL, nil

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
func updateArpCache(targetIp string, arp_entry arpEntry) bool {

	_, exist := arp_cache.arpMap[targetIp]

	if !exist {
		logWriter.Err(fmt.Sprintf("Update cache."))
		arp_cache.arpMap[targetIp] = arp_entry
		return true
	}
	logWriter.Err(fmt.Sprintf("Entry exists.."))

	return true
}

func printArpEntries() {
	logWriter.Err(fmt.Sprintf("************"))
	for ip, arp := range arp_cache.arpMap {
		logWriter.Err(fmt.Sprintf(ip, ":", arp.vlanid, ":", arp.macAddr))
		logger.Println(ip, ":", arp.vlanid, ":", arp.macAddr)
	}
	logWriter.Err(fmt.Sprintf("*************"))
}
