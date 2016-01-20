package server

import (
    "fmt"
    "git.apache.org/thrift.git/lib/go/thrift"
    nanomsg "github.com/op/go-nanomsg"
    "encoding/json"
    "io/ioutil"
    "strconv"
    "utils/ipcutils"
    "l3/ospf/config"
    "time"
    "log/syslog"
    "ribd"
//    "l3/rib/ribdCommonDefs"
    "asicdServices"
    "asicd/pluginManager/pluginCommon"
)

var (
    snapshot_len            int32 = 65549  //packet capture length
    promiscuous             bool = false  //mode
    timeout_pcap            time.Duration = 5 * time.Second
)

const (
    OSPF_HELLO_MIN_SIZE = 20
    OSPF_HEADER_SIZE = 24
    IP_HEADER_MIN_LEN = 20
    OSPF_PROTO_ID = 89
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

type OSPFServer struct {
    logger                  *syslog.Writer
    ribdClient              RibdClient
    asicdClient             AsicdClient
    portPropertyMap         map[int32]PortProperty
    vlanPropertyMap         map[uint16]VlanProperty
    IPIntfPropertyMap       map[string]IPIntfProperty
    ospfGlobalConf          GlobalConf
    GlobalConfigCh          chan config.GlobalConf
    AreaConfigCh            chan config.AreaConf
    IntfConfigCh            chan config.InterfaceConf
/*
    connRoutesTimer         *time.Timer
    ribSubSocket        *nanomsg.SubSocket
    ribSubSocketCh      chan []byte
    ribSubSocketErrCh   chan error
*/
    asicdSubSocket          *nanomsg.SubSocket
    asicdSubSocketCh        chan []byte
    asicdSubSocketErrCh     chan error
    AreaConfMap             map[AreaConfKey]AreaConf
    IntfConfMap             map[IntfConfKey]IntfConf
}

func NewOSPFServer(logger *syslog.Writer) *OSPFServer {
    ospfServer := &OSPFServer{}
    ospfServer.logger = logger
    ospfServer.GlobalConfigCh = make(chan config.GlobalConf)
    ospfServer.AreaConfigCh = make(chan config.AreaConf)
    ospfServer.IntfConfigCh = make(chan config.InterfaceConf)
    ospfServer.portPropertyMap = make(map[int32]PortProperty)
    ospfServer.vlanPropertyMap = make(map[uint16]VlanProperty)
    ospfServer.AreaConfMap = make(map[AreaConfKey]AreaConf)
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

func computeOspfCheckSum(ospfPkt []byte) (uint16) {
    var csum uint32

    for i := 0; i < len(ospfPkt); i+= 2 {
        csum += uint32(ospfPkt[i]) << 8
        csum += uint32(ospfPkt[i+1])
    }
    ospfChkSum := ^uint16((csum >> 16) + csum)
    return ospfChkSum
}

func (server *OSPFServer)StopSendRecvHelloPkts(intfConfKey IntfConfKey) {
    server.logger.Info("Stop Sending Hello Pkt")
    server.StopSendHelloPkt(intfConfKey)
    //server.logger.Info("Stop Receiving Hello Pkt")
    //ent.HelloPktRecvCh<-false
}

func (server *OSPFServer)StartSendRecvHelloPkts(intfConfKey IntfConfKey) {
    server.logger.Info("Start Sending Hello Pkt")
    server.StartSendHelloPkt(intfConfKey)
    server.logger.Info("Start Receiving Hello Pkt")
    //ent.HelloPktRecvCh<-true
}

func (server *OSPFServer)StopSendHelloPkt(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    if ent.SendHelloPktTimer == nil {
        server.logger.Err("No thread is there to stop.")
        return
    }
    ret := ent.SendHelloPktTimer.Stop()
    if ret == true {
        server.logger.Info("Successfully stopped sending Hello Pkt")
    } else if ret == false {
        server.logger.Info("Unable to stop sending Hello Pkt")
    }
    ent.SendHelloPktTimer = nil
    server.IntfConfMap[key] = ent
}

func (server *OSPFServer)StartSendHelloPkt(key IntfConfKey) {
    ent, _ := server.IntfConfMap[key]
    server.logger.Info(fmt.Sprintln("Started Send Hello Pkt Thread", ent.IfName))
    handle := ent.SendPcapHdl
    ospfHelloPkt := server.BuildHelloPkt(ent)
    helloInterval := time.Duration(ent.IfHelloInterval) * time.Second
    if handle == nil {
        server.logger.Err("Invalid pcap handle")
        return
    }
    SendHelloPktFunc := func() {
        if err := handle.WritePacketData(ospfHelloPkt); err != nil {
            server.logger.Err("Unable to send the hello pkt")
        }
        ent.SendHelloPktTimer.Reset(time.Duration(ent.IfHelloInterval) * time.Second)
    }
    ent.SendHelloPktTimer = time.AfterFunc(helloInterval, SendHelloPktFunc)
    server.IntfConfMap[key] = ent
}

func (server *OSPFServer)StartRecvHelloPkt(key IntfConfKey) {

}

func (server *OSPFServer)InitServer(paramFile string) {
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
    server.logger.Info("Listen for ASICd updates")
    server.listenForASICdUpdates(pluginCommon.PUB_SOCKET_ADDR)
    go server.createASICdSubscriber()

}

func (server *OSPFServer) StartServer(paramFile string) {
    server.InitServer(paramFile)
    for {
        select {
            case gConf := <-server.GlobalConfigCh:
                server.processGlobalConfig(gConf)
            case areaConf := <-server.AreaConfigCh:
                server.logger.Info(fmt.Sprintln("Received call for performing Area Configuration", areaConf))
                server.processAreaConfig(areaConf)
            case ifConf := <-server.IntfConfigCh:
                server.logger.Info(fmt.Sprintln("Received call for performing Intf Configuration", ifConf))
                server.processIntfConfig(ifConf)
            case asicdrxBuf := <-server.asicdSubSocketCh:
                server.processAsicdNotification(asicdrxBuf)
            case <-server.asicdSubSocketErrCh:
                ;
/*
            case ribrxBuf := <-server.ribSubSocketCh:
                server.processRibdNotification(ribdrxBuf)
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
