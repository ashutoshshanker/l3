// hw.go
package vxlan

import (
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"encoding/json"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"net"
	"strconv"
	"time"
	"utils/commonDefs"
	"utils/ipcutils"
)

type VXLANClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	VXLANClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var asicdclnt AsicdClient

// look up the various other daemons based on c string
func GetClientPort(paramsFile string, c string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		//StpLogger("ERROR", fmt.Sprintf("Error in reading configuration file:%s err:%s\n", paramsFile, err))
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		//StpLogger("ERROR", "Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == c {
			return client.Port
		}
	}
	return 0
}

// connect the the asic d
func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {

		for {
			asicdclnt.Address = "localhost:" + strconv.Itoa(port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			//StpLogger("INFO", fmt.Sprintf("found asicd at port %d Transport %#v PrtProtocolFactory %#v\n", port, asicdclnt.Transport, asicdclnt.PtrProtocolFactory))
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				//StpLogger("INFO", "connecting to asicd\n")
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				// lets gather all info needed from asicd such as the port
				//ConstructPortConfigMap()
				break
			} else {
				time.Sleep(time.Millisecond * 500)
			}
		}
	}
}

func (s *VXLANServer) getLoopbackInfo() (success bool, lbname string, mac net.HardwareAddr, ip net.IP) {
	// TODO this logic only assumes one loopback interface.  More logic is needed
	// to handle multiple  loopbacks configured.  The idea should be
	// that the lowest IP address is used.
	more := true
	for more {
		currMarker := asicdServices.Int(0)
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkLogicalIntfState(currMarker, 5)
		if err == nil {
			objCount := int(bulkInfo.Count)
			more = bool(bulkInfo.More)
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				ifindex := bulkInfo.LogicalIntfStateList[i].IfIndex
				lbname = bulkInfo.LogicalIntfStateList[i].Name
				if pluginCommon.GetTypeFromIfIndex(ifindex) == commonDefs.IfTypeLoopback {
					mac, _ = net.ParseMAC(bulkInfo.LogicalIntfStateList[i].SrcMac)
					ipV4ObjMore := true
					ipV4ObjCurrMarker := asicdServices.Int(0)
					for ipV4ObjMore {
						ipV4BulkInfo, _ := asicdclnt.ClientHdl.GetBulkIPv4Intf(ipV4ObjCurrMarker, 20)
						ipV4ObjCount := int(ipV4BulkInfo.Count)
						ipV4ObjCurrMarker = asicdServices.Int(bulkInfo.EndIdx)
						ipV4ObjMore = bool(ipV4BulkInfo.More)
						for j := 0; j < ipV4ObjCount; j++ {
							if ipV4BulkInfo.IPv4IntfList[j].IfIndex == ifindex {
								success = true
								ip = net.ParseIP(ipV4BulkInfo.IPv4IntfList[j].IpAddr)
								return success, lbname, mac, ip
							}
						}
					}
				}
			}
		}
	}
	return success, lbname, mac, ip
}
