package ovsdbHandler

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
 */
func (svr *BGPOvsdbHandler) getObjUUID(val interface{}) UUID {
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
func (svr *BGPOvsdbHandler) GetBGPRouterInfo() (uint32, UUID, error) {
	var asn uint32
	var id UUID

	vrfs, exists := svr.cache[OVSDB_VRF_TABLE]
	if !exists {
		return asn, id, errors.New("vrf table doesn't exists")
	}

	for _, vrf := range vrfs {
		// check vrf name
		if vrf.Fields["name"] == OVSDB_DEFAULT_VRF {
			// get BGP_Routers Map from the vrf fields
			bgpRouters := vrf.Fields[OVSDB_BGP_ROUTER_ENTRIES].(ovsdb.OvsMap).GoMap
			if len(bgpRouters) < 1 {
				return asn, id, errors.New("no bgp router configured")
			} else if len(bgpRouters) > 1 {
				return asn, id,
					errors.New("Multiple bgp routers configured on vrf_default")
			}
			for key, value := range bgpRouters {
				asn = uint32(key.(float64))
				id = svr.getObjUUID(value)
				if id == "" {
					return asn, id, errors.New("invalid uuid")
				}
				return asn, id, nil
			}
		}
	}

	return asn, id, errors.New("no entry found in vrf table")
}

/*  Get bgp neighbor uuids and addrs information
 */
func (svr *BGPOvsdbHandler) GetBGPNeighborInfo(rtUuid UUID) ([]net.IP,
	[]UUID, error) {
	bgpRouterEntries, exists := svr.cache[OVSDB_BGP_ROUTER_TABLE]
	if !exists {
		return nil, nil, errors.New("There is no bgp router table entry")
	}
	// scan through bgp router table and fetch all the addresses and uuids
	for key, value := range bgpRouterEntries {
		if sameUUID(rtUuid, key) {
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
				id := svr.getObjUUID(value)
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
	return nil, nil, nil
}

/*  BGP neighbor update in ovsdb... we will update our backend object
 */
func (svr *BGPOvsdbHandler) HandleBGPNeighborUpd(table ovsdb.TableUpdate) error {
	asn, bgpRouterUUID, err := svr.GetBGPRouterInfo()
	if err != nil {
		return err
	}
	fmt.Println("asn:", asn, "BGP_Router UUID:", bgpRouterUUID)
	neighborAddrs, neighborUUIDs, err := svr.GetBGPNeighborInfo(bgpRouterUUID)
	fmt.Println("neighborAddrs:", neighborAddrs, "uuid's:", neighborUUIDs)
	return nil
}

func (svr *BGPOvsdbHandler) HandleBGPRouteUpd() error {
	return nil
}
