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
func (ovsHdl *BGPOvsdbHandler) GetBGPRouterAsn(table ovsdb.TableUpdate) (*BGPOvsRouterInfo, error) {
	var asn uint32
	var id UUID
	// BGP Router ASN is stored in vrf table... not in BGP_Router table
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
			ovsHdl.logger.Info(fmt.Sprintln(bgpRouters))
			for key, value := range bgpRouters {
				asn = uint32(key.(float64))
				id = ovsHdl.getObjUUID(value)
				if id == "" {
					return nil, errors.New("invalid uuid")
				}
				rtrInfo := &BGPOvsRouterInfo{
					asn:  asn,
					uuid: id,
				}
				rtrInfo.routerId = ovsHdl.GetBGPRouterId(rtrInfo.uuid, table)
				return rtrInfo, nil
			}
		}
	}
	return nil, errors.New("no entry found in vrf table")
}

/*
func (ovsHdl *BGPOvsdbHandler) GetRouteInfo(ipAddr string) {

}
*/

/*  Lets get router id for the asn
 */
func (ovsHdl *BGPOvsdbHandler) GetBGPRouterId(rtUuid UUID, table ovsdb.TableUpdate) string {
	rtrId := ""
	ok := false

	for key, value := range table.Rows {
		// sanity check for router uuid
		if sameUUID(rtUuid, key) {
			rtrId, ok = value.New.Fields["router_id"].(string)
			if ok {
				ovsHdl.logger.Info("Router ID is " + rtrId)
				return rtrId
			}
		}
	}

	return rtrId
}

/*  Get bgp neighbor uuids and addrs information from bgp router table
 */
func (ovsHdl *BGPOvsdbHandler) GetBGPNeighInfoFromBgpRouter() ([]net.IP, []UUID, error) {
	var ok bool
	rtrId := ""
	bgpRouterEntries, exists := ovsHdl.cache[OVSDB_BGP_ROUTER_TABLE]
	if !exists {
		return nil, nil, errors.New("There is no bgp router table entry")
	}
	// scan through bgp router table and fetch all the addresses and uuids
	for key, value := range bgpRouterEntries {
		rtrId, ok = value.Fields["router_id"].(string)
		if ok && strings.Compare(rtrId, ovsHdl.routerInfo.routerId) != 0 {
			ovsHdl.logger.Err("Mis match in router id")
			return nil, nil, errors.New("Mismatch in router id")
		}
		if sameUUID(ovsHdl.routerInfo.uuid, key) {
			neighbors := value.Fields[OVSDB_BGP_NEIGHBOR_ENTRIES].(ovsdb.OvsMap).GoMap
			if len(neighbors) < 1 {
				return nil, nil, errors.New("no bgp neighbor configured")
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
					return nil, nil,
						errors.New("uuid schema has error")
				}
				uuids = append(uuids, id)
			}
			return addresses, uuids, nil
		}
	}
	return nil, nil, errors.New("Mis match in bgp router table and ovsdb cached routerInfo")
}

func (ovsHdl *BGPOvsdbHandler) CreateBgpNeighborInfo(addrs []net.IP, uuids []UUID,
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
				// @TODO: remove this neighbor config thrift call once interface for
				// listener side is already implemented
				neighborCfg := &bgpd.BGPNeighbor{
					PeerAS:          int32(newPeerAS),
					LocalAS:         int32(ovsHdl.routerInfo.asn),
					NeighborAddress: newNeighborAddr,
				}
				ovsHdl.logger.Info(fmt.Sprintln("PeerAS",
					newPeerAS))
				ovsHdl.logger.Info(fmt.Sprintln("Neighbor Addr",
					newNeighborAddr))
				newDesc, ok := value.New.Fields["description"].(string)
				if ok {
					ovsHdl.logger.Info(fmt.Sprintln("Description", newDesc))
					neighborCfg.Description = newDesc
				}

				/* Not Support yet from OVS-DB
				newLocalAS, ok := value.New.Fields["local_as"].(ovsdb.OvsSet)
				if ok {
					ovsHdl.logger.Info(fmt.Sprintln("Local AS:", newLocalAS))
				} else {
					// if not configured then we will use cached asn value from
					// routerInfo
					newLocalAS := ovsHdl.routerInfo.asn
				}
				*/

				newAdverInt, ok := value.New.Fields["advertisement_interval"].(float64)
				if ok {
					//@TODO: jgheewala talk with Harsha and figure out what is this
					//interval
					ovsHdl.logger.Info(fmt.Sprintln("Advertisement Interval",
						newAdverInt))
				}
				// CreateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor)
				/*
					type BGPNeighbor struct {
						ConfigObj
						PeerAS                  uint32
						LocalAS                 uint32
						AuthPassword            string
						Description             string
						NeighborAddress         string
						IfIndex                 int32
						RouteReflectorClusterId uint32
						RouteReflectorClient    bool
						MultiHopEnable          bool
						MultiHopTTL             uint8
						ConnectRetryTime        uint32
						HoldTime                uint32
						KeepaliveTime           uint32
						AddPathsRx              bool
						AddPathsMaxTx           uint8
						PeerGroup               string
						BfdEnable               bool
						MaxPrefixes             uint32
						MaxPrefixesThresholdPct uint8
						MaxPrefixesDisconnect   bool
						MaxPrefixesRestartTimer uint8
					}
				*/
				ovsHdl.logger.Info(fmt.Sprintln("OVS Neighbor config for thrift is",
					neighborCfg))
				ovsHdl.rpcHdl.CreateBGPNeighbor(neighborCfg)
			}
		}
	}
}

/*  Creating bgp global flexswitch object using BGP_Router information that was
 *  parse/collected from ovsdb update
 */
func (ovsHdl *BGPOvsdbHandler) CreateBgpGlobalConfig(rtrInfo *BGPOvsRouterInfo) *bgpd.BGPGlobal {
	bgpGlobal := &bgpd.BGPGlobal{
		ASNum:            int32(rtrInfo.asn),
		RouterId:         rtrInfo.routerId,
		UseMultiplePaths: true,
		EBGPMaxPaths:     32,
		IBGPMaxPaths:     32,
	}
	_, err := ovsHdl.rpcHdl.CreateBGPGlobal(bgpGlobal)
	if err != nil {
		if err.Error() == "BGP ASN already configured" {
			return nil
		}
	}
	return bgpGlobal
}

/*  BGP neighbor update in ovsdb... we will update our backend object
 */
func (ovsHdl *BGPOvsdbHandler) HandleBGPNeighborUpd(table ovsdb.TableUpdate) error {
	if ovsHdl.routerInfo.asn == 0 || ovsHdl.routerInfo.routerId == "" {
		ovsHdl.logger.Info("Configure ASN & Router ID before configuring bgp neighbor")
		return errors.New("Configure ASN & Router ID before configuring bgp neighbor")
	}
	// we got all neighbor address and neighbor UUID's
	neighborAddrs, neighborUUIDs, err := ovsHdl.GetBGPNeighInfoFromBgpRouter()
	if err != nil {
		return err
	}
	ovsHdl.CreateBgpNeighborInfo(neighborAddrs, neighborUUIDs, table)
	return nil
}

func (ovsHdl *BGPOvsdbHandler) updatePolicy(info map[interface{}]interface{}, update bool) {
	var rules []string
	for key, _ := range info {
		switch key {
		case "connected":
			rules = append(rules, "connected")
		case "static":
			rules = append(rules, "static")
		case "ospf":
			rules = append(rules, "ospf")
		}
	}
	if update {
		for idx := range rules {
			ovsHdl.policyMgr.AddRedistributePolicy(rules[idx])
		}
	} else {
		for idx := range rules {
			ovsHdl.policyMgr.RemoveRedistributePolicy(rules[idx])
		}
	}
}

func (ovsHdl *BGPOvsdbHandler) findAndRemovePolicy(new, old map[interface{}]interface{}) {
	var rules []string
	for key, _ := range old {
		_, ok := new[key]
		if ok {
			continue // rule is present in both new and old
		}
		switch key {
		case "connected":
			rules = append(rules, "connected")
		case "static":
			rules = append(rules, "static")
		case "ospf":
			rules = append(rules, "ospf")
		}
	}
	for idx := range rules {
		ovsHdl.policyMgr.RemoveRedistributePolicy(rules[idx])
	}
}

func (ovsHdl *BGPOvsdbHandler) handleRedistribute(old, new *map[interface{}]interface{}) {
	if old == nil && new != nil { // first time rule add
		ovsHdl.updatePolicy(*new, true /*meaning add or update*/)
	} else if old != nil && new == nil {
		ovsHdl.updatePolicy(*old, false /*meaning remove all policies*/)
	} else {
		// old is also present and new too

		/*
		 * Case 1) if len of new is greater than len of old then we can send out add
		 *	   policy, for e.g:
		 *	   router bgp 500
		 *		bgp router-id 10.1.10.1
		 *		redistribute connected
		 *	   The above config is already present.. And user wants to add more
		 *	   router bgp 500
		 *		redistribute static
		 *         In this case old ---> map[connected: [uuid]]
		 *			new ---> map[connected: [uuid], static[uuid]]
		 * Case 2) if len of new is less than len of old then we can send out take a diff
		 *	   and remove only the one policy, for e.g:
		 *	   router bgp 500
		 *		bgp router-id 10.1.10.1
		 *		redistribute connected
		 *		redistribute static
		 *	   The above config is already present.. And user removes static from
		 *	   router bgp 500
		 *		no redistribute static
		 *         In this case new ---> map[connected: [uuid]]
		 *			old ---> map[connected: [uuid], static[uuid]]
		 */
		if len(*new) > len(*old) {
			ovsHdl.updatePolicy(*new, true)
		} else {
			ovsHdl.findAndRemovePolicy(*new, *old)
		}
	}
}

func (ovsHdl *BGPOvsdbHandler) checkBgpRouterCfgUpd(uuid UUID, table ovsdb.TableUpdate) {
	ovsHdl.logger.Info("Check for updates")
	for key, value := range table.Rows {
		//ovsHdl.logger.Info(fmt.Sprintln("new value:", value.New))
		//ovsHdl.logger.Info(fmt.Sprintln("old value:", value.Old))
		//ovsHdl.logger.Info(fmt.Sprintln("uuid:", uuid, "key:", key))
		// sanity check for router uuid
		if sameUUID(uuid, key) {
			var old map[interface{}]interface{}
			var new map[interface{}]interface{}

			if value.Old.Fields["redistribute"] != nil {
				new = value.New.Fields["redistribute"].(ovsdb.OvsMap).GoMap
			}
			if value.Old.Fields["redistribute"] != nil {
				old = value.Old.Fields["redistribute"].(ovsdb.OvsMap).GoMap
			}

			if len(old) >= 1 && len(new) >= 1 {
				// means old value had policy and its updated to new
				ovsHdl.handleRedistribute(&old, &new)
			} else if len(new) >= 1 && len(old) < 1 {
				// means first time policy is getting configured
				ovsHdl.handleRedistribute(nil, &new)
			} else if len(old) >= 1 && len(new) < 1 {
				// means delete of all policies
				ovsHdl.handleRedistribute(&old, nil)
			}
		} else {
			ovsHdl.logger.Info("Key is mismatch")
		}
	}
}

func (ovsHdl *BGPOvsdbHandler) HandleBGPRouteUpd(table ovsdb.TableUpdate) error {
	var err error

	if ovsHdl.routerInfo == nil {
		ovsHdl.routerInfo, err = ovsHdl.GetBGPRouterAsn(table)
		if err != nil {
			return err
		}
		ovsHdl.logger.Info(fmt.Sprintln("Got BGP_Router Update asn:",
			ovsHdl.routerInfo.asn, "BGP_Router UUID:", ovsHdl.routerInfo.uuid))
	} else {
		ovsHdl.routerInfo.routerId = ovsHdl.GetBGPRouterId(ovsHdl.routerInfo.uuid, table)
	}
	if ovsHdl.routerInfo.routerId == "" {
		ovsHdl.logger.Info("Waiting for router id to be configured before starting bgp server")
		return nil
	}
	bgpGlobal := ovsHdl.CreateBgpGlobalConfig(ovsHdl.routerInfo)
	if bgpGlobal != nil {
		ovsHdl.logger.Info(fmt.Sprintln(bgpGlobal))
		return nil
	}
	// else check for update
	ovsHdl.checkBgpRouterCfgUpd(ovsHdl.routerInfo.uuid, table)
	return nil
}
