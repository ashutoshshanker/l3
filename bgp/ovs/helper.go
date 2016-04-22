package ovsMgr

import (
	"bgpd"
	"errors"
	"fmt"
	ovsdb "github.com/socketplane/libovsdb"
	"net"
	"strings"
)

const (
	OVSDB_DEFAULT_VRF          = "vrf_default"
	OVSDB_BGP_ROUTER_TABLE     = "BGP_Router"
	OVSDB_BGP_NEIGHBOR_TABLE   = "BGP_Neighbor"
	OVSDB_VRF_TABLE            = "VRF"
	OVSDB_BGP_NEIGHBOR_ENTRIES = "bgp_neighbors"
	OVSDB_BGP_ROUTER_ENTRIES   = "bgp_routers"
)

type UUID string

type BGPFlexSwitch struct {
	neighbor bgpd.BGPNeighbor
	global   bgpd.BGPGlobal
}

/*  Compare UUID so that we know whether the uuid we got is the same in the table
 *  or not
 */
func sameUUID(src UUID, dst string) bool {
	return (strings.Compare(string(src), dst) == 0)
}

/*  Get object uuid from the map
 *  for e.g:
 *	value: [uuid 4c682c17-8499-4abd-b359-ffaea8f2f79b]
 */
func (ovsHdl *BGPOvsdbHandler) getObjUUID(val interface{}) UUID {
	retVal, exists := val.([]interface{})
	if !exists {
		return ""
	}
	if len(retVal) != 2 || retVal[0].(string) != "uuid" {
		return ""
	}
	return UUID(retVal[1].(string))
}

/*  Lets get asn number for the local bgp and also get the ovsdb BGP_Router uuid
 */
func (ovsHdl *BGPOvsdbHandler) GetBGPRouterInfo() (*BGPOvsRouterInfo, error) { //(uint32, UUID, error) {
	var asn uint32
	var id UUID

	vrfs, exists := ovsHdl.cache[OVSDB_VRF_TABLE]
	if !exists {
		return nil, errors.New("vrf table doesn't exists")
	}

	for _, vrf := range vrfs {
		// check vrf name
		if vrf.Fields["name"] == OVSDB_DEFAULT_VRF {
			// get BGP_Routers Map from the vrf fields
			bgpRouters := vrf.Fields[OVSDB_BGP_ROUTER_ENTRIES].(ovsdb.OvsMap).GoMap
			if len(bgpRouters) < 1 {
				return nil, errors.New("no bgp router configured")
			} else if len(bgpRouters) > 1 {
				return nil, errors.New("Multiple bgp routers " +
					"configured on vrf_default")
			}
			for key, value := range bgpRouters {
				asn = uint32(key.(float64))
				id = ovsHdl.getObjUUID(value)
				if id == "" {
					return nil, errors.New("invalid uuid")
				}
				rtrInfo := &BGPOvsRouterInfo{
					asn:      asn,
					uuid:     id,
					routerId: "", //rtrId,
				}
				return rtrInfo, nil
			}
		}
	}
	return nil, errors.New("no entry found in vrf table")
}

/*  Get bgp neighbor uuids and addrs information
 */
func (ovsHdl *BGPOvsdbHandler) GetBGPNeighborInfo(rtUuid UUID) (string,
	[]net.IP, []UUID, error) {
	rtrId := ""
	var ok bool
	bgpRouterEntries, exists := ovsHdl.cache[OVSDB_BGP_ROUTER_TABLE]
	if !exists {
		return rtrId, nil, nil,
			errors.New("There is no bgp router table entry")
	}
	// scan through bgp router table and fetch all the addresses and uuids
	for key, value := range bgpRouterEntries {
		rtrId, ok = value.Fields["router_id"].(string)
		if ok {
			ovsHdl.logger.Info(fmt.Sprintln("router id", rtrId))
		}
		if sameUUID(rtUuid, key) {
			neighbors := value.Fields[OVSDB_BGP_NEIGHBOR_ENTRIES].(ovsdb.OvsMap).GoMap
			if len(neighbors) < 1 {
				return rtrId, nil, nil, errors.New("no bgp neighbor configured")
			}
			// Create slice of addresses and slice of UUID's which
			// defines all the entries of bgp neighbor in bgp router
			// table
			addresses := make([]net.IP, 0, len(neighbors))
			uuids := make([]UUID, 0, len(neighbors))
			for key, value := range neighbors {
				addresses = append(addresses, net.ParseIP(key.(string)))
				id := ovsHdl.getObjUUID(value)
				if id == "" {
					addresses = nil
					uuids = nil
					return rtrId, nil, nil,
						errors.New("uuid schema has error")
				}
				uuids = append(uuids, id)
			}
			return rtrId, addresses, uuids, nil
		}
	}
	return rtrId, nil, nil, nil
}

func (ovsHdl *BGPOvsdbHandler) DumpBgpNeighborInfo(addrs []net.IP, uuids []UUID,
	table ovsdb.TableUpdate) {
	for key, value := range table.Rows {
		for idx, uuid := range uuids {
			if sameUUID(uuid, key) {
				//ovsHdl.logger.Info(fmt.Sprintln("new value:", value.New))
				//ovsHdl.logger.Info(fmt.Sprintln("old value:", value.Old))
				//ovsHdl.logger.Info(fmt.Sprintln("uuid", uuid, "key uuid", key))
				newPeerAS, ok := value.New.Fields["remote_as"].(float64)
				if !ok {
					ovsHdl.logger.Warning("no asn")
					continue
				}
				newNeighborAddr := addrs[idx].String()
				ovsHdl.logger.Info(fmt.Sprintln("PeerAS",
					newPeerAS))
				ovsHdl.logger.Info(fmt.Sprintln("Neighbor Addr",
					newNeighborAddr))
				newDesc, ok := value.New.Fields["description"].(string)
				if ok {
					ovsHdl.logger.Info(fmt.Sprintln("Description", newDesc))
				}
				newLocalAS, ok := value.New.Fields["local_as"].(ovsdb.OvsSet)
				if ok {
					ovsHdl.logger.Info(fmt.Sprintln("Local AS:", newLocalAS))
				}

				newAdverInt, ok := value.New.Fields["advertisement_interval"].(float64)
				if ok {
					ovsHdl.logger.Info(fmt.Sprintln("Advertisement Interval",
						newAdverInt))
				}

				//neighborInfo := &bgpd.BGPNeighbor{}
			}
		}
	}
}

/*  Creating bgp global flexswitch object using BGP_Router information that was
 *  parse/collected from ovsdb update
 */
func (ovsHdl *BGPOvsdbHandler) CreateBgpGlobalConfig(
	rtrInfo *BGPOvsRouterInfo) *bgpd.BGPGlobal {
	bgpGlobal := &bgpd.BGPGlobal{
		ASNum:            int32(rtrInfo.asn),
		RouterId:         rtrInfo.routerId,
		UseMultiplePaths: true,
		EBGPMaxPaths:     32,
		IBGPMaxPaths:     32,
	}
	ovsHdl.rpcHdl.CreateBGPGlobal(bgpGlobal)
	return bgpGlobal
}

/*  BGP neighbor update in ovsdb... we will update our backend object
 */
func (ovsHdl *BGPOvsdbHandler) HandleBGPNeighborUpd(table ovsdb.TableUpdate) error {
	//asn, bgpRouterUUID, err := ovsHdl.GetBGPRouterInfo()
	routerInfo, err := ovsHdl.GetBGPRouterInfo()
	if err != nil {
		return err
	}
	ovsHdl.logger.Info(fmt.Sprintln("asn:", routerInfo.asn, "BGP_Router UUID:",
		routerInfo.uuid))
	rtrId, neighborAddrs, neighborUUIDs, err := ovsHdl.GetBGPNeighborInfo(routerInfo.uuid)
	if rtrId != "" {
		routerInfo.routerId = rtrId
	}
	if err != nil {
		return err
	}
	ovsHdl.routerInfo = routerInfo
	bgpGlobal := ovsHdl.CreateBgpGlobalConfig(ovsHdl.routerInfo)
	ovsHdl.logger.Info(fmt.Sprintln("neighborAddrs:", neighborAddrs, "uuid's:",
		neighborUUIDs))
	ovsHdl.logger.Info(fmt.Sprintln(bgpGlobal))
	ovsHdl.DumpBgpNeighborInfo(neighborAddrs, neighborUUIDs, table)
	return nil
}

func (ovsHdl *BGPOvsdbHandler) HandleBGPRouteUpd(table ovsdb.TableUpdate) error {
	return nil
}
