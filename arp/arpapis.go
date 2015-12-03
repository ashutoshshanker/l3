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
	//device_ip       string        = "40.1.1.1"
	device_ip       string = "10.0.2.15"
	filter_string   string = "arp host 10.1.10.1"
	filter_optimize int    = 0
	logWriter       *syslog.Writer
	log_err         error
)
var arp_cache *arpCache = &arpCache{}
var asicdClient AsicdClient //Thrift client to connect to asicd

/*** TEMP DEFINES **/
//var myMac = "00:11:22:33:44:55"
var myMac = "08:00:27:75:bc:4d"
var port_map = map[arpd.Int]int{
	200: 1,
	300: 2,
}

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
	vlan_id arpd.Int) (rc arpd.Int, err error) {

	cp := arp_cache
	handle, err = pcap.OpenLive(device, snapshot_len, promiscuous, timeout)
	if handle == nil {
		logWriter.Err(fmt.Sprintln("Server: No device found.: ", device))
		return 0, nil
	}
	cp.dev_handle = handle
	logWriter.Err(fmt.Sprintln("Server: Created listener port on ", device))

	go processPacket(cp, targetIp, vlan_id)
	logWriter.Err("ARP Request served")
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
			break
		}
	}
}

func initARPhandlerParams() {
	//init syslog
	logWriter, log_err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "ARPD_LOG")
	defer logWriter.Close()

	// Initialise arp cache.
	success := initArpCache(arp_cache)
	if success != true {
		logWriter.Err("server: Failed to initialise ARP cache")
	}

	// connect to asicd
	configFile := "params/clients.json"
	ConnectToClients(configFile)
}

func processPacket(cp *arpCache, targetIp string, vlanid arpd.Int) {
	logWriter.Err("Receive the ARP request")
	_, exist := cp.arpMap[targetIp]
	if !exist {
		myMac_addr, fail := getHWAddr(myMac)
		if fail != nil {
			logWriter.Err(fmt.Sprintf("corrupted my mac : ", myMac))
			return
		}
		// 1) send arp request
		success := sendArpReq(cp, targetIp, device_ip)
		if success != ARP_REQ_SUCCESS {
			logWriter.Err(fmt.Sprintf("Failed to send ARP request. for Ip : ", targetIp))
			return
		}

		logWriter.Err("Receive arp response")
		//2) get response
		err, destMAC := receiveArpResponse(targetIp, cp,
			myMac_addr, vlanid)
		if err != ARP_REQ_SUCCESS {
			logWriter.Err("Failed to receive arp response")
		}
		logWriter.Err(fmt.Sprintf("MAC entry as - ", destMAC))

		//3) Update asicd.
		if asicdClient.IsConnected {
			port_id := port_map[vlanid]
			rv, error := asicdClient.ClientHdl.CreateIPv4Neighbor(targetIp, destMAC.String(), (int32)(vlanid), (int32)(port_id))
			logWriter.Err(fmt.Sprintf("Asicd rv: ", rv, " error : ", error))
		} else {
			logWriter.Err("Asicd client is not connected.")
		}
		return
	}

	// get MAC from cache.
	arp_entry := cp.arpMap[targetIp]
	logWriter.Err(fmt.Sprintf("Exists MAC entry as - ", arp_entry.macAddr))
	printArpEntries(cp)

	return
}

/*
 *@fn sendArpReq
 *  Send the ARP request for ip targetIP
 */
func sendArpReq(cp *arpCache, targetIp string, myIp string) int {
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
	if err := cp.dev_handle.WritePacketData(buffer.Bytes()); err != nil {
		return ARP_ERR_REQ_FAIL
	}
	return ARP_REQ_SUCCESS
}

/*
 *@fn receiveArpResponse
 * Process ARP response from the interface for ARP
 * req sent for targetIp
 */
func receiveArpResponse(targetIp string, cp *arpCache, myMac net.HardwareAddr, vlanid arpd.Int) (err int, destMac net.HardwareAddr) {
	src := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
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

			destMac = net.HardwareAddr(arp.SourceHwAddress)
			/* update cache
			 */
			arp_entry := arpEntry{
				vlanid:  vlanid,
				macAddr: destMac,
			}
			updateArpCache(cp, net.IP(arp.SourceProtAddress).String(), arp_entry)
			logger.Println("ip ", net.IP(arp.SourceProtAddress), "mac", net.HardwareAddr(arp.SourceHwAddress))
			return ARP_REQ_SUCCESS, destMac
		}

	}
	return ARP_ERR_REQ_FAIL, nil

}

/*
 *@fn InitArpCache
 * Initiliase s/w cache. It also acts a reset API for timeout.
 */
func initArpCache(cp *arpCache) bool {
	cp.arpMap = make(map[string]arpEntry)
	logWriter.Err("InitArpCache done.")
	return true
}

/*
 * @fn UpdateArpCache
 *  Update IP to the ARP mapping for the hash table.
 */
func updateArpCache(cp *arpCache, targetIp string, arp_entry arpEntry) bool {

	_, exist := cp.arpMap[targetIp]

	if !exist {
		logWriter.Err(fmt.Sprintf("Update cache."))
		cp.arpMap[targetIp] = arp_entry
		return true
	}
	logWriter.Err(fmt.Sprintf("Entry exists.."))

	return true
}

func printArpEntries(cp *arpCache) {
	logWriter.Err(fmt.Sprintf("************"))
	for ip, arp := range cp.arpMap {
		logWriter.Err(fmt.Sprintf(ip, ":", arp.vlanid, ":", arp.macAddr))
		logger.Println(ip, ":", arp.vlanid, ":", arp.macAddr)
	}
	logWriter.Err(fmt.Sprintf("*************"))
}
