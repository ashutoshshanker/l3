package server

import (
    "fmt"
    "bytes"
    "git.apache.org/thrift.git/lib/go/thrift"
    nanomsg "github.com/op/go-nanomsg"
    "encoding/json"
    "io/ioutil"
    "strconv"
    "utils/ipcutils"
    "l3/ospf/config"
    //"l3/ospf/rpc"
    "time"
    "log/syslog"
    "ribd"
//    "l3/rib/ribdCommonDefs"
    "asicdServices"
    "asicd/asicdConstDefs"
/*
    "asicd/pluginManager/pluginCommon"
*/
    "net"
    "golang.org/x/net/ipv4"
   // "github.com/x/net/ipv4"
    "encoding/gob"
    "runtime"
)

const (
    OSPF_HELLO_MIN_SIZE = 20
    OSPF_HEADER_SIZE = 24
)

type ClientJson struct {
        Name string `json:Name`
        Port int    `json:Port`
}

type OspfClientBase struct {
        Address            string
        Transport          thrift.TTransport
        PtrProtocolFactory *thrift.TBinaryProtocolFactory
        IsConnected        bool
}

type AsicdClient struct {
        OspfClientBase
        ClientHdl *asicdServices.ASICDServicesClient
}

type RibdClient struct {
        OspfClientBase
        ClientHdl *ribd.RouteServiceClient
}

type PortProperty struct {
    Name    string
}

type IPIntfProperty struct {
    IntfName        string
    IntfType        uint8
    IntfIdx         uint8
}

type IntfConfKey struct {
    IPAddr          config.IpAddress
    IntfIdx         config.InterfaceIndexOrZero
}

type IntfConf struct {
    IfAreaId                []byte
    IfType                  config.IfType
    IfAdminStat             config.Status
    IfRtrPriority           uint8
    IfTransitDelay          config.UpToMaxAge
    IfRetransInterval       config.UpToMaxAge
    IfHelloInterval         uint16
    IfRtrDeadInterval       uint32
    IfPollInterval          config.PositiveInteger
    IfAuthKey               []byte
    IfMulticastForwarding   config.MulticastForwarding
    IfDemand                bool
    IfAuthType              uint16
    HelloPktSendCh          chan bool
    HelloPktRecvCh          chan bool
    HelloPktSendRecvState   bool
}

type GlobalConf struct {
    RouterId                    []byte
    AdminStat                   config.Status
    ASBdrRtrStatus              bool
    TOSSupport                  bool
    ExtLsdbLimit                int32
    MulticastExtensions         int32
    ExitOverflowInterval        config.PositiveInteger
    DemandExtensions            bool
    RFC1583Compatibility        bool
    ReferenceBandwidth          int32
    RestartSupport              config.RestartSupport
    RestartInterval             int32
    RestartStrictLsaChecking    bool
    StubRouterAdvertisement     config.AdvertiseAction
}


type OSPFHeader struct {
    ver         uint8
    pktType     uint8
    pktlen      uint16
    routerId    []byte
    areaId      []byte
    chksum      uint16
    authType    uint16
    authKey     []uint8
}

type OSPFHelloData struct {
    netMask             []byte
    helloInterval       uint16
    rtrDeadInterval     uint32
    deginatedRtr        []byte
    backupDesignatedRtr []byte
    neighbor            []byte
}

type OSPFServer struct {
    logger                  *syslog.Writer
    ribdClient              RibdClient
    asicdClient             AsicdClient
    portPropertyMap         map[int]PortProperty
    IPIntfPropertyMap       map[string]IPIntfProperty
    ospfGlobalConf          GlobalConf
    GlobalConfigCh          chan config.GlobalConf
    IntfConfigCh            chan config.InterfaceConf
    connRoutesTimer         *time.Timer

/*
    ribSubSocket        *nanomsg.SubSocket
    ribSubSocketCh      chan []byte
    ribSubSocketErrCh   chan error
*/

    asicdSubSocket          *nanomsg.SubSocket
    asicdSubSocketCh        chan []byte
    asicdSubSocketErrCh     chan error
    IntfConfMap             map[IntfConfKey]IntfConf
}

func NewOSPFServer(logger *syslog.Writer) *OSPFServer {
    ospfServer := &OSPFServer{}
    ospfServer.logger = logger
    ospfServer.GlobalConfigCh = make(chan config.GlobalConf)
    ospfServer.IntfConfigCh = make(chan config.InterfaceConf)
    ospfServer.portPropertyMap = make(map[int]PortProperty)
    ospfServer.IntfConfMap = make(map[IntfConfKey]IntfConf)
/*
    ospfServer.ribSubSocketCh = make(chan []byte)
    ospfServer.ribSubSocketErrCh = make(chan error)
    ospfServer.connRoutesTimer = time.NewTimer(time.Duration(10) * time.Second)
    ospfServer.connRoutesTimer.Stop()
*/
    ospfServer.asicdSubSocketCh = make(chan []byte)
    ospfServer.asicdSubSocketErrCh = make(chan error)

    return ospfServer
}

/*
func (server *OSPFServer) listenForRIBUpdates(address string) error {
    var err error
    if server.ribSubSocket, err = nanomsg.NewSubSocket(); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to create RIB subscribe socket, error:", err))
        return err
    }

    if err = server.ribSubSocket.Subscribe(""); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on RIB subscribe socket, error:", err))
        return err
    }

    if _, err = server.ribSubSocket.Connect(address); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to connect to RIB publisher socket, address:", address, "error:", err))
        return err
    }

    server.logger.Info(fmt.Sprintln("Connected to RIB publisher at address:", address))
    if err = server.ribSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to set the buffer size for RIB publisher socket, error:", err))
        return err
    }
    return nil
}
*/

func (server *OSPFServer) updateGlobalConf(gConf config.GlobalConf) {
    routerId := convertAreaOrRouterId(string(gConf.RouterId))
    if routerId == nil {
        server.logger.Err("Invalid Router Id")
        return
    }
    server.ospfGlobalConf.RouterId = routerId
    server.ospfGlobalConf.AdminStat = gConf.AdminStat
    server.ospfGlobalConf.ASBdrRtrStatus = gConf.ASBdrRtrStatus
    server.ospfGlobalConf.TOSSupport = gConf.TOSSupport
    server.ospfGlobalConf.ExtLsdbLimit = gConf.ExtLsdbLimit
    server.ospfGlobalConf.MulticastExtensions = gConf.MulticastExtensions
    server.ospfGlobalConf.ExitOverflowInterval = gConf.ExitOverflowInterval
    server.ospfGlobalConf.RFC1583Compatibility = gConf.RFC1583Compatibility
    server.ospfGlobalConf.ReferenceBandwidth = gConf.ReferenceBandwidth
    server.ospfGlobalConf.RestartSupport = gConf.RestartSupport
    server.ospfGlobalConf.RestartInterval= gConf.RestartInterval
    server.ospfGlobalConf.RestartStrictLsaChecking = gConf.RestartStrictLsaChecking
    server.ospfGlobalConf.StubRouterAdvertisement = gConf.StubRouterAdvertisement
}

func (server *OSPFServer) initOspfGlobalConfDefault() {
    routerId := convertAreaOrRouterId("0.0.0.0")
    if routerId == nil {
        server.logger.Err("Invalid Router Id")
        return
    }
    server.ospfGlobalConf.RouterId = routerId
    server.ospfGlobalConf.AdminStat = config.Status(2) // disabled
    server.ospfGlobalConf.ASBdrRtrStatus = false
    server.ospfGlobalConf.TOSSupport = false
    server.ospfGlobalConf.ExtLsdbLimit = -1
    server.ospfGlobalConf.MulticastExtensions = 0
    server.ospfGlobalConf.ExitOverflowInterval = 0
    server.ospfGlobalConf.RFC1583Compatibility = false
    server.ospfGlobalConf.ReferenceBandwidth = 100000 // Default value 100 MBPS
    server.ospfGlobalConf.RestartSupport = 1 // none
    server.ospfGlobalConf.RestartInterval= 0
    server.ospfGlobalConf.RestartStrictLsaChecking = false
    server.ospfGlobalConf.StubRouterAdvertisement = 1 //doNotAdvertise
}

func (server *OSPFServer) ConnectToClients(paramsFile string) {
    var clientsList []ClientJson

    bytes, err := ioutil.ReadFile(paramsFile)
    if err != nil {
        server.logger.Info("Error in reading configuration file")
        return
    }

    err = json.Unmarshal(bytes, &clientsList)
    if err != nil {
            server.logger.Info("Error in Unmarshalling Json")
            return
    }

    for _, client := range clientsList {
        server.logger.Info("#### Client name is ")
        server.logger.Info(client.Name)
        if client.Name == "asicd" {
            server.logger.Info(fmt.Sprintln("found asicd at port", client.Port))
            server.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
            server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(server.asicdClient.Address)
            if server.asicdClient.Transport != nil && server.asicdClient.PtrProtocolFactory != nil {
                server.logger.Info("connecting to asicd")
                server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
                server.asicdClient.IsConnected = true
            }
        } else if client.Name == "ribd" {
            server.logger.Info(fmt.Sprintln("found ribd at port", client.Port))
            server.ribdClient.Address = "localhost:" + strconv.Itoa(client.Port)
            server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(server.ribdClient.Address)
            if server.ribdClient.Transport != nil && server.ribdClient.PtrProtocolFactory != nil {
                server.logger.Info("connecting to ribd")
                server.ribdClient.ClientHdl = ribd.NewRouteServiceClientFactory(server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory)
                server.ribdClient.IsConnected = true
            }
        }
    }
}

func (server *OSPFServer) BuildPortPropertyMap() {
    currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
    if server.asicdClient.IsConnected {
        server.logger.Info("Calling asicd for port property")
        count := 10
        for {
            bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPortConfig(int64(currMarker), int64(count))
            if bulkInfo == nil {
                return
            }
            objCount := int(bulkInfo.ObjCount)
            more := bool(bulkInfo.More)
            currMarker = bulkInfo.NextMarker
            for i := 0; i < objCount; i++ {
                portNum := int(bulkInfo.PortConfigList[i].PortNum)
                ent := server.portPropertyMap[portNum]
                ent.Name = bulkInfo.PortConfigList[i].Name
                server.portPropertyMap[portNum] = ent
            }
            if more == false {
                return
            }
        }
    }
}

/*
func createRIBSubscriber() {
    for {
        server.logger.Info("Read on RIB subscriber socket...")
        ribrxBuf, err := server.ribSubSocket.Recv(0)
        if err != nil {
            server.logger.Err(fmt.Sprintln("Recv on RIB subscriber socket failed with error:", err))
            server.ribSubSocketErrCh <- err
            continue
        }
        server.logger.Info(fmt.Sprintln("RIB subscriber recv returned:", ribrxBuf))
        server.ribSubSocketCh <- ribrxBuf
    }
}
*/

func (server *OSPFServer)createASICdSubscriber() {
    for {
        server.logger.Info("Read on ASICd subscriber socket...")
        asicdrxBuf, err := server.asicdSubSocket.Recv(0)
        if err != nil {
            server.logger.Err(fmt.Sprintln("Recv on ASICd subscriber socket failed with error:", err))
            server.asicdSubSocketErrCh <- err
            continue
        }
        server.logger.Info(fmt.Sprintln("ASIC subscriber recv returned:", asicdrxBuf))
        server.asicdSubSocketCh <- asicdrxBuf
    }
}

func (server *OSPFServer) listenForASICdUpdates(address string) error {
    var err error
    if server.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to create ASICd subscribe socket, error:", err))
        return err
    }

    if err = server.asicdSubSocket.Subscribe(""); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on ASICd subscribe socket, error:", err))
        return err
    }

    if _, err = server.asicdSubSocket.Connect(address); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to connect to ASICd publisher socket, address:", address, "error:", err))
        return err
    }

    server.logger.Info(fmt.Sprintln("Connected to ASICd publisher at address:", address))
    if err = server.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
        server.logger.Err(fmt.Sprintln("Failed to set the buffer size for ASICd publisher socket, error:", err))
        return err
    }
    return nil
}


func (server *OSPFServer)initDefaultIntfConf(key IntfConfKey) {
    ent, exist := server.IntfConfMap[key]
    if !exist {
        areaId := convertAreaOrRouterId("0.0.0.0")
        if areaId == nil {
            return
        }
        ent.IfAreaId = areaId
        ent.IfType = config.Broadcast
        ent.IfAdminStat = config.Enabled
        ent.IfRtrPriority = uint8(config.DesignatedRouterPriority(1))
        ent.IfTransitDelay = config.UpToMaxAge(1)
        ent.IfRetransInterval = config.UpToMaxAge(5)
        ent.IfHelloInterval = uint16(config.HelloRange(10))
        ent.IfRtrDeadInterval = uint32(config.PositiveInteger(40))
        ent.IfPollInterval = config.PositiveInteger(120)
        authKey := convertAuthKey("0.0.0.0.0.0.0.0")
        if authKey == nil {
            return
        }
        ent.IfAuthKey = authKey
        ent.IfMulticastForwarding = config.Blocked
        ent.IfDemand = false
        ent.IfAuthType = uint16(config.NoAuth)
        ent.HelloPktSendRecvState = false
        ent.HelloPktSendCh = make(chan bool)
        ent.HelloPktRecvCh = make(chan bool)
        server.IntfConfMap[key] = ent
    }
}

/*
func (server *OSPFServer)createIPIntfConfMap(msg pluginCommon.IPv4IntfNotifyMsg) {
    ip, _, err := net.ParseCIDR(msg.IpAddr)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to parse IP address", msg.IpAddr))
        return
    }

    // Set ifIdx = 0 for time being --- Need to be revisited
    intfConfKey := IntfConfKey {
        IPAddr:     config.IpAddress(ip.String()),
        //IntfIdx:    int(msg.IfIdx),
        IntfIdx:    config.InterfaceIndexOrZero(0),
    }
    server.initDefaultIntfConf(intfConfKey)
}

func (server *OSPFServer)deleteIPIntfConfMap(msg pluginCommon.IPv4IntfNotifyMsg) {
    ip, _, err := net.ParseCIDR(msg.IpAddr)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to parse IP address", msg.IpAddr))
        return
    }

    // Set ifIdx = 0 for time being --- Need to be revisited
    intfConfKey := IntfConfKey {
        IPAddr:     config.IpAddress(ip.String()),
        //IntfIdx:    int(msg.IfIdx),
        IntfIdx:    config.InterfaceIndexOrZero(0),
    }
    delete(server.IntfConfMap, intfConfKey)
}
*/

func (server *OSPFServer)updateIPIntfConfMap(ifConf config.InterfaceConf) {
    intfConfKey := IntfConfKey {
        IPAddr:             ifConf.IfIpAddress,
        IntfIdx:            config.InterfaceIndexOrZero(ifConf.AddressLessIf),
    }

    ent, exist := server.IntfConfMap[intfConfKey]
    //if exist {
    // HACK: This thing need to go away we can update only when we already have entry
    if !exist {
        areaId := convertAreaOrRouterId(string(ifConf.IfAreaId))
        if areaId == nil {
            server.logger.Err("Invalid areaId")
            return
        }
        ent.IfAreaId = areaId
        ent.IfType = ifConf.IfType
        ent.IfAdminStat = ifConf.IfAdminStat
        ent.IfRtrPriority = uint8(ifConf.IfRtrPriority)
        ent.IfTransitDelay = ifConf.IfTransitDelay
        ent.IfRetransInterval = ifConf.IfRetransInterval
        ent.IfHelloInterval = uint16(ifConf.IfHelloInterval)
        ent.IfRtrDeadInterval = uint32(ifConf.IfRtrDeadInterval)
        ent.IfPollInterval = ifConf.IfPollInterval
        authKey := convertAuthKey(string(ifConf.IfAuthKey))
        if authKey == nil {
            server.logger.Err("Invalid authKey")
            return
        }
        ent.IfAuthKey = authKey
        ent.IfMulticastForwarding = ifConf.IfMulticastForwarding
        ent.IfDemand = ifConf.IfDemand
        ent.IfAuthType = uint16(ifConf.IfAuthType)
        server.IntfConfMap[intfConfKey] = ent
    }
}

func (server *OSPFServer)RecvHelloPkt(ifName string) {
/*
    var filter string = "not ether proto 0x8809"
    local_handle, err := pcap.OpenLive(ifName, snapshot_len, promiscuous, timeout_pcap)
*/
}

func encodeOspfHdr(ospfHdr OSPFHeader) ([]byte) {
    encBuf := new(bytes.Buffer)
    err := gob.NewEncoder(encBuf).Encode(ospfHdr)
    if err != nil {
        return nil
    }

    return encBuf.Bytes()
}


func (server *OSPFServer)SendHelloPkt(ifName string, intfConfKey IntfConfKey) {
    c, err := net.ListenPacket("ip4:89", "0.0.0.0") // OSPF for IPv4
    if err != nil {
        server.logger.Err(fmt.Sprintln("Error listen for packet:", err))
        return
    }
    defer c.Close()
    r, err := ipv4.NewRawConn(c)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to open new raw conn:", err))
        return
    }

    iface, err := net.InterfaceByName(ifName)
    if err != nil {
        server.logger.Err(fmt.Sprintln("Unable to get Interface:", err))
        return
    }
    allSPFRouters := net.IPAddr{IP: net.IPv4(224, 0, 0, 5)}
    if err := r.JoinGroup(iface, &allSPFRouters); err != nil {
        server.logger.Err(fmt.Sprintln("Unable to Join allSPFRouters group:", err))
        return
    }
    defer r.LeaveGroup(iface, &allSPFRouters)

    hello := make([]byte, 24) // fake hello data, you need to implement this
   // ospf := make([]byte, 24)  // fake ospf header, you need to implement this
    ent := server.IntfConfMap[intfConfKey]
    ospfHdr := OSPFHeader {
        ver:            2,
        pktType:        1,
        pktlen:         0,
        routerId:       server.ospfGlobalConf.RouterId,
        areaId:         ent.IfAreaId,
        chksum:         0,
        authType:       ent.IfAuthType,
        authKey:        ent.IfAuthKey,
    }
    //ospf[0] = 2               // version 2
   // ospf[1] = 1               // hello packet
    pktlen := OSPF_HEADER_SIZE
    pktlen = pktlen + OSPF_HELLO_MIN_SIZE

    ospfHdr.pktlen = uint16(pktlen)

    ospfEncHdr := encodeOspfHdr(ospfHdr)
    if ospfEncHdr == nil {
        server.logger.Err("Unable to encode ospfHdr")
        return
    }

    ospf := append(ospfEncHdr, hello...)
    iph := &ipv4.Header{
        Version:  ipv4.Version,
        Len:      ipv4.HeaderLen,
        TOS:      0xc0, // DSCP CS6
        TotalLen: ipv4.HeaderLen + len(ospf),
        TTL:      1,
        Protocol: 89,
        Dst:      allSPFRouters.IP.To4(),
    }

    var cm *ipv4.ControlMessage
    switch runtime.GOOS {
    case "darwin", "linux":
        cm = &ipv4.ControlMessage{IfIndex: iface.Index}
    default:
        if err := r.SetMulticastInterface(iface); err != nil {
            server.logger.Err(fmt.Sprintln("Unable to set Multicast Interface:", err))
            return
        }
    }
    if err := r.WriteTo(iph, ospf, cm); err != nil {
        server.logger.Err(fmt.Sprintln("Unable to WriteTo:", err))
        return
    }
}

func (server *OSPFServer)StopSendRecvHelloPkts(intfConfKey IntfConfKey) {
    ent, _ := server.IntfConfMap[intfConfKey]
    if ent.HelloPktSendRecvState == false {
        return
    }

    server.logger.Info("Stop Receiving Hello Pkt")
    ent.HelloPktRecvCh<-false
    server.logger.Info("Stop Sending Hello Pkt")
    ent.HelloPktSendCh<-false
    ent.HelloPktSendRecvState = false
    server.IntfConfMap[intfConfKey] = ent
}

func (server *OSPFServer)StartSendRecvHelloPkts(intfConfKey IntfConfKey) {

    ent, _ := server.IntfConfMap[intfConfKey]
    if ent.HelloPktSendRecvState == true {
        return
    }

    server.logger.Info("Start Sending Hello Pkt")
    ent.HelloPktSendCh<-true
    server.logger.Info("Start Receiving Hello Pkt")
    ent.HelloPktRecvCh<-true
    ent.HelloPktSendRecvState = true
    server.IntfConfMap[intfConfKey] = ent
}

func (server *OSPFServer) StartServer(paramFile string) {
    server.logger.Info(fmt.Sprintln("Starting Ospf Server"))
    server.ConnectToClients(paramFile)
    server.BuildPortPropertyMap()
    server.initOspfGlobalConfDefault()
    server.logger.Info(fmt.Sprintln("GlobalConf:", server.ospfGlobalConf))

/*
    server.logger.Info("Listen for RIBd updates")
    server.listenForRIBUpdates(ribdCommonDefs.PUB_SOCKET_ADDR)
    go createRIBSubscriber()
    server.connRoutesTimer.Reset(time.Duration(10) * time.Second)
*/


/*
    server.logger.Info("Listen for ASICd updates")
    server.listenForASICdUpdates(pluginCommon.PUB_SOCKET_ADDR)
    go server.createASICdSubscriber()
*/

    for {
        select {
            case gConf := <-server.GlobalConfigCh:
                var localIntfStateMap map[IntfConfKey]bool = make(map[IntfConfKey]bool)
                for key, ent := range server.IntfConfMap {
                    localIntfStateMap[key] = ent.HelloPktSendRecvState
                    if ent.HelloPktSendRecvState == true {
                        server.StopSendRecvHelloPkts(key)
                    }
                }
                server.logger.Info(fmt.Sprintln("Received call for performing Global Configuration", gConf))
                server.updateGlobalConf(gConf)
                server.logger.Info(fmt.Sprintln("GlobalConf:", server.ospfGlobalConf))
                if gConf.AdminStat == config.Enabled {
                    for key, ent := range localIntfStateMap {
                        if ent == true {
                            server.StartSendRecvHelloPkts(key)
                        }
                    }
                } else {
                    for key, ent := range localIntfStateMap {
                        intfConfMapEntry := server.IntfConfMap[key]
                        intfConfMapEntry.HelloPktSendRecvState = ent
                        server.IntfConfMap[key] = intfConfMapEntry
                    }
                }
            case ifConf := <-server.IntfConfigCh:
                server.logger.Info(fmt.Sprintln("Received call for performing Intf Configuration", ifConf))
                intfConfKey := IntfConfKey {
                    IPAddr:             ifConf.IfIpAddress,
                    IntfIdx:            config.InterfaceIndexOrZero(ifConf.AddressLessIf),
                }
                _, exist := server.IntfConfMap[intfConfKey]
                if !exist {
                    server.logger.Err("No such L3 interface exists")
                    continue
                }
                server.StopSendRecvHelloPkts(intfConfKey)
                server.updateIPIntfConfMap(ifConf)
                // Hack to Start SEND AND RECV HELLO PKT on SVI4 interface
                server.SendHelloPkt("SVI4", intfConfKey)
                server.RecvHelloPkt("SVI4")

                server.logger.Info(fmt.Sprintln("InterfaceConf:", server.IntfConfMap))
                if ifConf.IfAdminStat == config.Enabled &&
                    server.ospfGlobalConf.AdminStat == config.Enabled {
                    // Stop the SendRecvHelloPkts thread if running
                    // And then start SendRecvHelloPkts thread for Neighbor Discovery
                    // Second argument= true 
                    server.StartSendRecvHelloPkts(intfConfKey)
                } else if ifConf.IfAdminStat == config.Disabled {
                    // Stop the SendRecvHelloPkts thread if running
                    // Second argument=false
                    server.StopSendRecvHelloPkts(intfConfKey)
                }
/*
            case asicdrxBuf := <-server.asicdSubSocketCh:
                var msg pluginCommon.AsicdNotification
                err := json.Unmarshal(asicdrxBuf, &msg)
                if err != nil {
                    server.logger.Err(fmt.Sprintln("Unable to unmarshal asicdrxBuf:", asicdrxBuf))
                    continue
                }
                if msg.MsgType == pluginCommon.NOTIFY_IPV4INTF_CREATE ||
                    msg.MsgType == pluginCommon.NOTIFY_IPV4INTF_DELETE {
                    var ipv4IntfMsg pluginCommon.IPv4IntfNotifyMsg
                    err = json.Unmarshal(msg.Msg, &ipv4IntfMsg)
                    if err != nil {
                        server.logger.Err(fmt.Sprintln("Unable to unmarshal msg:", msg.Msg))
                        continue
                    }
                    if msg.MsgType == pluginCommon.NOTIFY_IPV4INTF_CREATE {
                        server.createIPIntfConfMap(ipv4IntfMsg)
                    } else {
                        server.deleteIPIntfConfMap(ipv4IntfMsg)
                    }
                }
            case <-server.asicdSubSocketErrCh:
                ;
*/
/*
            case ribrxBuf := <-server.ribSubSocketCh:
                var route ribdCommonDefs.RoutelistInfo
                routes := make([]*ribd.Routes, 0, 1)
                reader := bytes.NewReader(ribrxBuf)
                decoder := json.NewDecoder(reader)
                msg := ribdCommonDefs.RibdNotifyMsg{}
                for err := decoder.Decode(&msg); err == nil; err = decoder.Decode(&msg) {
                    err = json.Unmarshal(msg.MsgBuf, &route)
                    if err != nil {
                            server.logger.Err("Err in processing routes from RIB")
                    }
                    server.logger.Info(fmt.Sprintln("Remove connected route, dest:", route.RouteInfo.Ipaddr, "netmask:", route.RouteInfo.Mask, "nexthop:", route.RouteInfo.NextHopIp))
                    routes = append(routes, &route.RouteInfo)
                }
                //server.ProcessConnectedRoutes(make([]*ribd.Routes, 0), routes)
            case <-server.connRoutesTimer.C:
                routes, _ := server.ribdClient.ClientHdl.GetConnectedRoutesInfo()
                server.logger.Info(fmt.Sprintln("Received Connected Routes:", routes))
                //server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))
                //server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

            case <-server.ribSubSocketErrCh:
                ;
*/
        }
    }
}
